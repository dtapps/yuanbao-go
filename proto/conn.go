package proto

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
)

// 消息类型名称常量
const (
	MsgTypeConnMsg      = "trpc.yuanbao.conn_common.ConnMsg"
	MsgTypeAuthBindReq  = "trpc.yuanbao.conn_common.AuthBindReq"
	MsgTypeAuthBindRsp  = "trpc.yuanbao.conn_common.AuthBindRsp"
	MsgTypePingReq      = "trpc.yuanbao.conn_common.PingReq"
	MsgTypePingRsp      = "trpc.yuanbao.conn_common.PingRsp"
	MsgTypeKickoutMsg   = "trpc.yuanbao.conn_common.KickoutMsg"
	MsgTypeDirectedPush = "trpc.yuanbao.conn_common.DirectedPush"
	MsgTypePushMsg      = "trpc.yuanbao.conn_common.PushMsg"
)

// 命令类型
type CmdType int32

const (
	CmdTypeRequest  int32 = 0
	CmdTypeResponse int32 = 1
	CmdTypePush     int32 = 2
	CmdTypePushAck  int32 = 3
)

// 命令
type Cmd string

const (
	CmdAuthBind   Cmd = "auth-bind"
	CmdPing       Cmd = "ping"
	CmdKickout    Cmd = "kickout"
	CmdUpdateMeta Cmd = "update-meta"
)

// 模块名
const (
	ModuleConnAccess Cmd = "conn_access"
)

// 响应码
type RetCode int32

const (
	RetCodeSuccess           RetCode = 0
	RetCodeAuthFail          RetCode = 40100
	RetCodeAuthTokenInvalid  RetCode = 41103
	RetCodeAuthTokenExpired  RetCode = 41104
	RetCodeAlreadyAuth       RetCode = 41101
	RetCodeAuthRetryable     RetCode = 50400
	RetCodeBackendReturnFail RetCode = 90003
)

func (c RetCode) String() string {
	switch c {
	case RetCodeSuccess:
		return "SUCCESS"
	case RetCodeAuthFail:
		return "AUTH_FAIL"
	case RetCodeAuthTokenInvalid:
		return "AUTH_TOKEN_INVALID"
	case RetCodeAuthTokenExpired:
		return "AUTH_TOKEN_EXPIRED"
	case RetCodeAlreadyAuth:
		return "ALREADY_AUTH"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", c)
	}
}

// Head 消息头
type Head struct {
	CmdType int32  `protobuf:"varint,1,opt,name=cmdType" json:"cmdType,omitempty"`
	Cmd     string `protobuf:"bytes,2,opt,name=cmd" json:"cmd,omitempty"`
	SeqNo   uint32 `protobuf:"varint,3,opt,name=seqNo" json:"seqNo,omitempty"`
	MsgId   string `protobuf:"bytes,4,opt,name=msgId" json:"msgId,omitempty"`
	Module  string `protobuf:"bytes,5,opt,name=module" json:"module,omitempty"`
	NeedAck bool   `protobuf:"varint,6,opt,name=needAck" json:"needAck,omitempty"`
	Status  int32  `protobuf:"varint,10,opt,name=status" json:"status,omitempty"`
}

