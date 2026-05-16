package types

// 消息类型枚举
type ClawMsgType int

const (
	ClawMsgUnknown ClawMsgType = 0
	ClawMsgGroup   ClawMsgType = 1
	ClawMsgPrivate ClawMsgType = 2
)

// CmdType 命令类型
type CmdType uint32

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
	BizCmdSyncInformation      BizCmd = "sync_information"
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

// Module 模块名枚举
type Module string

// 模块名
const (
	ModuleConnAccess           Module = "conn_access"
	ModuleYuanbaoOpenClawProxy Module = "yuanbao_openclaw_proxy"
)

// BotCommand 机器人命令
type BotCommand struct {
	Name        string `json:"name"`        // 命令名称，如 /help
	Description string `json:"description"` // 命令描述
}

// SyncCommandsData 同步命令数据
type SyncCommandsData struct {
	BotCommands    []BotCommand `json:"bot_commands"`    // 机器人命令列表
	PluginCommands []BotCommand `json:"plugin_commands"` // 插件命令列表
}

// SyncInformationType 同步信息类型
type SyncInformationType int32

const (
	SyncTypeUnspecified SyncInformationType = 0 // 未指定
	SyncTypeCommands    SyncInformationType = 1 // 命令列表
)

// SyncInformationData 同步信息数据
type SyncInformationData struct {
	SyncType      SyncInformationType `json:"sync_type"`      // 同步类型
	BotVersion    string              `json:"bot_version"`    // 机器人版本
	PluginVersion string              `json:"plugin_version"` // 插件版本
	CommandData   SyncCommandsData    `json:"command_data"`   // 命令数据
}

// SyncInformationResponse 同步信息响应
type SyncInformationResponse struct {
	Code    int32  `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
}
