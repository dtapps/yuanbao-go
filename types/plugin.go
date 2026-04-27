package types

const (
	Version = "1.0.6"
)

// OnReadyData 连接就绪
type OnReadyData struct {
	ConnectID string // 连接编号
	Timestamp int64  // 连接时间
}

// WsAuthData WebSocket认证信息
type WsAuthData struct {
	BizID    string // 业务ID
	BotID    string // Bot ID
	Source   string // 来源
	Token    string // 令牌
	RouteEnv string // 路由环境
	Version  string // 版本号
}
