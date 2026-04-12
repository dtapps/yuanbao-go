package proto

import (
	"github.com/golang/protobuf/proto"
)

// 业务消息类型名称
const (
	MsgTypeSendC2CMessageReq       = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendC2CMessageReq"
	MsgTypeSendC2CMessageRsp       = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendC2CMessageRsp"
	MsgTypeSendGroupMessageReq     = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendGroupMessageReq"
	MsgTypeSendGroupMessageRsp     = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendGroupMessageRsp"
	MsgTypeInboundMessagePush      = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.InboundMessagePush"
	MsgTypeGetGroupMemberListReq   = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.GetGroupMemberListReq"
	MsgTypeGetGroupMemberListRsp   = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.GetGroupMemberListRsp"
	MsgTypeQueryGroupInfoReq       = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.QueryGroupInfoReq"
	MsgTypeQueryGroupInfoRsp       = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.QueryGroupInfoRsp"
	MsgTypeSendPrivateHeartbeatReq = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendPrivateHeartbeatReq"
	MsgTypeSendPrivateHeartbeatRsp = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendPrivateHeartbeatRsp"
	MsgTypeSendGroupHeartbeatReq   = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendGroupHeartbeatReq"
	MsgTypeSendGroupHeartbeatRsp   = "trpc.yuanbao.yuanbao_conn.yuanbao_openclaw_proxy.SendGroupHeartbeatRsp"
)

// ImageInfo 图片信息
type ImageInfo struct {
	Type   uint32 `protobuf:"varint,1,opt,name=type" json:"type,omitempty"`
	Size   uint32 `protobuf:"varint,2,opt,name=size" json:"size,omitempty"`
	Width  uint32 `protobuf:"varint,3,opt,name=width" json:"width,omitempty"`
	Height uint32 `protobuf:"varint,4,opt,name=height" json:"height,omitempty"`
	URL    string `protobuf:"bytes,5,opt,name=url" json:"url,omitempty"`
}

// MsgContent 消息内容
type MsgContent struct {
	Text           string       `protobuf:"bytes,1,opt,name=text" json:"text,omitempty"`
	UUID           string       `protobuf:"bytes,2,opt,name=uuid" json:"uuid,omitempty"`
	ImageFormat    uint32       `protobuf:"varint,3,opt,name=imageFormat" json:"imageFormat,omitempty"`
	Data           string       `protobuf:"bytes,4,opt,name=data" json:"data,omitempty"`
	Desc           string       `protobuf:"bytes,5,opt,name=desc" json:"desc,omitempty"`
	Ext            string       `protobuf:"bytes,6,opt,name=ext" json:"ext,omitempty"`
	Sound          string       `protobuf:"bytes,7,opt,name=sound" json:"sound,omitempty"`
	ImageInfoArray []*ImageInfo `protobuf:"bytes,8,rep,name=imageInfoArray" json:"imageInfoArray,omitempty"`
	Index          uint32       `protobuf:"varint,9,opt,name=index" json:"index,omitempty"`
	URL            string       `protobuf:"bytes,10,opt,name=url" json:"url,omitempty"`
	FileSize       uint32       `protobuf:"varint,11,opt,name=fileSize" json:"fileSize,omitempty"`
	FileName       string       `protobuf:"bytes,12,opt,name=fileName" json:"fileName,omitempty"`
}

// MsgBodyElement 消息体元素
type MsgBodyElement struct {
	MsgType    string      `protobuf:"bytes,1,opt,name=msgType" json:"msgType,omitempty"`
	MsgContent *MsgContent `protobuf:"bytes,2,opt,name=msgContent" json:"msgContent,omitempty"`
}

// LogInfoExt 日志扩展信息
type LogInfoExt struct {
	TraceId string `protobuf:"bytes,1,opt,name=traceId" json:"traceId,omitempty"`
}

