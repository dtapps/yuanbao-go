package http

// import (
// 	"crypto/hmac"
// 	"crypto/rand"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"math/big"
// 	"net/http"
// 	"strings"
// 	"time"

// 	"github.com/dtapps/yuanbao-go/account"
// 	"github.com/dtapps/yuanbao-go/logger"
// 	"github.com/dtapps/yuanbao-go/token"
// 	"github.com/dtapps/yuanbao-go/types"
// )

// const (
// 	// API路径
// 	SignTokenPath    = "/api/v5/robotLogic/sign-token"
// 	UploadInfoPath   = "/api/resource/genUploadInfo"
// 	DownloadInfoPath = "/api/resource/v1/download"

// 	// 可重试的错误码
// 	RetryableSignCode = 10099
// 	// 最大重试次数
// 	SignMaxRetries = 3
// 	// 重试延迟
// 	SignRetryDelayMs = 1000
// 	// Token刷新提前量
// 	CacheRefreshMarginMs = 5 * 60 * 1000
// 	// 最大安全超时
// 	MaxSafeTimeoutMs = 24 * 24 * 3600 * 1000
// 	// HTTP认证重试最大次数
// 	HTTPAuthRetryMax = 1
// )

// var (
// 	pluginVersion   = "1.0.0"
// 	openclawVersion = "1.0.0"
// )

// // SetVersion 设置版本信息
// func SetVersion(plugin, openclaw string) {
// 	pluginVersion = plugin
// 	openclawVersion = openclaw
// }

// // GetPluginVersion 获取插件版本
// func GetPluginVersion() string {
// 	return pluginVersion
// }

// // GetOpenclawVersion 获取OpenClaw版本
// func GetOpenclawVersion() string {
// 	return openclawVersion
// }

// // GetOperationSystem 获取操作系统
// func GetOperationSystem() string {
// 	return "Go"
// }

// // ApiClient HTTP客户端
// type ApiClient struct {
// 	account *types.Account
// 	log     *logger.Logger
// }

// // NewClient 创建HTTP客户端
// func NewApiClient(acc *types.Account) *ApiClient {
// 	return &ApiClient{
// 		account: acc,
// 		log:     logger.New("http"),
// 	}
// }

// // SignTokenResponse 签票响应
// type SignTokenResponse struct {
// 	Code int               `json:"code"`
// 	Msg  string            `json:"msg"`
// 	Data *types.AuthResult `json:"data"`
// }

// // GetSignToken 获取签票
// func (c *ApiClient) GetSignToken() (*types.AuthResult, error) {
// 	// 检查缓存
// 	cache := token.GetTokenCacheManager()
// 	if cached := cache.Get(c.account.AccountID); cached != nil {
// 		c.log.Info("使用缓存Token", logger.F("accountId", c.account.AccountID))
// 		return cached, nil
// 	}

// 	// 执行签票
// 	for attempt := 0; attempt <= SignMaxRetries; attempt++ {
// 		result, err := c.doFetchSignToken()
// 		if err != nil {
// 			if strings.Contains(err.Error(), fmt.Sprintf("code=%d", RetryableSignCode)) && attempt < SignMaxRetries {
// 				c.log.Warn("签票可重试", map[string]any{"attempt": attempt, "error": err.Error()})
// 				time.Sleep(time.Duration(SignRetryDelayMs) * time.Millisecond)
// 				continue
// 			}
// 			return nil, err
// 		}

// 		// 缓存Token
// 		if result.Duration > 0 {
// 			cache.Set(c.account.AccountID, result, int64(result.Duration)*1000)
// 		}

// 		// 缓存Bot ID
// 		if result.BotID != "" {
// 			account.GetManager().AddBotID(c.account.AccountID, result.BotID)
// 		}

// 		return result, nil
// 	}

// 	return nil, fmt.Errorf("签票失败: 超过最大重试次数")
// }

// // ForceRefreshSignToken 强制刷新Token
// func (c *ApiClient) ForceRefreshSignToken() (*types.AuthResult, error) {
// 	cache := token.GetTokenCacheManager()
// 	cache.Clear(c.account.AccountID)
// 	return c.GetSignToken()
// }

// // doFetchSignToken 执行签票请求
// func (c *ApiClient) doFetchSignToken() (*types.AuthResult, error) {
// 	if c.account.AppKey == "" || c.account.AppSecret == "" {
// 		return nil, fmt.Errorf("签票失败: 缺少AppKey或AppSecret")
// 	}

// 	url := fmt.Sprintf("https://%s%s", c.account.ApiDomain, SignTokenPath)

// 	// 生成随机数
// 	nonceBytes := make([]byte, 16)
// 	rand.Read(nonceBytes)
// 	nonce := hex.EncodeToString(nonceBytes)

// 	// 生成时间戳（北京时间 = UTC+8）
// 	loc := time.FixedZone("CST", 8*3600)
// 	bjTime := time.Now().In(loc)
// 	timestamp := bjTime.Format("2006-01-02T15:04:05+08:00")

// 	// 计算签名
// 	signature := c.computeSignature(nonce, timestamp)

// 	// 构建请求体
// 	body := map[string]any{
// 		"app_key":   c.account.AppKey,
// 		"nonce":     nonce,
// 		"signature": signature,
// 		"timestamp": timestamp,
// 	}

// 	bodyBytes, _ := json.Marshal(body)

// 	// 构建请求
// 	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
// 	if err != nil {
// 		return nil, fmt.Errorf("创建请求失败: %w", err)
// 	}

// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("X-AppVersion", pluginVersion)
// 	req.Header.Set("X-OperationSystem", "Go")
// 	req.Header.Set("X-Instance-Id", "16")
// 	req.Header.Set("X-Bot-Version", openclawVersion)

