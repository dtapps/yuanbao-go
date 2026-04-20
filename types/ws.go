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
