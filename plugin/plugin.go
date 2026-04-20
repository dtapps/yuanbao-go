package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/member"
	"github.com/dtapps/yuanbao-go/message"
	"github.com/dtapps/yuanbao-go/token"
	"github.com/dtapps/yuanbao-go/types"
	"github.com/dtapps/yuanbao-go/ws"
	bizProto "github.com/dtapps/yuanbao-go/wsproto/biz"
)

// Plugin 插件
type Plugin struct {
	name      string
	version   string
	accountID string
	account   *types.Account
	config    *types.Config
	client    *ws.WsClient
	runtime   *Runtime

	// 日志
	log *logger.Logger

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// Runtime 运行时
type Runtime struct {
	channel *ChannelRuntime
	// onMessage 消息推送
	onMessage func(msg *types.InboundMessage, chatType types.ChatType)
	// onConnected 连接成功
	onConnected func()
	// onDisconnected 断开连接
	onDisconnected func()
	// onError 错误
	onError func(err error)
}

// ChannelRuntime 通道运行时
type ChannelRuntime struct {
	// 发送文本
	SendText func(text string) error
}

// NewPlugin 创建插件
func NewPlugin(accountID string, account *types.Account, cfg *types.Config) *Plugin {
	ctx, cancel := context.WithCancel(context.Background())

	return &Plugin{
		name:      "yuanbao",
		version:   types.Version,
		accountID: accountID,
		account:   account,
		config:    cfg,
		log:       logger.New("plugin"),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动插件
func (p *Plugin) Start() error {
	p.log.Info("启动插件", logger.F("accountID", p.accountID))
	p.log.Info("配置信息",
		logger.F("appID", p.account.AppID),
		logger.FS("appSecret", p.account.AppSecret),
		logger.F("tokenEndpoint", p.account.TokenEndpoint),
		logger.F("wsEndpoint", p.account.WSEndpoint),
	)

	// 获取 Token 管理器
	tokenMgr := token.GetManager(p.accountID)

	// 获取新的 Token
	tokenData, err := tokenMgr.FetchToken(
		p.account.AppID,
		p.account.AppSecret,
		p.account.TokenEndpoint,
	)
	if err != nil {
		return fmt.Errorf("获取Token失败: %w", err)
	}

	// 更新Bot ID
	if tokenData.BotID != "" {
		p.account.BotID = tokenData.BotID
	}

	// 创建WS客户端
	p.client = ws.NewWsClient(
		p.account.WSEndpoint,
		p.account.AccountID,
		p.account.BotID,
		p,
	)

	// 设置认证信息
	p.client.SetAuth(&types.WsAuthData{
		BizID:    "ybBot",                   // 业务ID
		BotID:    p.account.BotID,           // Bot ID
		Source:   tokenData.Source,          // 来源
		Token:    tokenData.Token,           // 令牌
		RouteEnv: p.account.Config.RouteEnv, // 路由环境
		Version:  types.Version,             // 版本号
	})

	// 设置重连配置
	p.client.SetReconnectConfig(
		p.account.WsMaxReconnectAttempts,
		types.DefaultReconnectDelays,
	)

	// 启动客户端
	return p.client.Connect()
}

// Stop 停止插件
func (p *Plugin) Stop() error {
	p.log.Info("停止插件", logger.F("accountID", p.accountID))

	p.cancel()

	// 清空成员管理
	memberMgr := member.GetManager(p.accountID)
	memberMgr.Clear()

	// 断开WS连接
	if p.client != nil {
		if err := p.client.Disconnect(); err != nil {
			p.log.Error("断开连接失败", logger.F("error", err.Error()))
		}
	}

	return nil
}

// GetAccountId 获取账号ID
func (p *Plugin) GetAccountId() string {
	return p.accountID
}

// GetBotId 获取BotID
func (p *Plugin) GetBotId() string {
	return p.account.BotID
}

// GetState 获取状态
func (p *Plugin) GetState() string {
	if p.client == nil {
		return types.ConnectionStateDisconnected.String()
	}
	return p.client.GetState()
}

// OnReady 连接就绪
func (p *Plugin) OnReady(data *types.OnReadyData) {
	p.log.Info("连接就绪", logger.F("ConnectID", data.ConnectID))

	if p.runtime != nil && p.runtime.onConnected != nil {
		p.runtime.onConnected()
	}
}

// OnDispatch 消息推送
func (p *Plugin) OnDispatch(msg *bizProto.InboundMessagePush) {
	p.log.Info("消息推送", logger.F("MsgId", msg.MsgId))

	// 聊天类型
	chatType := message.InferChatType(msg)

	// 转换为 InboundMessage
	inbound := message.ToInboundMessage(msg)

	// 群消息需要 @ 机器人才触发回调
	if chatType == types.ChatTypeGroup && p.account.Config.RequireMention != nil {
		if *p.account.Config.RequireMention && len(inbound.AtList) == 0 {
			p.log.Warn("群消息未@机器人，跳过",
				logger.F("groupCode", msg.GroupCode),
				logger.F("from", msg.FromAccount),
			)
			return
		}
	}

	// 设置账号ID
	inbound.AccountID = p.account.AccountID

	// 设置应用ID
	inbound.AppID = p.account.AppID

	// 设置应用BotID
	inbound.BotID = p.account.BotID

	// 增加成员
	memberMgr := member.GetManager(p.accountID)
	if _, err := memberMgr.AddUser(&types.MemberAddUserRequest{
		UserID:   inbound.SenderID,
		Nickname: inbound.SenderName,
	}); err != nil {
		p.log.Error("添加成员失败", logger.F("error", err.Error()))
	}
	if inbound.GroupID != "" {
		if _, err := memberMgr.AddGroupUser(&types.GroupAddUserRequest{
			GroupID:  inbound.GroupID,
			UserID:   inbound.SenderID,
			Nickname: inbound.SenderName,
		}); err != nil {
			p.log.Error("添加群成员失败", logger.F("error", err.Error()))
		}
	}

	// 调用消息处理回调
	if p.runtime != nil && p.runtime.onMessage != nil {
		p.runtime.onMessage(inbound, chatType)
	}
}

// OnStateChange 状态变化
func (p *Plugin) OnStateChange(state string) {
	p.log.Warn("状态变化", logger.F("state", state))

	if state == types.ConnectionStateDisconnected.String() && p.runtime != nil && p.runtime.onDisconnected != nil {
		p.runtime.onDisconnected()
	}
}

// OnError 错误
func (p *Plugin) OnError(err error) {
	p.log.Error("错误", logger.F("error", err.Error()))

	if p.runtime != nil {
		p.runtime.onError(err)
	}
}

// OnClose 关闭
func (p *Plugin) OnClose(code int, reason string) {
	p.log.Warn("关闭",
		logger.F("code", code),
		logger.F("reason", reason),
	)
}

// OnKickout 被踢
func (p *Plugin) OnKickout(code int, reason string) {
	p.log.Warn("被踢",
		logger.F("code", code),
		logger.F("reason", reason),
	)
}

// OnAuthFailed 认证失败
func (p *Plugin) OnAuthFailed(code int) (*types.WsAuthData, error) {
	p.log.Warn("认证失败，尝试刷新Token", logger.F("code", code))

	// 获取 Token 管理器
	tokenMgr := token.GetManager(p.accountID)

	// 获取新的 Token
	tokenData, err := tokenMgr.FetchToken(
		p.account.AppID,
		p.account.AppSecret,
		p.account.TokenEndpoint,
	)
	if err != nil {
		return nil, err
	}

	// 更新Bot ID
	if tokenData.BotID != "" {
		p.account.BotID = tokenData.BotID
	}

	return &types.WsAuthData{
		BizID:    "ybBot",                   // 业务ID
		BotID:    p.account.BotID,           // Bot ID
		Source:   tokenData.Source,          // 来源
		Token:    tokenData.Token,           // 令牌
		RouteEnv: p.account.Config.RouteEnv, // 路由环境
		Version:  types.Version,             // 版本号
	}, nil
}

// SendMessage 发送消息
func (p *Plugin) SendMessage(msg *types.OutboundC2CMessage) (string, error) {
	if p.client == nil || p.client.GetState() != types.ConnectionStateConnected.String() {
		err := errors.New("not connected")
		p.log.Error("发送消息", logger.F("error", err.Error()))
		return "", err
	}

	messageID, err := p.client.SendC2CMessage(msg.ToUserID, msg.Text)
	if err != nil {
		p.log.Error("发送消息", logger.F("error", err.Error()))
		return "", err
	}

	p.log.Debug("发送消息",
		logger.F("messageID", messageID),
		logger.F("toUserID", msg.ToUserID),
		logger.F("content", msg.Text),
	)

	return messageID, nil
}

// SendGroupMessage 发送群消息
func (p *Plugin) SendGroupMessage(msg *types.OutboundGroupMessage) (string, error) {
	if p.client == nil || p.client.GetState() != types.ConnectionStateConnected.String() {
		err := errors.New("not connected")
		p.log.Error("发送群消息", logger.F("error", err.Error()))
		return "", err
	}

	messageID, err := p.client.SendGroupMessage(msg.ToGroupID, msg.Text, msg.AtList)
	if err != nil {
		p.log.Error("发送群消息", logger.F("error", err.Error()))
		return "", err
	}

	p.log.Debug("发送群消息",
		logger.F("messageID", messageID),
		logger.F("toGroupID", msg.ToGroupID),
		logger.F("content", msg.Text),
		logger.F("atList", msg.AtList),
	)

	return messageID, nil
}

// SetRuntime 设置运行时
func (p *Plugin) SetRuntime(runtime *Runtime) {
	p.runtime = runtime
}

// SetOnMessage 设置消息处理回调
func (p *Plugin) SetOnMessage(fn func(msg *types.InboundMessage, chatType types.ChatType)) {
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

// SetOnError 设置错误回调
func (p *Plugin) SetOnError(fn func(err error)) {
	if p.runtime == nil {
		p.runtime = &Runtime{}
	}
	p.runtime.onError = fn
}

// GetMember 获取成员管理
func (p *Plugin) GetMember() *member.Manager {
	return member.GetManager(p.accountID)
}

// GetTokenManager 获取 Token 管理器
func (p *Plugin) GetTokenManager() *token.Manager {
	return token.GetManager(p.accountID)
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
func (m *PluginManager) CreatePlugin(accountID string, account *types.Account, cfg *types.Config) *Plugin {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止已有插件
	if existing, ok := m.plugins[accountID]; ok {
		if err := existing.Stop(); err != nil {
			m.log.Error("停止已有插件失败", logger.F("accountID", accountID), logger.F("error", err.Error()))
		}
	}

	plugin := NewPlugin(accountID, account, cfg)
	m.plugins[accountID] = plugin

	return plugin
}

// GetPlugin 获取插件
func (m *PluginManager) GetPlugin(accountID string) *Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.plugins[accountID]
}

// StopPlugin 停止插件
func (m *PluginManager) StopPlugin(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, ok := m.plugins[accountID]
	if !ok {
		return nil
	}

	err := plugin.Stop()
	delete(m.plugins, accountID)
	return err
}

// StopAll 停止所有插件
func (m *PluginManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, plugin := range m.plugins {
		if err := plugin.Stop(); err != nil {
			m.log.Error("停止插件失败", logger.F("error", err.Error()))
		}
	}

	m.plugins = make(map[string]*Plugin)
	return nil
}

// CreateAndStart 创建并启动插件
func CreateAndStart(manager *PluginManager, accountID string, account *types.Account, cfg *types.Config) (*Plugin, error) {
	plugin := manager.CreatePlugin(accountID, account, cfg)

	if err := plugin.Start(); err != nil {
		return nil, err
	}

	return plugin, nil
}

// StopAndRemove 停止并移除插件
func StopAndRemove(manager *PluginManager, accountID string) error {
	return manager.StopPlugin(accountID)
}

// GetPluginByAccountId 根据账号ID获取插件
func GetPluginByAccountId(manager *PluginManager, accountID string) *Plugin {
	return manager.GetPlugin(accountID)
}
