package ws

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/proto"
	"github.com/dtapps/yuanbao-go/types"
	goproto "github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	// 默认心跳间隔(秒)
	DefaultHeartbeatInterval = 5
	// 默认心跳超时次数阈值
	HeartbeatTimeoutThreshold = 2
	// 默认发送超时(毫秒)
	DefaultSendTimeoutMs = 30000
	// 默认最大重连次数
	DefaultMaxReconnectAttempts = 100
	// 默认重连延迟
	DefaultReconnectDelays = "1s,2s,5s,10s,30s,60s"
	// 不重连的关闭码
	InstanceId = "16"
)

// WsClientCallback WebSocket客户端回调接口
type WsClientCallback interface {
	OnReady(data *types.AuthReadyData)
	OnDispatch(pushEvent *PushEvent)
	OnStateChange(state string)
	OnError(err error)
	OnClose(code int, reason string)
	OnKickout(data *types.KickoutMsg)
	OnAuthFailed(code int) (*types.WsAuth, error)
}

// PushEvent 推送事件
type PushEvent struct {
	Cmd      string `json:"cmd,omitempty"`
	Module   string `json:"module,omitempty"`
	MsgID    string `json:"msgId,omitempty"`
	Type     uint32 `json:"type,omitempty"`
	Content  string `json:"content,omitempty"`
	RawData  []byte `json:"rawData,omitempty"`
	ConnData []byte `json:"connData,omitempty"`
}

// WsClient WebSocket客户端
type WsClient struct {
	mu        sync.RWMutex
	conn      *websocket.Conn
	url       string
	state     string
	auth      *types.WsAuth
	accountId string
	botId     string

	// 心跳相关
	heartbeatInterval     int
	heartbeatTimer        *time.Timer
	heartbeatAckReceived  bool
	lastHeartbeatAt       int64
	heartbeatTimeoutCount int

	// 重连相关
	reconnectAttempts    int
	maxReconnectAttempts int
	reconnectDelays      []time.Duration
	reconnectTimer       *time.Timer

	// 待响应的请求
	pendingRequests map[string]*PendingRequest

	// 回调
	callback WsClientCallback

	// 日志
	log *logger.Logger

	// 序列号
	seqNo uint32

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 连接ID
	connectId string

	lastHeartbeatAck time.Time
}

// PendingRequest 待响应的请求
type PendingRequest struct {
	resolveCh chan any
	timeout   time.Duration
	decoder   func(data []byte, msgId string) any
	Resolve   func(any)
	Reject    func(error)
	Timer     *time.Timer
	Decoder   func([]byte, string) any
}

// NewWsClient 创建WebSocket客户端
func NewWsClient(url string, accountId string, botId string, callback WsClientCallback) *WsClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WsClient{
		url:                  url,
		accountId:            accountId,
		botId:                botId,
		state:                "disconnected",
		heartbeatInterval:    DefaultHeartbeatInterval,
		maxReconnectAttempts: DefaultMaxReconnectAttempts,
		reconnectDelays:      parseDelays(DefaultReconnectDelays),
		pendingRequests:      make(map[string]*PendingRequest),
		callback:             callback,
		log:                  logger.New("ws"),
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// SetAuth 设置认证信息
func (c *WsClient) SetAuth(auth *types.WsAuth) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.auth = auth
}

// SetReconnectConfig 设置重连配置
func (c *WsClient) SetReconnectConfig(maxAttempts int, delays string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxReconnectAttempts = maxAttempts
	c.reconnectDelays = parseDelays(delays)
}

// Connect 连接
func (c *WsClient) Connect() error {
	c.mu.Lock()
	if c.state == "disconnected" {
		c.state = "connecting"
	}
	c.mu.Unlock()

	return c.doConnect()
}

// doConnect 执行连接
func (c *WsClient) doConnect() error {
	c.log.Info("正在连接WebSocket", logger.F("url", c.url))

	// 创建WebSocket连接
	header := http.Header{}
	header.Set("Origin", "https://yuanbao.tencent.com")

	conn, resp, err := websocket.DefaultDialer.Dial(c.url, header)
	if err != nil {
		c.log.Error("WebSocket连接失败", logger.F("error", err.Error()))
		c.scheduleReconnect()
		return err
	}

	if resp != nil {
		resp.Body.Close()
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// 设置处理器
	conn.SetCloseHandler(func(code int, text string) error {
		c.handleClose(code, text)
		return nil
	})

	// 启动读取协程
	go c.readLoop()

	// 发送认证
	c.sendAuthBind()

	return nil
}

// readLoop 读取循环
func (c *WsClient) readLoop() {
	defer func() {
		// 连接断开时触发重连
		c.handleClose(1006, "read loop exit")
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			_, data, err := conn.ReadMessage()
			if err != nil {
				// 忽略 "use of closed" 错误，这是正常关闭
				if !strings.Contains(err.Error(), "use of closed") {
					c.log.Error("读取消息失败", logger.F("error", err.Error()))
				}
				return
			}

			// 调试：打印原始消息
			c.log.Debug("收到原始消息", logger.F("length", len(data)))

			c.handleMessage(data)
		}
	}
}

// handleMessage 处理消息
func (c *WsClient) handleMessage(data []byte) {
	c.mu.Lock()
	c.heartbeatAckReceived = true
	c.heartbeatTimeoutCount = 0
	c.mu.Unlock()

	connMsg, err := c.decodeConnMsg(data)
	if err != nil {
		c.log.Error("解析ConnMsg失败", logger.F("error", err.Error()))
		return
	}

	if connMsg.Head == nil {
		c.log.Warn("ConnMsg head为空")
		return
	}

	cmdType := connMsg.Head.CmdType
	c.log.Info("收到消息", logger.F("cmdType", cmdType), logger.F("cmd", connMsg.Head.Cmd), logger.F("msgId", connMsg.Head.MsgId))

	switch cmdType {
	case 1: // Response
		c.handleResponse(connMsg)
	case 2: // Push
		c.handlePush(connMsg)
	default:
		c.log.Debug("未处理的cmdType", logger.F("cmdType", cmdType))
	}
}

// decodeConnMsg 解码ConnMsg
func (c *WsClient) decodeConnMsg(data []byte) (*proto.ConnMsgWrapper, error) {
	msg, err := proto.DecodePB("trpc.yuanbao.conn_common.ConnMsg", data)
	if err != nil {
		return nil, err
	}

	connMsg, ok := msg.(*proto.ConnMsgWrapper)
	if !ok {
		return nil, fmt.Errorf("类型断言失败")
	}

	return connMsg, nil
}