// HeadMeta 头部元数据
type HeadMeta struct {
	Key   string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

// AuthInfo 认证信息
type AuthInfo struct {
	Uid    string `protobuf:"bytes,1,opt,name=uid" json:"uid,omitempty"`
	Source string `protobuf:"bytes,2,opt,name=source" json:"source,omitempty"`
	Token  string `protobuf:"bytes,3,opt,name=token" json:"token,omitempty"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	AppVersion         string `protobuf:"bytes,1,opt,name=appVersion" json:"appVersion,omitempty"`
	AppOperationSystem string `protobuf:"bytes,2,opt,name=appOperationSystem" json:"appOperationSystem,omitempty"`
	InstanceId         string `protobuf:"bytes,10,opt,name=instanceId" json:"instanceId,omitempty"`
	BotVersion         string `protobuf:"bytes,opt,name=botVersion" json:"botVersion,omitempty"`
}

// AuthBindReq 认证绑定请求
type AuthBindReq struct {
	BizId      string      `protobuf:"bytes,1,opt,name=bizId" json:"bizId,omitempty"`
	AuthInfo   *AuthInfo   `protobuf:"bytes,2,opt,name=authInfo" json:"authInfo,omitempty"`
	DeviceInfo *DeviceInfo `protobuf:"bytes,3,opt,name=deviceInfo" json:"deviceInfo,omitempty"`
	EnvName    string      `protobuf:"bytes,opt,name=envName" json:"envName,omitempty"`
}

// AuthBindRsp 认证绑定响应
type AuthBindRsp struct {
	Code      int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message   string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	ConnectId string `protobuf:"bytes,3,opt,name=connectId" json:"connectId,omitempty"`
	Timestamp int64  `protobuf:"varint,4,opt,name=timestamp" json:"timestamp,omitempty"`
	ClientIp  string `protobuf:"bytes,5,opt,name=clientIp" json:"clientIp,omitempty"`
}

// PingReq Ping请求
type PingReq struct {
}

// PingRsp Ping响应
type PingRsp struct {
	HeartInterval uint32 `protobuf:"varint,1,opt,name=heartInterval" json:"heartInterval,omitempty"`
}

// KickoutMsg 踢出消息
type KickoutMsg struct {
	Status          int32  `protobuf:"varint,1,opt,name=status" json:"status,omitempty"`
	Reason          string `protobuf:"bytes,2,opt,name=reason" json:"reason,omitempty"`
	OtherDeviceName string `protobuf:"bytes,3,opt,name=otherDeviceName" json:"otherDeviceName,omitempty"`
}

// DirectedPush 定向推送
type DirectedPush struct {
	Type    uint32 `protobuf:"varint,1,opt,name=type" json:"type,omitempty"`
	Content string `protobuf:"bytes,2,opt,name=content" json:"content,omitempty"`
}

// PushMsg 推送消息
type PushMsg struct {
	Cmd    string `protobuf:"bytes,1,opt,name=cmd" json:"cmd,omitempty"`
	Module string `protobuf:"bytes,2,opt,name=module" json:"module,omitempty"`
	MsgId  string `protobuf:"bytes,3,opt,name=msgId" json:"msgId,omitempty"`
	Data   []byte `protobuf:"bytes,4,opt,name=data" json:"data,omitempty"`
}

// 全局消息注册表
var (
	registry   = make(map[string]func() proto.Message)
	registerMu sync.RWMutex
)

// Register 注册protobuf消息类型
func Register(name string, fn func() proto.Message) {
	registerMu.Lock()
	defer registerMu.Unlock()
	registry[name] = fn
}

// GetMessageType 获取消息类型构造函数
func GetMessageType(name string) (func() proto.Message, bool) {
	registerMu.RLock()
	defer registerMu.RUnlock()
	fn, ok := registry[name]
	return fn, ok
}

// EncodePB 编码protobuf消息
func EncodePB(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

// DecodePB 解码protobuf消息
func DecodePB(name string, data []byte) (proto.Message, error) {
	registerMu.RLock()
	fn, ok := registry[name]
	registerMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown message type: %s", name)
	}

	msg := fn()
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// 初始化消息类型注册表
func init() {
	// 连接相关消息
	Register(MsgTypeConnMsg, func() proto.Message { return &ConnMsgWrapper{} })
	Register(MsgTypeAuthBindReq, func() proto.Message { return &AuthBindReqWrapper{} })
	Register(MsgTypeAuthBindRsp, func() proto.Message { return &AuthBindRspWrapper{} })
	Register(MsgTypePingReq, func() proto.Message { return &PingReqWrapper{} })
	Register(MsgTypePingRsp, func() proto.Message { return &PingRspWrapper{} })
	Register(MsgTypeKickoutMsg, func() proto.Message { return &KickoutMsgWrapper{} })
	Register(MsgTypeDirectedPush, func() proto.Message { return &DirectedPushWrapper{} })
	Register(MsgTypePushMsg, func() proto.Message { return &PushMsgWrapper{} })
}

// ConnMsgWrapper 连接消息包装
type ConnMsgWrapper struct {
	Head *HeadWrapper `protobuf:"bytes,1,opt,name=head" json:"head,omitempty"`
	Data []byte       `protobuf:"bytes,2,opt,name=data" json:"data,omitempty"`
}

func (m *ConnMsgWrapper) Reset()         { *m = ConnMsgWrapper{} }
func (m *ConnMsgWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *ConnMsgWrapper) ProtoMessage()  {}

// HeadWrapper 消息头包装
type HeadWrapper struct {
	CmdType int32  `protobuf:"varint,1,opt,name=cmdType" json:"cmdType,omitempty"`
	Cmd     string `protobuf:"bytes,2,opt,name=cmd" json:"cmd,omitempty"`
	SeqNo   uint32 `protobuf:"varint,3,opt,name=seqNo" json:"seqNo,omitempty"`
	MsgId   string `protobuf:"bytes,4,opt,name=msgId" json:"msgId,omitempty"`
	Module  string `protobuf:"bytes,5,opt,name=module" json:"module,omitempty"`
	NeedAck bool   `protobuf:"varint,6,opt,name=needAck" json:"needAck,omitempty"`
	Status  int32  `protobuf:"varint,10,opt,name=status" json:"status,omitempty"`
}

func (m *HeadWrapper) Reset()         {}
func (m *HeadWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *HeadWrapper) ProtoMessage()  {}

// AuthBindReqWrapper 认证绑定请求包装
type AuthBindReqWrapper struct {
	BizId      string             `protobuf:"bytes,1,opt,name=bizId" json:"bizId,omitempty"`
	AuthInfo   *AuthInfoWrapper   `protobuf:"bytes,2,opt,name=authInfo" json:"authInfo,omitempty"`
	DeviceInfo *DeviceInfoWrapper `protobuf:"bytes,3,opt,name=deviceInfo" json:"deviceInfo,omitempty"`
	EnvName    string             `protobuf:"bytes,5,opt,name=envName" json:"envName,omitempty"`
}

func (m *AuthBindReqWrapper) Reset()         {}
func (m *AuthBindReqWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *AuthBindReqWrapper) ProtoMessage()  {}

// AuthInfoWrapper 认证信息包装
type AuthInfoWrapper struct {
	Uid    string `protobuf:"bytes,1,opt,name=uid" json:"uid,omitempty"`
	Source string `protobuf:"bytes,2,opt,name=source" json:"source,omitempty"`
	Token  string `protobuf:"bytes,3,opt,name=token" json:"token,omitempty"`
}

func (m *AuthInfoWrapper) Reset()         {}
func (m *AuthInfoWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *AuthInfoWrapper) ProtoMessage()  {}

// DeviceInfoWrapper 设备信息包装
type DeviceInfoWrapper struct {
	AppVersion         string `protobuf:"bytes,1,opt,name=appVersion" json:"appVersion,omitempty"`
	AppOperationSystem string `protobuf:"bytes,2,opt,name=appOperationSystem" json:"appOperationSystem,omitempty"`
	InstanceId         string `protobuf:"bytes,10,opt,name=instanceId" json:"instanceId,omitempty"`
	BotVersion         string `protobuf:"bytes,11,opt,name=botVersion" json:"botVersion,omitempty"`
}

func (m *DeviceInfoWrapper) Reset()         {}
func (m *DeviceInfoWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *DeviceInfoWrapper) ProtoMessage()  {}

// AuthBindRspWrapper 认证绑定响应包装
type AuthBindRspWrapper struct {
	Code      int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message   string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
	ConnectId string `protobuf:"bytes,3,opt,name=connectId" json:"connectId,omitempty"`
	Timestamp int64  `protobuf:"varint,4,opt,name=timestamp" json:"timestamp,omitempty"`
	ClientIp  string `protobuf:"bytes,5,opt,name=clientIp" json:"clientIp,omitempty"`
}

func (m *AuthBindRspWrapper) Reset()         {}
func (m *AuthBindRspWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *AuthBindRspWrapper) ProtoMessage()  {}

// PingReqWrapper Ping请求包装
type PingReqWrapper struct{}

func (m *PingReqWrapper) Reset()         {}
func (m *PingReqWrapper) String() string { return "{}" }
func (m *PingReqWrapper) ProtoMessage()  {}

// PingRspWrapper Ping响应包装
type PingRspWrapper struct {
	HeartInterval uint32 `protobuf:"varint,1,opt,name=heartInterval" json:"heartInterval,omitempty"`
}

func (m *PingRspWrapper) Reset()         {}
func (m *PingRspWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *PingRspWrapper) ProtoMessage()  {}

// KickoutMsgWrapper 踢出消息包装
type KickoutMsgWrapper struct {
	Status          int32  `protobuf:"varint,1,opt,name=status" json:"status,omitempty"`
	Reason          string `protobuf:"bytes,2,opt,name=reason" json:"reason,omitempty"`
	OtherDeviceName string `protobuf:"bytes,3,opt,name=otherDeviceName" json:"otherDeviceName,omitempty"`
}

func (m *KickoutMsgWrapper) Reset()         {}
func (m *KickoutMsgWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *KickoutMsgWrapper) ProtoMessage()  {}

// DirectedPushWrapper 定向推送包装
type DirectedPushWrapper struct {
	Type    uint32 `protobuf:"varint,1,opt,name=type" json:"type,omitempty"`
	Content string `protobuf:"bytes,2,opt,name=content" json:"content,omitempty"`
}

func (m *DirectedPushWrapper) Reset()         {}
func (m *DirectedPushWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *DirectedPushWrapper) ProtoMessage()  {}

// PushMsgWrapper 推送消息包装
type PushMsgWrapper struct {
	Cmd    string `protobuf:"bytes,1,opt,name=cmd" json:"cmd,omitempty"`
	Module string `protobuf:"bytes,2,opt,name=module" json:"module,omitempty"`
	MsgId  string `protobuf:"bytes,3,opt,name=msgId" json:"msgId,omitempty"`
	Data   []byte `protobuf:"bytes,4,opt,name=data" json:"data,omitempty"`
}

func (m *PushMsgWrapper) Reset()         {}
func (m *PushMsgWrapper) String() string { return fmt.Sprintf("%+v", *m) }
func (m *PushMsgWrapper) ProtoMessage()  {}

// ConnMsgHeadWrapper 消息头包装
type ConnMsgHeadWrapper struct {
	MsgId   string `protobuf:"bytes,1,opt,name=msgId" json:msgId,omitempty"`
	SeqNo   uint32 `protobuf:"varint,2,opt,name=seqNo" json:seqNo,omitempty"`
	Cmd     string `protobuf:"bytes,3,opt,name=cmd" json:cmd,omitempty"`
	CmdType int32  `protobuf:"varint,4,opt,name=cmdType" json:cmdType,omitempty"` // 必须有这个
	Module  string `protobuf:"bytes,5,opt,name=module" json:module,omitempty"`
	Status  int32  `protobuf:"varint,6,opt,name=status" json:status,omitempty"`
}
