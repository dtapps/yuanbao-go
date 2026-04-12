package account

import (
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/config"
	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
)

// Account 账号信息
type Account struct {
	AccountID              string
	Name                   string
	Enabled                bool
	Configured             bool
	AppKey                 string
	AppSecret              string
	BotID                  string
	Token                  string
	ApiDomain              string
	WsGatewayUrl           string
	WsMaxReconnectAttempts int
	OverflowPolicy         string
	ReplyToMode            string
	MediaMaxMb             int
	MaxChars               int
	HistoryLimit           int
	DisableBlockStreaming  bool
	RequireMention         bool
	FallbackReply          string
	MarkdownHintEnabled    bool
	Config                 *config.YuanbaoConfig
}

// Manager 账号管理器
type Manager struct {
	mu       sync.RWMutex
	accounts map[string]*Account
	botIds   map[string]string // accountId -> botId
	log      *logger.Logger
}

// 全局账号管理器
var (
	globalManager *Manager
	managerOnce   sync.Once
)

// GetManager 获取全局账号管理器
func GetManager() *Manager {
	managerOnce.Do(func() {
		globalManager = NewManager()
	})
	return globalManager
}

// NewManager 创建账号管理器
func NewManager() *Manager {
	return &Manager{
		accounts: make(map[string]*Account),
		botIds:   make(map[string]string),
		log:      logger.New("account"),
	}
}

// ResolveAccount 解析账号
func (m *Manager) ResolveAccount(cfg *config.Config, accountId string) *Account {
	m.mu.Lock()
	defer m.mu.Unlock()

	account := &Account{
		AccountID: accountId,
	}

	// 合并配置
	if cfg != nil && cfg.Yuanbao != nil {
		accountConfig := cfg.Yuanbao

		if accountConfig.Enabled != nil {
			account.Enabled = *accountConfig.Enabled
		} else {
			account.Enabled = true
		}

		account.AppKey = accountConfig.AppKey
		account.AppSecret = accountConfig.AppSecret
		account.Token = accountConfig.Token
		account.ApiDomain = accountConfig.ApiDomain
		if account.ApiDomain == "" {
			account.ApiDomain = "bot.yuanbao.tencent.com"
		}
		account.WsGatewayUrl = accountConfig.WsUrl
		if account.WsGatewayUrl == "" {
			account.WsGatewayUrl = "wss://bot-wss.yuanbao.tencent.com/wss/connection"
		}

		account.OverflowPolicy = accountConfig.OverflowPolicy
		if account.OverflowPolicy == "" {
			account.OverflowPolicy = "split"
		}
		account.ReplyToMode = accountConfig.ReplyToMode
		if account.ReplyToMode == "" {
			account.ReplyToMode = "first"
		}

		account.MediaMaxMb = 20
		if accountConfig.MediaMaxMb > 0 {
			account.MediaMaxMb = accountConfig.MediaMaxMb
		}

		account.HistoryLimit = 100
		if accountConfig.HistoryLimit > 0 {
			account.HistoryLimit = accountConfig.HistoryLimit
		}

		if accountConfig.DisableBlockStreaming != nil {
			account.DisableBlockStreaming = *accountConfig.DisableBlockStreaming
		}

		if accountConfig.RequireMention != nil {
			account.RequireMention = *accountConfig.RequireMention
		} else {
			account.RequireMention = true
		}

		account.FallbackReply = accountConfig.FallbackReply
		if accountConfig.MarkdownHintEnabled != nil {
			account.MarkdownHintEnabled = *accountConfig.MarkdownHintEnabled
		} else {
			account.MarkdownHintEnabled = true
		}

		account.WsMaxReconnectAttempts = 100
		if accountConfig.WsMaxReconnectAttempts > 0 {
			account.WsMaxReconnectAttempts = accountConfig.WsMaxReconnectAttempts
		}

		account.Configured = account.AppKey != "" && account.AppSecret != ""
		account.Config = accountConfig
	}

	// 检查 botId 缓存
	if botId, ok := m.botIds[accountId]; ok {
		account.BotID = botId
	}

	m.log.Debug("解析账号", logger.F("accountId", accountId), logger.F("configured", account.Configured))

	return account
}

// SetBotId 设置 Bot ID
func (m *Manager) SetBotId(accountId, botId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.botIds[accountId] = botId
	m.log.Info("设置BotID", logger.F("accountId", accountId), logger.F("botId", botId))
}

