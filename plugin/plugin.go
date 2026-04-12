package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/account"
	"github.com/dtapps/yuanbao-go/config"
	"github.com/dtapps/yuanbao-go/http"
	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/member"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/dtapps/yuanbao-go/outbound"
	"github.com/dtapps/yuanbao-go/types"
	"github.com/dtapps/yuanbao-go/ws"
)

// Plugin 元宝插件
type Plugin struct {
	name      string
	version   string
	accountId string
	account   *account.Account
	config    *config.Config
	client    *ws.WsClient
	runtime   *Runtime
	log       *logger.Logger
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// Runtime 运行时
type Runtime struct {
	channel        *ChannelRuntime
	onMessage      func(msg *types.InboundMessage, chatType string)
	onConnected    func()
	onDisconnected func()
}

// ChannelRuntime 通道运行时
type ChannelRuntime struct {
	// 文本分块
	ChunkMarkdownText func(text string, limit int) []string
	// 发送文本
	SendText func(text string) error
	// 发送消息体
	SendMsgBody func(msgBody []types.MsgBodyElement) error
}

// NewPlugin 创建插件
func NewPlugin(accountId string, account *account.Account, cfg *config.Config) *Plugin {
	ctx, cancel := context.WithCancel(context.Background())

	return &Plugin{
		name:      "yuanbao",
		version:   "1.0.0",
		accountId: accountId,
		account:   account,
		config:    cfg,
		log:       logger.New("plugin"),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动插件
func (p *Plugin) Start() error {
	p.log.Info("启动元宝插件", logger.F("accountId", p.accountId))
	p.log.Info("配置信息", map[string]any{
		"appKey":    maskString(p.account.AppKey),
		"appSecret": maskString(p.account.AppSecret),
		"apiDomain": p.account.ApiDomain,
	})

	// 创建HTTP客户端
	httpClient := http.NewClient(p.account)

	// 获取Token
	tokenData, err := httpClient.GetSignToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %w", err)
	}

	// 更新Bot ID
	if tokenData.BotID != "" {
		p.account.BotID = tokenData.BotID
	}

	// 创建WS客户端
	p.client = ws.NewWsClient(p.account.WsGatewayUrl, p.account.AccountID, p.account.BotID, p)

	// 设置认证信息
	p.client.SetAuth(&types.WsAuth{
		BizID:    "ybBot",
		UID:      p.account.BotID,
		Source:   tokenData.Source,
		Token:    tokenData.Token,
		RouteEnv: p.account.Config.RouteEnv,
	})

	// 设置重连配置
	p.client.SetReconnectConfig(p.account.WsMaxReconnectAttempts, "1s,2s,5s,10s,30s,60s")

	// 初始化出站队列
	outbound.InitQueue(p.accountId, &outbound.QueueConfig{
		Strategy:  "merge-text",
		MaxChars:  p.account.MaxChars,
		MinChars:  p.account.HistoryLimit,
		ChunkText: outbound.ChunkMarkdownText,
	})

	// 启动客户端
	return p.client.Connect()
}

// Stop 停止插件
func (p *Plugin) Stop() error {
	p.log.Info("停止元宝插件", logger.F("accountId", p.accountId))

	p.cancel()

	// 销毁出站队列
	outbound.DestroyQueue(p.accountId)

	// 移除成员管理
	member.RemoveMember(p.accountId)

	// 断开WS连接
	if p.client != nil {
		p.client.Disconnect()
	}

	return nil
}

// GetAccountId 获取账号ID
func (p *Plugin) GetAccountId() string {
	return p.accountId
}

// GetState 获取状态
func (p *Plugin) GetState() string {
	if p.client == nil {
		return "disconnected"
	}
	return p.client.GetState()
}

// WsClientCallback 实现

// OnReady 连接就绪
func (p *Plugin) OnReady(data *types.AuthReadyData) {
	p.log.Info("WebSocket连接就绪", map[string]any{"connectId": data.ConnectId})

	if p.runtime != nil && p.runtime.onConnected != nil {
		p.runtime.onConnected()
	}
}

// OnDispatch 消息推送
func (p *Plugin) OnDispatch(pushEvent *ws.PushEvent) {
	p.log.Debug("收到推送", map[string]any{"cmd": pushEvent.Cmd, "module": pushEvent.Module})

	// 解析消息
	msg := p.decodePushEvent(pushEvent)
	if msg == nil {
		return
	}

	chatType := message.InferChatType(msg)
	if !message.HasValidMsgFields(msg) {
		return
	}

	// 群消息需要 @ 机器人才触发回调
	if chatType == "group" && p.account.Config.RequireMention != nil && *p.account.Config.RequireMention && !msg.IsAtBot {
		p.log.Debug("群消息未@机器人，跳过", map[string]any{"groupCode": msg.GroupCode, "from": msg.FromAccount})
		return
	}

	p.log.Info("收到消息", map[string]any{"chatType": chatType, "from": msg.FromAccount})

	// 调用消息处理回调
	if p.runtime != nil && p.runtime.onMessage != nil {
		p.runtime.onMessage(msg, chatType)
	}
}

// OnStateChange 状态变化
func (p *Plugin) OnStateChange(state string) {
	p.log.Info("WebSocket状态变化", logger.F("state", state))

	if state == "disconnected" && p.runtime != nil && p.runtime.onDisconnected != nil {
		p.runtime.onDisconnected()
	}
}

// OnError 错误
func (p *Plugin) OnError(err error) {
	p.log.Error("WebSocket错误", logger.F("error", err.Error()))
}

// OnClose 关闭
func (p *Plugin) OnClose(code int, reason string) {
	p.log.Info("WebSocket关闭", map[string]any{"code": code, "reason": reason})
}

// OnKickout 被踢
func (p *Plugin) OnKickout(data *types.KickoutMsg) {
	p.log.Warn("被踢下线", map[string]any{"status": data.Status, "reason": data.Reason})
}

// OnAuthFailed 认证失败
func (p *Plugin) OnAuthFailed(code int) (*types.WsAuth, error) {
	p.log.Warn("认证失败，尝试刷新Token", logger.F("code", code))

	// 创建HTTP客户端
	httpClient := http.NewClient(p.account)

	// 强制刷新Token
	tokenData, err := httpClient.ForceRefreshSignToken()
	if err != nil {
		return nil, err
	}

	// 更新Bot ID
	if tokenData.BotID != "" {
		p.account.BotID = tokenData.BotID
	}

	return &types.WsAuth{
		BizID:    "ybBot",
		UID:      p.account.BotID,
		Source:   tokenData.Source,
		Token:    tokenData.Token,
		RouteEnv: p.account.Config.RouteEnv,
	}, nil
}

// decodePushEvent 解码推送事件
func (p *Plugin) decodePushEvent(event *ws.PushEvent) *types.InboundMessage {
	// 优先从connData解析
	if len(event.ConnData) > 0 {
		msg, err := ws.DecodeInboundMessageWithBotId(event.ConnData, p.account.BotID)
		if err == nil && msg != nil {
			return msg
		}
	}

	// 尝试从rawData解析JSON
	if len(event.RawData) > 0 {
		msg, err := ws.DecodeInboundMessageFromJSONWithBotId(event.RawData, p.account.BotID)
		if err == nil && msg != nil {
			return msg
		}
	}

	// 尝试从content解析
	if event.Content != "" {
		return p.decodeFromContent(event.Content)
	}

	return nil
}

// decodeFromContent 从content解析
func (p *Plugin) decodeFromContent(content string) *types.InboundMessage {
	// 实际实现中需要解析content
	return nil
}

// SendMessage 发送消息
func (p *Plugin) SendMessage(to string, text string) (*message.SendResult, error) {
	if p.client == nil || p.client.GetState() != "connected" {
		p.log.Warn("发送消息失败：未连接")
		return &message.SendResult{Ok: false, Error: fmt.Errorf("not connected")}, nil
	}

	// 解析目标
	chatType, targetId := parseTarget(to)

	p.log.Debug("发送消息", logger.F("chatType", chatType), logger.F("target", targetId), logger.F("text", text))

	msgBody := message.BuildOutboundMsgBodyFromText(text)

	var result any
	var err error

	if chatType == types.ChatTypeGroup {
		result, err = p.client.SendGroupMessageSimple(targetId, msgBody)
	} else {
		result, err = p.client.SendC2CMessageSimple(targetId, msgBody)
	}

	if err != nil {
		p.log.Error("发送消息失败", logger.F("error", err.Error()))
		return &message.SendResult{Ok: false, Error: err}, nil
	}

	if r, ok := result.(*types.SendMessageResult); ok {
		if r.Code != 0 {
			p.log.Error("发送消息失败", logger.F("code", r.Code), logger.F("message", r.Message))
		}
		return &message.SendResult{
			Ok:        r.Code == 0,
			MessageID: r.MsgID,
			Error:     nil,
		}, nil
	}

	return &message.SendResult{Ok: true}, nil
}

// SendGroupMessage 发送群消息
func (p *Plugin) SendGroupMessage(groupCode string, text string, refMsgId string) (*message.SendResult, error) {
	if p.client == nil || p.client.GetState() != "connected" {
		p.log.Warn("发送群消息失败：未连接")
		return &message.SendResult{Ok: false, Error: fmt.Errorf("not connected")}, nil
	}

	p.log.Debug("发送群消息", logger.F("groupCode", groupCode), logger.F("text", text))

	msgBody := message.BuildOutboundMsgBodyFromText(text)

	result, err := p.client.SendGroupMessageSimple(groupCode, msgBody)
	if err != nil {
		p.log.Error("发送群消息失败", logger.F("error", err.Error()))
		return &message.SendResult{Ok: false, Error: err}, nil
	}

	if result.Code != 0 {
		p.log.Error("发送群消息失败", logger.F("code", result.Code), logger.F("message", result.Message))
	}

	return &message.SendResult{
		Ok:        result.Code == 0,
		MessageID: result.MsgID,
		Error:     nil,
	}, nil
}

// parseTarget 解析目标
func parseTarget(target string) (types.ChatType, string) {
	if len(target) > 6 && target[:6] == "group:" {
		return types.ChatTypeGroup, target[6:]
	}
	return types.ChatTypeC2C, target
}

// SetRuntime 设置运行时
func (p *Plugin) SetRuntime(runtime *Runtime) {
	p.runtime = runtime
}

// SetOnMessage 设置消息处理回调
func (p *Plugin) SetOnMessage(fn func(msg *types.InboundMessage, chatType string)) {
	if p.runtime == nil {
		p.runtime = &Runtime{}
	}
	p.runtime.onMessage = fn
}

// SetOnConnected 设置连接成功回调
func (p *Plugin) SetOnConnected(fn func()) {
	if p.runtime == nil {
		p.runtime = &Runtime{}
	}
	p.runtime.onConnected = fn
}

// SetOnDisconnected 设置断开连接回调
func (p *Plugin) SetOnDisconnected(fn func()) {
	if p.runtime == nil {
		p.runtime = &Runtime{}
	}
	p.runtime.onDisconnected = fn
}

// GetMember 获取成员管理
func (p *Plugin) GetMember() *member.Member {
	return member.GetMember(p.accountId)
}

// PluginManager 插件管理器
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]*Plugin
	log     *logger.Logger
}