// SendC2CMessageReq 发送C2C消息请求
type SendC2CMessageReq struct {
	MsgId       string            `protobuf:"bytes,1,opt,name=msgId" json:"msgId,omitempty"`
	ToAccount   string            `protobuf:"bytes,2,opt,name=toAccount" json:"toAccount,omitempty"`
	FromAccount string            `protobuf:"bytes,3,opt,name=fromAccount" json:"fromAccount,omitempty"`
	MsgRandom   uint32            `protobuf:"varint,4,opt,name=msgRandom" json:"msgRandom,omitempty"`
	MsgBody     []*MsgBodyElement `protobuf:"bytes,5,rep,name=msgBody" json:"msgBody,omitempty"`
	GroupCode   string            `protobuf:"bytes,6,opt,name=groupCode" json:"groupCode,omitempty"`
	MsgSeq      uint64            `protobuf:"varint,7,opt,name=msgSeq" json:"msgSeq,omitempty"`
	LogExt      *LogInfoExt       `protobuf:"bytes,8,opt,name=logExt" json:"logExt,omitempty"`
}

// SendC2CMessageRsp 发送C2C消息响应
type SendC2CMessageRsp struct {
	Code    int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
}

// SendGroupMessageReq 发送群消息请求
type SendGroupMessageReq struct {
	MsgId       string            `protobuf:"bytes,1,opt,name=msgId" json:"msgId,omitempty"`
	GroupCode   string            `protobuf:"bytes,2,opt,name=groupCode" json:"groupCode,omitempty"`
	FromAccount string            `protobuf:"bytes,3,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount   string            `protobuf:"bytes,4,opt,name=toAccount" json:"toAccount,omitempty"`
	Random      string            `protobuf:"bytes,5,opt,name=random" json:"random,omitempty"`
	MsgBody     []*MsgBodyElement `protobuf:"bytes,6,rep,name=msgBody" json:"msgBody,omitempty"`
	RefMsgId    string            `protobuf:"bytes,7,opt,name=refMsgId" json:"refMsgId,omitempty"`
	MsgSeq      uint64            `protobuf:"varint,8,opt,name=msgSeq" json:"msgSeq,omitempty"`
	LogExt      *LogInfoExt       `protobuf:"bytes,9,opt,name=logExt" json:"logExt,omitempty"`
}

// SendGroupMessageRsp 发送群消息响应
type SendGroupMessageRsp struct {
	Code    int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
}

// EnumCLawMsgType 消息类型枚举
type EnumCLawMsgType int32

const (
	CLAW_MSG_UNKNOWN EnumCLawMsgType = 0
	CLAW_MSG_GROUP   EnumCLawMsgType = 1
	CLAW_MSG_PRIVATE EnumCLawMsgType = 2
)

// ImMsgSeq 消息序列
type ImMsgSeq struct {
	MsgSeq uint64 `protobuf:"varint,1,opt,name=msgSeq" json:"msgSeq,omitempty"`
	MsgId  string `protobuf:"bytes,2,opt,name=msgId" json:"msgId,omitempty"`
}

// InboundMessagePush 入站消息推送
type InboundMessagePush struct {
	CallbackCommand      string            `protobuf:"bytes,1,opt,name=callbackCommand" json:"callbackCommand,omitempty"`
	FromAccount          string            `protobuf:"bytes,2,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount            string            `protobuf:"bytes,3,opt,name=toAccount" json:"toAccount,omitempty"`
	SenderNickname       string            `protobuf:"bytes,4,opt,name=senderNickname" json:"senderNickname,omitempty"`
	GroupId              string            `protobuf:"bytes,5,opt,name=groupId" json:"groupId,omitempty"`
	GroupCode            string            `protobuf:"bytes,6,opt,name=groupCode" json:"groupCode,omitempty"`
	GroupName            string            `protobuf:"bytes,7,opt,name=groupName" json:"groupName,omitempty"`
	MsgSeq               uint32            `protobuf:"varint,8,opt,name=msgSeq" json:"msgSeq,omitempty"`
	MsgRandom            uint32            `protobuf:"varint,9,opt,name=msgRandom" json:"msgRandom,omitempty"`
	MsgTime              uint32            `protobuf:"varint,10,opt,name=msgTime" json:"msgTime,omitempty"`
	MsgKey               string            `protobuf:"bytes,11,opt,name=msgKey" json:"msgKey,omitempty"`
	MsgId                string            `protobuf:"bytes,12,opt,name=msgId" json:"msgId,omitempty"`
	MsgBody              []*MsgBodyElement `protobuf:"bytes,13,rep,name=msgBody" json:"msgBody,omitempty"`
	CloudCustomData      string            `protobuf:"bytes,14,opt,name=cloudCustomData" json:"cloudCustomData,omitempty"`
	EventTime            uint32            `protobuf:"varint,15,opt,name=eventTime" json:"eventTime,omitempty"`
	BotOwnerId           string            `protobuf:"bytes,16,opt,name=botOwnerId" json:"botOwnerId,omitempty"`
	RecallMsgSeqList     []*ImMsgSeq       `protobuf:"bytes,17,rep,name=recallMsgSeqList" json:"recallMsgSeqList,omitempty"`
	ClawMsgType          EnumCLawMsgType   `protobuf:"varint,18,opt,name=clawMsgType" json:"clawMsgType,omitempty"`
	PrivateFromGroupCode string            `protobuf:"bytes,19,opt,name=privateFromGroupCode" json:"privateFromGroupCode,omitempty"`
	LogExt               *LogInfoExt       `protobuf:"bytes,20,opt,name=logExt" json:"logExt,omitempty"`
}

