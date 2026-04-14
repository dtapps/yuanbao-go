package types

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