// handleResponse 处理响应
func (c *WsClient) handleResponse(connMsg *proto.ConnMsgWrapper) {
	cmd := connMsg.Head.Cmd

	switch cmd {
	case "auth-bind":
		c.handleAuthBindResponse(connMsg)
	case "ping":
		c.handlePingResponse(connMsg)
	default:
		// 业务响应
		c.handleBusinessResponse(connMsg)
	}
}

// handlePush 处理推送
func (c *WsClient) handlePush(connMsg *proto.ConnMsgWrapper) {
	data := connMsg.Data

	// 发送ACK
	if connMsg.Head.NeedAck {
		c.sendPushAck(connMsg.Head)
	}

	// 尝试解析为KickoutMsg
	kickout, err := c.decodePB("trpc.yuanbao.conn_common.KickoutMsg", data)
	if err == nil && kickout != nil {
		if km, ok := kickout.(*proto.KickoutMsgWrapper); ok {
			c.log.Warn("被踢下线", logger.F("status", km.Status))
			if c.callback != nil {
				c.callback.OnKickout(&types.KickoutMsg{
					Status:          int32(km.Status),
					Reason:          km.Reason,
					OtherDeviceName: km.OtherDeviceName,
				})
			}
			return
		}
	}

	// 尝试解析为PushMsg
	pushMsg, err := c.decodePB("trpc.yuanbao.conn_common.PushMsg", data)
	if err == nil && pushMsg != nil {
		if pm, ok := pushMsg.(*proto.PushMsgWrapper); ok {
			event := &PushEvent{
				Cmd:      pm.Cmd,
				Module:   pm.Module,
				MsgID:    pm.MsgId,
				ConnData: pm.Data,
			}
			if c.callback != nil {
				c.callback.OnDispatch(event)
			}
			return
		}
	}

	// 尝试解析为DirectedPush
	directed, err := c.decodePB("trpc.yuanbao.conn_common.DirectedPush", data)
	if err == nil && directed != nil {
		if dp, ok := directed.(*proto.DirectedPushWrapper); ok {
			event := &PushEvent{
				Type:    dp.Type,
				Content: dp.Content,
				Cmd:     connMsg.Head.Cmd,
				Module:  connMsg.Head.Module,
				MsgID:   connMsg.Head.MsgId,
			}
			if c.callback != nil {
				c.callback.OnDispatch(event)
			}
			return
		}
	}

	// 通用推送
	event := &PushEvent{
		Cmd:     connMsg.Head.Cmd,
		Module:  connMsg.Head.Module,
		MsgID:   connMsg.Head.MsgId,
		RawData: data,
	}
	if c.callback != nil {
		c.callback.OnDispatch(event)
	}
}

// sendPushAck 发送推送ACK
func (c *WsClient) sendPushAck(head *proto.HeadWrapper) {
	ack := &proto.ConnMsgWrapper{
		Head: &proto.HeadWrapper{
			CmdType: 3, // PushAck
			Cmd:     head.Cmd,
			SeqNo:   c.nextSeqNo(),
			MsgId:   head.MsgId,
			Module:  head.Module,
		},
	}

	data, err := proto.EncodePB(ack)
	if err != nil {
		c.log.Error("编码PushAck失败", logger.F("error", err.Error()))
		return
	}

	c.sendBinary(data)
}