// Member 成员信息
type Member struct {
	UserId   string `protobuf:"bytes,1,opt,name=userId" json:"userId,omitempty"`
	NickName string `protobuf:"bytes,2,opt,name=nickName" json:"nickName,omitempty"`
	UserType int32  `protobuf:"varint,3,opt,name=userType" json:"userType,omitempty"`
}

// GetGroupMemberListReq 获取群成员列表请求
type GetGroupMemberListReq struct {
	GroupCode string `protobuf:"bytes,1,opt,name=groupCode" json:"groupCode,omitempty"`
}

// GetGroupMemberListRsp 获取群成员列表响应
type GetGroupMemberListRsp struct {
	Code       int32     `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message    string    `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	MemberList []*Member `protobuf:"bytes,3,rep,name=memberList" json:"memberList,omitempty"`
}

// QueryGroupInfoReq 查询群信息请求
type QueryGroupInfoReq struct {
	GroupCode string `protobuf:"bytes,1,opt,name=groupCode" json:"groupCode,omitempty"`
}

// QueryGroupInfoRsp 查询群信息响应
type QueryGroupInfoRsp struct {
	Code      int32      `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Msg       string     `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
	GroupInfo *GroupInfo `protobuf:"bytes,3,opt,name=groupInfo" json:"groupInfo,omitempty"`
}

// GroupInfo 群信息
type GroupInfo struct {
	GroupName          string `protobuf:"bytes,1,opt,name=groupName" json:"groupName,omitempty"`
	GroupOwnerUserId   string `protobuf:"bytes,2,opt,name=groupOwnerUserId" json:"groupOwnerUserId,omitempty"`
	GroupOwnerNickname string `protobuf:"bytes,3,opt,name=groupOwnerNickname" json:"groupOwnerNickname,omitempty"`
	GroupSize          int32  `protobuf:"varint,4,opt,name=groupSize" json:"groupSize,omitempty"`
}

// EnumHeartbeat 心跳枚举
type EnumHeartbeat int32

const (
	HEARTBEAT_UNKNOWN EnumHeartbeat = 0
	HEARTBEAT_RUNNING EnumHeartbeat = 1
	HEARTBEAT_FINISH  EnumHeartbeat = 2
)

// SendPrivateHeartbeatReq 发送私聊心跳请求
type SendPrivateHeartbeatReq struct {
	FromAccount string        `protobuf:"bytes,1,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount   string        `protobuf:"bytes,2,opt,name=toAccount" json:"toAccount,omitempty"`
	Heartbeat   EnumHeartbeat `protobuf:"varint,3,opt,name=heartbeat" json:"heartbeat,omitempty"`
}

// SendPrivateHeartbeatRsp 发送私聊心跳响应
type SendPrivateHeartbeatRsp struct {
	Code int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
}

// SendGroupHeartbeatReq 发送群聊心跳请求
type SendGroupHeartbeatReq struct {
	FromAccount string        `protobuf:"bytes,1,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount   string        `protobuf:"bytes,2,opt,name=toAccount" json:"toAccount,omitempty"`
	GroupCode   string        `protobuf:"bytes,3,opt,name=groupCode" json:"groupCode,omitempty"`
	SendTime    int64         `protobuf:"varint,4,opt,name=sendTime" json:"sendTime,omitempty"`
	Heartbeat   EnumHeartbeat `protobuf:"varint,5,opt,name=heartbeat" json:"heartbeat,omitempty"`
}

// SendGroupHeartbeatRsp 发送群聊心跳响应
type SendGroupHeartbeatRsp struct {
	Code int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
}

