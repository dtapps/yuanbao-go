package ws

import (
	"fmt"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/dtapps/yuanbao-go/types"
	connProto "github.com/dtapps/yuanbao-go/wsproto/conn"
	"github.com/gorilla/websocket"
)

// SendC2CMessage 发送C2C消息（有序队列模式）
// 无论多少个 goroutine 并发调用，消息都严格按入队顺序发出。
func (c *WsClient) SendC2CMessage(toAccount string, text string) (string, error) {
	// 确保 sender 协程已启动（懒加载，只启动一次）
	c.startSender()

	// 构建发送任务（闭包捕获参数）
	task := sendTask{
		execute: func() (string, error) {
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return "", fmt.Errorf("连接未建立")
			}

			params := message.BuildInboundMessageC2CMessageParams{
				SeqNo:       c.generateNextSeqNo(),
				ToAccount:   toAccount,
				FromAccount: c.botID,
				MsgSeq:      c.generateNextMsgSeq(),
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
				logger.F("msgSeq", params.MsgSeq),
			)

			return messageID, nil
		},
		result: make(chan sendResult, 1),
	}

	// 入队（快速非阻塞，因为有缓冲）
	select {
	case c.sendQueue <- task:
		// 入队成功，等待结果
	case <-c.ctx.Done():
		return "", fmt.Errorf("客户端已关闭")
	}

	// 阻塞等待发送完成并获取结果
	result := <-task.result
	return result.msgID, result.err
}

// SendGroupMessage 发送群消息（有序队列模式）
// 无论多少个 goroutine 并发调用，消息都严格按入队顺序发出。
func (c *WsClient) SendGroupMessage(groupCode string, text string, atList []types.AtInfo) (string, error) {
	// 确保 sender 协程已启动
	c.startSender()

	// 构建发送任务
	task := sendTask{
		execute: func() (string, error) {
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return "", fmt.Errorf("连接未建立")
			}

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
			)

			return messageID, nil
		},
		result: make(chan sendResult, 1),
	}

	// 入队
	select {
	case c.sendQueue <- task:
	case <-c.ctx.Done():
		return "", fmt.Errorf("客户端已关闭")
	}

	// 等待结果
	result := <-task.result
	return result.msgID, result.err
}

// sendAuthBindMessage 发送认证绑定消息
func (c *WsClient) sendAuthBindMessage() {
	// 使用 mu 保护 conn 引用（TOCTOU），但不使用 sendMu 避免阻塞心跳
	c.mu.Lock()
	conn := c.conn
	auth := c.auth
	c.mu.Unlock()

	if conn == nil || auth == nil {
		if auth == nil {
			c.log.Error("认证信息为空")
		}
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
	// 使用 mu 保护 conn 引用（TOCTOU），但不使用 sendMu 避免阻塞业务消息
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

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
