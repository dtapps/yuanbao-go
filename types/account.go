package types

// Account 账号信息
type Account struct {
	AccountID              string // 账号ID
	Enabled                bool   // 是否启用
	Configured             bool   // 是否已配置
	AppID                  string // 应用ID
	AppSecret              string // 应用密钥
	BotID                  string // Bot ID
	TokenEndpoint          string // Token 端点
	WSEndpoint             string // WebSocket 端点
	WsMaxReconnectAttempts int    // WebSocket 最大重连尝试次数
	OverflowPolicy         string
	ReplyToMode            string
	MediaMaxMb             int
	MaxChars               int
	HistoryLimit           int
	DisableBlockStreaming  bool
	RequireMention         bool
	FallbackReply          string
	MarkdownHintEnabled    bool

	Config *YuanbaoConfig
}

// AccountListAccountsResponse 列出所有账号 响应
type AccountListAccountsResponse struct {
	Total    int        // 总数
	Accounts []*Account // 账号列表
}

// AccountListBotIDsResponse 列出所有 Bot ID 响应
type AccountListBotIDsResponse struct {
	Total  int       // 总数
	BotIDs []*string // Bot ID列表
}