// handleAuthBindResponse 处理认证绑定响应
func (c *WsClient) handleAuthBindResponse(connMsg *proto.ConnMsgWrapper) {
	var rsp *proto.AuthBindRspWrapper

	if len(connMsg.Data) > 0 {
		decoded, err := c.decodePB("trpc.yuanbao.conn_common.AuthBindRsp", connMsg.Data)
		if err != nil {
			c.log.Error("解码AuthBindRsp失败", logger.F("error", err.Error()))
			return
		}
		var ok bool
		rsp, ok = decoded.(*proto.AuthBindRspWrapper)
		if !ok {
			c.log.Error("AuthBindRsp类型断言失败")
			return
		}
	}

	status := connMsg.Head.Status
	code := int32(0)
	if rsp != nil {
		code = rsp.Code
	}

	// 检查错误码
	if status != 0 && status != 41101 { // 41101 = ALREADY_AUTH
		c.log.Error("认证失败", map[string]any{"status": status, "code": code})

		if rsp != nil && c.shouldRefreshToken(int(code)) {
			// 刷新token并重连
			c.mu.Lock()
			c.state = "reconnecting"
			c.mu.Unlock()

			go c.refreshTokenAndReconnect()
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
	c.connectId = ""
	if rsp != nil {
		c.connectId = rsp.ConnectId
	}
	c.state = "connected"
	c.reconnectAttempts = 0
	c.mu.Unlock()

	c.log.Info("认证成功", logger.F("connectId", c.connectId))

	// 启动心跳
	c.startHeartbeat(true)

	// 回调
	if c.callback != nil {
		result := &types.AuthReadyData{
			ConnectId: c.connectId,
			Timestamp: time.Now().Unix(),
		}
		if rsp != nil {
			result.Timestamp = rsp.Timestamp
		}
		c.callback.OnReady(result)
	}

	if c.callback != nil {
		c.callback.OnStateChange("connected")
	}
}

// handlePingResponse 处理Ping响应
func (c *WsClient) handlePingResponse(connMsg *proto.ConnMsgWrapper) {
	c.mu.Lock()
	c.heartbeatAckReceived = true
	c.heartbeatTimeoutCount = 0
	c.mu.Unlock()

	var interval uint32 = DefaultHeartbeatInterval
	if len(connMsg.Data) > 0 {
		rsp, err := c.decodePB("trpc.yuanbao.conn_common.PingRsp", connMsg.Data)
		if err == nil {
			if pr, ok := rsp.(*proto.PingRspWrapper); ok && pr.HeartInterval > 1 {
				interval = pr.HeartInterval
			}
		}
	}

	c.mu.Lock()
	c.heartbeatInterval = int(interval)
	c.mu.Unlock()

	c.startHeartbeat(false)
}

// handleBusinessResponse 处理业务响应
func (c *WsClient) handleBusinessResponse(connMsg *proto.ConnMsgWrapper) {
	msgId := connMsg.Head.MsgId
	if msgId == "" {
		c.log.Debug("handleBusinessResponse: msgId为空，跳过")
		return
	}

	c.log.Info("收到业务响应", logger.F("msgId", msgId), logger.F("cmd", connMsg.Head.Cmd), logger.F("dataLen", len(connMsg.Data)), logger.F("status", connMsg.Head.Status))

	c.mu.Lock()
	pending, exists := c.pendingRequests[msgId]
	c.mu.Unlock()

	c.log.Info("查找pending请求", logger.F("msgId", msgId), logger.F("exists", exists), logger.F("pending", pending))

	if !exists {
		c.log.Warn("收到无匹配的业务回包", logger.F("msgId", msgId), logger.F("cmd", connMsg.Head.Cmd))
		return
	}

	if pending.Timer != nil {
		pending.Timer.Stop()
	}

	var result any
	if len(connMsg.Data) > 0 {
		decoder := pending.Decoder
		if decoder == nil {
			decoder = defaultMessageDecoder
		}
		result = decoder(connMsg.Data, msgId)
	} else {
		result = map[string]any{
			"msgId":  msgId,
			"code":   connMsg.Head.Status,
			"status": connMsg.Head.Status,
		}
	}

	pending.Resolve(result)
	c.log.Info("已调用Resolve", logger.F("msgId", msgId))
}

// sendAuthBind 发送认证绑定
func (c *WsClient) sendAuthBind() {
	c.mu.RLock()
	auth := c.auth
	c.mu.RUnlock()

	if auth == nil {
		c.log.Error("认证信息为空")
		return
	}

	msgId := c.generateMsgId()

	authReq := &proto.AuthBindReqWrapper{
		BizId: auth.BizID,
		AuthInfo: &proto.AuthInfoWrapper{
			Uid:    auth.UID,
			Source: auth.Source,
			Token:  auth.Token,
		},
		DeviceInfo: &proto.DeviceInfoWrapper{
			AppVersion:         "1.0.0",
			AppOperationSystem: "Go",
			InstanceId:         InstanceId,
			BotVersion:         "1.0.0",
		},
	}

	if auth.RouteEnv != "" {
		authReq.EnvName = auth.RouteEnv
	}

	data, err := proto.EncodePB(authReq)
	if err != nil {
		c.log.Error("编码AuthBindReq失败", logger.F("error", err.Error()))
		return
	}

	connMsg := &proto.ConnMsgWrapper{
		Head: &proto.HeadWrapper{
			CmdType: 0, // Request
			Cmd:     "auth-bind",
			SeqNo:   c.nextSeqNo(),
			MsgId:   msgId,
			Module:  "conn_access",
		},
		Data: data,
	}

	msgData, err := proto.EncodePB(connMsg)
	if err != nil {
		c.log.Error("编码ConnMsg失败", logger.F("error", err.Error()))
		return
	}

	c.sendBinary(msgData)
}

// sendPing 发送Ping
func (c *WsClient) sendPing() {
	c.mu.Lock()
	if !c.heartbeatAckReceived {
		c.heartbeatTimeoutCount++
		if c.heartbeatTimeoutCount >= HeartbeatTimeoutThreshold {
			c.log.Warn("心跳连续超时，触发重连", logger.F("count", c.heartbeatTimeoutCount))
			if c.conn != nil {
				c.conn.Close()
			}
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
		c.scheduleNextPingCheck()
		return
	}
	c.mu.Unlock()

	msgId := c.generateMsgId()

	pingReq := &proto.PingReqWrapper{}
	data, err := proto.EncodePB(pingReq)
	if err != nil {
		c.log.Error("编码PingReq失败", logger.F("error", err.Error()))
		return
	}

	connMsg := &proto.ConnMsgWrapper{
		Head: &proto.HeadWrapper{
			CmdType: 0,
			Cmd:     "ping",
			SeqNo:   c.nextSeqNo(),
			MsgId:   msgId,
			Module:  "conn_access",
		},
		Data: data,
	}

	msgData, err := proto.EncodePB(connMsg)
	if err != nil {
		c.log.Error("编码ConnMsg失败", logger.F("error", err.Error()))
		return
	}

	c.mu.Lock()
	c.heartbeatAckReceived = false
	c.lastHeartbeatAt = time.Now().UnixMilli()
	c.mu.Unlock()

	c.sendBinary(msgData)
	c.log.Debug("心跳已发送")
}

// startHeartbeat 启动心跳
func (c *WsClient) startHeartbeat(isFirst bool) {
	c.mu.Lock()
	if c.heartbeatTimer != nil {
		c.heartbeatTimer.Stop()
	}
	c.mu.Unlock()

	delayMs := 5000
	if !isFirst {
		c.mu.RLock()
		delayMs = (c.heartbeatInterval - 1) * 1000
		c.mu.RUnlock()
	}

	c.mu.Lock()
	c.heartbeatTimer = time.AfterFunc(time.Duration(delayMs)*time.Millisecond, func() {
		c.sendPing()
	})
	c.mu.Unlock()
}

// scheduleNextPingCheck 安排下一次Ping检查
func (c *WsClient) scheduleNextPingCheck() {
	c.mu.Lock()
	interval := c.heartbeatInterval
	c.mu.Unlock()

	delayMs := (interval - 1) * 1000

	c.mu.Lock()
	c.heartbeatTimer = time.AfterFunc(time.Duration(delayMs)*time.Millisecond, func() {
		c.sendPing()
	})
	c.mu.Unlock()
}

// scheduleReconnect 安排重连
func (c *WsClient) scheduleReconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 超过最大重连次数判断
	c.reconnectAttempts++
	if c.reconnectAttempts > c.maxReconnectAttempts {
		c.log.Error("超过最大重连次数", logger.F("attempts", c.reconnectAttempts))
		c.state = "disconnected"
		return
	}

	delay := c.getReconnectDelay()
	c.log.Info("准备重连", map[string]any{
		"delay_ms": delay.Milliseconds(),
		"attempt":  c.reconnectAttempts,
	})

	// 在锁内安全清理并重置定时器
	if c.reconnectTimer != nil {
		c.reconnectTimer.Stop()
	}
	c.reconnectTimer = time.AfterFunc(delay, func() {
		c.doConnect()
	})
}

// refreshTokenAndReconnect 刷新token并重连
func (c *WsClient) refreshTokenAndReconnect() {
	if c.callback == nil {
		return
	}

	// 调用回调获取新的认证信息
	newAuth, err := c.callback.OnAuthFailed(41103) // AUTH_TOKEN_INVALID
	if err != nil || newAuth == nil {
		c.log.Error("刷新token失败")
		c.mu.Lock()
		c.state = "disconnected"
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.auth = newAuth
	c.mu.Unlock()

	c.scheduleReconnect()
}

// getReconnectDelay 获取重连延迟
func (c *WsClient) getReconnectDelay() time.Duration {
	// c.mu.RLock()
	// defer c.mu.RUnlock()

	delays := c.reconnectDelays
	index := c.reconnectAttempts
	if index >= len(delays) {
		index = len(delays) - 1
	}

	return delays[index]
}

// shouldRefreshToken 判断是否应该刷新token
func (c *WsClient) shouldRefreshToken(code int) bool {
	authFailedCodes := map[int]bool{
		41103: true, // AUTH_TOKEN_INVALID
		41104: true, // AUTH_TOKEN_EXPIRED
		41108: true, // AUTH_TOKEN_FORCED_EXPIRATION
	}

	retryableCodes := map[int]bool{
		50400: true, // INNER_SVR_FAIL
		50503: true, // OVERLOAD_CONTROL
		90001: true, // NET_FAIL
		90003: true, // BACKEND_RETURN_FAIL
	}

	return authFailedCodes[code] || retryableCodes[code]
}

// sendBinary 发送二进制数据
func (c *WsClient) sendBinary(data []byte) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	err := conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		c.log.Error("发送消息失败", logger.F("error", err.Error()))
		return err
	}

	return nil
}

// close 关闭连接
func (c *WsClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stopHeartbeatLocked()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// handleClose 处理关闭
func (c *WsClient) handleClose(code int, reason string) {
	c.mu.Lock()
	// 状态守卫：如果已经在重连或手动断开，直接忽略后续信号
	if c.state == "reconnecting" || c.state == "disconnected" {
		c.mu.Unlock()
		return
	}

	c.log.Info("处理连接关闭", map[string]any{"code": code, "reason": reason})

	// 立即变更状态为重连中，封死其他竞争入口
	c.state = "reconnecting"

	// 使用不加锁的私有方法清理定时器，避免死锁
	c.stopHeartbeatLocked()
	c.mu.Unlock()

	// 回调给外部（锁外执行）
	if c.callback != nil {
		c.callback.OnClose(code, reason)
	}

	// 检查不可重连码
	noReconnectCodes := map[int]bool{4012: true, 4013: true, 4014: true, 4018: true, 4019: true, 4021: true}
	if noReconnectCodes[code] {
		c.mu.Lock()
		c.state = "disconnected"
		c.mu.Unlock()
		return
	}

	// 统一由 handleClose 触发重连计划
	c.scheduleReconnect()
}

// stopHeartbeatLocked 内部心跳停止逻辑 (加锁)
func (c *WsClient) stopHeartbeatLocked() {
	if c.heartbeatTimer != nil {
		c.heartbeatTimer.Stop()
		c.heartbeatTimer = nil
	}
}

// stopHeartbeat 对外暴露的心跳停止方法 (加锁)
func (c *WsClient) stopHeartbeat() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopHeartbeatLocked()
}

// Disconnect 断开连接
func (c *WsClient) Disconnect() {
	c.mu.Lock()
	c.state = "disconnected"
	c.mu.Unlock()

	c.cancel()
	c.close()
}

// GetState 获取状态
func (c *WsClient) GetState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetConnectId 获取连接ID
func (c *WsClient) GetConnectId() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connectId
}

