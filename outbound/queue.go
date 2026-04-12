package outbound

import (
	"context"
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
)

// QueueConfig 队列配置
type QueueConfig struct {
	Strategy  string // "immediate" | "merge-text"
	MaxChars  int
	MinChars  int
	ChunkText func(text string, limit int) []string
}

// QueueItem 队列项
type QueueItem struct {
	Type      string // "text" | "media" | "sticker"
	Text      string
	MediaURL  string
	StickerID string
}

// Session 会话
type Session interface {
	Push(item *QueueItem) error
	Flush() error
	Abort()
	EmitReplyHeartbeat(heartbeat types.WsHeartbeat)
	DrainNow() <-chan struct{}
}

// QueueManager 队列管理器
type QueueManager struct {
	mu        sync.RWMutex
	accountId string
	config    *QueueConfig
	sessions  map[string]*session
	log       *logger.Logger
}

// NewQueueManager 创建队列管理器
func NewQueueManager(accountId string, config *QueueConfig) *QueueManager {
	if config.MinChars == 0 {
		config.MinChars = 2800
	}
	if config.MaxChars == 0 {
		config.MaxChars = 3000
	}
	if config.ChunkText == nil {
		config.ChunkText = defaultChunkText
	}

	return &QueueManager{
		accountId: accountId,
		config:    config,
		sessions:  make(map[string]*session),
		log:       logger.New("outbound-queue"),
	}
}

// GetOrCreateSession 获取或创建会话
func (m *QueueManager) GetOrCreateSession(sessionKey string, options *SessionOptions) Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[sessionKey]; ok {
		return s
	}

	s := &session{
		key:        sessionKey,
		config:     m.config,
		options:    options,
		queueItems: make([]*QueueItem, 0),
		abortCh:    make(chan struct{}),
		doneCh:     make(chan struct{}),
		log:        m.log,
	}

	m.sessions[sessionKey] = s
	go s.run()

	return s
}

// GetSession 获取会话
func (m *QueueManager) GetSession(sessionKey string) Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionKey]
}

// UnregisterSession 注销会话
func (m *QueueManager) UnregisterSession(sessionKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[sessionKey]; ok {
		s.Abort()
		delete(m.sessions, sessionKey)
	}
}

// Destroy 销毁管理器
func (m *QueueManager) Destroy() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, s := range m.sessions {
		s.Abort()
		delete(m.sessions, key)
	}
}

// SessionOptions 会话选项
type SessionOptions struct {
	ChatType       string // "c2c" | "group"
	Account        any
	Target         string
	FromAccount    string
	RefMsgId       string
	RefFromAccount string
	ToAccount      string
	GroupCode      string
}

// session 会话实现
type session struct {
	key        string
	config     *QueueConfig
	options    *SessionOptions
	queueItems []*QueueItem
	abortCh    chan struct{}
	doneCh     chan struct{}
	mu         sync.Mutex
	aborted    bool
	sent       bool
	log        *logger.Logger
}

// run 运行会话
func (s *session) run() {
	defer close(s.doneCh)

	<-s.abortCh

	s.mu.Lock()
	s.aborted = true
	s.mu.Unlock()
}

// Push 添加队列项
func (s *session) Push(item *QueueItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.aborted {
		return nil
	}

	s.queueItems = append(s.queueItems, item)
	return nil
}

// Flush 刷新队列
func (s *session) Flush() error {
	s.mu.Lock()
	if s.aborted {
		s.mu.Unlock()
		return nil
	}

	items := s.queueItems
	s.queueItems = nil
	s.mu.Unlock()

	// 发送消息
	for _, item := range items {
		if item.Type == "text" && item.Text != "" {
			s.sendText(item.Text)
		}
	}

	s.mu.Lock()
	s.sent = true
	s.mu.Unlock()

	close(s.abortCh)
	<-s.doneCh

	return nil
}

// Abort 中止会话
func (s *session) Abort() {
	s.mu.Lock()
	if s.aborted {
		s.mu.Unlock()
		return
	}
	s.aborted = true
	s.mu.Unlock()

	select {
	case s.abortCh <- struct{}{}:
	default:
	}

	<-s.doneCh
}

// EmitReplyHeartbeat 发送回复心跳
func (s *session) EmitReplyHeartbeat(heartbeat types.WsHeartbeat) {
	// 实际实现中需要调用WS客户端发送心跳
}

// DrainNow 立即排空
func (s *session) DrainNow() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		s.Flush()
		close(ch)
	}()
	return ch
}

// sendText 发送文本
func (s *session) sendText(text string) {
	// 实际实现中需要调用WS客户端发送消息
	s.log.Info("发送文本", map[string]any{"target": s.options.Target, "text": text})
}

// sendMedia 发送媒体
func (s *session) sendMedia(mediaURL string) {
	// 实际实现中需要处理媒体上传和发送
	s.log.Info("发送媒体", map[string]any{"target": s.options.Target, "url": mediaURL})
}

// sendSticker 发送表情
func (s *session) sendSticker(stickerId string) {
	// 实际实现中需要处理表情发送
	s.log.Info("发送表情", map[string]any{"target": s.options.Target, "stickerId": stickerId})
}

