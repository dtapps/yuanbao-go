package types

// YuanbaoConfig 元宝配置结构
type YuanbaoConfig struct {
	Enabled                *bool  `json:"enabled,omitempty"`
	AppKey                 string `json:"appKey,omitempty"`
	AppSecret              string `json:"appSecret,omitempty"`
	Token                  string `json:"token,omitempty"`
	OverflowPolicy         string `json:"overflowPolicy,omitempty"`         // "stop" | "split"
	ReplyToMode            string `json:"replyToMode,omitempty"`            // "off" | "first" | "all"
	OutboundQueueStrategy  string `json:"outboundQueueStrategy,omitempty"`  // "immediate" | "merge-text"
	MinChars               int    `json:"minChars,omitempty"`               // 消息聚合最小字符数
	MaxChars               int    `json:"maxChars,omitempty"`               // 单条消息最大字符数
	IdleMs                 int    `json:"idleMs,omitempty"`                 // 空闲自动发送超时
	MediaMaxMb             int    `json:"mediaMaxMb,omitempty"`             // 媒体文件大小上限
	HistoryLimit           int    `json:"historyLimit,omitempty"`           // 群聊上下文历史条数
	DisableBlockStreaming  *bool  `json:"disableBlockStreaming,omitempty"`  // 禁用分块流式输出
	RequireMention         *bool  `json:"requireMention,omitempty"`         // 群聊需要@机器人
	FallbackReply          string `json:"fallbackReply,omitempty"`          // 兜底回复文案
	MarkdownHintEnabled    *bool  `json:"markdownHintEnabled,omitempty"`    // 注入Markdown格式指令
	ApiDomain              string `json:"apiDomain,omitempty"`              // API域名
	WsUrl                  string `json:"wsUrl,omitempty"`                  // WebSocket网关URL
	WsMaxReconnectAttempts int    `json:"wsMaxReconnectAttempts,omitempty"` // 最大重连次数
	RouteEnv               string `json:"routeEnv,omitempty"`               // 路由环境
}

// Config 全局配置
type Config struct {
	Yuanbao *YuanbaoConfig `json:"yuanbao,omitempty"`
}
