package types

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
	Config                 *YuanbaoConfig
}
