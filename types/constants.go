package types

import "time"

const (
	// 默认的 API 端点
	DefaultWSEndpoint    = "wss://bot-wss.yuanbao.tencent.com/wss/connection"
	DefaultWSOrigin      = "https://yuanbao.tencent.com"
	DefaultTokenEndpoint = "https://bot.yuanbao.tencent.com/api/v5/robotLogic/sign-token"

	// TokenExpireDuration token 过期时间（秒）
	TokenExpireDuration = 24 * time.Hour // 24小时
	// TokenRefreshBuffer token 刷新缓冲时间
	TokenRefreshBuffer = 60 * time.Second // 60秒

	// TokenFetchMaxRetries 获取 Token 最大重试次数
	TokenFetchMaxRetries = 3
	// TokenFetchBaseDelay 获取 Token 基础延迟（毫秒）
	TokenFetchBaseDelay = 1000
	// TokenFetchMaxDelay 获取 Token 最大延迟（毫秒）
	TokenFetchMaxDelay = 5000

	// 默认心跳间隔(秒)
	DefaultHeartbeatInterval = 30 * time.Second
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

// ConnectionState 连接状态
type ConnectionState string

const (
	ConnectionStateIdle         ConnectionState = "idle"         // 空闲
	ConnectionStateConnecting   ConnectionState = "connecting"   // 连接中
	ConnectionStateConnected    ConnectionState = "connected"    // 已连接
	ConnectionStateBackoff      ConnectionState = "backoff"      // 重试中
	ConnectionStateReconnecting ConnectionState = "reconnecting" // 重连中
	ConnectionStateError        ConnectionState = "error"        // 错误
	ConnectionStateStopped      ConnectionState = "stopped"      // 已停止
	ConnectionStateDisconnected ConnectionState = "disconnected" // 已断开
)

// String 连接状态 字符串
func (c ConnectionState) String() string {
	return string(c)
}