// SendAndWait 发送并等待响应
func (c *WsClient) SendAndWait(cmd, module string, data []byte, timeoutMs int) (any, error) {
	return c.SendAndWaitWithDecoder(cmd, module, data, timeoutMs, nil)
}

// SendAndWaitWithDecoder 发送并等待响应（支持自定义解码器）
func (c *WsClient) SendAndWaitWithDecoder(cmd, module string, data []byte, timeoutMs int, decoder func([]byte, string) any) (any, error) {
	msgId := c.generateMsgId()

	connMsg := &proto.ConnMsgWrapper{
		Head: &proto.HeadWrapper{
			CmdType: 0,
			Cmd:     cmd,
			SeqNo:   c.nextSeqNo(),
			MsgId:   msgId,
			Module:  module,
		},
		Data: data,
	}

	msgData, err := proto.EncodePB(connMsg)
	if err != nil {
		return nil, fmt.Errorf("编码失败: %w", err)
	}

	c.log.Info("SendAndWaitWithDecoder: 注册pending请求", logger.F("msgId", msgId), logger.F("cmd", cmd))

	resolveCh := make(chan any, 1)
	rejectCh := make(chan error, 1)

	c.mu.Lock()
	pending := &PendingRequest{
		Timer:   nil,
		Decoder: decoder,
	}

	// 设置 Resolve 和 Reject，它们会在调用时自动清理
	pending.Resolve = func(v any) {
		// 先停止计时器
		if pending.Timer != nil {
			pending.Timer.Stop()
		}
		// 删除 pending
		c.mu.Lock()
		delete(c.pendingRequests, msgId)
		c.mu.Unlock()
		// 发送结果
		select {
		case resolveCh <- v:
		default:
		}
	}

	pending.Reject = func(e error) {
		c.mu.Lock()
		delete(c.pendingRequests, msgId)
		c.mu.Unlock()
		select {
		case rejectCh <- e:
		default:
		}
	}

	pending.Timer = time.AfterFunc(time.Duration(timeoutMs)*time.Millisecond, func() {
		pending.Reject(fmt.Errorf("timeout after %dms", timeoutMs))
	})

	c.pendingRequests[msgId] = pending
	c.mu.Unlock()

	if err := c.sendBinary(msgData); err != nil {
		c.mu.Lock()
		delete(c.pendingRequests, msgId)
		c.mu.Unlock()
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	c.log.Info("消息已发送，等待响应...", logger.F("msgId", msgId))

	select {
	case result := <-resolveCh:
		return result, nil
	case err := <-rejectCh:
		return nil, err
	}
}

func (c *WsClient) SendAndWaitWithDecoderAndId(cmd, module string, data []byte, timeoutMs int, decoder func([]byte, string) any, msgId string) (any, error) {
	if timeoutMs <= 0 {
		timeoutMs = DefaultSendTimeoutMs
	}

	c.log.Info("SendAndWaitWithDecoderAndId: 注册pending请求", logger.F("msgId", msgId), logger.F("cmd", cmd))

	resolveCh := make(chan any, 1)
	rejectCh := make(chan error, 1)

	c.mu.Lock()
	c.pendingRequests[msgId] = &PendingRequest{
		Resolve: func(v any) {
			c.mu.Lock()
			delete(c.pendingRequests, msgId)
			c.mu.Unlock()
			resolveCh <- v
		},
		Reject: func(e error) {
			c.mu.Lock()
			delete(c.pendingRequests, msgId)
			c.mu.Unlock()
			rejectCh <- e
		},
		Timer: time.AfterFunc(time.Duration(timeoutMs)*time.Millisecond, func() {
			rejectCh <- fmt.Errorf("timeout after %dms", timeoutMs)
		}),
		Decoder: decoder,
	}
	c.mu.Unlock()

	// 使用同一个 msgId 发送
	err := c.sendBusinessConnMsg(cmd, module, msgId, data)
	if err != nil {
		c.mu.Lock()
		delete(c.pendingRequests, msgId)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case result := <-resolveCh:
		c.log.Info("收到响应成功", logger.F("msgId", msgId))
		return result, nil
	case err := <-rejectCh:
		// 超时了，但检查一下 pending 是否还存在（可能被响应处理删除了）
		c.mu.RLock()
		_, exists := c.pendingRequests[msgId]
		c.mu.RUnlock()
		c.log.Info("收到reject", logger.F("msgId", msgId), logger.F("exists", exists))
		if exists {
			return nil, err
		}
		// pending 不存在了，说明响应已经处理了，忽略超时错误
		return nil, fmt.Errorf("请求已处理")
	}
}

// SendC2CMessage 发送C2C消息
func (c *WsClient) SendC2CMessage(data *types.SendC2CMessageReq) (any, error) {
	wrapper := c.toSendC2CMessageReqWrapper(data)
	msgData, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, fmt.Errorf("编码SendC2CMessageReq失败: %w", err)
	}

	return c.SendAndWait("send_c2c_message", "yuanbao_openclaw_proxy", msgData, DefaultSendTimeoutMs)
}

