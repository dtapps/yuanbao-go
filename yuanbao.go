package yuanbao

import (
	"github.com/dtapps/yuanbao-go/account"
	"github.com/dtapps/yuanbao-go/member"
	"github.com/dtapps/yuanbao-go/plugin"
	"github.com/dtapps/yuanbao-go/types"
)

// Client 元宝客户端
type Client struct {
	plugin *plugin.Plugin
}

// NewClient 创建新客户端
func NewClient(accountId string, cfg *types.Config) (*Client, error) {
	// 解析账号
	acc := account.GetManager().ResolveAccount(cfg, accountId)

	if !acc.Configured {
		return nil, ErrAccountNotConfigured
	}

	// 创建并启动插件
	p, err := plugin.CreateAndStart(accountId, acc, cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		plugin: p,
	}, nil
}

// OnMessage 设置消息处理回调
func (c *Client) OnMessage(handler func(msg *types.InboundMessage, chatType string)) {
	c.plugin.SetOnMessage(handler)
}

// OnConnected 设置连接成功回调
func (c *Client) OnConnected(handler func()) {
	c.plugin.SetOnConnected(handler)
}

// OnDisconnected 设置断开连接回调
func (c *Client) OnDisconnected(handler func()) {
	c.plugin.SetOnDisconnected(handler)
}

// SendMessage 发送消息
func (c *Client) SendMessage(to string, text string) error {
	_, err := c.plugin.SendMessage(to, text)
	return err
}

// SendGroupMessage 发送群消息
func (c *Client) SendGroupMessage(groupCode string, text string) error {
	_, err := c.plugin.SendGroupMessage(groupCode, text, "")
	return err
}

// SendGroupMessageWithRef 发送群消息（带引用）
func (c *Client) SendGroupMessageWithRef(groupCode string, text string, refMsgId string) error {
	_, err := c.plugin.SendGroupMessage(groupCode, text, refMsgId)
	return err
}

// GetState 获取连接状态
func (c *Client) GetState() string {
	return c.plugin.GetState()
}

// GetMember 获取成员管理
func (c *Client) GetMember() member.MemberInterface {
	return c.plugin.GetMember()
}

// Stop 停止客户端
func (c *Client) Stop() error {
	return c.plugin.Stop()
}

// 错误定义

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

var (
	ErrAccountNotConfigured = &Error{Code: 1, Message: "账号未配置"}
	ErrNotConnected         = &Error{Code: 2, Message: "未连接到服务器"}
	ErrSendFailed           = &Error{Code: 3, Message: "发送消息失败"}
)
