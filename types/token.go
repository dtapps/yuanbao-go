package types

// TokenCache Token缓存
type TokenCache struct {
	Token      string // Token
	AcquiredAt int64  // 获取时间（秒）
	ExpiresAt  int64  // 过期时间（秒）
}

// TokenResult Token结果
type TokenResult struct {
	AppID      string // 应用ID
	BotID      string // 机器人ID
	Source     string // 来源
	Token      string // Token
	ExpiresIn  int64  // 过期时长（秒）
	AcquiredAt int64  // 获取时间（秒）
}

// TokenRequest Token 请求
type TokenRequest struct {
	AppKey    string `json:"app_key"`   // 应用密钥
	Nonce     string `json:"nonce"`     // 随机数
	Signature string `json:"signature"` // 签名
	Timestamp string `json:"timestamp"` // 时间戳（秒）
}

// TokenResponse Token 响应
type TokenResponse struct {
	Code int       `json:"code,omitempty"` // 状态码
	Msg  string    `json:"msg,omitempty"`  // 状态描述
	Data TokenData `json:"data"`           // Token数据
}

// TokenData Token数据
type TokenData struct {
	BotID      string `json:"bot_id,omitempty"`      // 机器人ID
	Source     string `json:"source,omitempty"`      // 来源
	Token      string `json:"token,omitempty"`       // Token
	Product    string `json:"product,omitempty"`     // 产品
	Duration   int    `json:"duration,omitempty"`    // 过期时间（秒）
	CreateType int    `json:"create_type,omitempty"` // 创建类型
}
