# Yuanbao Go SDK

腾讯元宝智能机器人 Go 语言客户端，支持 WebSocket 长连接、私聊和群聊消息收发。

参考 [openclaw-plugin-yuanbao](https://www.npmjs.com/package/openclaw-plugin-yuanbao) 实现

## 安装

```bash
go get github.com/dtapps/yuanbao-go
```

## 获取凭证

1. 登录 [腾讯元宝](https://bot.yuanbao.tencent.com) App
2. 创建应用获取 `AppID` 和 `AppSecret`

## 使用示例

```go
package main

import (
	"fmt"
	"log"

	yuanbao "github.com/dtapps/yuanbao-go"
	"github.com/dtapps/yuanbao-go/config"
	"github.com/dtapps/yuanbao-go/types"
)

func boolPtr(b bool) *bool {
	return &b
}

func main() {
	// 日志
	logger.SetLevel(logger.LevelDebug)
	l := logger.GetLogger("demo")

	// 获取默认配置
	defaultCfg := config.DefaultConfig()
	defaultCfg.AppID = "your-app-id"
	defaultCfg.AppSecret = "your-app-secret"
	defaultCfg.RequireMention = boolPtr(true) // 群消息需要 @ 机器人才触发回调

	// 创建客户端
	client, err := yuanbao.NewClient("default", &types.Config{
		Yuanbao: defaultCfg,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Stop()

	// 设置连接状态回调
	client.OnConnected(func() {
		fmt.Println("✓ 已连接到元宝服务器")
	})

	client.OnDisconnected(func() {
		fmt.Println("✗ 已断开连接")
	})

	// 设置消息处理
	client.OnMessage(func(msg *types.InboundMessage, chatType types.ChatType) {
		// 提取文本内容
		content := ""
		for _, segment := range msg.Content {
			content += segment.Text
		}

		fmt.Printf("[%s] 收到消息: %s\n", chatType, content)

		// 自动回复
		reply := fmt.Sprintf("收到: %s", content)
		if chatType == types.ChatTypeGroup {
			atList := make([]types.AtInfo, 0, len(msg.AtList))
			if len(msg.AtList) > 0 {
				atList = append(atList, types.AtInfo{
					UserID:   msg.SenderID,
					UserName: msg.SenderName,
				})
			}
			_, err := client.SendGroupMessage(&types.OutboundGroupMessage{
				ToGroupID: msg.GroupID,
				Text:      reply,
				AtList:    atList,
			})
			if err != nil {
				fmt.Println("发送群消息失败:", err)
			}
		} else {
			_, err := client.SendMessage(&types.OutboundC2CMessage{
				ToUserID: msg.RecipientID,
				Text:     reply,
			})
			if err != nil {
				fmt.Println("发送私聊消息失败:", err)
			}
		}
		fmt.Println("已回复:", reply)
	})

	fmt.Println("正在连接...")

	select {}
}
```

## 配置

| 参数 | 必填 | 说明 |
|------|------|------|
| AppID | 是 | 应用ID |
| AppSecret | 是 | 应用密钥 |
| Enabled | 否 | 是否启用，默认 true |
| WSEndpoint | 否 | WebSocket 端点，默认 wss://bot-wss.yuanbao.tencent.com/wss/connection |
| TokenEndpoint | 否 | Token 端点，默认 bot.yuanbao.tencent.com |
| OverflowPolicy | 否 | 消息溢出策略："stop" 或 "split"，默认 "split" |
| ReplyToMode | 否 | 引用回复模式："off"、"first" 或 "all"，默认 "first" |
| OutboundQueueStrategy | 否 | 出站队列策略："immediate" 或 "merge-text"，默认 "merge-text" |
| MinChars | 否 | 消息聚合最小字符数，默认 2800 |
| MaxChars | 否 | 单条消息最大字符数，默认 3000 |
| IdleMs | 否 | 空闲自动发送超时（毫秒），默认 5000 |
| MediaMaxMb | 否 | 媒体文件大小上限（MB），默认 20 |
| HistoryLimit | 否 | 群聊上下文历史条数，默认 100 |
| DisableBlockStreaming | 否 | 禁用分块流式输出，默认 false |
| RequireMention | 否 | 群聊是否需要@机器人，默认 true |
| FallbackReply | 否 | 兜底回复文案 |
| MarkdownHintEnabled | 否 | 注入 Markdown 格式指令，默认 true |
| WsMaxReconnectAttempts | 否 | 最大重连次数，默认 100 |
| RouteEnv | 否 | 路由环境 |

## License

MIT
