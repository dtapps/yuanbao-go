package member

import (
	"sync"
	"time"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/ws"
)

const (
	// SessionTTLMs 会话TTL
	SessionTTLMs = 24 * 60 * 60 * 1000 // 24小时
	// GroupCacheTTLMs 群缓存TTL
	GroupCacheTTLMs = 5 * 60 * 1000 // 5分钟
)

// MemberInterface 成员管理接口
type MemberInterface interface {
	RecordUser(groupCode, userId, nickName string)
	RecordC2cUser(userId, nickName string)
	QueryMembers(groupCode, nameFilter string) []*UserRecord
	LookupUsers(groupCode, nameFilter string) []*UserRecord
	QueryGroupOwner(groupCode string) *UserRecord
	QueryGroupInfo(groupCode string) *GroupInfo
	QueryYuanbaoUserId(groupCode string) string
}

// UserRecord 用户记录
type UserRecord struct {
	UserID   string
	NickName string
	LastSeen int64
	UserType int32
}

// SessionMember 会话成员
type SessionMember struct {
	mu         sync.RWMutex
	groupUsers map[string]map[string]*UserRecord // groupCode -> userId -> UserRecord
}

// NewSessionMember 创建会话成员
func NewSessionMember() *SessionMember {
	return &SessionMember{
		groupUsers: make(map[string]map[string]*UserRecord),
	}
}

// RecordUser 记录用户
func (m *SessionMember) RecordUser(groupCode, userId, nickName string) {
	if userId == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groupUsers[groupCode]; !ok {
		m.groupUsers[groupCode] = make(map[string]*UserRecord)
	}

	m.groupUsers[groupCode][userId] = &UserRecord{
		UserID:   userId,
		NickName: nickName,
		LastSeen: time.Now().UnixMilli(),
	}

	m.cleanExpired()
}

// LookupUsers 查找用户
func (m *SessionMember) LookupUsers(groupCode, nameFilter string) []*UserRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := m.groupUsers[groupCode]
	if len(users) == 0 {
		return nil
	}

	var results []*UserRecord
	for _, user := range users {
		if nameFilter == "" {
			results = append(results, user)
		} else if containsIgnoreCase(user.NickName, nameFilter) {
			results = append(results, user)
		}
	}

	// 按最后活跃时间排序
	sortByLastSeen(results)
	return results
}

// LookupUserByNickName 按昵称查找用户
func (m *SessionMember) LookupUserByNickName(groupCode, nickName string) *UserRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := m.groupUsers[groupCode]
	target := toLower(nickName)

	for _, user := range users {
		if toLower(user.NickName) == target {
			return user
		}
	}

	return nil
}

// LookupUserById 按ID查找用户
func (m *SessionMember) LookupUserById(groupCode, userId string) *UserRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := m.groupUsers[groupCode]
	return users[userId]
}

// UpsertUser 插入或更新用户
func (m *SessionMember) UpsertUser(groupCode string, record *UserRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groupUsers[groupCode]; !ok {
		m.groupUsers[groupCode] = make(map[string]*UserRecord)
	}

	m.groupUsers[groupCode][record.UserID] = record
}

// ListGroupCodes 列出群组代码
func (m *SessionMember) ListGroupCodes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var codes []string
	for code := range m.groupUsers {
		codes = append(codes, code)
	}
	return codes
}

// cleanExpired 清理过期数据
func (m *SessionMember) cleanExpired() {
	now := time.Now().UnixMilli()

	for code, users := range m.groupUsers {
		for id, user := range users {
			if now-user.LastSeen > SessionTTLMs {
				delete(users, id)
			}
		}

		if len(users) == 0 {
			delete(m.groupUsers, code)
		}
	}
}

// GroupMember 群组成员
type GroupMember struct {
	accountId    string
	session      *SessionMember
	cache        map[string]*groupCacheEntry // groupCode -> cache entry
	ownerCache   map[string]*ownerCacheEntry
	infoCache    map[string]*infoCacheEntry
	clientGetter func(accountId string) *ws.WsClient
	log          *logger.Logger
	mu           sync.RWMutex
}

type groupCacheEntry struct {
	Members   []*UserRecord
	FetchedAt int64
}

type ownerCacheEntry struct {
	Owner     *UserRecord
	FetchedAt int64
}

type infoCacheEntry struct {
	Info      *GroupInfo
	FetchedAt int64
}