// SendGroupMessage 发送群消息
func (c *WsClient) SendGroupMessage(data *types.SendGroupMessageReq) (any, error) {
	wrapper := c.toSendGroupMessageReqWrapper(data)
	msgData, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, fmt.Errorf("编码SendGroupMessageReq失败: %w", err)
	}

	return c.SendAndWait("send_group_message", "yuanbao_openclaw_proxy", msgData, DefaultSendTimeoutMs)
}

// QueryGroupInfo 查询群信息
func (c *WsClient) QueryGroupInfo(data *types.QueryGroupInfoReq) (*types.QueryGroupInfoResult, error) {
	wrapper := c.toQueryGroupInfoReqWrapper(data)
	msgData, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, fmt.Errorf("编码QueryGroupInfoReq失败: %w", err)
	}

	result, err := c.SendAndWait("query_group_info", "yuanbao_openclaw_proxy", msgData, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	return c.decodeQueryGroupInfoRsp(result)
}

// GetGroupMemberList 获取群成员列表
func (c *WsClient) GetGroupMemberList(data *types.GetGroupMemberListReq) (*types.GetGroupMemberListResult, error) {
	wrapper := c.toGetGroupMemberListReqWrapper(data)
	msgData, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, fmt.Errorf("编码GetGroupMemberListReq失败: %w", err)
	}

	result, err := c.SendAndWait("get_group_member_list", "yuanbao_openclaw_proxy", msgData, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	return c.decodeGetGroupMemberListRsp(result)
}

// SendPrivateHeartbeat 发送私聊心跳
func (c *WsClient) SendPrivateHeartbeat(data *types.SendPrivateHeartbeatReq) (*types.SendHeartbeatResult, error) {
	wrapper := c.toSendPrivateHeartbeatReqWrapper(data)
	msgData, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, fmt.Errorf("编码失败: %w", err)
	}

	result, err := c.SendAndWait("send_private_heartbeat", "yuanbao_openclaw_proxy", msgData, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	return c.decodeSendPrivateHeartbeatRsp(result)
}

// SendGroupHeartbeat 发送群聊心跳
func (c *WsClient) SendGroupHeartbeat(data *types.SendGroupHeartbeatReq) (*types.SendHeartbeatResult, error) {
	wrapper := c.toSendGroupHeartbeatReqWrapper(data)
	msgData, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, fmt.Errorf("编码失败: %w", err)
	}

	result, err := c.SendAndWait("send_group_heartbeat", "yuanbao_openclaw_proxy", msgData, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	return c.decodeSendGroupHeartbeatRsp(result)
}

// decodePB 解码protobuf
func (c *WsClient) decodePB(name string, data []byte) (goproto.Message, error) {
	return proto.DecodePB(name, data)
}

// 解码响应
func (c *WsClient) decodeSendC2CMessageRsp(data []byte, msgId string) any {
	c.log.Info("开始解码响应", logger.F("msgId", msgId), logger.F("dataLen", len(data)))

	rsp, err := c.decodePB("trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendC2CMessageRsp", data)
	if err != nil {
		c.log.Error("PB解码失败", logger.F("error", err.Error()))
		return nil
	}

	if r, ok := rsp.(*proto.SendC2CMessageRspWrapper); ok {
		c.log.Info("解码成功", logger.F("Code", r.Code), logger.F("Message", r.Message))
		return &types.SendMessageResult{
			MsgID:   msgId,
			Code:    r.Code,
			Message: r.Message,
		}
	}

	c.log.Error("类型断言失败")
	return nil
}

func (c *WsClient) decodeSendGroupMessageRsp(data []byte, msgId string) any {
	c.log.Info("解码群消息响应", logger.F("msgId", msgId), logger.F("dataLen", len(data)))

	rsp, err := c.decodePB("trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendGroupMessageRsp", data)
	if err != nil {
		c.log.Error("PB解码失败", logger.F("error", err.Error()))
		return nil
	}

	if r, ok := rsp.(*proto.SendGroupMessageRspWrapper); ok {
		c.log.Info("群消息解码成功", logger.F("Code", r.Code), logger.F("Message", r.Message))
		return &types.SendMessageResult{
			MsgID:   msgId,
			Code:    r.Code,
			Message: r.Message,
		}
	}

	c.log.Error("类型断言失败")
	return nil
}

func (c *WsClient) decodeQueryGroupInfoRsp(result any) (*types.QueryGroupInfoResult, error) {
	rsp, ok := result.(*proto.QueryGroupInfoRspWrapper)
	if !ok {
		return nil, fmt.Errorf("类型断言失败")
	}

	r := &types.QueryGroupInfoResult{
		Code: rsp.Code,
		Msg:  rsp.Msg,
	}

	if rsp.GroupInfo != nil {
		r.GroupInfo = &types.GroupInfo{
			GroupName:          rsp.GroupInfo.GroupName,
			GroupOwnerUserID:   rsp.GroupInfo.GroupOwnerUserId,
			GroupOwnerNickname: rsp.GroupInfo.GroupOwnerNickname,
			GroupSize:          rsp.GroupInfo.GroupSize,
		}
	}

	return r, nil
}

