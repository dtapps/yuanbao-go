package ws

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/message"
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
	manualDisconnect     bool // 是否手动断开，用于区分主动关闭和异常断开

	// 回调
	callback WsClientCallback

	// 日志
	log *logger.Logger

	// 序列号
	seqNo uint32

	// 业务消息序列号
	msgSeq uint64

	// === 有序发送队列 ===
	// 所有业务消息（C2C/Group）通过此队列串行发送，
	// 保证无论调用方如何并发，消息都按入队顺序发出。
	sendQueue  chan sendTask // 任务队列（有缓冲）
	sendOnce   sync.Once     // 确保 sender 只启动一次
	senderDone chan struct{} // 用于通知 sender 退出

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 连接ID
	connectID string
}

// sendTask 有序发送任务
type sendTask struct {
	execute func() (string, error) // 实际的发送逻辑
	result  chan sendResult        // 结果回传通道
}

// sendResult 发送结果
type sendResult struct {
	msgID string
	err   error
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
		sendQueue:            make(chan sendTask, 256), // 有缓冲队列，调用方可快速入队
		senderDone:           make(chan struct{}),
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
		int(types.RetCodeAuthTokenInvalid):          true, // Token 无效
		int(types.RetCodeAuthTokenExpired):          true, // Token 过期
		int(types.RetCodeAuthTokenForcedExpiration): true, // Token 强制过期
	}

	retryableCodes := map[int]bool{
		int(types.RetCodeInnerSvrFail):    true, // 服务器内部错误
		int(types.RetCodeOverloadControl): true, // 过载控制
		int(types.RetCodeNetFail):         true, // 网络失败
		int(types.RetCodeBackendFail):     true, // 后端返回失败
	}

	return authFailedCodes[code] || retryableCodes[code]
}

// close 关闭连接
func (c *WsClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Warn("正在关闭连接")

	// 停止有序发送队列的 sender 协程
	select {
	case <-c.senderDone:
		// 已经停止
	default:
		close(c.senderDone)
	}

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
	c.manualDisconnect = true
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

// startSender 启动有序发送协程（懒加载，只启动一次）
// 所有 SendC2CMessage/SendGroupMessage 的请求都会进入队列，
// 由该协程按 FIFO 顺序逐条执行，保证消息有序性。
func (c *WsClient) startSender() {
	c.sendOnce.Do(func() {
		go c.senderLoop()
	})
}

// senderLoop 有序发送主循环
// 从 sendQueue 中按序取出任务并执行，保证消息严格按入队顺序发出。
func (c *WsClient) senderLoop() {
	c.log.Info("[Sender] 有序发送协程已启动")
	for {
		select {
		case <-c.senderDone:
			// 客户端关闭，退出循环
			// 排空剩余任务
			for len(c.sendQueue) > 0 {
				task := <-c.sendQueue
				task.result <- sendResult{err: fmt.Errorf("客户端已关闭")}
			}
			c.log.Info("[Sender] 有序发送协程已停止")
			return

		case task := <-c.sendQueue:
			msgID, err := task.execute()
			task.result <- sendResult{msgID: msgID, err: err}
		}
	}
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

// generateNextMsgSeq 生成下一个业务消息序号
func (c *WsClient) generateNextMsgSeq() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.msgSeq++
	if c.msgSeq == 0 {
		c.msgSeq = 1
	}
	return c.msgSeq
}

// SyncInformation 同步信息（如命令列表）到服务器
func (c *WsClient) SyncInformation(data *types.SyncInformationData) error {
	c.log.Info("[同步] 发送 SyncInformation 请求",
		logger.F("syncType", data.SyncType),
		logger.F("botVersion", data.BotVersion),
		logger.F("pluginVersion", data.PluginVersion),
	)

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("连接未建立")
	}

	// 构建 proto 消息
	req := &bizProto.SyncInformationReq{
		SyncType:      bizProto.SyncInformationType(data.SyncType),
		BotVersion:    data.BotVersion,
		PluginVersion: data.PluginVersion,
		CommandData: &bizProto.SyncCommandsData{
			BotCommands:    botCommandsToProto(data.CommandData.BotCommands),
			PluginCommands: botCommandsToProto(data.CommandData.PluginCommands),
		},
	}

	// 编码为二进制
	encoded, err := utils.EncodeBizPB(req)
	if err != nil {
		c.log.Error("编码 SyncInformationReq 失败", logger.F("error", err.Error()))
		return fmt.Errorf("编码失败：%w", err)
	}

	// 构建业务请求消息
	reqParams := message.BuildBizRequestParams{
		SeqNo:   c.generateNextSeqNo(),
		Cmd:     string(types.BizCmdSyncInformation),
		Module:  string(types.ModuleYuanbaoOpenClawProxy),
		MsgID:   message.GenerateMsgID(),
		Payload: encoded,
	}

	reqData, err := message.BuildBizRequestMessage(reqParams)
	if err != nil {
		c.log.Error("构建 SyncInformation 请求失败", logger.F("error", err.Error()))
		return fmt.Errorf("构建请求失败：%w", err)
	}

	// 发送请求
	if err := conn.WriteMessage(websocket.BinaryMessage, reqData); err != nil {
		c.log.Error("发送 SyncInformation 失败", logger.F("error", err.Error()))
		return fmt.Errorf("发送失败：%w", err)
	}

	c.log.Info("[同步] SyncInformation 发送成功")
	return nil
}

// botCommandsToProto BotCommand 转换为 proto 类型
func botCommandsToProto(commands []types.BotCommand) []*bizProto.Command {
	result := make([]*bizProto.Command, 0, len(commands))
	for _, cmd := range commands {
		result = append(result, &bizProto.Command{
			Name:        cmd.Name,
			Description: cmd.Description,
		})
	}
	return result
}