// SendC2CMessageReqWrapper 私聊消息请求包装
type SendC2CMessageReqWrapper struct {
	MsgId       string                   `protobuf:"bytes,1,opt,name=msgId" json:msgId,omitempty"`
	ToAccount   string                   `protobuf:"bytes,2,opt,name=toAccount" json:toAccount,omitempty"`
	FromAccount string                   `protobuf:"bytes,3,opt,name=fromAccount" json:fromAccount,omitempty"`
	MsgRandom   uint32                   `protobuf:"varint,4,opt,name=msgRandom" json:msgRandom,omitempty"`
	MsgBody     []*MsgBodyElementWrapper `protobuf:"bytes,5,rep,name=msgBody" json:msgBody,omitempty"`
	GroupCode   string                   `protobuf:"bytes,6,opt,name=groupCode" json:groupCode,omitempty"`
	MsgSeq      uint64                   `protobuf:"varint,7,opt,name=msgSeq" json:msgSeq,omitempty"`
}

type SendC2CMessageRspWrapper struct {
	Code    int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
}

type SendGroupMessageReqWrapper struct {
	MsgId       string                   `protobuf:"bytes,1,opt,name=msgId" json:"msgId,omitempty"`
	GroupCode   string                   `protobuf:"bytes,2,opt,name=groupCode" json:"groupCode,omitempty"`
	FromAccount string                   `protobuf:"bytes,3,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount   string                   `protobuf:"bytes,4,opt,name=toAccount" json:"toAccount,omitempty"`
	Random      string                   `protobuf:"bytes,5,opt,name=random" json:"random,omitempty"`
	MsgBody     []*MsgBodyElementWrapper `protobuf:"bytes,6,rep,name=msgBody" json:"msgBody,omitempty"`
	RefMsgId    string                   `protobuf:"bytes,7,opt,name=refMsgId" json:"refMsgId,omitempty"`
	MsgSeq      uint64                   `protobuf:"varint,8,opt,name=msgSeq" json:"msgSeq,omitempty"`
	LogExt      *LogInfoExtWrapper       `protobuf:"bytes,9,opt,name=logExt" json:"logExt,omitempty"`
}

type SendGroupMessageRspWrapper struct {
	Code    int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
}

type InboundMessagePushWrapper struct {
	CallbackCommand      string                   `protobuf:"bytes,1,opt,name=callbackCommand" json:"callbackCommand,omitempty"`
	FromAccount          string                   `protobuf:"bytes,2,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount            string                   `protobuf:"bytes,3,opt,name=toAccount" json:"toAccount,omitempty"`
	SenderNickname       string                   `protobuf:"bytes,4,opt,name=senderNickname" json:"senderNickname,omitempty"`
	GroupId              string                   `protobuf:"bytes,5,opt,name=groupId" json:"groupId,omitempty"`
	GroupCode            string                   `protobuf:"bytes,6,opt,name=groupCode" json:"groupCode,omitempty"`
	GroupName            string                   `protobuf:"bytes,7,opt,name=groupName" json:"groupName,omitempty"`
	MsgSeq               uint32                   `protobuf:"varint,8,opt,name=msgSeq" json:"msgSeq,omitempty"`
	MsgRandom            uint32                   `protobuf:"varint,9,opt,name=msgRandom" json:"msgRandom,omitempty"`
	MsgTime              uint32                   `protobuf:"varint,10,opt,name=msgTime" json:"msgTime,omitempty"`
	MsgKey               string                   `protobuf:"bytes,11,opt,name=msgKey" json:"msgKey,omitempty"`
	MsgId                string                   `protobuf:"bytes,12,opt,name=msgId" json:"msgId,omitempty"`
	MsgBody              []*MsgBodyElementWrapper `protobuf:"bytes,13,rep,name=msgBody" json:"msgBody,omitempty"`
	CloudCustomData      string                   `protobuf:"bytes,14,opt,name=cloudCustomData" json:"cloudCustomData,omitempty"`
	EventTime            uint32                   `protobuf:"varint,15,opt,name=eventTime" json:"eventTime,omitempty"`
	BotOwnerId           string                   `protobuf:"bytes,16,opt,name=botOwnerId" json:"botOwnerId,omitempty"`
	RecallMsgSeqList     []*ImMsgSeqWrapper       `protobuf:"bytes,17,rep,name=recallMsgSeqList" json:"recallMsgSeqList,omitempty"`
	ClawMsgType          int32                    `protobuf:"varint,18,opt,name=clawMsgType" json:"clawMsgType,omitempty"`
	PrivateFromGroupCode string                   `protobuf:"bytes,19,opt,name=privateFromGroupCode" json:"privateFromGroupCode,omitempty"`
	LogExt               *LogInfoExtWrapper       `protobuf:"bytes,20,opt,name=logExt" json:"logExt,omitempty"`
}