func (c *WsClient) decodeGetGroupMemberListRsp(result any) (*types.GetGroupMemberListResult, error) {
	rsp, ok := result.(*proto.GetGroupMemberListRspWrapper)
	if !ok {
		return nil, fmt.Errorf("类型断言失败")
	}

	r := &types.GetGroupMemberListResult{
		Code:    rsp.Code,
		Message: rsp.Message,
	}

	for _, m := range rsp.MemberList {
		r.MemberList = append(r.MemberList, types.Member{
			UserID:   m.UserId,
			NickName: m.NickName,
			UserType: m.UserType,
		})
	}

	return r, nil
}

func (c *WsClient) decodeSendPrivateHeartbeatRsp(result any) (*types.SendHeartbeatResult, error) {
	rsp, ok := result.(*proto.SendPrivateHeartbeatRspWrapper)
	if !ok {
		return nil, fmt.Errorf("类型断言失败")
	}

	return &types.SendHeartbeatResult{
		Code:    rsp.Code,
		Msg:     rsp.Msg,
		Message: rsp.Msg,
	}, nil
}

func (c *WsClient) decodeSendGroupHeartbeatRsp(result any) (*types.SendHeartbeatResult, error) {
	rsp, ok := result.(*proto.SendGroupHeartbeatRspWrapper)
	if !ok {
		return nil, fmt.Errorf("类型断言失败")
	}

	return &types.SendHeartbeatResult{
		Code:    rsp.Code,
		Msg:     rsp.Msg,
		Message: rsp.Msg,
	}, nil
}

// generateMsgId 生成消息ID
func (c *WsClient) generateMsgId() string {
	// 生成32位UUID
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// nextSeqNo 生成序列号
func (c *WsClient) nextSeqNo() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seqNo++
	if c.seqNo == 0 {
		c.seqNo = 1
	}
	return c.seqNo
}

// 辅助函数

func parseDelays(s string) []time.Duration {
	parts := strings.Split(s, ",")
	delays := make([]time.Duration, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		d, err := time.ParseDuration(p)
		if err == nil {
			delays = append(delays, d)
		}
	}

	if len(delays) == 0 {
		return []time.Duration{1 * time.Second}
	}

	return delays
}

func defaultMessageDecoder(data []byte, msgId string) any {
	// 尝试解码为SendMessageResult
	var result types.SendMessageResult
	result.MsgID = msgId

	if len(data) >= 4 {
		result.Code = int32(binary.LittleEndian.Uint32(data[:4]))
	}

	return &result
}

// SendC2CMessageReq 实现兼容的类型
func (c *WsClient) toSendC2CMessageReqWrapper(data *types.SendC2CMessageReq) *proto.SendC2CMessageReqWrapper {
	msgId := data.MsgID
	if msgId == "" {
		msgId = c.generateMsgId()
	}

	msgRandom := data.MsgRandom
	if msgRandom == 0 {
		msgRandom = uint32(time.Now().UnixNano() / 1e6)
	}

	req := &proto.SendC2CMessageReqWrapper{
		MsgId:       msgId,
		ToAccount:   data.ToAccount,
		FromAccount: data.FromAccount, // 使用传入的值，为空时服务器自动处理
		MsgRandom:   msgRandom,
		MsgSeq:      data.MsgSeq,
	}

	for _, elem := range data.MsgBody {
		req.MsgBody = append(req.MsgBody, &proto.MsgBodyElementWrapper{
			MsgType: elem.MsgType,
			MsgContent: &proto.MsgContentWrapper{
				Text: elem.MsgContent.Text,
			},
		})
	}

	return req
}

func (c *WsClient) toSendGroupMessageReqWrapper(data *types.SendGroupMessageReq) *proto.SendGroupMessageReqWrapper {
	req := &proto.SendGroupMessageReqWrapper{
		MsgId:       data.MsgID,
		GroupCode:   data.GroupCode,
		FromAccount: data.FromAccount,
		ToAccount:   data.ToAccount,
		Random:      data.Random,
		RefMsgId:    data.RefMsgID,
		MsgSeq:      data.MsgSeq,
	}

	if data.LogExt != nil {
		req.LogExt = &proto.LogInfoExtWrapper{
			TraceId: data.LogExt.TraceId,
		}
	}

	for _, elem := range data.MsgBody {
		req.MsgBody = append(req.MsgBody, &proto.MsgBodyElementWrapper{
			MsgType: elem.MsgType,
			MsgContent: &proto.MsgContentWrapper{
				Text:        elem.MsgContent.Text,
				UUID:        elem.MsgContent.UUID,
				ImageFormat: elem.MsgContent.ImageFormat,
				Data:        elem.MsgContent.Data,
				Desc:        elem.MsgContent.Desc,
				Ext:         elem.MsgContent.Ext,
				Sound:       elem.MsgContent.Sound,
				Index:       elem.MsgContent.Index,
				URL:         elem.MsgContent.URL,
				FileSize:    elem.MsgContent.FileSize,
				FileName:    elem.MsgContent.FileName,
			},
		})
	}

	return req
}

func (c *WsClient) toQueryGroupInfoReqWrapper(data *types.QueryGroupInfoReq) *proto.QueryGroupInfoReqWrapper {
	return &proto.QueryGroupInfoReqWrapper{
		GroupCode: data.GroupCode,
	}
}

func (c *WsClient) toGetGroupMemberListReqWrapper(data *types.GetGroupMemberListReq) *proto.GetGroupMemberListReqWrapper {
	return &proto.GetGroupMemberListReqWrapper{
		GroupCode: data.GroupCode,
	}
}

func (c *WsClient) toSendPrivateHeartbeatReqWrapper(data *types.SendPrivateHeartbeatReq) *proto.SendPrivateHeartbeatReqWrapper {
	return &proto.SendPrivateHeartbeatReqWrapper{
		FromAccount: data.FromAccount,
		ToAccount:   data.ToAccount,
		Heartbeat:   int32(data.Heartbeat),
	}
}