// GroupInfo 群组信息
type GroupInfo struct {
	GroupName     string
	OwnerUserId   string
	OwnerNickName string
	GroupSize     int32
}

// NewGroupMember 创建群组成员
func NewGroupMember(accountId string, session *SessionMember) *GroupMember {
	return &GroupMember{
		accountId:  accountId,
		session:    session,
		cache:      make(map[string]*groupCacheEntry),
		ownerCache: make(map[string]*ownerCacheEntry),
		infoCache:  make(map[string]*infoCacheEntry),
		log:        logger.New("member:group"),
	}
}

// GetMembers 获取成员列表
func (m *GroupMember) GetMembers(groupCode string) []*UserRecord {
	m.mu.RLock()
	if cached, ok := m.cache[groupCode]; ok {
		if time.Now().UnixMilli()-cached.FetchedAt < GroupCacheTTLMs {
			m.mu.RUnlock()
			return cached.Members
		}
	}
	m.mu.RUnlock()

	// 从API获取
	members := m.fetchFromApi(groupCode)

	m.mu.Lock()
	m.cache[groupCode] = &groupCacheEntry{
		Members:   members,
		FetchedAt: time.Now().UnixMilli(),
	}
	m.mu.Unlock()

	return members
}

// LookupUsers 查找用户
func (m *GroupMember) LookupUsers(groupCode, nameFilter string) []*UserRecord {
	m.mu.RLock()
	cached, ok := m.cache[groupCode]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	var results []*UserRecord
	for _, user := range cached.Members {
		if nameFilter == "" || containsIgnoreCase(user.NickName, nameFilter) {
			results = append(results, user)
		}
	}
	return results
}

// LookupUserByNickName 按昵称查找用户
func (m *GroupMember) LookupUserByNickName(groupCode, nickName string) *UserRecord {
	m.mu.RLock()
	cached, ok := m.cache[groupCode]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	target := toLower(nickName)
	for _, user := range cached.Members {
		if toLower(user.NickName) == target {
			return user
		}
	}
	return nil
}

// HasCachedData 是否有缓存数据
func (m *GroupMember) HasCachedData(groupCode string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.cache[groupCode]
	return ok
}

// ListCachedGroupCodes 列出缓存的群组代码
func (m *GroupMember) ListCachedGroupCodes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var codes []string
	for code := range m.cache {
		codes = append(codes, code)
	}
	return codes
}

// Refresh 刷新缓存
func (m *GroupMember) Refresh(groupCode string) []*UserRecord {
	m.mu.Lock()
	delete(m.cache, groupCode)
	m.mu.Unlock()

	return m.GetMembers(groupCode)
}

// QueryGroupOwner 查询群主
func (m *GroupMember) QueryGroupOwner(groupCode string) *UserRecord {
	m.mu.RLock()
	if cached, ok := m.ownerCache[groupCode]; ok {
		if time.Now().UnixMilli()-cached.FetchedAt < GroupCacheTTLMs {
			m.mu.RUnlock()
			return cached.Owner
		}
	}
	m.mu.RUnlock()

	// 获取WS客户端
	client := m.getClient()
	if client == nil || client.GetState() != "connected" {
		m.mu.RLock()
		cached := m.ownerCache[groupCode]
		m.mu.RUnlock()
		if cached != nil {
			return cached.Owner
		}
		return nil
	}

	// 调用API
	rsp, err := client.QueryGroupInfoSimple(groupCode)
	if err != nil || rsp == nil || rsp.GroupInfo == nil {
		m.log.Warn("查询群主失败", map[string]any{"groupCode": groupCode, "error": err})
		m.mu.RLock()
		cached := m.ownerCache[groupCode]
		m.mu.RUnlock()
		if cached != nil {
			return cached.Owner
		}
		return nil
	}

	owner := &UserRecord{
		UserID:   rsp.GroupInfo.GroupOwnerUserID,
		NickName: rsp.GroupInfo.GroupOwnerNickname,
	}

	m.mu.Lock()
	m.ownerCache[groupCode] = &ownerCacheEntry{
		Owner:     owner,
		FetchedAt: time.Now().UnixMilli(),
	}
	m.mu.Unlock()

	return owner
}

