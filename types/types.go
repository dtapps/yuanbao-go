package types

// OutboundC2CMessage 发送C2C消息
type OutboundC2CMessage struct {
	ToUserID string `json:"to_user_id,omitempty"` // 接收用户ID
	Text     string `json:"text,omitempty"`       // 消息内容
}

// OutboundGroupMessage 发送群消息
type OutboundGroupMessage struct {
	ToGroupID    string   `json:"to_group_id,omitempty"`    // 接收群ID
	Text         string   `json:"text,omitempty"`           // 消息内容
	AtList       []AtInfo `json:"at_list,omitempty"`        // 被@的列表
	RefMessageID string   `json:"ref_message_id,omitempty"` // 引用消息ID
}

// InboundMessage 收到的消息
type InboundMessage struct {
	AccountID string `json:"account_id,omitempty"` // 账号ID
	AppID     string `json:"app_id,omitempty"`     // 应用ID
	BotID     string `json:"bot_id,omitempty"`     // BotID
	MessageID string `json:"message_id,omitempty"` // 消息ID

	SenderID   string `json:"sender_id,omitempty"`   // 发送者ID
	SenderName string `json:"sender_name,omitempty"` // 发送者名称
	Timestamp  int64  `json:"timestamp,omitempty"`   // 发送时间戳

	GroupID   string `json:"group_id,omitempty"`   // 群ID
	GroupCode string `json:"group_code,omitempty"` // 群码
	GroupName string `json:"group_name,omitempty"` // 群名称

	RecipientID string `json:"recipient_id,omitempty"` // 接收者ID

	Content []MessageSegment `json:"content,omitempty"` // 消息内容

	AtList []AtInfo `json:"at_list,omitempty"` // 被@的列表

	RawMessage []byte `json:"-"` // 原始数据
}

// 消息内容
type MessageSegment struct {
	Type     string `json:"type"`                // 消息类型 text | image | file
	Text     string `json:"text,omitempty"`      // 文本内容
	Url      string `json:"url,omitempty"`       // 远程资源链接
	Data     string `json:"data,omitempty"`      // Base64 编码的内容
	FileName string `json:"file_name,omitempty"` // 文件名
	FileSize int64  `json:"file_size,omitempty"` // 文件大小
	MimeType string `json:"mime_type,omitempty"` // MIME 类型
}

// AtInfo 被@的列表
type AtInfo struct {
	UserID   string `json:"user_id,omitempty"`   // 用户ID
	UserName string `json:"user_name,omitempty"` // 用户名称
}