type GetGroupMemberListReqWrapper struct {
	GroupCode string `protobuf:"bytes,1,opt,name=groupCode" json:"groupCode,omitempty"`
}

type GetGroupMemberListRspWrapper struct {
	Code       int32            `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message    string           `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	MemberList []*MemberWrapper `protobuf:"bytes,3,rep,name=memberList" json:"memberList,omitempty"`
}

type MemberWrapper struct {
	UserId   string `protobuf:"bytes,1,opt,name=userId" json:"userId,omitempty"`
	NickName string `protobuf:"bytes,2,opt,name=nickName" json:"nickName,omitempty"`
	UserType int32  `protobuf:"varint,3,opt,name=userType" json:"userType,omitempty"`
}

type QueryGroupInfoReqWrapper struct {
	GroupCode string `protobuf:"bytes,1,opt,name=groupCode" json:"groupCode,omitempty"`
}

type QueryGroupInfoRspWrapper struct {
	Code      int32             `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Msg       string            `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
	GroupInfo *GroupInfoWrapper `protobuf:"bytes,3,opt,name=groupInfo" json:"groupInfo,omitempty"`
}

type GroupInfoWrapper struct {
	GroupName          string `protobuf:"bytes,1,opt,name=groupName" json:"groupName,omitempty"`
	GroupOwnerUserId   string `protobuf:"bytes,2,opt,name=groupOwnerUserId" json:"groupOwnerUserId,omitempty"`
	GroupOwnerNickname string `protobuf:"bytes,3,opt,name=groupOwnerNickname" json:"groupOwnerNickname,omitempty"`
	GroupSize          int32  `protobuf:"varint,4,opt,name=groupSize" json:"groupSize,omitempty"`
}

type SendPrivateHeartbeatReqWrapper struct {
	FromAccount string `protobuf:"bytes,1,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount   string `protobuf:"bytes,2,opt,name=toAccount" json:"toAccount,omitempty"`
	Heartbeat   int32  `protobuf:"varint,3,opt,name=heartbeat" json:"heartbeat,omitempty"`
}

type SendPrivateHeartbeatRspWrapper struct {
	Code int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
}

type SendGroupHeartbeatReqWrapper struct {
	FromAccount string `protobuf:"bytes,1,opt,name=fromAccount" json:"fromAccount,omitempty"`
	ToAccount   string `protobuf:"bytes,2,opt,name=toAccount" json:"toAccount,omitempty"`
	GroupCode   string `protobuf:"bytes,3,opt,name=groupCode" json:"groupCode,omitempty"`
	SendTime    int64  `protobuf:"varint,4,opt,name=sendTime" json:"sendTime,omitempty"`
	Heartbeat   int32  `protobuf:"varint,5,opt,name=heartbeat" json:"heartbeat,omitempty"`
}

type SendGroupHeartbeatRspWrapper struct {
	Code int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg" json:"msg,omitempty"`
}

type MsgBodyElementWrapper struct {
	MsgType    string             `protobuf:"bytes,1,opt,name=msgType" json:"msgType,omitempty"`
	MsgContent *MsgContentWrapper `protobuf:"bytes,2,opt,name=msgContent" json:"msgContent,omitempty"`
}