// NewPluginManager 创建插件管理器
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]*Plugin),
		log:     logger.New("plugin-manager"),
	}
}

// CreatePlugin 创建插件
func (m *PluginManager) CreatePlugin(accountId string, account *account.Account, cfg *config.Config) *Plugin {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止已有插件
	if existing, ok := m.plugins[accountId]; ok {
		existing.Stop()
	}

	plugin := NewPlugin(accountId, account, cfg)
	m.plugins[accountId] = plugin

	return plugin
}

// GetPlugin 获取插件
func (m *PluginManager) GetPlugin(accountId string) *Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[accountId]
}

// StopPlugin 停止插件
func (m *PluginManager) StopPlugin(accountId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[accountId]
	if !ok {
		return nil
	}

	err := plugin.Stop()
	delete(m.plugins, accountId)
	return err
}

// StopAll 停止所有插件
func (m *PluginManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, plugin := range m.plugins {
		plugin.Stop()
	}

	m.plugins = make(map[string]*Plugin)
	return nil
}

// 全局插件管理器
var (
	globalPluginManager *PluginManager
	pluginManagerOnce   sync.Once
)

// GetPluginManager 获取全局插件管理器
func GetPluginManager() *PluginManager {
	pluginManagerOnce.Do(func() {
		globalPluginManager = NewPluginManager()
	})
	return globalPluginManager
}

