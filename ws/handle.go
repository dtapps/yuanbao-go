package ws

import (
	"fmt"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/dtapps/yuanbao-go/types"
	bizProto "github.com/dtapps/yuanbao-go/wsproto/biz"
	connProto "github.com/dtapps/yuanbao-go/wsproto/conn"
	"google.golang.org/protobuf/proto"
)

// handleClose 处理关闭
func (c *WsClient) handleClose(code int, reason string) {
	c.log.Info("处理关闭",
		logger.F("code", code),
		logger.F("reason", reason),
	)

	c.mu.Lock()
	wasConnected := c.state == types.ConnectionStateConnected.String()
	c.mu.Unlock()

	c.stopHeartbeat()

	c.close()

	c.mu.Lock()
	c.state = types.ConnectionStateDisconnected.String()
	c.mu.Unlock()

	if c.callback != nil {
		c.callback.OnClose(code, reason)
	}

	// 安排重连
	// 1000 = 正常关闭，不需要重连
	if wasConnected && code != 1000 {
		c.ScheduleReconnect()
	}
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
		case "sync_information":
			// 同步信息响应
			c.handleResponseTypeSyncInformation(connMsg)
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
		c.log.Warn("未处理的cmdType",
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
	c.log.Debug("认证绑定响应",
		logger.F("authBind", authBind),
	)

	// 状态码 (status)，通常用于表示连接层是否成功
	status := connMsg.Head.Status

	// 业务错误码 (0 代表 SUCCESS)
	code := authBind.Code

	// 检查错误码
	if status != 0 && status != int32(types.RetCodeAlreadyAuth) {
		c.log.Error("认证失败",
			logger.F("status", status),
			logger.F("code", code),
		)

		// 检查是否是 token 过期，需要刷新 token
		if c.shouldRefreshToken(int(status)) && c.callback != nil {
			c.log.Warn("Token 过期，尝试刷新",
				logger.F("status", status),
			)
			c.close()
			newAuth, err := c.callback.OnAuthFailed(int(status))
			if err != nil {
				c.log.Error("刷新 Token 失败", logger.F("error", err))
				c.callback.OnError(fmt.Errorf("auth failed and refresh token failed: status=%d, err=%v", status, err))
				return
			}
			if newAuth != nil {
				c.log.Info("Token 刷新成功，重新连接")
				c.auth = newAuth
				c.ScheduleReconnect()
			}
			return
		}

		c.close()
		if c.callback != nil {
			c.callback.OnError(fmt.Errorf("auth failed: status=%d, code=%d", status, code))
		}
		return
	}

	// 检查业务层错误码
	if code != 0 && code != int32(types.RetCodeAlreadyAuth) {
		c.log.Error("认证业务失败",
			logger.F("code", code),
			logger.F("message", authBind.Message),
		)

		// 检查是否是 token 过期
		if c.shouldRefreshToken(int(code)) && c.callback != nil {
			c.log.Warn("Token 过期，尝试刷新",
				logger.F("code", code),
			)
			c.close()
			newAuth, err := c.callback.OnAuthFailed(int(code))
			if err != nil {
				c.log.Error("刷新 Token 失败", logger.F("error", err))
				c.callback.OnError(fmt.Errorf("auth failed and refresh token failed: code=%d, err=%v", code, err))
				return
			}
			if newAuth != nil {
				c.log.Info("Token 刷新成功，重新连接")
				c.auth = newAuth
				c.ScheduleReconnect()
			}
			return
		}

		c.close()
		if c.callback != nil {
			c.callback.OnError(fmt.Errorf("auth failed: code=%d, message=%s", code, authBind.Message))
		}
		return
	}

	// 认证成功
	var connectID string
	c.mu.Lock()
	c.connectID = authBind.ConnectId
	connectID = c.connectID // 在锁内读取，避免竞态
	c.state = types.ConnectionStateConnected.String()
	c.reconnectAttempts = 0
	c.mu.Unlock()

	// 启动心跳
	c.startHeartbeat()

	// 回调
	if c.callback != nil {
		result := &types.OnReadyData{
			ConnectID: connectID,
			Timestamp: time.Now().Unix(),
		}
		c.callback.OnReady(result)
		c.callback.OnStateChange(types.ConnectionStateConnected.String())
	}
}

// handleResponseTypePing 处理响应Ping类型
func (c *WsClient) handleResponseTypePing(connMsg *connProto.ConnMsg) {
	c.mu.Lock()
	c.heartbeatAckReceived = true
	c.heartbeatTimeoutCount = 0
	c.lastHeartbeatAt = time.Now().UnixMilli()
	c.mu.Unlock()

	// 解析 ping 消息
	ping, err := message.ParsePingMessage(connMsg.Data)
	if err != nil {
		c.log.Error("解析消息失败", logger.F("error", err))
		return
	}
	c.log.Debug("Ping响应",
		logger.F("ping", ping),
	)
}

// handleResponseTypeSyncInformation 处理同步信息响应
func (c *WsClient) handleResponseTypeSyncInformation(connMsg *connProto.ConnMsg) {
	// 解析 SyncInformation 响应
	rsp := &bizProto.SyncInformationRsp{}
	err := proto.Unmarshal(connMsg.Data, rsp)
	if err != nil {
		c.log.Error("解析 SyncInformation 响应失败", logger.F("error", err))
		return
	}

	c.log.Info("[同步] SyncInformation 响应",
		logger.F("code", rsp.Code),
		logger.F("msg", rsp.Msg),
	)

	if rsp.Code != 0 {
		c.log.Warn("[同步] 返回非零码",
			logger.F("code", rsp.Code),
			logger.F("msg", rsp.Msg),
		)
	} else {
		c.log.Info("[同步] 命令同步成功")
	}
}

// handleResponseTypeInboundMessage 处理消息推送
func (c *WsClient) handleResponseTypeInboundMessage(connMsg *connProto.ConnMsg) {

	// 解析 inbound_message 消息
	inboundMessage, err := message.ParseInboundMessageMessage(connMsg.Data)
	if err != nil {
		c.log.Error("解析 inbound_message 消息失败", logger.F("error", err))
		return
	}
	c.log.Debug("消息推送",
		logger.F("inboundMessage", inboundMessage),
	)

	if c.callback != nil {
		c.callback.OnDispatch(inboundMessage)
	}
}
