package types

// 消息类型枚举
type ClawMsgType int

const (
	ClawMsgUnknown ClawMsgType = 0
	ClawMsgGroup   ClawMsgType = 1
	ClawMsgPrivate ClawMsgType = 2
)

// CmdType 命令类型
type CmdType int

const (
	CmdTypeRequest  CmdType = 0
	CmdTypeResponse CmdType = 1
	CmdTypePush     CmdType = 2
	CmdTypePushAck  CmdType = 3
)

// Cmd 命令枚举
type Cmd string

const (
	CmdAuthBind   Cmd = "auth-bind"
	CmdPing       Cmd = "ping"
	CmdKickout    Cmd = "kickout"
	CmdUpdateMeta Cmd = "update-meta"
)

// BizCmd 业务命令枚举
type BizCmd string

const (
	BizCmdSendC2CMessage       BizCmd = "send_c2c_message"
	BizCmdSendGroupMessage     BizCmd = "send_group_message"
	BizCmdQueryGroupInfo       BizCmd = "query_group_info"
	BizCmdGetGroupMemberList   BizCmd = "get_group_member_list"
	BizCmdSendPrivateHeartbeat BizCmd = "send_private_heartbeat"
	BizCmdSendGroupHeartbeat   BizCmd = "send_group_heartbeat"
)

// WsHeartbeat WebSocket心跳状态
type WsHeartbeat int

const (
	WsHeartbeatUnknown WsHeartbeat = 0
	WsHeartbeatRunning WsHeartbeat = 1
	WsHeartbeatFinish  WsHeartbeat = 2
)

// ChatType 聊天类型
type ChatType string

const (
	ChatTypeC2C   ChatType = "c2c"
	ChatTypeGroup ChatType = "group"
)

// MsgBodyElement 消息体元素
type MsgBodyElement struct {
	MsgType    string     `json:"msg_type"`
	MsgContent MsgContent `json:"msg_content"`
}

// MsgContent 消息内容
type MsgContent struct {
	Text           string      `json:"text,omitempty"`
	UUID           string      `json:"uuid,omitempty"`
	ImageFormat    uint32      `json:"image_format,omitempty"`
	Data           string      `json:"data,omitempty"`
	Desc           string      `json:"desc,omitempty"`
	Ext            string      `json:"ext,omitempty"`
	Sound          string      `json:"sound,omitempty"`
	ImageInfoArray []ImageInfo `json:"image_info_array,omitempty"`
	Index          uint32      `json:"index,omitempty"`
	URL            string      `json:"url,omitempty"`
	FileSize       uint32      `json:"file_size,omitempty"`
	FileName       string      `json:"file_name,omitempty"`
}

// ImageInfo 图片信息
type ImageInfo struct {
	Type   uint32 `json:"type"`
	Size   uint32 `json:"size"`
	Width  uint32 `json:"width"`
	Height uint32 `json:"height"`
	URL    string `json:"url"`
}

// InboundMessage 收到的消息
type InboundMessage struct {
	CallbackCommand      string           `json:"callback_command,omitempty"`
	FromAccount          string           `json:"from_account,omitempty"`
	ToAccount            string           `json:"to_account,omitempty"`
	SenderNickname       string           `json:"sender_nickname,omitempty"`
	GroupID              string           `json:"group_id,omitempty"`
	GroupCode            string           `json:"group_code,omitempty"`
	GroupName            string           `json:"group_name,omitempty"`
	MsgSeq               uint32           `json:"msg_seq,omitempty"`
	MsgRandom            uint32           `json:"msg_random,omitempty"`
	MsgTime              uint32           `json:"msg_time,omitempty"`
	MsgKey               string           `json:"msg_key,omitempty"`
	MsgID                string           `json:"msg_id,omitempty"`
	MsgBody              []MsgBodyElement `json:"msg_body,omitempty"`
	CloudCustomData      string           `json:"cloud_custom_data,omitempty"`
	EventTime            uint32           `json:"event_time,omitempty"`
	BotOwnerID           string           `json:"bot_owner_id,omitempty"`
	RecallMsgSeqList     []ImMsgSeq       `json:"recall_msg_seq_list,omitempty"`
	ClawMsgType          ClawMsgType      `json:"claw_msg_type,omitempty"`
	PrivateFromGroupCode string           `json:"private_from_group_code,omitempty"`
	TraceID              string           `json:"trace_id,omitempty"`
	SeqID                string           `json:"seq_id,omitempty"`
	IsAtBot              bool             `json:"is_at_bot,omitempty"`
}

// ImMsgSeq 消息序列
type ImMsgSeq struct {
	MsgSeq uint64 `json:"msg_seq"`
	MsgID  string `json:"msg_id"`
}

