package message

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dtapps/yuanbao-go/types"
)

// ExtractResult 提取结果
type ExtractResult struct {
	RawBody     string
	Text        string
	Medias      []MediaInfo
	IsAtBot     bool
	Mentions    []MentionInfo
	BotUsername string
}

// MediaInfo 媒体信息
type MediaInfo struct {
	Type     string
	URL      string
	UUID     string
	Size     uint32
	FileName string
}

// MentionInfo @提及信息
type MentionInfo struct {
	UserID   string
	NickName string
	Text     string
}

// ExtractTextFromMsgBody 从消息体提取文本
func ExtractTextFromMsgBody(msgBody []types.MsgBodyElement) ExtractResult {
	result := ExtractResult{
		Medias:   make([]MediaInfo, 0),
		Mentions: make([]MentionInfo, 0),
	}

	var textParts []string
	var atBotUserId string

	for _, elem := range msgBody {
		switch elem.MsgType {
		case "TIMTextElem":
			if elem.MsgContent.Text != "" {
				textParts = append(textParts, elem.MsgContent.Text)
			}

		case "TIMCustomElem":
			// 解析自定义消息
			data := elem.MsgContent.Data
			if data != "" {
				var customData map[string]any
				if err := json.Unmarshal([]byte(data), &customData); err == nil {
					// 检查是否是@消息
					if elemType, ok := customData["elem_type"].(float64); ok && elemType == 1002 {
						if text, ok := customData["text"].(string); ok {
							textParts = append(textParts, text)
						}
						if userId, ok := customData["user_id"].(string); ok {
							atBotUserId = userId
						}
					}
				}
			}

		case "TIMImageElem":
			media := MediaInfo{
				Type: "image",
				URL:  elem.MsgContent.URL,
				UUID: elem.MsgContent.UUID,
			}
			result.Medias = append(result.Medias, media)

		case "TIMFileElem":
			media := MediaInfo{
				Type:     "file",
				UUID:     elem.MsgContent.UUID,
				FileName: elem.MsgContent.FileName,
				Size:     elem.MsgContent.FileSize,
			}
			result.Medias = append(result.Medias, media)

		case "TIMVideoFileElem":
			media := MediaInfo{
				Type: "video",
				UUID: elem.MsgContent.UUID,
			}
			result.Medias = append(result.Medias, media)

		case "TIMSoundElem":
			media := MediaInfo{
				Type: "sound",
				UUID: elem.MsgContent.UUID,
			}
			result.Medias = append(result.Medias, media)
		}
	}

	result.RawBody = strings.Join(textParts, "")
	result.Text = strings.TrimSpace(result.RawBody)
	result.IsAtBot = atBotUserId != ""
	result.BotUsername = atBotUserId

	return result
}

// BuildTextMsgBody 构建文本消息体
func BuildTextMsgBody(text string) []types.MsgBodyElement {
	return []types.MsgBodyElement{
		{
			MsgType: "TIMTextElem",
			MsgContent: types.MsgContent{
				Text: text,
			},
		},
	}
}

// BuildImageMsgBody 构建图片消息体
func BuildImageMsgBody(url, uuid string, size uint32) []types.MsgBodyElement {
	return []types.MsgBodyElement{
		{
			MsgType: "TIMImageElem",
			MsgContent: types.MsgContent{
				UUID:     uuid,
				URL:      url,
				FileSize: size,
			},
		},
	}
}

// BuildFileMsgBody 构建文件消息体
func BuildFileMsgBody(url, uuid, fileName string, size uint32) []types.MsgBodyElement {
	return []types.MsgBodyElement{
		{
			MsgType: "TIMFileElem",
			MsgContent: types.MsgContent{
				UUID:     uuid,
				URL:      url,
				FileName: fileName,
				FileSize: size,
			},
		},
	}
}

// BuildCustomMsgBody 构建自定义消息体
func BuildCustomMsgBody(data string) []types.MsgBodyElement {
	return []types.MsgBodyElement{
		{
			MsgType: "TIMCustomElem",
			MsgContent: types.MsgContent{
				Data: data,
			},
		},
	}
}

// BuildAtUserMsgBody 构建@用户消息体
func BuildAtUserMsgBody(userId, nickName string) []types.MsgBodyElement {
	data, _ := json.Marshal(map[string]any{
		"elem_type": 1002,
		"text":      "@" + nickName,
		"user_id":   userId,
	})

	return []types.MsgBodyElement{
		{
			MsgType: "TIMCustomElem",
			MsgContent: types.MsgContent{
				Data: string(data),
			},
		},
	}
}

// PrepareOutboundContent 准备发送内容
func PrepareOutboundContent(text string) []OutboundItem {
	items := make([]OutboundItem, 0)

	if text == "" {
		return items
	}

	// 处理@提及
	atRegex := regexp.MustCompile(`(?:\s|^)@(\S+?)(?:\s|$)`)
	matches := atRegex.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		items = append(items, OutboundItem{
			Type: "text",
			Data: map[string]any{
				"text": text,
			},
		})
		return items
	}

	lastIndex := 0
	for _, match := range matches {
		// 添加匹配前的文本
		if match[0] > lastIndex {
			before := strings.TrimSpace(text[lastIndex:match[0]])
			if before != "" {
				items = append(items, OutboundItem{
					Type: "text",
					Data: map[string]any{
						"text": before,
					},
				})
			}
		}

		// 添加@提及
		nickName := text[match[2]:match[3]]
		items = append(items, OutboundItem{
			Type: "custom",
			Data: map[string]any{
				"elem_type": 1002,
				"text":      "@" + nickName,
				"user_id":   nickName, // 实际使用中需要解析为真实userId
			},
		})

		lastIndex = match[1]
	}

	// 添加剩余文本
	if lastIndex < len(text) {
		remaining := strings.TrimSpace(text[lastIndex:])
		if remaining != "" {
			items = append(items, OutboundItem{
				Type: "text",
				Data: map[string]any{
					"text": remaining,
				},
			})
		}
	}

	return items
}

