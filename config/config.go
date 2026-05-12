package config

import "github.com/dtapps/yuanbao-go/types"

// DefaultConfig 返回默认配置
func DefaultConfig() *types.YuanbaoConfig {
	return &types.YuanbaoConfig{
		Enabled:                new(true),
		WSEndpoint:             types.DefaultWSEndpoint,
		TokenEndpoint:          types.DefaultTokenEndpoint,
		OverflowPolicy:         "split",
		ReplyToMode:            "first",
		OutboundQueueStrategy:  "merge-text",
		MinChars:               2800,
		MaxChars:               3000,
		IdleMs:                 5000,
		MediaMaxMb:             20,
		HistoryLimit:           100,
		DisableBlockStreaming:  new(false),
		RequireMention:         new(true),
		FallbackReply:          "暂时无法解答，你可以换个问题问问我哦",
		MarkdownHintEnabled:    new(true),
		WsMaxReconnectAttempts: 100,
	}
}