// SendMessageResult 发送消息结果
type SendMessageResult struct {
	MsgID   string `json:"msgId"`
	Code    int32  `json:"code"`
	Message string `json:"message,omitempty"`
}

// GroupInfo 群信息
type GroupInfo struct {
	GroupName          string `json:"group_name"`
	GroupOwnerUserID   string `json:"group_owner_user_id"`
	GroupOwnerNickname string `json:"group_owner_nickname"`
	GroupSize          int32  `json:"group_size"`
}

// QueryGroupInfoResult 查询群信息结果
type QueryGroupInfoResult struct {
	MsgID     string     `json:"msgId"`
	Code      int32      `json:"code"`
	Msg       string     `json:"msg,omitempty"`
	GroupInfo *GroupInfo `json:"group_info,omitempty"`
}

// Member 成员信息
type Member struct {
	UserID   string `json:"user_id"`
	NickName string `json:"nick_name"`
	UserType int32  `json:"user_type"`
}

// GetGroupMemberListResult 获取群成员列表结果
type GetGroupMemberListResult struct {
	MsgID      string   `json:"msgId"`
	Code       int32    `json:"code"`
	Message    string   `json:"message,omitempty"`
	MemberList []Member `json:"member_list"`
}

// SendHeartbeatResult 发送心跳结果
type SendHeartbeatResult struct {
	MsgID   string `json:"msgId"`
	Code    int32  `json:"code"`
	Msg     string `json:"msg,omitempty"`
	Message string `json:"message,omitempty"`
}

// AuthResult 认证结果
type AuthResult struct {
	BotID    string `json:"bot_id"`
	Duration int    `json:"duration"`
	Product  string `json:"product"`
	Source   string `json:"source"`
	Token    string `json:"token"`
}

// WsAuth WebSocket认证信息
type WsAuth struct {
	BizID    string `json:"bizId"`
	UID      string `json:"uid"`
	Source   string `json:"source"`
	Token    string `json:"token"`
	RouteEnv string `json:"routeEnv,omitempty"`
}

// KickoutMsg 踢出消息
type KickoutMsg struct {
	Status          int32  `json:"status"`
	Reason          string `json:"reason"`
	OtherDeviceName string `json:"otherDeviceName,omitempty"`
}

// AuthReadyData 认证就绪数据
type AuthReadyData struct {
	ConnectId string `json:"connectId"`
	Timestamp int64  `json:"timestamp"`
	ClientIp  string `json:"clientIp,omitempty"`
}

// SendC2CMessageReq 发送C2C消息请求
type SendC2CMessageReq struct {
	MsgID       string           `json:"msgId,omitempty"`
	ToAccount   string           `json:"toAccount"`
	FromAccount string           `json:"fromAccount,omitempty"`
	MsgRandom   uint32           `json:"msgRandom,omitempty"`
	MsgBody     []MsgBodyElement `json:"msgBody,omitempty"`
	GroupCode   string           `json:"groupCode,omitempty"`
	MsgSeq      uint64           `json:"msgSeq,omitempty"`
	LogExt      *LogInfoExt      `json:"logExt,omitempty"`
}

// SendGroupMessageReq 发送群消息请求
type SendGroupMessageReq struct {
	MsgID       string           `json:"msgId,omitempty"`
	GroupCode   string           `json:"groupCode"`
	FromAccount string           `json:"fromAccount,omitempty"`
	ToAccount   string           `json:"toAccount,omitempty"`
	Random      string           `json:"random,omitempty"`
	MsgBody     []MsgBodyElement `json:"msgBody,omitempty"`
	RefMsgID    string           `json:"refMsgId,omitempty"`
	MsgSeq      uint64           `json:"msgSeq,omitempty"`
	LogExt      *LogInfoExt      `json:"logExt,omitempty"`
}

// QueryGroupInfoReq 查询群信息请求
type QueryGroupInfoReq struct {
	GroupCode string `json:"groupCode"`
}

// GetGroupMemberListReq 获取群成员列表请求
type GetGroupMemberListReq struct {
	GroupCode string `json:"groupCode"`
}

// SendPrivateHeartbeatReq 发送私聊心跳请求
type SendPrivateHeartbeatReq struct {
	FromAccount string      `json:"fromAccount"`
	ToAccount   string      `json:"toAccount"`
	Heartbeat   WsHeartbeat `json:"heartbeat"`
}

// SendGroupHeartbeatReq 发送群聊心跳请求
type SendGroupHeartbeatReq struct {
	FromAccount string      `json:"fromAccount"`
	ToAccount   string      `json:"toAccount"`
	GroupCode   string      `json:"groupCode"`
	SendTime    int64       `json:"sendTime"`
	Heartbeat   WsHeartbeat `json:"heartbeat"`
}

// LogInfoExt 日志扩展信息
type LogInfoExt struct {
	TraceId string `json:"traceId,omitempty"`
}