// QueryGroupInfo 查询群信息
func (m *GroupMember) QueryGroupInfo(groupCode string) *GroupInfo {
	m.mu.RLock()
	if cached, ok := m.infoCache[groupCode]; ok {
		if time.Now().UnixMilli()-cached.FetchedAt < GroupCacheTTLMs {
			m.mu.RUnlock()
			return cached.Info
		}
	}
	m.mu.RUnlock()

	// 获取WS客户端
	client := m.getClient()
	if client == nil || client.GetState() != "connected" {
		m.mu.RLock()
		cached := m.infoCache[groupCode]
		m.mu.RUnlock()
		if cached != nil {
			return cached.Info
		}
		return nil
	}

	// 调用API
	rsp, err := client.QueryGroupInfoSimple(groupCode)
	if err != nil || rsp == nil || rsp.GroupInfo == nil {
		m.log.Warn("查询群信息失败", map[string]any{"groupCode": groupCode, "error": err})
		m.mu.RLock()
		cached := m.infoCache[groupCode]
		m.mu.RUnlock()
		if cached != nil {
			return cached.Info
		}
		return nil
	}

	info := &GroupInfo{
		GroupName:     rsp.GroupInfo.GroupName,
		OwnerUserId:   rsp.GroupInfo.GroupOwnerUserID,
		OwnerNickName: rsp.GroupInfo.GroupOwnerNickname,
		GroupSize:     rsp.GroupInfo.GroupSize,
	}

	m.mu.Lock()
	m.infoCache[groupCode] = &infoCacheEntry{
		Info:      info,
		FetchedAt: time.Now().UnixMilli(),
	}
	m.mu.Unlock()

	return info
}

// fetchFromApi 从API获取成员
func (m *GroupMember) fetchFromApi(groupCode string) []*UserRecord {
	client := m.getClient()
	if client == nil || client.GetState() != "connected" {
		return nil
	}

	rsp, err := client.GetGroupMemberListSimple(groupCode)
	if err != nil || rsp == nil || rsp.Code != 0 {
		return nil
	}

	var members []*UserRecord
	now := time.Now().UnixMilli()

	for _, member := range rsp.MemberList {
		record := &UserRecord{
			UserID:   member.UserID,
			NickName: member.NickName,
			UserType: member.UserType,
			LastSeen: now,
		}

		// 合并会话数据
		if existing := m.session.LookupUserById(groupCode, member.UserID); existing != nil {
			record.LastSeen = existing.LastSeen
		}

		m.session.UpsertUser(groupCode, record)
		members = append(members, record)
	}

	return members
}

// getClient 获取WS客户端
func (m *GroupMember) getClient() *ws.WsClient {
	// 实际实现中需要从运行时获取
	return nil
}

// SetClientGetter 设置客户端获取器
func (m *GroupMember) SetClientGetter(getter func(accountId string) *ws.WsClient) {
	m.clientGetter = getter
}

// Member 成员管理
type Member struct {
	accountId          string
	session            *SessionMember
	group              *GroupMember
	c2cUsers           map[string]*UserRecord
	yuanbaoUserIdCache string
	mu                 sync.RWMutex
}

// NewMember 创建成员管理
func NewMember(accountId string) *Member {
	session := NewSessionMember()
	return &Member{
		accountId: accountId,
		session:   session,
		group:     NewGroupMember(accountId, session),
		c2cUsers:  make(map[string]*UserRecord),
	}
}

// RecordUser 记录用户
func (m *Member) RecordUser(groupCode, userId, nickName string) {
	m.session.RecordUser(groupCode, userId, nickName)
}

// RecordC2cUser 记录C2C用户
func (m *Member) RecordC2cUser(userId, nickName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if userId == "" {
		return
	}

	m.c2cUsers[userId] = &UserRecord{
		UserID:   userId,
		NickName: nickName,
		LastSeen: time.Now().UnixMilli(),
	}
}

// ListC2cUsers 列出C2C用户
func (m *Member) ListC2cUsers() []*UserRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var users []*UserRecord
	for _, user := range m.c2cUsers {
		users = append(users, user)
	}

	sortByLastSeen(users)
	return users
}

// QueryMembers 查询成员
func (m *Member) QueryMembers(groupCode, nameFilter string) []*UserRecord {
	members := m.group.GetMembers(groupCode)
	if len(members) > 0 {
		if nameFilter == "" {
			return members
		}

		var filtered []*UserRecord
		for _, user := range members {
			if containsIgnoreCase(user.NickName, nameFilter) {
				filtered = append(filtered, user)
			}
		}

		if len(filtered) > 0 {
			return filtered
		}
	}

	return m.session.LookupUsers(groupCode, nameFilter)
}