// 全局队列管理器
var (
	queueManagers = sync.Map{}
)

// InitQueue 初始化队列
func InitQueue(accountId string, config *QueueConfig) *QueueManager {
	manager := NewQueueManager(accountId, config)
	queueManagers.Store(accountId, manager)
	return manager
}

// GetQueue 获取队列
func GetQueue(accountId string) *QueueManager {
	if manager, ok := queueManagers.Load(accountId); ok {
		return manager.(*QueueManager)
	}
	return nil
}

// DestroyQueue 销毁队列
func DestroyQueue(accountId string) {
	if manager, ok := queueManagers.LoadAndDelete(accountId); ok {
		manager.(*QueueManager).Destroy()
	}
}

// MergeTextSession 合并文本会话
type MergeTextSession struct {
	*session
	textBuffer string
	sendChain  chan struct{}
}

// NewMergeTextSession 创建合并文本会话
func NewMergeTextSession(options *SessionOptions, config *QueueConfig) *MergeTextSession {
	s := &MergeTextSession{
		session: &session{
			key:        "",
			config:     config,
			options:    options,
			queueItems: make([]*QueueItem, 0),
			abortCh:    make(chan struct{}),
			doneCh:     make(chan struct{}),
			log:        logger.New("merge-text-session"),
		},
		sendChain: make(chan struct{}, 1),
	}

	go s.run()
	return s
}

// run 运行合并文本会话
func (s *MergeTextSession) run() {
	ticker := time.NewTicker(time.Duration(s.config.MinChars/100) * time.Millisecond)
	defer ticker.Stop()
	defer close(s.doneCh)

	for {
		select {
		case <-s.abortCh:
			return
		case <-ticker.C:
			s.drainBuffer(false)
		case <-s.sendChain:
			s.drainBuffer(true)
		}
	}
}

// Push 添加队列项
func (s *MergeTextSession) Push(item *QueueItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.aborted {
		return nil
	}

	if item.Type == "text" {
		s.textBuffer += item.Text
	} else {
		// 非文本项，先发送缓冲区
		s.drainBuffer(true)
		s.queueItems = append(s.queueItems, item)
	}

	// 触发发送检查
	select {
	case s.sendChain <- struct{}{}:
	default:
	}

	return nil
}

// drainBuffer 排空缓冲区
func (s *MergeTextSession) drainBuffer(force bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.textBuffer == "" {
		return
	}

	// 检查是否需要发送
	if !force && len(s.textBuffer) < s.config.MinChars {
		return
	}

	// 分块发送
	chunks := s.config.ChunkText(s.textBuffer, s.config.MaxChars)
	for i, chunk := range chunks {
		if !force && i == len(chunks)-1 && len(s.textBuffer) < s.config.MinChars {
			// 最后一个块且未强制发送且文本较短，保留在缓冲区
			continue
		}
		s.sendText(chunk)
	}

	s.textBuffer = ""
}

// Flush 刷新
func (s *MergeTextSession) Flush() error {
	s.mu.Lock()
	if s.aborted {
		s.mu.Unlock()
		return nil
	}

	items := s.queueItems
	s.queueItems = nil
	text := s.textBuffer
	s.textBuffer = ""
	s.mu.Unlock()

	// 发送文本
	if text != "" {
		chunks := s.config.ChunkText(text, s.config.MaxChars)
		for _, chunk := range chunks {
			s.sendText(chunk)
		}
	}

	// 发送其他项
	for _, item := range items {
		if item.Type == "text" {
			s.sendText(item.Text)
		} else if item.Type == "media" {
			s.sendMedia(item.MediaURL)
		} else if item.Type == "sticker" {
			s.sendSticker(item.StickerID)
		}
	}

	close(s.abortCh)
	<-s.doneCh

	return nil
}

// 辅助函数

func defaultChunkText(text string, limit int) []string {
	if len(text) <= limit {
		return []string{text}
	}

	chunks := make([]string, 0)
	for i := 0; i < len(text); i += limit {
		end := min(i+limit, len(text))
		chunks = append(chunks, text[i:end])
	}

	return chunks
}

// ChunkMarkdownText 分块Markdown文本
func ChunkMarkdownText(text string, limit int) []string {
	// 简单的Markdown感知分块
	lines := make([]string, 0, len(text)/limit+1)
	current := ""

	for _, line := range splitLines(text) {
		if len(current)+len(line) > limit {
			if current != "" {
				chunks := defaultChunkText(current, limit)
				lines = append(lines, chunks...)
				current = ""
			}

			// 如果单行超过限制，直接分块
			if len(line) > limit {
				chunks := defaultChunkText(line, limit)
				lines = append(lines, chunks[:len(chunks)-1]...)
				current = chunks[len(chunks)-1]
			} else {
				current = line
			}
		} else {
			current += line
		}
	}

	if current != "" {
		chunks := defaultChunkText(current, limit)
		lines = append(lines, chunks...)
	}

	return lines
}

func splitLines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i+1])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

// WithContext 带上下文的队列操作
func WithContext(ctx context.Context, fn func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fn()
	}
}