// 	if c.account.Config != nil && c.account.Config.RouteEnv != "" {
// 		req.Header.Set("x-route-env", c.account.Config.RouteEnv)
// 	}

// 	c.log.Info("正在签票", logger.F("url", url))

// 	// 发送请求
// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("签票请求失败: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	respBody, _ := io.ReadAll(resp.Body)

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("签票HTTP错误: %d %s", resp.StatusCode, string(respBody))
// 	}

// 	var result SignTokenResponse
// 	if err := json.Unmarshal(respBody, &result); err != nil {
// 		return nil, fmt.Errorf("解析响应失败: %w", err)
// 	}

// 	if result.Code != 0 {
// 		return nil, fmt.Errorf("签票业务错误: code=%d, msg=%s", result.Code, result.Msg)
// 	}

// 	c.log.Info("签票成功", logger.F("bot_id", result.Data.BotID))

// 	return result.Data, nil
// }

// // computeSignature 计算签名
// func (c *ApiClient) computeSignature(nonce, timestamp string) string {
// 	plain := nonce + timestamp + c.account.AppKey + c.account.AppSecret
// 	h := hmac.New(sha256.New, []byte(c.account.AppSecret))
// 	h.Write([]byte(plain))
// 	return hex.EncodeToString(h.Sum(nil))
// }

// // VerifySignature 验证签名
// func VerifySignature(expected, actual string) bool {
// 	h := hmac.New(sha256.New, []byte(actual))
// 	h.Write([]byte(expected))
// 	return hex.EncodeToString(h.Sum(nil)) == actual
// }

// // YuanbaoPost 发送POST请求
// func (c *ApiClient) YuanbaoPost(path string, body any) ([]byte, error) {
// 	return c.doRequest("POST", path, body)
// }

// // YuanbaoGet 发送GET请求
// func (c *ApiClient) YuanbaoGet(path string, params map[string]string) ([]byte, error) {
// 	return c.doRequest("GET", path+formatQueryParams(params), nil)
// }

// // doRequest 执行请求
// func (c *ApiClient) doRequest(method, path string, body any) ([]byte, error) {
// 	url := fmt.Sprintf("https://%s%s", c.account.ApiDomain, path)
// 	c.log.Debug("发起HTTP请求", logger.F("method", method), logger.F("url", url))

// 	for attempt := 0; attempt <= HTTPAuthRetryMax; attempt++ {
// 		// 获取认证头
// 		authResult, err := c.GetSignToken()
// 		if err != nil {
// 			c.log.Error("获取签票失败", logger.F("error", err.Error()))
// 			return nil, err
// 		}

// 		var bodyBytes []byte
// 		if body != nil {
// 			bodyBytes, _ = json.Marshal(body)
// 		}

// 		req, err := http.NewRequest(method, url, strings.NewReader(string(bodyBytes)))
// 		if err != nil {
// 			return nil, fmt.Errorf("创建请求失败: %w", err)
// 		}

// 		req.Header.Set("Content-Type", "application/json")
// 		req.Header.Set("X-ID", authResult.BotID)
// 		req.Header.Set("X-Token", authResult.Token)
// 		req.Header.Set("X-Source", authResult.Source)
// 		if authResult.Source == "" {
// 			req.Header.Set("X-Source", "web")
// 		}
// 		req.Header.Set("X-AppVersion", pluginVersion)
// 		req.Header.Set("X-OperationSystem", "Go")
// 		req.Header.Set("X-Instance-Id", "16")
// 		req.Header.Set("X-Bot-Version", openclawVersion)

// 		if c.account.Config != nil && c.account.Config.RouteEnv != "" {
// 			req.Header.Set("X-Route-Env", c.account.Config.RouteEnv)
// 		}

// 		resp, err := http.DefaultClient.Do(req)
// 		if err != nil {
// 			c.log.Error("HTTP请求失败", logger.F("error", err.Error()))
// 			return nil, fmt.Errorf("请求失败: %w", err)
// 		}
// 		defer resp.Body.Close()

// 		respBody, _ := io.ReadAll(resp.Body)

// 		// 401需要刷新token重试
// 		if resp.StatusCode == http.StatusUnauthorized && attempt < HTTPAuthRetryMax {
// 			c.log.Warn("收到401，刷新Token后重试")
// 			token.GetTokenCacheManager().Clear(c.account.AccountID)
// 			continue
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
// 		}

// 		// 解析响应
// 		var result map[string]any
// 		if err := json.Unmarshal(respBody, &result); err != nil {
// 			return nil, fmt.Errorf("解析响应失败: %w", err)
// 		}

// 		if code, ok := result["code"].(float64); ok && code != 0 {
// 			return nil, fmt.Errorf("业务错误: code=%d", int(code))
// 		}

// 		return respBody, nil
// 	}

// 	return nil, fmt.Errorf("请求失败: 超过最大重试次数")
// }

// // formatQueryParams 格式化查询参数
// func formatQueryParams(params map[string]string) string {
// 	if len(params) == 0 {
// 		return ""
// 	}

// 	parts := make([]string, 0, len(params))
// 	for k, v := range params {
// 		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
// 	}
// 	return "?" + strings.Join(parts, "&")
// }

// // GenerateNonce 生成随机数
// func GenerateNonce(length int) (string, error) {
// 	const chars = "0123456789abcdef"
// 	result := make([]byte, length)
// 	for i := range result {
// 		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
// 		if err != nil {
// 			return "", err
// 		}
// 		result[i] = chars[n.Int64()]
// 	}
// 	return string(result), nil
// }
