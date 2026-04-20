package ws

import (
	"fmt"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/dtapps/yuanbao-go/types"
	connProto "github.com/dtapps/yuanbao-go/wsproto/conn"
	"github.com/gorilla/websocket"
)

// SendC2CMessage 发送C2C消息
func (c *WsClient) SendC2CMessage(toAccount string, text string) (string, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return "", fmt.Errorf("连接未建立")
	}

	// 构建 inbound_message C2C 消息
	params := message.BuildInboundMessageC2CMessageParams{
		SeqNo:       c.generateNextSeqNo(),
		ToAccount:   toAccount,
		FromAccount: c.botID,
		MsgSeq:      0,
		Text:        text,
	}
	messageID, data, err := message.BuildInboundMessageC2CMessage(params)
	if err != nil {
		return "", fmt.Errorf("构建C2C消息失败: %w", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return "", fmt.Errorf("发送C2C消息失败: %w", err)
	}

	c.log.Debug("发送C2C消息成功",
		logger.F("messageID", messageID),
		logger.F("data", string(data)),
	)

	return messageID, nil
}

// SendGroupMessage 发送群消息
func (c *WsClient) SendGroupMessage(groupCode string, text string, atList []types.AtInfo) (string, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return "", fmt.Errorf("连接未建立")
	}

	// 构建 inbound_message Group 消息
	params := message.BuildInboundMessageGroupMessageParams{
		SeqNo:       c.generateNextSeqNo(),
		GroupCode:   groupCode,
		FromAccount: c.botID,
		Text:        text,
		AtList:      atList,
	}
	messageID, data, err := message.BuildInboundMessageGroupMessage(params)
	if err != nil {
		return "", fmt.Errorf("构建群消息失败: %w", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return "", fmt.Errorf("发送群消息失败: %w", err)
	}

	c.log.Debug("发送群消息成功",
		logger.F("messageID", messageID),
		logger.F("data", string(data)),
	)

	return messageID, nil
}

// sendAuthBindMessage 发送认证绑定消息
func (c *WsClient) sendAuthBindMessage() {
	c.mu.RLock()
	conn := c.conn
	auth := c.auth
	c.mu.RUnlock()

	if auth == nil {
		c.log.Error("认证信息为空")
		return
	}

	// 构建 auth-bind 请求消息
	params := message.BuildAuthBindRequestMessageParams{
		SeqNo:    c.generateNextSeqNo(),
		BizID:    auth.BizID,
		UID:      auth.BotID,
		Source:   auth.Source,
		Token:    auth.Token,
		RouteEnv: auth.RouteEnv,
	}
	messageID, data, err := message.BuildAuthBindRequestMessage(params)
	if err != nil {
		c.log.Error("构建认证绑定消息失败", logger.F("error", err))
		return
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		c.log.Error("发送认证绑定消息失败", logger.F("error", err))
		return
	}

	c.log.Info("发送认证绑定消息成功",
		logger.F("messageID", messageID),
		logger.F("data", string(data)),
	)
}

// sendPushAckMessage 发送ACK消息
func (c *WsClient) sendPushAckMessage(connMsg *connProto.ConnMsg) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	// 构建 PushAck 消息
	params := message.BuildPushAckMessageParams{
		SeqNo: c.generateNextSeqNo(),
	}
	msg, err := message.BuildPushAckMessage(connMsg.Head, params)
	if err != nil {
		c.log.Error("构建PushAck消息失败", logger.F("error", err))
		return
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
		c.log.Error("发送PushAck消息失败", logger.F("error", err))
		return
	}

	c.log.Debug("发送PushACK消息成功",
		logger.F("seqNo", params.SeqNo),
		logger.F("messageID", connMsg.Head.MsgId),
	)
}
