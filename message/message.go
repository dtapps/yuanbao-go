package message

import (
	"encoding/json"
	"fmt"

	"github.com/dtapps/yuanbao-go/types"
	bizProto "github.com/dtapps/yuanbao-go/wsproto/biz"
	connProto "github.com/dtapps/yuanbao-go/wsproto/conn"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/proto"
)

// InferChatType 推断聊天类型
func InferChatType(msg *bizProto.InboundMessagePush) types.ChatType {
	if msg.GroupCode != "" {
		return types.ChatTypeGroup
	}

	callbackCmd := msg.CallbackCommand
	if callbackCmd == "Group.CallbackAfterRecallMsg" || callbackCmd == "Group.CallbackAfterSendMsg" {
		return types.ChatTypeGroup
	}

	return types.ChatTypeC2C
}

// ToInboundMessage WsMessageMsg 转换为 InboundMessage
func ToInboundMessage(m *bizProto.InboundMessagePush) *types.InboundMessage {

	// 注意：
	//	C2C 中 ToAccount 为小龙虾 ID，如果设置 RecipientID 为 ToAccount 就报错

	// 结构体转换为[]byte
	rawMessage, _ := json.Marshal(m)

	inbound := &types.InboundMessage{
		MessageID: m.MsgId, // 消息ID

		SenderID:   m.FromAccount,    // 发送者ID
		SenderName: m.SenderNickname, // 发送者名称
		Timestamp:  int64(m.MsgTime), // 发送时间戳

		GroupID:   m.GroupId,   // 群ID
		GroupCode: m.GroupCode, // 群码
		GroupName: m.GroupName, // 群名称

		RecipientID: m.FromAccount, // 接收者ID

		RawMessage: rawMessage, // 原始数据
	}

	// 处理消息内容
	for _, item := range m.MsgBody {
		if item.MsgType == "TIMTextElem" {
			inbound.Content = append(inbound.Content, types.MessageSegment{
				Type: "text",               // 消息类型 text | image | file
				Text: item.MsgContent.Text, // 文本内容
			})
		}
		if item.MsgType == "TIMCustomElem" {
			elemType := gjson.Get(item.MsgContent.Data, "elem_type").Int()
			if elemType == 1002 {
				// 被 @ 了
				inbound.AtList = append(inbound.AtList, types.AtInfo{
					UserID:   gjson.Get(item.MsgContent.Data, "user_id").String(),   // 用户ID
					UserName: gjson.Get(item.MsgContent.Data, "user_name").String(), // 用户名称
				})
			}
		}
		if item.MsgType == "TIMImageElem" {
			for _, image := range item.MsgContent.ImageInfoArray {
				inbound.Content = append(inbound.Content, types.MessageSegment{
					Type:     "image",           // 消息类型 text | image | file
					Url:      image.Url,         // 远程资源链接
					FileSize: int64(image.Size), // 文件大小
				})
			}
		}
		if item.MsgType == "TIMFileElem" {
			inbound.Content = append(inbound.Content, types.MessageSegment{
				Type:     "file",                          // 消息类型 text | image | file
				Url:      item.MsgContent.Url,             // 远程资源链接
				FileName: item.MsgContent.FileName,        // 文件名
				FileSize: int64(item.MsgContent.FileSize), // 文件大小
			})
		}
		if item.MsgType == "TIMFaceElem" {
			// 表情消息，暂不处理
			_ = item.MsgType // 避免空分支警告
		}
	}

	return inbound
}

type BuildPushAckMessageParams struct {
	SeqNo uint32
}

