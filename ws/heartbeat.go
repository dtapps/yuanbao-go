package ws

import (
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/gorilla/websocket"
)

// startHeartbeat 启动心跳
func (c *WsClient) startHeartbeat() {
	c.log.Info("启动心跳定时器")

	c.stopHeartbeat()

	c.mu.Lock()
	interval := c.heartbeatInterval
	c.mu.Unlock()

	c.heartbeatTimer = time.AfterFunc(interval, func() {
		c.sendHeartbeatMessage()
	})
}

// stopHeartbeatLocked 停止并置空心跳定时器
func (c *WsClient) stopHeartbeatLocked() {
	if c.heartbeatTimer != nil {
		c.heartbeatTimer.Stop()
		c.heartbeatTimer = nil
	}
}

// stopHeartbeat 停止并置空心跳定时器 (加锁)
func (c *WsClient) stopHeartbeat() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopHeartbeatLocked()
}

// sendHeartbeatMessage 发送心跳消息
func (c *WsClient) sendHeartbeatMessage() {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	now := time.Now().UnixMilli()
	c.mu.Lock()
	if now-c.lastHeartbeatAt > int64(c.heartbeatInterval.Milliseconds()*2) {
		c.heartbeatTimeoutCount++
	}
	c.lastHeartbeatAt = now
	c.heartbeatCount++
	c.mu.Unlock()

	// 构建 ping 请求消息
	params := message.BuildPingRequestMessageParams{
		SeqNo: c.generateNextSeqNo(),
	}
	messageID, data, err := message.BuildPingRequestMessage(params)
	if err != nil {
		c.log.Error("构建心跳消息失败", logger.F("error", err.Error()))
		return
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		c.log.Error("发送心跳消息失败", logger.F("error", err.Error()))
		return
	}

	c.log.Info("发送心跳消息成功",
		logger.F("messageID", messageID),
		logger.F("data", string(data)),
	)

	c.mu.Lock()
	c.heartbeatTimer = time.AfterFunc(c.heartbeatInterval, func() {
		c.sendHeartbeatMessage()
	})
	c.mu.Unlock()
}
