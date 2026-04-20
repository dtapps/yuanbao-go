package ws

import (
	"context"
	"encoding/hex"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
	"github.com/dtapps/yuanbao-go/utils"
	bizProto "github.com/dtapps/yuanbao-go/wsproto/biz"
	"github.com/gorilla/websocket"
)

// WsClientCallback WebSocket客户端回调接口
type WsClientCallback interface {
	// OnReady 连接就绪
	OnReady(data *types.OnReadyData)
	// OnDispatch 消息推送
	OnDispatch(msg *bizProto.InboundMessagePush)
	// OnStateChange 状态变化
	OnStateChange(state string)
	// OnError 错误
	OnError(err error)
	// OnClose 关闭连接
	OnClose(code int, reason string)
	// OnKickout 被踢
	OnKickout(code int, reason string)
	// OnAuthFailed 认证失败
	OnAuthFailed(code int) (*types.WsAuthData, error)
}

// defaultDialer 全局默认 WebSocket Dialer
var defaultDialer = &websocket.Dialer{
	Proxy:            http.ProxyFromEnvironment,
	HandshakeTimeout: 45 * time.Second,
}

// SetDefaultDialer 设置全局默认 WebSocket Dialer
func SetDefaultDialer(dialer *websocket.Dialer) {
	if dialer != nil {
		defaultDialer = dialer
	}
}

// WsClient WebSocket客户端
type WsClient struct {
	mu        sync.RWMutex
	conn      *websocket.Conn
	url       string
	state     string
	auth      *types.WsAuthData
	accountID string
	botID     string

	// 心跳相关
	heartbeatInterval     time.Duration // 心跳间隔(秒)
	heartbeatTimer        *time.Timer   // 心跳定时器
	heartbeatAckReceived  bool          // 是否收到心跳确认
	lastHeartbeatAt       int64         // 上次心跳时间
	heartbeatTimeoutCount int           // 心跳超时次数
	heartbeatCount        int           // 心跳次数

	// 重连相关
	reconnectAttempts    int
	maxReconnectAttempts int
	reconnectDelays      []time.Duration
	reconnectTimer       *time.Timer

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
	connectID string
}

// NewWsClient 创建WebSocket客户端
func NewWsClient(url string, accountID string, botID string, callback WsClientCallback) *WsClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WsClient{
		url:                  url,
		accountID:            accountID,
		botID:                botID,
		state:                types.ConnectionStateDisconnected.String(),
		heartbeatInterval:    types.DefaultHeartbeatInterval,
		maxReconnectAttempts: types.DefaultMaxReconnectAttempts,
		reconnectDelays:      utils.ParseDelays(types.DefaultReconnectDelays),
		callback:             callback,
		log:                  logger.New("ws"),
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// SetAuth 设置认证信息
func (c *WsClient) SetAuth(auth *types.WsAuthData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.auth = auth
}

// SetReconnectConfig 设置重连配置
func (c *WsClient) SetReconnectConfig(maxAttempts int, delays string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxReconnectAttempts = maxAttempts
	c.reconnectDelays = utils.ParseDelays(delays)
}

// Connect 连接
func (c *WsClient) Connect() error {
	c.mu.Lock()
	if c.state == types.ConnectionStateDisconnected.String() {
		c.state = types.ConnectionStateConnecting.String()
	}
	c.mu.Unlock()

	return c.doConnect()
}

// doConnect 执行连接
func (c *WsClient) doConnect() error {
	c.log.Info("正在连接", logger.FS("url", c.url))

	// 创建WebSocket连接
	header := http.Header{}
	header.Set("Origin", types.DefaultWSOrigin)

	conn, resp, err := defaultDialer.Dial(c.url, header)
	if err != nil {
		c.log.Error("连接失败", logger.F("error", err.Error()))
		c.ScheduleReconnect()
		return err
	}

	if resp != nil {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("关闭响应体失败", logger.F("error", err.Error()))
		}
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// 启动读取协程
	go c.readLoop()

	// 发送认证消息
	c.sendAuthBindMessage()

	return nil
}

// readLoop 读取循环
func (c *WsClient) readLoop() {
	closeHandled := false
	defer func() {
		if !closeHandled {
			c.handleClose(1006, "read loop exit")
		}
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
				if closeErr, ok := err.(*websocket.CloseError); ok {
					closeHandled = true
					c.handleClose(closeErr.Code, closeErr.Text)
					return
				}
				// 忽略 "use of closed" 错误，这是正常关闭连接
				if !strings.Contains(err.Error(), "use of closed") {
					c.log.Error("读取消息失败", logger.F("error", err.Error()))
				}
				return
			}

			// HEX 只打印前 64 字节
			limit := min(len(data), 64)
			c.log.Debug("收到原始消息",
				logger.F("length", len(data)),
				logger.F("data", string(data)),
				logger.F("hex", hex.EncodeToString(data[:limit])),
			)

			c.handleMessage(data)
		}
	}
}

// ScheduleReconnect 安排重连
func (c *WsClient) ScheduleReconnect() {
	c.mu.Lock()
	if c.state == types.ConnectionStateConnecting.String() {
		c.mu.Unlock()
		return
	}
	c.state = types.ConnectionStateConnecting.String()
	c.mu.Unlock()

	c.stopHeartbeat()

	c.mu.Lock()
	c.reconnectAttempts++
	if c.maxReconnectAttempts > 0 && c.reconnectAttempts > c.maxReconnectAttempts {
		c.log.Error("达到最大重连次数", logger.F("attempts", c.reconnectAttempts))
		c.state = types.ConnectionStateDisconnected.String()
		c.mu.Unlock()
		return
	}

	idx := c.reconnectAttempts - 1
	if idx >= len(c.reconnectDelays) {
		idx = len(c.reconnectDelays) - 1
	}
	if idx < 0 {
		idx = 0
	}
	delay := c.reconnectDelays[idx] + time.Duration(rand.Intn(1000))*time.Millisecond

	c.log.Info("计划重连",
		logger.F("delay", delay),
		logger.F("attempts", c.reconnectAttempts),
	)

	c.reconnectTimer = time.AfterFunc(delay, func() {
		if err := c.doConnect(); err != nil {
			c.log.Error("重连失败", logger.F("error", err.Error()))
		}
	})
	c.mu.Unlock()
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

// close 关闭连接
func (c *WsClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Warn("正在关闭连接")

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.log.Error("关闭连接失败", logger.F("error", err.Error()))
		}
		c.conn = nil
	}

	c.stopHeartbeatLocked()

	// 停止并置空重连定时器
	if c.reconnectTimer != nil {
		c.reconnectTimer.Stop()
		c.reconnectTimer = nil
	}
}

// Disconnect 断开连接
func (c *WsClient) Disconnect() error {
	c.mu.Lock()
	c.state = types.ConnectionStateDisconnected.String()
	c.mu.Unlock()

	c.cancel()
	c.close()

	return nil
}

// GetState 获取状态
func (c *WsClient) GetState() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetConnectID 获取连接ID
func (c *WsClient) GetConnectID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connectID
}

// generateNextSeqNo 生成下一个序列号
func (c *WsClient) generateNextSeqNo() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.seqNo++
	if c.seqNo == 0 {
		c.seqNo = 1
	}
	return c.seqNo
}