// BuildPushAckMessage 构建 PushAck 消息
func BuildPushAckMessage(head *connProto.Head, params BuildPushAckMessageParams) ([]byte, error) {

	// 公共数据
	connMsg := &connProto.ConnMsg{
		Head: &connProto.Head{
			CmdType: uint32(types.CmdTypePushAck),
			Cmd:     head.Cmd,
			SeqNo:   params.SeqNo,
			MsgId:   head.MsgId,
			Module:  head.Module,
		},
	}

	// Protobuf 编码
	data, err := proto.Marshal(connMsg)
	if err != nil {
		return nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	return data, nil
}

// ParsePingMessage 解析 ping 消息
func ParsePingMessage(data []byte) (*connProto.PingRsp, error) {

	if len(data) == 0 {
		return nil, fmt.Errorf("空数据")
	}

	ping := &connProto.PingRsp{}

	// 解析数据
	if err := UnmarshalAny(data, ping); err != nil {
		return nil, err
	}

	return ping, nil
}

type BuildPingRequestMessageParams struct {
	SeqNo uint32
}

// BuildPingRequestMessage 构建 ping 请求消息
func BuildPingRequestMessage(params BuildPingRequestMessageParams) (string, []byte, error) {

	// 业务数据
	pingReq := &connProto.PingReq{}

	// Protobuf 编码
	reqData, err := proto.Marshal(pingReq)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	// 公共数据
	connMsg := &connProto.ConnMsg{
		Head: &connProto.Head{
			CmdType: uint32(types.CmdTypeRequest),
			Cmd:     string(types.CmdPing),
			SeqNo:   params.SeqNo,
			MsgId:   GenerateMsgID(),
			Module:  string(types.ModuleConnAccess),
		},
		Data: reqData,
	}

	// Protobuf 编码
	data, err := proto.Marshal(connMsg)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	return connMsg.Head.MsgId, data, nil
}

type BuildAuthBindRequestMessageParams struct {
	SeqNo    uint32
	BizID    string
	UID      string
	Source   string
	Token    string
	RouteEnv string
}

// BuildAuthBindRequestMessage 构建 auth-bind 请求消息
func BuildAuthBindRequestMessage(params BuildAuthBindRequestMessageParams) (string, []byte, error) {

	// 业务数据
	authBindReq := &connProto.AuthBindReq{
		BizId: params.BizID,
		AuthInfo: &connProto.AuthInfo{
			Uid:    params.UID,
			Source: params.Source,
			Token:  params.Token,
		},
		DeviceInfo: &connProto.DeviceInfo{
			AppVersion:         "1.0.0",
			AppOperationSystem: "Go",
			InstanceId:         types.InstanceId,
			BotVersion:         "1.0.0",
		},
	}
	if params.RouteEnv != "" {
		authBindReq.EnvName = params.RouteEnv
	}

	// Protobuf 编码
	reqData, err := proto.Marshal(authBindReq)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	// 公共数据
	connMsg := &connProto.ConnMsg{
		Head: &connProto.Head{
			CmdType: uint32(types.CmdTypeRequest),
			Cmd:     string(types.CmdAuthBind),
			SeqNo:   params.SeqNo,
			MsgId:   GenerateMsgID(),
			Module:  string(types.ModuleConnAccess),
		},
		Data: reqData,
	}

	// Protobuf 编码
	data, err := proto.Marshal(connMsg)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	return connMsg.Head.MsgId, data, nil
}

// ParseInboundMessageMessage 解析 inbound_message 消息
func ParseInboundMessageMessage(data []byte) (*bizProto.InboundMessagePush, error) {

	if len(data) == 0 {
		return nil, fmt.Errorf("空数据")
	}

	inbound := &bizProto.InboundMessagePush{}

	// 解析数据
	if err := UnmarshalAny(data, inbound); err != nil {
		return nil, err
	}

	return inbound, nil
}

type BuildInboundMessageC2CMessageParams struct {
	SeqNo       uint32
	ToAccount   string
	FromAccount string
	MsgSeq      uint64
	Text        string
}

// BuildInboundMessageC2CMessage 构建 inbound_message C2C 消息
func BuildInboundMessageC2CMessage(params BuildInboundMessageC2CMessageParams) (string, []byte, error) {

	// 业务数据
	c2cMessageReq := &bizProto.SendC2CMessageReq{
		MsgId:       GenerateMsgID(),
		ToAccount:   params.ToAccount,
		FromAccount: params.FromAccount,
		MsgRandom:   GenerateMsgRandom(),
		MsgBody:     PrepareMsgBodyElement(params.Text, nil),
		MsgSeq:      params.MsgSeq,
	}

	// Protobuf 编码
	reqData, err := proto.Marshal(c2cMessageReq)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	// 公共数据
	connMsg := &connProto.ConnMsg{
		Head: &connProto.Head{
			CmdType: uint32(types.CmdTypeRequest),
			Cmd:     string(types.BizCmdSendC2CMessage),
			SeqNo:   params.SeqNo,
			MsgId:   c2cMessageReq.MsgId,
			Module:  string(types.ModuleYuanbaoOpenClawProxy),
		},
		Data: reqData,
	}

	// Protobuf 编码
	data, err := proto.Marshal(connMsg)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	return c2cMessageReq.MsgId, data, nil
}

type BuildInboundMessageGroupMessageParams struct {
	SeqNo       uint32
	GroupCode   string
	FromAccount string
	RefMsgID    string
	TraceID     string
	Text        string
	AtList      []types.AtInfo
}

// BuildInboundMessageGroupMessage 构建 inbound_message Group 消息
func BuildInboundMessageGroupMessage(params BuildInboundMessageGroupMessageParams) (string, []byte, error) {

	// 业务数据
	groupMessageReq := &bizProto.SendGroupMessageReq{
		MsgId:       GenerateMsgID(),
		GroupCode:   params.GroupCode,
		FromAccount: params.FromAccount,
		Random:      GenerateRandom(),
		RefMsgId:    params.RefMsgID,
		MsgBody:     PrepareMsgBodyElement(params.Text, params.AtList),
	}

	if groupMessageReq.LogExt != nil {
		groupMessageReq.LogExt = &bizProto.LogInfoExt{
			TraceId: params.TraceID,
		}
	}

	// Protobuf 编码
	reqData, err := proto.Marshal(groupMessageReq)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	// 公共数据
	connMsg := &connProto.ConnMsg{
		Head: &connProto.Head{
			CmdType: uint32(types.CmdTypeRequest),
			Cmd:     string(types.BizCmdSendGroupMessage),
			SeqNo:   params.SeqNo,
			MsgId:   groupMessageReq.MsgId,
			Module:  string(types.ModuleYuanbaoOpenClawProxy),
		},
		Data: reqData,
	}

	// Protobuf 编码
	data, err := proto.Marshal(connMsg)
	if err != nil {
		return "", nil, fmt.Errorf("protobuf 编码失败: %w", err)
	}

	return groupMessageReq.MsgId, data, nil
}

// PrepareMsgBodyElement 准备消息体元素
func PrepareMsgBodyElement(text string, atList []types.AtInfo) []*bizProto.MsgBodyElement {

	items := make([]*bizProto.MsgBodyElement, 0)
	if text == "" {
		return items
	}

	// 添加主文本
	if text != "" {
		items = append(items, createTextElement(text))
	}

	// 添加 @ 列表
	for _, at := range atList {
		items = append(items, createAtElement(at.UserID, at.UserName))
	}

	return items
}

func createTextElement(text string) *bizProto.MsgBodyElement {
	return &bizProto.MsgBodyElement{
		MsgType: "TIMTextElem",
		MsgContent: &bizProto.MsgContent{
			Text: text,
		},
	}
}

func createAtElement(userID string, nickname string) *bizProto.MsgBodyElement {
	internalData := map[string]any{
		"elem_type": 1002,
		"text":      "@" + nickname,
		"user_id":   userID,
		"content":   "",
	}

	dataBytes, _ := json.Marshal(internalData)

	return &bizProto.MsgBodyElement{
		MsgType: "TIMCustomElem",
		MsgContent: &bizProto.MsgContent{
			Data: string(dataBytes),
			Desc: "@" + nickname,
		},
	}
}