// 便捷函数

// CreateAndStart 创建并启动插件
func CreateAndStart(accountId string, account *account.Account, cfg *config.Config) (*Plugin, error) {
	manager := GetPluginManager()
	plugin := manager.CreatePlugin(accountId, account, cfg)

	if err := plugin.Start(); err != nil {
		return nil, err
	}

	return plugin, nil
}

// StopAndRemove 停止并移除插件
func StopAndRemove(accountId string) error {
	return GetPluginManager().StopPlugin(accountId)
}

// GetPluginByAccountId 根据账号ID获取插件
func GetPluginByAccountId(accountId string) *Plugin {
	return GetPluginManager().GetPlugin(accountId)
}

// RunWithContext 运行直到上下文取消
func RunWithContext(ctx context.Context, accountId string, account *account.Account, cfg *config.Config) error {
	plugin, err := CreateAndStart(accountId, account, cfg)
	if err != nil {
		return err
	}

	// 设置消息处理
	plugin.SetOnMessage(func(msg *types.InboundMessage, chatType string) {
		handleMessage(msg, chatType, plugin)
	})

	// 运行直到取消
	<-ctx.Done()

	return plugin.Stop()
}

// handleMessage 处理消息
func handleMessage(msg *types.InboundMessage, chatType string, plugin *Plugin) {
	// 提取消息内容
	result := message.ExtractTextFromMsgBody(msg.MsgBody)

	if result.Text == "" && len(result.Medias) == 0 {
		return
	}

	// 检查是否是@机器人的消息
	if chatType == "group" && result.IsAtBot {
		// 群聊中@机器人的消息，生成回复
		// 实际实现中需要调用AI服务
		generateAndSendReply(msg, result, chatType, plugin)
	} else if chatType == "c2c" {
		// 私聊消息，生成回复
		generateAndSendReply(msg, result, chatType, plugin)
	}
}

// generateAndSendReply 生成并发送回复
func generateAndSendReply(msg *types.InboundMessage, result message.ExtractResult, chatType string, plugin *Plugin) {
	// 获取成员管理
	mem := plugin.GetMember()

	if chatType == "group" {
		mem.RecordUser(msg.GroupCode, msg.FromAccount, msg.SenderNickname)
	} else {
		mem.RecordC2cUser(msg.FromAccount, msg.SenderNickname)
	}

	// 发送"正在输入"状态
	// 实际实现中需要发送心跳

	// 模拟AI回复
	time.Sleep(1 * time.Second)

	// 生成回复
	reply := fmt.Sprintf("收到你的消息: %s", result.Text)

	// 发送回复
	if chatType == "group" {
		plugin.SendGroupMessage(msg.GroupCode, reply, msg.MsgID)
	} else {
		plugin.SendMessage(msg.FromAccount, reply)
	}

	// 发送回复完成状态
	// 实际实现中需要发送心跳
}

// maskString 遮蔽字符串
func maskString(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "..." + s[len(s)-3:]
}