// GetBotId 获取 Bot ID
func (m *Manager) GetBotId(accountId string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.botIds[accountId]
}

// GetAccount 获取账号
func (m *Manager) GetAccount(accountId string) *Account {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.accounts[accountId]
}

// SetAccount 设置账号
func (m *Manager) SetAccount(accountId string, account *Account) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accounts[accountId] = account
}

// DeleteAccount 删除账号
func (m *Manager) DeleteAccount(accountId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.accounts, accountId)
	delete(m.botIds, accountId)
}

// ListAccounts 列出所有账号
func (m *Manager) ListAccounts() []*Account {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accounts := make([]*Account, 0, len(m.accounts))
	for _, account := range m.accounts {
		accounts = append(accounts, account)
	}
	return accounts
}

// TokenCache Token缓存
type TokenCache struct {
	mu        sync.RWMutex
	data      *types.AuthResult
	expiresAt int64
}

// TokenCacheManager Token缓存管理器
type TokenCacheManager struct {
	mu     sync.RWMutex
	caches map[string]*TokenCache
	log    *logger.Logger
}

// 全局Token缓存管理器
var (
	globalTokenCache *TokenCacheManager
	tokenCacheOnce   sync.Once
)

// GetTokenCacheManager 获取Token缓存管理器
func GetTokenCacheManager() *TokenCacheManager {
	tokenCacheOnce.Do(func() {
		globalTokenCache = NewTokenCacheManager()
	})
	return globalTokenCache
}

// NewTokenCacheManager 创建Token缓存管理器
func NewTokenCacheManager() *TokenCacheManager {
	return &TokenCacheManager{
		caches: make(map[string]*TokenCache),
		log:    logger.New("token-cache"),
	}
}

// Get 获取缓存的Token
func (m *TokenCacheManager) Get(accountId string) *types.AuthResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cache, ok := m.caches[accountId]
	if !ok {
		m.log.Debug("Token缓存未命中", logger.F("accountId", accountId))
		return nil
	}

	if time.Now().UnixMilli() > cache.expiresAt {
		m.log.Debug("Token缓存已过期", logger.F("accountId", accountId))
		return nil
	}

	m.log.Debug("Token缓存命中", logger.F("accountId", accountId))
	return cache.data
}

// Set 设置Token缓存
func (m *TokenCacheManager) Set(accountId string, data *types.AuthResult, durationMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	expiresAt := time.Now().UnixMilli() + durationMs
	if durationMs <= 0 {
		expiresAt = time.Now().Add(24 * time.Hour).UnixMilli()
	}

	m.caches[accountId] = &TokenCache{
		data:      data,
		expiresAt: expiresAt,
	}

	m.log.Info("Token已缓存", logger.F("accountId", accountId), logger.F("durationMs", durationMs))
}

// Clear 清除缓存
func (m *TokenCacheManager) Clear(accountId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.caches, accountId)
}

// ClearAll 清除所有缓存
func (m *TokenCacheManager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.caches = make(map[string]*TokenCache)
}

// BotIdCache Bot ID缓存
type BotIdCache struct {
	mu        sync.RWMutex
	botIds    map[string]string // accountId -> botId
	expiresAt map[string]int64
}

// 全局Bot ID缓存
var (
	globalBotIdCache *BotIdCache
	botIdCacheOnce   sync.Once
)

// GetBotIdCache 获取Bot ID缓存
func GetBotIdCache() *BotIdCache {
	botIdCacheOnce.Do(func() {
		globalBotIdCache = &BotIdCache{
			botIds:    make(map[string]string),
			expiresAt: make(map[string]int64),
		}
	})
	return globalBotIdCache
}

// Get 获取Bot ID
func (c *BotIdCache) Get(accountId string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Now().UnixMilli() > c.expiresAt[accountId] {
		return ""
	}

	return c.botIds[accountId]
}

// Set 设置Bot ID
func (c *BotIdCache) Set(accountId, botId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.botIds[accountId] = botId
	// 默认24小时过期
	c.expiresAt[accountId] = time.Now().Add(24 * time.Hour).UnixMilli()
}

// ResolveOverflowPolicy 解析溢出策略
func ResolveOverflowPolicy(raw string) string {
	if raw == "stop" {
		return "stop"
	}
	return "split"
}

// ResolveReplyToMode 解析回复模式
func ResolveReplyToMode(raw string) string {
	switch raw {
	case "off", "all":
		return raw
	default:
		return "first"
	}
}
