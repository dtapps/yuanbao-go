package token

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/http"
	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
)

// Manager Token 管理器
type Manager struct {
	mu         sync.RWMutex
	cache      *types.TokenCache
	httpClient *http.Client
	log        *logger.Logger
	callback   types.TokenCallback // Token 回调
}

// 全局 Token 管理器
var (
	globalManagers map[string]*Manager
	globalMu       sync.RWMutex
	managerOnce    sync.Once
)

// GetManager 获取指定账号的 Token 管理器
func GetManager(accountID string) *Manager {
	managerOnce.Do(func() {
		globalManagers = make(map[string]*Manager)
	})

	globalMu.RLock()
	mgr, ok := globalManagers[accountID]
	globalMu.RUnlock()
	if ok {
		return mgr
	}

	globalMu.Lock()
	defer globalMu.Unlock()

	if mgr, ok = globalManagers[accountID]; ok {
		return mgr
	}

	mgr = NewManager()
	globalManagers[accountID] = mgr
	return mgr
}

// NewManager 创建 Token 管理器
func NewManager() *Manager {
	return &Manager{
		cache:      nil,
		httpClient: http.NewClient(),
		log:        logger.New("token"),
		callback:   nil,
	}
}

// SetCallback 设置 Token 回调
func (m *Manager) SetCallback(callback types.TokenCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callback = callback
}

// GetCallback 获取 Token 回调
func (m *Manager) GetCallback() types.TokenCallback {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callback
}

// isTokenValid 检查 Token 是否有效
func (m *Manager) isTokenValid() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cache == nil {
		return false
	}

	expiresAt := m.cache.ExpiresAt
	if expiresAt == 0 {
		expiresAt = m.cache.AcquiredAt + int64(types.TokenExpireDuration/time.Second)
	}

	expiresAt = expiresAt - int64(types.TokenRefreshBuffer)

	now := time.Now().Unix()
	isValid := now < expiresAt
	return isValid
}

// GetValidToken 获取有效的 Token（自动刷新）
func (m *Manager) GetValidToken(appID string, appSecret string, tokenEndpoint string) (string, error) {

	if m.isTokenValid() {
		m.mu.RLock()
		token := m.cache.Token
		m.mu.RUnlock()
		return token, nil
	}

	result, err := m.FetchToken(appID, appSecret, tokenEndpoint)
	if err != nil {
		return "", err
	}

	return result.Token, nil
}

// FetchToken 获取新的 Token
func (m *Manager) FetchToken(appID string, appSecret string, tokenEndpoint string) (*types.TokenResult, error) {

	if tokenEndpoint == "" {
		tokenEndpoint = types.DefaultTokenEndpoint
	}

	var lastErr error

	for attempt := 0; attempt <= types.TokenFetchMaxRetries; attempt++ {
		var response *types.TokenResponse
		nonce, timestamp, signature := m.prepareTokenRequest(appID, appSecret)
		err := m.httpClient.PostJSON(tokenEndpoint, map[string]string{
			"X-AppVersion":      types.Version,
			"X-OperationSystem": "Go",
			"X-Instance-Id":     "16",
			"X-Bot-Version":     types.Version,
		}, types.TokenRequest{
			AppKey:    appID,
			Nonce:     nonce,
			Signature: signature,
			Timestamp: timestamp,
		}, &response)

		if err != nil {
			lastErr = err

			if attempt < types.TokenFetchMaxRetries {
				delay := min(time.Duration(types.TokenFetchBaseDelay)*time.Duration(1<<attempt), types.TokenFetchMaxDelay)
				time.Sleep(delay)
				continue
			}

			// 调用回调（失败）
			m.callCallback(&types.TokenCallbackData{
				Status: "error",
				AppID:  appID,
				Error:  lastErr,
			})

			return nil, lastErr
		}

		if response.Data.Token == "" {
			lastErr = fmt.Errorf("响应中缺少 token")

			if attempt < types.TokenFetchMaxRetries {
				delay := min(time.Duration(types.TokenFetchBaseDelay)*time.Duration(1<<attempt), types.TokenFetchMaxDelay)
				time.Sleep(delay)
				continue
			}

			// 调用回调（失败）
			m.callCallback(&types.TokenCallbackData{
				Status: "error",
				AppID:  appID,
				Error:  lastErr,
			})

			return nil, lastErr
		}

		result := &types.TokenResult{
			AppID:      appID,                         // 应用 ID
			BotID:      response.Data.BotID,           // 机器人 ID
			Source:     response.Data.Source,          // 来源
			Token:      response.Data.Token,           // Token
			AcquiredAt: time.Now().Unix(),             // 获取时间（秒）
			ExpiresIn:  int64(response.Data.Duration), // 过期时长（秒）
		}

		m.mu.Lock()
		m.cache = &types.TokenCache{
			Token:      result.Token,                         // Token
			AcquiredAt: result.AcquiredAt,                    // 获取时间（秒）
			ExpiresAt:  result.AcquiredAt + result.ExpiresIn, // 过期时间（秒）
		}
		m.mu.Unlock()

		// 调用回调（成功）
		m.callCallback(&types.TokenCallbackData{
			Status:     "success",
			Token:      result.Token,
			AppID:      appID,
			BotID:      result.BotID,
			Source:     result.Source,
			ExpiresIn:  result.ExpiresIn,
			AcquiredAt: result.AcquiredAt,
			ExpiresAt:  result.AcquiredAt + result.ExpiresIn,
		})

		return result, nil
	}

	return nil, lastErr
}

// callCallback 调用 Token 回调
func (m *Manager) callCallback(data *types.TokenCallbackData) {
	m.mu.RLock()
	callback := m.callback
	m.mu.RUnlock()

	if callback != nil {
		go callback(data)
	}
}

// prepareTokenRequest 准备 Token 请求
func (m *Manager) prepareTokenRequest(appID string, appSecret string) (string, string, string) {

	// 生成随机数
	nonceBytes := make([]byte, 16)
	rand.Read(nonceBytes)
	nonce := hex.EncodeToString(nonceBytes)

	// 生成时间戳（北京时间 = UTC+8）
	loc := time.FixedZone("CST", 8*3600)
	bjTime := time.Now().In(loc)
	timestamp := bjTime.Format("2006-01-02T15:04:05+08:00")

	// 计算签名
	signature := computeSignature(appID, appSecret, nonce, timestamp)

	return nonce, timestamp, signature
}

// GetCachedToken 获取缓存的 Token
func (m *Manager) GetCachedToken() *types.TokenCache {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cache == nil {
		return nil
	}

	return &types.TokenCache{
		Token:      m.cache.Token,      // Token
		AcquiredAt: m.cache.AcquiredAt, // 获取时间（秒）
		ExpiresAt:  m.cache.ExpiresAt,  // 过期时间（秒）
	}
}

// ClearCache 清除缓存的 Token
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache = nil
}

// computeSignature 计算签名
func computeSignature(appKey string, appSecret string, nonce string, timestamp string) string {
	plain := nonce + timestamp + appKey + appSecret
	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write([]byte(plain))
	return hex.EncodeToString(h.Sum(nil))
}
