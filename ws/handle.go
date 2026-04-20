package ws

import (
	"fmt"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/dtapps/yuanbao-go/types"
	connProto "github.com/dtapps/yuanbao-go/wsproto/conn"
	"google.golang.org/protobuf/proto"
)

// handleClose 处理关闭
func (c *WsClient) handleClose(code int, reason string) {
	c.log.Info("处理关闭",
		logger.F("code", code),
		logger.F("reason", reason),
	)

	// c.mu.Lock()
	// wasConnected := c.state == types.ConnectionStateConnected.String()
	// c.mu.Unlock()

	c.stopHeartbeat()

	c.close()

	c.mu.Lock()
	c.state = types.ConnectionStateDisconnected.String()
	c.mu.Unlock()

	if c.callback != nil {
		c.callback.OnClose(code, reason)
	}

	// 安排重连
	// code=1008 reason=Invalid or expired token
	// if wasConnected || code != 1008 {
	// 	c.ScheduleReconnect()
	// }
}

// handleMessage 处理消息
func (c *WsClient) handleMessage(data []byte) {
	c.mu.Lock()
	c.heartbeatAckReceived = true
	c.heartbeatTimeoutCount = 0
	c.mu.Unlock()

	// Protobuf 解析
	connMsg := &connProto.ConnMsg{}
	err := proto.Unmarshal(data, connMsg)
	if err != nil {
		c.log.Error("解析消息失败", logger.F("error", err.Error()))
		return
	}

	switch connMsg.Head.CmdType {
	case 1:
		// 响应
		switch connMsg.Head.Cmd {
		case "auth-bind":
			// 认证绑定
			c.handleResponseTypeAuthBind(connMsg)
		case "ping":
			// Ping
			c.handleResponseTypePing(connMsg)
		case "send_c2c_message":
			// 发送私消息
		case "send_group_message":
			// 发送群消息
		}
	case 2:
		// 推送
		if connMsg.Head.NeedAck {
			// 发送ACK
			c.sendPushAckMessage(connMsg)
		}
		switch connMsg.Head.Cmd {
		case "inbound_message":
			// 消息
			c.handleResponseTypeInboundMessage(connMsg)
			return
		}
	default:
		c.log.Debug("未处理的cmdType",
			logger.F("cmd", connMsg.Head.Cmd),
			logger.F("cmdType", connMsg.Head.CmdType),
		)
	}
}

// handleResponseTypeAuthBind 处理响应认证绑定类型
func (c *WsClient) handleResponseTypeAuthBind(connMsg *connProto.ConnMsg) {

	// Protobuf 解析
	authBind := &connProto.AuthBindRsp{}
	err := proto.Unmarshal(connMsg.Data, authBind)
	if err != nil {
		c.log.Error("解析消息失败", logger.F("error", err))
		return
	}

	// 状态码 (status)，通常用于表示连接层是否成功
	status := connMsg.Head.Status

	// 业务错误码 (0 代表 SUCCESS)
	code := authBind.Code

	// 检查错误码
	if status != 0 && status != 41101 {
		// 41101 = ALREADY_AUTH
		c.log.Error("认证失败",
			logger.F("status", status),
			logger.F("code", code),
		)

		if authBind != nil && c.shouldRefreshToken(int(code)) {
			// 刷新token并重连
			c.mu.Lock()
			c.state = types.ConnectionStateReconnecting.String()
			c.mu.Unlock()

			// go c.refreshTokenAndReconnect()
			return
		}

		c.close()
		if c.callback != nil {
			c.callback.OnError(fmt.Errorf("auth failed: status=%d, code=%d", status, code))
		}
		return
	}

	// 认证成功
	c.mu.Lock()
	c.connectID = authBind.ConnectId
	c.state = types.ConnectionStateConnected.String()
	c.reconnectAttempts = 0
	c.mu.Unlock()

	// 启动心跳
	c.startHeartbeat()

	// 回调
	if c.callback != nil {
		result := &types.OnReadyData{
			ConnectID: c.connectID,
			Timestamp: time.Now().Unix(),
		}
		result.Timestamp = int64(authBind.Timestamp)
		c.callback.OnReady(result)
		c.callback.OnStateChange(types.ConnectionStateConnected.String())
	}
}

// handleResponseTypePing 处理响应Ping类型
func (c *WsClient) handleResponseTypePing(connMsg *connProto.ConnMsg) {
	c.mu.Lock()
	c.heartbeatAckReceived = true
	c.heartbeatTimeoutCount = 0
	c.mu.Unlock()

	// 解析 ping 消息
	_, err := message.ParsePingMessage(connMsg.Data)
	if err != nil {
		c.log.Error("解析消息失败", logger.F("error", err))
		return
	}

	c.lastHeartbeatAt = time.Now().UnixMilli()
}

// handleResponseTypeInboundMessage 处理业务响应
func (c *WsClient) handleResponseTypeInboundMessage(connMsg *connProto.ConnMsg) {

	// 解析 inbound_message 消息
	inboundMessage, err := message.ParseInboundMessageMessage(connMsg.Data)
	if err != nil {
		c.log.Error("解析 inbound_message 消息失败", logger.F("error", err))
		return
	}

	c.log.Debug("解析 inbound_message 消息成功",
		logger.F("inboundMessage", inboundMessage),
	)

	if c.callback != nil {
		c.callback.OnDispatch(inboundMessage)
	}
}