// OutboundItem 出站消息项
type OutboundItem struct {
	Type string
	Data map[string]any
}

// BuildOutboundMsgBody 构建出站消息体
func BuildOutboundMsgBody(items []OutboundItem) []types.MsgBodyElement {
	msgBody := make([]types.MsgBodyElement, 0)

	for _, item := range items {
		switch item.Type {
		case "text":
			if text, ok := item.Data["text"].(string); ok && text != "" {
				msgBody = append(msgBody, BuildTextMsgBody(text)...)
			}

		case "custom":
			if data, ok := item.Data["data"].(string); ok {
				msgBody = append(msgBody, BuildCustomMsgBody(data)...)
			} else if jsonData, err := json.Marshal(item.Data); err == nil {
				msgBody = append(msgBody, BuildCustomMsgBody(string(jsonData))...)
			}

		case "image":
			url := getString(item.Data, "url")
			uuid := getString(item.Data, "uuid")
			size := getUint32(item.Data, "size")
			msgBody = append(msgBody, BuildImageMsgBody(url, uuid, size)...)

		case "file":
			url := getString(item.Data, "url")
			uuid := getString(item.Data, "uuid")
			fileName := getString(item.Data, "fileName")
			size := getUint32(item.Data, "size")
			msgBody = append(msgBody, BuildFileMsgBody(url, uuid, fileName, size)...)
		}
	}

	return msgBody
}

// BuildOutboundMsgBodyFromText 从文本构建出站消息体
func BuildOutboundMsgBodyFromText(text string) []types.MsgBodyElement {
	items := PrepareOutboundContent(text)
	return BuildOutboundMsgBody(items)
}

// QuoteInfo 引用信息
type QuoteInfo struct {
	ID             string
	SenderID       string
	SenderNickname string
	Desc           string
}

// ParseQuoteFromCloudCustomData 解析引用信息
func ParseQuoteFromCloudCustomData(cloudCustomData string) *QuoteInfo {
	if cloudCustomData == "" {
		return nil
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(cloudCustomData), &data); err != nil {
		return nil
	}

	// 检查是否有引用数据
	quoteData, ok := data["Quote"].(map[string]any)
	if !ok {
		return nil
	}

	info := &QuoteInfo{}

	if id, ok := quoteData["MsgID"].(string); ok {
		info.ID = id
	}
	if senderId, ok := quoteData["UserID"].(string); ok {
		info.SenderID = senderId
	}
	if senderNick, ok := quoteData["NickName"].(string); ok {
		info.SenderNickname = senderNick
	}
	if desc, ok := quoteData["Desc"].(string); ok {
		info.Desc = desc
	}

	return info
}

// FormatQuoteContext 格式化引用上下文
func FormatQuoteContext(info *QuoteInfo) string {
	if info == nil {
		return ""
	}

	desc := info.Desc
	if desc == "" {
		desc = "[引用消息]"
	}

	sender := info.SenderNickname
	if sender == "" {
		sender = info.SenderID
	}

	return fmt.Sprintf("> %s\n> — @%s\n\n", desc, sender)
}

// SendResult 发送结果
type SendResult struct {
	Ok        bool
	MessageID string
	Error     error
}

// InferChatType 推断聊天类型
func InferChatType(msg *types.InboundMessage) string {
	if msg.GroupCode != "" {
		return "group"
	}

	callbackCmd := msg.CallbackCommand
	if callbackCmd == "Group.CallbackAfterRecallMsg" || callbackCmd == "Group.CallbackAfterSendMsg" {
		return "group"
	}

	return "c2c"
}

// HasValidMsgFields 检查消息是否有有效字段
func HasValidMsgFields(msg *types.InboundMessage) bool {
	return msg.CallbackCommand != "" || msg.FromAccount != "" || len(msg.MsgBody) > 0
}

// 辅助函数

func getString(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func getUint32(data map[string]any, key string) uint32 {
	if v, ok := data[key].(float64); ok {
		return uint32(v)
	}
	return 0
}

// NormalizeMarkdownText 规范化Markdown文本
func NormalizeMarkdownText(text string) string {
	// 移除首尾的代码块标记
	for {
		if strings.HasPrefix(text, "```") && strings.HasSuffix(text, "```") {
			text = strings.TrimPrefix(text, "```")
			text = strings.TrimSuffix(text, "```")
			text = strings.Trim(text, "\n")
			continue
		}
		break
	}

	return text
}

// StripOuterMarkdownFence 移除外层Markdown代码块
func StripOuterMarkdownFence(text string) string {
	return NormalizeMarkdownText(text)
}

// Constants 常量
const (
	MarkdownHintText = "请直接输出Markdown内容，不需要用代码块包裹"

	OverflowNoticeText = "⚠️ 消息过长已截断，如需查看完整内容请单独发送"

	FinalTextChunkLimit = 3000
)