type MsgContentWrapper struct {
	Text           string              `protobuf:"bytes,1,opt,name=text" json:"text,omitempty"`
	Image          []byte              `protobuf:"bytes,2,opt,name=image" json:image,omitempty"`
	UUID           string              `protobuf:"bytes,2,opt,name=uuid" json:"uuid,omitempty"`
	ImageFormat    uint32              `protobuf:"varint,3,opt,name=imageFormat" json:"imageFormat,omitempty"`
	Data           string              `protobuf:"bytes,4,opt,name=data" json:"data,omitempty"`
	Desc           string              `protobuf:"bytes,5,opt,name=desc" json:"desc,omitempty"`
	Ext            string              `protobuf:"bytes,6,opt,name=ext" json:"ext,omitempty"`
	Sound          string              `protobuf:"bytes,7,opt,name=sound" json:"sound,omitempty"`
	ImageInfoArray []*ImageInfoWrapper `protobuf:"bytes,8,rep,name=imageInfoArray" json:"imageInfoArray,omitempty"`
	Index          uint32              `protobuf:"varint,9,opt,name=index" json:"index,omitempty"`
	URL            string              `protobuf:"bytes,10,opt,name=url" json:"url,omitempty"`
	FileSize       uint32              `protobuf:"varint,11,opt,name=fileSize" json:"fileSize,omitempty"`
	FileName       string              `protobuf:"bytes,12,opt,name=fileName" json:"fileName,omitempty"`
}

type ImageInfoWrapper struct {
	Type   uint32 `protobuf:"varint,1,opt,name=type" json:"type,omitempty"`
	Size   uint32 `protobuf:"varint,2,opt,name=size" json:"size,omitempty"`
	Width  uint32 `protobuf:"varint,3,opt,name=width" json:"width,omitempty"`
	Height uint32 `protobuf:"varint,4,opt,name=height" json:"height,omitempty"`
	URL    string `protobuf:"bytes,5,opt,name=url" json:"url,omitempty"`
}

type LogInfoExtWrapper struct {
	TraceId string `protobuf:"bytes,1,opt,name=traceId" json:"traceId,omitempty"`
}

type ImMsgSeqWrapper struct {
	MsgSeq uint64 `protobuf:"varint,1,opt,name=msgSeq" json:"msgSeq,omitempty"`
	MsgId  string `protobuf:"bytes,2,opt,name=msgId" json:"msgId,omitempty"`
}

// proto方法实现
func (m *SendC2CMessageReqWrapper) Reset()         { *m = SendC2CMessageReqWrapper{} }
func (m *SendC2CMessageReqWrapper) String() string { return "" }
func (m *SendC2CMessageReqWrapper) ProtoMessage()  {}

func (m *SendC2CMessageRspWrapper) Reset()         { *m = SendC2CMessageRspWrapper{} }
func (m *SendC2CMessageRspWrapper) String() string { return "" }
func (m *SendC2CMessageRspWrapper) ProtoMessage()  {}

func (m *SendGroupMessageReqWrapper) Reset()         { *m = SendGroupMessageReqWrapper{} }
func (m *SendGroupMessageReqWrapper) String() string { return "" }
func (m *SendGroupMessageReqWrapper) ProtoMessage()  {}

func (m *SendGroupMessageRspWrapper) Reset()         { *m = SendGroupMessageRspWrapper{} }
func (m *SendGroupMessageRspWrapper) String() string { return "" }
func (m *SendGroupMessageRspWrapper) ProtoMessage()  {}

func (m *InboundMessagePushWrapper) Reset()         { *m = InboundMessagePushWrapper{} }
func (m *InboundMessagePushWrapper) String() string { return "" }
func (m *InboundMessagePushWrapper) ProtoMessage()  {}

func (m *GetGroupMemberListReqWrapper) Reset()         { *m = GetGroupMemberListReqWrapper{} }
func (m *GetGroupMemberListReqWrapper) String() string { return "" }
func (m *GetGroupMemberListReqWrapper) ProtoMessage()  {}

func (m *GetGroupMemberListRspWrapper) Reset()         { *m = GetGroupMemberListRspWrapper{} }
func (m *GetGroupMemberListRspWrapper) String() string { return "" }
func (m *GetGroupMemberListRspWrapper) ProtoMessage()  {}

func (m *MemberWrapper) Reset()         { *m = MemberWrapper{} }
func (m *MemberWrapper) String() string { return "" }
func (m *MemberWrapper) ProtoMessage()  {}

func (m *QueryGroupInfoReqWrapper) Reset()         { *m = QueryGroupInfoReqWrapper{} }
func (m *QueryGroupInfoReqWrapper) String() string { return "" }
func (m *QueryGroupInfoReqWrapper) ProtoMessage()  {}

func (m *QueryGroupInfoRspWrapper) Reset()         { *m = QueryGroupInfoRspWrapper{} }
func (m *QueryGroupInfoRspWrapper) String() string { return "" }
func (m *QueryGroupInfoRspWrapper) ProtoMessage()  {}

