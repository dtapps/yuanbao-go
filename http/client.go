package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
)

// Client HTTP 客户端
type Client struct {
	client *http.Client
	log    *logger.Logger
}

// defaultHTTPClient 全局默认 HTTP 客户端
var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// SetDefaultHTTPClient 设置全局默认 HTTP 客户端
func SetDefaultHTTPClient(client *http.Client) {
	if client != nil {
		defaultHTTPClient = client
	}
}

// NewClient 创建 HTTP 客户端（使用全局默认配置）
func NewClient() *Client {
	return &Client{
		client: defaultHTTPClient,
		log:    logger.New("http"),
	}
}

// GetJSON 发送 GET 请求并解析 JSON 响应
func (c *Client) GetJSON(url string, result any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("关闭响应体失败",
				logger.F("url", url),
				logger.F("method", req.Method),
				logger.F("error", err.Error()),
			)
		}
	}()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.log.Error("请求失败",
			logger.F("url", url),
			logger.F("method", req.Method),
			logger.F("status_code", resp.StatusCode),
			logger.F("response_body", string(respBody)),
		)
		return fmt.Errorf("请求失败: %d %s - %s", resp.StatusCode, resp.Status, string(respBody))
	}

	c.log.Debug("请求成功",
		logger.F("url", url),
		logger.F("method", req.Method),
		logger.F("status_code", resp.StatusCode),
		logger.F("response_body", string(respBody)),
	)
	return json.Unmarshal(respBody, result)
}

// PostJSON 发送 POST 请求并解析 JSON 响应
func (c *Client) PostJSON(url string, headers map[string]string, body any, result any) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("关闭响应体失败",
				logger.F("url", url),
				logger.F("method", req.Method),
				logger.FS("headers", req.Header.Clone()),
				logger.F("error", err.Error()),
			)
		}
	}()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.log.Error("请求失败",
			logger.F("url", url),
			logger.F("method", req.Method),
			logger.FS("headers", req.Header.Clone()),
			logger.F("status_code", resp.StatusCode),
			logger.FS("request_body", string(jsonData)),
			logger.FS("response_body", string(respBody)),
		)
		return fmt.Errorf("请求失败: %d %s - %s", resp.StatusCode, resp.Status, string(respBody))
	}

	c.log.Debug("请求成功",
		logger.F("url", url),
		logger.F("method", req.Method),
		logger.FS("headers", req.Header.Clone()),
		logger.F("status_code", resp.StatusCode),
		logger.FS("request_body", string(jsonData)),
		logger.FS("response_body", string(respBody)),
	)
	return json.Unmarshal(respBody, result)
}

func AddQueryParams(baseURL string, params map[string]any) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()
	for k, v := range params {
		if v == nil || v == "" {
			continue
		}
		q.Set(k, fmt.Sprintf("%v", v))
	}

	u.RawQuery = q.Encode()
	return u.String()
}