func (c *WsClient) toSendGroupHeartbeatReqWrapper(data *types.SendGroupHeartbeatReq) *proto.SendGroupHeartbeatReqWrapper {
	return &proto.SendGroupHeartbeatReqWrapper{
		FromAccount: data.FromAccount,
		ToAccount:   data.ToAccount,
		GroupCode:   data.GroupCode,
		SendTime:    data.SendTime,
		Heartbeat:   int32(data.Heartbeat),
	}
}

func (c *WsClient) SendC2CMessageSimple(toAccount string, msgBody []types.MsgBodyElement) (*types.SendMessageResult, error) {
	msgId := c.generateMsgId()
	c.log.Info("准备发送消息", logger.F("msgId", msgId), logger.F("to", toAccount))

	req := &types.SendC2CMessageReq{
		MsgID:       msgId,
		ToAccount:   toAccount,
		FromAccount: c.botId,
		MsgRandom:   uint32(time.Now().UnixNano() / 1e6),
		MsgBody:     msgBody,
	}

	wrapper := c.toSendC2CMessageReqWrapper(req)
	data, err := goproto.Marshal(wrapper)
	if err != nil {
		return nil, err
	}

	connMsg := &proto.ConnMsgWrapper{
		Head: &proto.HeadWrapper{
			CmdType: 0,
			Cmd:     "send_c2c_message",
			SeqNo:   c.nextSeqNo(),
			MsgId:   msgId,
			Module:  "yuanbao_openclaw_proxy",
		},
		Data: data,
	}

	msgData, err := proto.EncodePB(connMsg)
	if err != nil {
		return nil, fmt.Errorf("编码失败: %w", err)
	}

	// 直接发送，不等待响应
	if err := c.sendBinary(msgData); err != nil {
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	// 返回成功
	return &types.SendMessageResult{
		MsgID:   msgId,
		Code:    0,
		Message: "succ",
	}, nil
}

func (c *WsClient) SendGroupMessageSimple(groupCode string, msgBody []types.MsgBodyElement) (*types.SendMessageResult, error) {
	req := &types.SendGroupMessageReq{
		GroupCode:   groupCode,
		FromAccount: c.botId,
		Random:      fmt.Sprintf("%d", time.Now().UnixNano()/1e6),
		MsgBody:     msgBody,
	}

	wrapper := c.toSendGroupMessageReqWrapper(req)
	data, err := proto.EncodePB(wrapper)
	if err != nil {
		return nil, err
	}

	// 生成 msgId
	msgId := c.generateMsgId()

	connMsg := &proto.ConnMsgWrapper{
		Head: &proto.HeadWrapper{
			CmdType: 0,
			Cmd:     "send_group_message",
			SeqNo:   c.nextSeqNo(),
			MsgId:   msgId,
			Module:  "yuanbao_openclaw_proxy",
		},
		Data: data,
	}

	msgData, err := proto.EncodePB(connMsg)
	if err != nil {
		return nil, fmt.Errorf("编码失败: %w", err)
	}

	// 直接发送，不等待响应
	if err := c.sendBinary(msgData); err != nil {
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	// 返回成功
	return &types.SendMessageResult{
		MsgID:   msgId,
		Code:    0,
		Message: "succ",
	}, nil
}

// SendPrivateHeartbeatSimple 便捷方法
func (c *WsClient) SendPrivateHeartbeatSimple(fromAccount, toAccount string, heartbeat types.WsHeartbeat) (*types.SendHeartbeatResult, error) {
	req := &proto.SendPrivateHeartbeatReqWrapper{
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		Heartbeat:   int32(heartbeat),
	}

	data, err := proto.EncodePB(req)
	if err != nil {
		return nil, err
	}

	result, err := c.SendAndWait("send_private_heartbeat", "yuanbao_openclaw_proxy", data, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	if r, ok := result.(*types.SendHeartbeatResult); ok {
		return r, nil
	}

	return nil, fmt.Errorf("unexpected result type")
}

// SendGroupHeartbeatSimple 便捷方法
func (c *WsClient) SendGroupHeartbeatSimple(fromAccount, toAccount, groupCode string, heartbeat types.WsHeartbeat) (*types.SendHeartbeatResult, error) {
	req := &proto.SendGroupHeartbeatReqWrapper{
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		GroupCode:   groupCode,
		SendTime:    time.Now().Unix(),
		Heartbeat:   int32(heartbeat),
	}

	data, err := proto.EncodePB(req)
	if err != nil {
		return nil, err
	}

	result, err := c.SendAndWait("send_group_heartbeat", "yuanbao_openclaw_proxy", data, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	if r, ok := result.(*types.SendHeartbeatResult); ok {
		return r, nil
	}

	return nil, fmt.Errorf("unexpected result type")
}

// QueryGroupInfoSimple 便捷方法
func (c *WsClient) QueryGroupInfoSimple(groupCode string) (*types.QueryGroupInfoResult, error) {
	req := &proto.QueryGroupInfoReqWrapper{
		GroupCode: groupCode,
	}

	data, err := proto.EncodePB(req)
	if err != nil {
		return nil, err
	}

	result, err := c.SendAndWait("query_group_info", "yuanbao_openclaw_proxy", data, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	return c.decodeQueryGroupInfoRsp(result)
}

// GetGroupMemberListSimple 便捷方法
func (c *WsClient) GetGroupMemberListSimple(groupCode string) (*types.GetGroupMemberListResult, error) {
	req := &proto.GetGroupMemberListReqWrapper{
		GroupCode: groupCode,
	}

	data, err := proto.EncodePB(req)
	if err != nil {
		return nil, err
	}

	result, err := c.SendAndWait("get_group_member_list", "yuanbao_openclaw_proxy", data, DefaultSendTimeoutMs)
	if err != nil {
		return nil, err
	}

	return c.decodeGetGroupMemberListRsp(result)
}

// 解码入站消息
func DecodeInboundMessage(data []byte) (*types.InboundMessage, error) {
	return DecodeInboundMessageWithBotId(data, "")
}

func DecodeInboundMessageWithBotId(data []byte, botId string) (*types.InboundMessage, error) {
	msg, err := proto.DecodePB("trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.InboundMessagePush", data)
	if err != nil {
		return nil, err
	}

	r, ok := msg.(*proto.InboundMessagePushWrapper)
	if !ok {
		return nil, fmt.Errorf("类型断言失败")
	}

	inbound := &types.InboundMessage{
		CallbackCommand:      r.CallbackCommand,
		FromAccount:          r.FromAccount,
		ToAccount:            r.ToAccount,
		SenderNickname:       r.SenderNickname,
		GroupID:              r.GroupId,
		GroupCode:            r.GroupCode,
		GroupName:            r.GroupName,
		MsgSeq:               r.MsgSeq,
		MsgRandom:            r.MsgRandom,
		MsgTime:              r.MsgTime,
		MsgKey:               r.MsgKey,
		MsgID:                r.MsgId,
		CloudCustomData:      r.CloudCustomData,
		EventTime:            r.EventTime,
		BotOwnerID:           r.BotOwnerId,
		PrivateFromGroupCode: r.PrivateFromGroupCode,
	}

	isAtBot := false

	for _, elem := range r.MsgBody {
		content := &types.MsgContent{
			Text:        elem.MsgContent.Text,
			UUID:        elem.MsgContent.UUID,
			ImageFormat: elem.MsgContent.ImageFormat,
			Data:        elem.MsgContent.Data,
			Desc:        elem.MsgContent.Desc,
			Ext:         elem.MsgContent.Ext,
			Sound:       elem.MsgContent.Sound,
			Index:       elem.MsgContent.Index,
			URL:         elem.MsgContent.URL,
			FileSize:    elem.MsgContent.FileSize,
			FileName:    elem.MsgContent.FileName,
		}

		if elem.MsgType == "TIMCustomElem" && elem.MsgContent.Data != "" && botId != "" {
			var customData map[string]any
			if err := json.Unmarshal([]byte(elem.MsgContent.Data), &customData); err == nil {
				if elemType, ok := customData["elem_type"].(float64); ok && int(elemType) == 1002 {
					if userId, ok := customData["user_id"].(string); ok {
						if userId == botId {
							isAtBot = true
						}
					}
				}
			}
		}

		inbound.MsgBody = append(inbound.MsgBody, types.MsgBodyElement{
			MsgType:    elem.MsgType,
			MsgContent: *content,
		})
	}

	inbound.IsAtBot = isAtBot

	if r.LogExt != nil {
		inbound.TraceID = r.LogExt.TraceId
	}

	return inbound, nil
}

// JSON 解析入站消息
func DecodeInboundMessageFromJSON(data []byte) (*types.InboundMessage, error) {
	return DecodeInboundMessageFromJSONWithBotId(data, "")
}

func DecodeInboundMessageFromJSONWithBotId(data []byte, botId string) (*types.InboundMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	inbound := &types.InboundMessage{}
	isAtBot := false

	if v, ok := raw["callback_command"].(string); ok {
		inbound.CallbackCommand = v
	}
	if v, ok := raw["from_account"].(string); ok {
		inbound.FromAccount = v
	}
	if v, ok := raw["to_account"].(string); ok {
		inbound.ToAccount = v
	}
	if v, ok := raw["sender_nickname"].(string); ok {
		inbound.SenderNickname = v
	}
	if v, ok := raw["group_code"].(string); ok {
		inbound.GroupCode = v
	}
	if v, ok := raw["group_name"].(string); ok {
		inbound.GroupName = v
	}
	if v, ok := raw["msg_id"].(string); ok {
		inbound.MsgID = v
	}
	if v, ok := raw["msg_key"].(string); ok {
		inbound.MsgKey = v
	}
	if v, ok := raw["msg_seq"].(float64); ok {
		inbound.MsgSeq = uint32(v)
	}

	// 解析 msg_body
	if body, ok := raw["msg_body"].([]any); ok {
		for _, elem := range body {
			if elemMap, ok := elem.(map[string]any); ok {
				bodyElem := types.MsgBodyElement{}
				if v, ok := elemMap["msg_type"].(string); ok {
					bodyElem.MsgType = v
				}
				if content, ok := elemMap["msg_content"].(map[string]any); ok {
					bodyElem.MsgContent = types.MsgContent{}
					if v, ok := content["text"].(string); ok {
						bodyElem.MsgContent.Text = v
					}
					// 检测 @ 消息 (TIMCustomElem with elem_type=1002)
					if bodyElem.MsgType == "TIMCustomElem" {
						if dataStr, ok := content["data"].(string); ok && botId != "" {
							var customData map[string]any
							if err := json.Unmarshal([]byte(dataStr), &customData); err == nil {
								if elemType, ok := customData["elem_type"].(float64); ok && int(elemType) == 1002 {
									if userId, ok := customData["user_id"].(string); ok {
										if userId == botId {
											isAtBot = true
										}
									}
								}
							}
						}
					}
				}
				inbound.MsgBody = append(inbound.MsgBody, bodyElem)
			}
		}
	}

	inbound.IsAtBot = isAtBot

	return inbound, nil
}

// sendBusinessConnMsg 构造业务 Head 并发送二进制包
func (c *WsClient) sendBusinessConnMsg(cmd, module, msgId string, data []byte) error {
	head := &proto.HeadWrapper{
		MsgId:   msgId,
		SeqNo:   uint32(time.Now().Unix()),
		Cmd:     cmd,
		CmdType: int32(proto.CmdTypeRequest),
		Module:  module,
	}

	connMsg := &proto.ConnMsgWrapper{
		Head: head,
		Data: data,
	}

	encoded, err := goproto.Marshal(connMsg)
	if err != nil {
		return fmt.Errorf("encode business conn msg failed: %v", err)
	}

	c.log.Info("发送二进制消息", logger.F("cmd", cmd), logger.F("msgId", msgId), logger.F("dataLen", len(data)), logger.F("encodedLen", len(encoded)))

	err = c.sendBinary(encoded)
	if err != nil {
		c.log.Error("sendBinary失败", logger.F("error", err.Error()))
		return err
	}

	c.log.Info("消息已发送，等待响应...", logger.F("msgId", msgId))
	return nil
}

// client.go 末尾
func (c *WsClient) handleBusinessRsp(head *proto.HeadWrapper, data []byte) {
	c.mu.Lock()
	pending, ok := c.pendingRequests[head.MsgId]
	if ok {
		delete(c.pendingRequests, head.MsgId)
	}
	c.mu.Unlock()

	if !ok {
		c.log.Warn("收到回包但找不到对应的本地请求", logger.F("msgId", head.MsgId), logger.F("cmd", head.Cmd))
		return
	}

	if pending.decoder != nil {
		result := pending.decoder(data, head.MsgId)
		// 打印解码结果，方便排查
		c.log.Info("回包解析完成", logger.F("msgId", head.MsgId))

		select {
		case pending.resolveCh <- result:
		default:
			c.log.Error("resolveCh 写入失败", logger.F("msgId", head.MsgId))
		}
	}
}