func (m *GroupInfoWrapper) Reset()         { *m = GroupInfoWrapper{} }
func (m *GroupInfoWrapper) String() string { return "" }
func (m *GroupInfoWrapper) ProtoMessage()  {}

func (m *SendPrivateHeartbeatReqWrapper) Reset()         { *m = SendPrivateHeartbeatReqWrapper{} }
func (m *SendPrivateHeartbeatReqWrapper) String() string { return "" }
func (m *SendPrivateHeartbeatReqWrapper) ProtoMessage()  {}

func (m *SendPrivateHeartbeatRspWrapper) Reset()         { *m = SendPrivateHeartbeatRspWrapper{} }
func (m *SendPrivateHeartbeatRspWrapper) String() string { return "" }
func (m *SendPrivateHeartbeatRspWrapper) ProtoMessage()  {}

func (m *SendGroupHeartbeatReqWrapper) Reset()         { *m = SendGroupHeartbeatReqWrapper{} }
func (m *SendGroupHeartbeatReqWrapper) String() string { return "" }
func (m *SendGroupHeartbeatReqWrapper) ProtoMessage()  {}

func (m *SendGroupHeartbeatRspWrapper) Reset()         { *m = SendGroupHeartbeatRspWrapper{} }
func (m *SendGroupHeartbeatRspWrapper) String() string { return "" }
func (m *SendGroupHeartbeatRspWrapper) ProtoMessage()  {}

func (m *MsgBodyElementWrapper) Reset()         { *m = MsgBodyElementWrapper{} }
func (m *MsgBodyElementWrapper) String() string { return "" }
func (m *MsgBodyElementWrapper) ProtoMessage()  {}

func (m *MsgContentWrapper) Reset()         { *m = MsgContentWrapper{} }
func (m *MsgContentWrapper) String() string { return "" }
func (m *MsgContentWrapper) ProtoMessage()  {}

func (m *ImageInfoWrapper) Reset()         { *m = ImageInfoWrapper{} }
func (m *ImageInfoWrapper) String() string { return "" }
func (m *ImageInfoWrapper) ProtoMessage()  {}

func (m *LogInfoExtWrapper) Reset()         { *m = LogInfoExtWrapper{} }
func (m *LogInfoExtWrapper) String() string { return "" }
func (m *LogInfoExtWrapper) ProtoMessage()  {}

func (m *ImMsgSeqWrapper) Reset()         { *m = ImMsgSeqWrapper{} }
func (m *ImMsgSeqWrapper) String() string { return "" }
func (m *ImMsgSeqWrapper) ProtoMessage()  {}

// 初始化业务消息类型注册表
func init() {
	Register(MsgTypeSendC2CMessageReq, func() proto.Message { return &SendC2CMessageReqWrapper{} })
	Register(MsgTypeSendC2CMessageRsp, func() proto.Message { return &SendC2CMessageRspWrapper{} })
	Register(MsgTypeSendGroupMessageReq, func() proto.Message { return &SendGroupMessageReqWrapper{} })
	Register(MsgTypeSendGroupMessageRsp, func() proto.Message { return &SendGroupMessageRspWrapper{} })
	Register(MsgTypeInboundMessagePush, func() proto.Message { return &InboundMessagePushWrapper{} })
	Register(MsgTypeGetGroupMemberListReq, func() proto.Message { return &GetGroupMemberListReqWrapper{} })
	Register(MsgTypeGetGroupMemberListRsp, func() proto.Message { return &GetGroupMemberListRspWrapper{} })
	Register(MsgTypeQueryGroupInfoReq, func() proto.Message { return &QueryGroupInfoReqWrapper{} })
	Register(MsgTypeQueryGroupInfoRsp, func() proto.Message { return &QueryGroupInfoRspWrapper{} })
	Register(MsgTypeSendPrivateHeartbeatReq, func() proto.Message { return &SendPrivateHeartbeatReqWrapper{} })
	Register(MsgTypeSendPrivateHeartbeatRsp, func() proto.Message { return &SendPrivateHeartbeatRspWrapper{} })
	Register(MsgTypeSendGroupHeartbeatReq, func() proto.Message { return &SendGroupHeartbeatReqWrapper{} })
	Register(MsgTypeSendGroupHeartbeatRsp, func() proto.Message { return &SendGroupHeartbeatRspWrapper{} })
}
