package config

import "github.com/dtapps/yuanbao-go/types"

// DefaultConfig 返回默认配置
func DefaultConfig() *types.YuanbaoConfig {
	return &types.YuanbaoConfig{
		OverflowPolicy:         "split",
		ReplyToMode:            "first",
		OutboundQueueStrategy:  "merge-text",
		MinChars:               2800,
		MaxChars:               3000,
		IdleMs:                 5000,
		MediaMaxMb:             20,
		HistoryLimit:           100,
		DisableBlockStreaming:  boolPtr(false),
		RequireMention:         boolPtr(true),
		FallbackReply:          "暂时无法解答，你可以换个问题问问我哦",
		MarkdownHintEnabled:    boolPtr(true),
		ApiDomain:              "bot.yuanbao.tencent.com",
		WsUrl:                  "wss://bot-wss.yuanbao.tencent.com/wss/connection",
		WsMaxReconnectAttempts: 100,
	}
}

// boolPtr 返回布尔指针
func boolPtr(b bool) *bool {
	return &b
}
