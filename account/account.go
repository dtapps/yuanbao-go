package account

import (
	"fmt"
	"sync"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
)

// Manager 账号管理器
type Manager struct {
	mu       sync.RWMutex
	accounts map[string]*types.Account // accountID -> Account
	botIDs   map[string]string         // accountID -> botID
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
		accounts: make(map[string]*types.Account),
		botIDs:   make(map[string]string),
		log:      logger.New("account"),
	}
}

// ResolveAccount 解析账号
func (m *Manager) ResolveAccount(cfg *types.Config, accountID string) *types.Account {
	m.mu.Lock()
	defer m.mu.Unlock()

	account := &types.Account{
		AccountID: accountID,
	}

	// 合并配置
	if cfg != nil && cfg.Yuanbao != nil {
		yuanbaoConfig := cfg.Yuanbao

		if yuanbaoConfig.Enabled != nil {
			account.Enabled = *yuanbaoConfig.Enabled
		}

		account.WSEndpoint = types.DefaultWSEndpoint
		if account.WSEndpoint == "" {
			account.WSEndpoint = types.DefaultWSEndpoint
		}

		account.TokenEndpoint = types.DefaultTokenEndpoint
		if account.TokenEndpoint == "" {
			account.TokenEndpoint = types.DefaultTokenEndpoint
		}

		account.AppID = yuanbaoConfig.AppID
		account.AppSecret = yuanbaoConfig.AppSecret
		if account.AppID != "" && account.AppSecret != "" {
			account.Configured = true
		}

		account.OverflowPolicy = yuanbaoConfig.OverflowPolicy
		if account.OverflowPolicy == "" {
			account.OverflowPolicy = "split"
		}
		account.ReplyToMode = yuanbaoConfig.ReplyToMode
		if account.ReplyToMode == "" {
			account.ReplyToMode = "first"
		}

		account.MediaMaxMb = 20
		if yuanbaoConfig.MediaMaxMb > 0 {
			account.MediaMaxMb = yuanbaoConfig.MediaMaxMb
		}

		account.HistoryLimit = 100
		if yuanbaoConfig.HistoryLimit > 0 {
			account.HistoryLimit = yuanbaoConfig.HistoryLimit
		}

		if yuanbaoConfig.DisableBlockStreaming != nil {
			account.DisableBlockStreaming = *yuanbaoConfig.DisableBlockStreaming
		}

		if yuanbaoConfig.RequireMention != nil {
			account.RequireMention = *yuanbaoConfig.RequireMention
		} else {
			account.RequireMention = true
		}

		account.FallbackReply = yuanbaoConfig.FallbackReply
		if yuanbaoConfig.MarkdownHintEnabled != nil {
			account.MarkdownHintEnabled = *yuanbaoConfig.MarkdownHintEnabled
		} else {
			account.MarkdownHintEnabled = true
		}

		account.WsMaxReconnectAttempts = 100
		if yuanbaoConfig.WsMaxReconnectAttempts > 0 {
			account.WsMaxReconnectAttempts = yuanbaoConfig.WsMaxReconnectAttempts
		}

		account.Config = yuanbaoConfig
	}

	// 检查 botId 缓存
	if botId, ok := m.botIDs[accountID]; ok {
		account.BotID = botId
	}

	m.log.Debug("解析账号",
		logger.F("accountID", accountID),
		logger.F("configured", account.Configured),
	)

	return account
}

// AddAccount 设置账号
func (m *Manager) AddAccount(accountID string, account *types.Account) (*types.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.accounts[accountID] = account

	return account, nil
}

// UpdateAccount 更新账号
func (m *Manager) UpdateAccount(accountID string, account *types.Account) (*types.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.accounts[accountID] = account

	return account, nil
}

// DeleteAccount 删除账号
func (m *Manager) DeleteAccount(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.accounts, accountID)
	delete(m.botIDs, accountID)

	return nil
}

// ListAccounts 列出所有账号
func (m *Manager) ListAccounts() *types.AccountListAccountsResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accounts := make([]*types.Account, 0, len(m.accounts))
	for _, account := range m.accounts {
		accounts = append(accounts, account)
	}

	return &types.AccountListAccountsResponse{
		Total:    len(m.accounts),
		Accounts: accounts,
	}
}

// GetAccount 获取账号
func (m *Manager) GetAccount(accountID string) (*types.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if account, ok := m.accounts[accountID]; ok {
		return account, nil
	}

	return &types.Account{AccountID: accountID}, fmt.Errorf("account not found: %s", accountID)
}

// AddBotID 添加 Bot ID
func (m *Manager) AddBotID(accountID string, botID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.botIDs[accountID] = botID

	return nil
}

// UpdateBotID 更新 Bot ID
func (m *Manager) UpdateBotID(accountID string, botID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.botIDs[accountID] = botID

	return nil
}

// DeleteBotID 删除 Bot ID
func (m *Manager) DeleteBotID(accountID string, botID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.botIDs, accountID)

	return nil
}

// ListBotIDs 列出所有 Bot ID
func (m *Manager) ListBotIDs() *types.AccountListBotIDsResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	botIDs := make([]*string, 0, len(m.botIDs))
	for accountID := range m.botIDs {
		botIDs = append(botIDs, &accountID)
	}

	return &types.AccountListBotIDsResponse{
		Total:  len(m.botIDs),
		BotIDs: botIDs,
	}
}

// GetBotID 获取 Bot ID
func (m *Manager) GetBotID(accountID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if botID, ok := m.botIDs[accountID]; ok {
		return botID, nil
	}

	return "", fmt.Errorf("botID not found: %s", accountID)
}
