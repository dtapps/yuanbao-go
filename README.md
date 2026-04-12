# YuanBao Go 客户端

腾讯元宝智能机器人 Go 语言客户端，支持私聊和群聊功能。

参考 [腾讯元宝智能机器人频道插件](https://www.npmjs.com/package/openclaw-plugin-yuanbao) 实现

## 功能特性

- WebSocket 长连接
- Protobuf 消息编解码
- 私聊消息收发
- 群聊消息收发
- @提及处理
- 消息引用回复
- 媒体消息支持
- 成员管理
- 自动重连

## 安装

```bash
go get github.com/dtapps/yuanbao-go
```

## 快速开始

### 环境要求

- Go 1.21+

### 基本使用

```go
package main

import (
	"fmt"
	"log"

	yuanbao "github.com/dtapps/yuanbao-go"
	"github.com/dtapps/yuanbao-go/config"
	"github.com/dtapps/yuanbao-go/types"
)

func main() {
	// 创建配置
	cfg := &config.Config{
		Yuanbao: &config.YuanbaoConfig{
			AppKey:    os.Getenv("YUANBAO_APP_KEY"),
			AppSecret: os.Getenv("YUANBAO_APP_SECRET"),
		},
	}

	// 创建客户端
	client, err := yuanbao.NewClient("default", cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Stop()

	// 设置消息处理
	client.OnMessage(func(msg *types.InboundMessage, chatType string) {
		from := msg.FromAccount
		if msg.SenderNickname != "" {
			from = msg.SenderNickname
		}

		var text string
		for _, elem := range msg.MsgBody {
			if elem.MsgType == "TIMTextElem" {
				text = elem.MsgContent.Text
				break
			}
		}

		if chatType == "group" {
			fmt.Printf("[%s] 群:%s(%s) 用户:%s(%s): %s\n", chatType, msg.GroupCode, msg.GroupName, msg.SenderNickname, msg.FromAccount, text)
		} else {
			fmt.Printf("[%s] 用户:%s(%s): %s\n", chatType, msg.SenderNickname, msg.FromAccount, text)
		}

		// 自动回复
		reply := fmt.Sprintf("收到: %s", text)
		if chatType == "group" {
			err := client.SendGroupMessage(msg.GroupCode, reply)
			if err != nil {
				fmt.Println("发送群消息失败", err)
			}
		} else {
			err := client.SendMessage(msg.FromAccount, reply)
			if err != nil {
				fmt.Println("发送私聊消息失败", err)
			}
		}
		fmt.Println("已回复:", reply)
	})

	// 设置连接状态
	client.OnConnected(func() {
		fmt.Println("✓ 已连接到元宝服务器")
	})

	client.OnDisconnected(func() {
		fmt.Println("✗ 已断开连接")
	})

	fmt.Println("正在连接...")

	select {}
}
```

## 配置选项

```go
cfg := &config.Config{
    Yuanbao: &config.YuanbaoConfig{
        AppKey:                  "your-app-key",
        AppSecret:               "your-app-secret",
        Token:                   "optional-token",
        ApiDomain:               "bot.yuanbao.tencent.com", // 默认值
        WsUrl:                  "wss://bot-wss.yuanbao.tencent.com/wss/connection", // 默认值
        WsMaxReconnectAttempts: 100, // 默认值
        
        // 消息策略
        OverflowPolicy:         "split", // "stop" | "split"
        ReplyToMode:           "first", // "off" | "first" | "all"
        OutboundQueueStrategy: "merge-text", // "immediate" | "merge-text"
        MinChars:              2800, // 消息聚合最小字符数
        MaxChars:              3000, // 单条消息最大字符数
        
        // 群聊设置
        RequireMention:        true,  // 群聊是否需要@机器人
        HistoryLimit:          100,   // 群聊上下文历史条数
        
        // 其他
        MediaMaxMb:            20,    // 媒体文件大小上限
        DisableBlockStreaming: false, // 禁用分块流式输出
        FallbackReply:         "暂时无法解答，你可以换个问题问问我哦",
        MarkdownHintEnabled:   true,  // 注入Markdown格式指令
    },
}
```

## API 文档

### Client

#### NewClient(accountId string, cfg *config.Config) (*Client, error)

创建新客户端。

#### OnMessage(handler func(msg *types.InboundMessage, chatType string))

设置消息处理回调。

#### OnConnected(handler func())

设置连接成功回调。

#### OnDisconnected(handler func())

设置断开连接回调。

#### SendMessage(to string, text string) error

发送私聊消息。

#### SendGroupMessage(groupCode string, text string) error

发送群聊消息。

#### GetState() string

获取连接状态。

#### Stop() error

停止客户端。

### Member

#### RecordUser(groupCode, userId, nickName string)

记录群用户。

#### RecordC2cUser(userId, nickName string)

记录 C2C 用户。

#### QueryMembers(groupCode, nameFilter string) []*UserRecord

查询群成员。

#### QueryGroupOwner(groupCode string) *UserRecord

查询群主。

#### QueryGroupInfo(groupCode string) *GroupInfo

查询群信息。

## 消息类型

```go
type InboundMessage struct {
    CallbackCommand      string            // 回调命令
    FromAccount         string            // 发送者账号
    ToAccount           string            // 接收者账号
    SenderNickname      string            // 发送者昵称
    GroupCode           string            // 群代码
    GroupName           string            // 群名称
    MsgSeq              uint32            // 消息序列号
    MsgID               string            // 消息ID
    MsgBody            []MsgBodyElement   // 消息体
    CloudCustomData     string            // 自定义数据
    BotOwnerID         string            // Bot所有者ID
    TraceID            string            // 追踪ID
}

type MsgBodyElement struct {
    MsgType   string      // 消息类型
    MsgContent MsgContent // 消息内容
}

type MsgContent struct {
    Text        string // 文本内容
    UUID        string // 文件UUID
    ImageFormat uint32 // 图片格式
    URL         string // 资源URL
    FileSize    uint32 // 文件大小
    FileName    string // 文件名
}
```

## 示例程序

运行示例程序：

```bash
export YUANBAO_APP_KEY=your-app-key
export YUANBAO_APP_SECRET=your-app-secret
go run ./cmd/example/main.go
```

## 许可证

MIT License