// QueryUserIdsByNickName 按昵称查询用户ID
func (m *Member) QueryUserIdsByNickName(nickName, groupCode string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]bool)
	var results []string

	// 从C2C用户中查找
	for _, user := range m.c2cUsers {
		if containsIgnoreCase(user.NickName, nickName) && !seen[user.UserID] {
			seen[user.UserID] = true
			results = append(results, user.UserID)
		}
	}

	// 从群组成员中查找
	if groupCode != "" {
		members := m.group.GetMembers(groupCode)
		for _, user := range members {
			if containsIgnoreCase(user.NickName, nickName) && !seen[user.UserID] {
				seen[user.UserID] = true
				results = append(results, user.UserID)
			}
		}
	}

	return results
}

// LookupUsers 查找用户
func (m *Member) LookupUsers(groupCode, nameFilter string) []*UserRecord {
	users := m.group.LookupUsers(groupCode, nameFilter)
	if len(users) > 0 {
		return users
	}
	return m.session.LookupUsers(groupCode, nameFilter)
}

// LookupUserByNickName 按昵称查找用户
func (m *Member) LookupUserByNickName(groupCode, nickName string) *UserRecord {
	if user := m.group.LookupUserByNickName(groupCode, nickName); user != nil {
		return user
	}
	return m.session.LookupUserByNickName(groupCode, nickName)
}

// QueryGroupOwner 查询群主
func (m *Member) QueryGroupOwner(groupCode string) *UserRecord {
	return m.group.QueryGroupOwner(groupCode)
}

// QueryGroupInfo 查询群信息
func (m *Member) QueryGroupInfo(groupCode string) *GroupInfo {
	return m.group.QueryGroupInfo(groupCode)
}

// QueryYuanbaoUserId 查询元宝用户ID
func (m *Member) QueryYuanbaoUserId(groupCode string) string {
	m.mu.RLock()
	if m.yuanbaoUserIdCache != "" {
		m.mu.RUnlock()
		return m.yuanbaoUserIdCache
	}
	m.mu.RUnlock()

	if groupCode == "" {
		return ""
	}

	members := m.group.GetMembers(groupCode)
	for _, member := range members {
		if member.UserType == 2 || member.UserType == 3 { // 2=元宝, 3=机器人
			m.mu.Lock()
			m.yuanbaoUserIdCache = member.UserID
			m.mu.Unlock()
			return member.UserID
		}
	}

	return ""
}

// ListGroupCodes 列出群组代码
func (m *Member) ListGroupCodes() []string {
	return m.session.ListGroupCodes()
}

// FormatRecords 格式化记录
func (m *Member) FormatRecords(records []*UserRecord) []map[string]any {
	var result []map[string]any
	for _, record := range records {
		result = append(result, map[string]any{
			"userId":   record.UserID,
			"nickName": record.NickName,
			"lastSeen": time.UnixMilli(record.LastSeen).Format(time.RFC3339),
		})
	}
	return result
}

// 全局成员管理
var (
	activeMembers sync.Map
)

// GetMember 获取成员管理
func GetMember(accountId string) *Member {
	if member, ok := activeMembers.Load(accountId); ok {
		return member.(*Member)
	}

	member := NewMember(accountId)
	activeMembers.Store(accountId, member)
	return member
}

// RemoveMember 移除成员管理
func RemoveMember(accountId string) {
	activeMembers.Delete(accountId)
}

// GetAllActiveMembers 获取所有活跃成员
func GetAllActiveMembers() map[string]*Member {
	result := make(map[string]*Member)
	activeMembers.Range(func(key, value any) bool {
		result[key.(string)] = value.(*Member)
		return true
	})
	return result
}

// 辅助函数

func containsIgnoreCase(s, substr string) bool {
	return stringsContains(toLower(s), toLower(substr))
}

func toLower(s string) string {
	// 简单的Unicode小写转换
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringsContainsImpl(s, substr))
}

func stringsContainsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sortByLastSeen(records []*UserRecord) {
	// 简单的冒泡排序
	for i := range records {
		for j := i + 1; j < len(records); j++ {
			if records[j].LastSeen > records[i].LastSeen {
				records[i], records[j] = records[j], records[i]
			}
		}
	}
}
