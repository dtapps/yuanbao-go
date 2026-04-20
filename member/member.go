package member

import (
	"fmt"
	"sync"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
)

// Manager 成员管理器
type Manager struct {
	mu      sync.RWMutex
	members map[string]*types.Member // userID -> Member
	groups  map[string]*types.Group  // groupID -> Group
	log     *logger.Logger
}

// 全局成员管理器
var (
	globalManagers map[string]*Manager
	globalMu       sync.RWMutex
	managerOnce    sync.Once
)

// GetManager 获取指定账号的成员管理器
func GetManager(accountID string) *Manager {
	managerOnce.Do(func() {
		globalManagers = make(map[string]*Manager)
	})

	globalMu.RLock()
	mgr, ok := globalManagers[accountID]
	globalMu.RUnlock()
	if ok {
		return mgr
	}

	globalMu.Lock()
	defer globalMu.Unlock()

	if mgr, ok = globalManagers[accountID]; ok {
		return mgr
	}

	mgr = NewManager()
	globalManagers[accountID] = mgr
	return mgr
}

// NewManager 创建成员管理器
func NewManager() *Manager {
	return &Manager{
		members: make(map[string]*types.Member),
		groups:  make(map[string]*types.Group),
		log:     logger.New("member"),
	}
}

// AddUser 添加成员
func (m *Manager) AddUser(req *types.MemberAddUserRequest) (*types.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	member := &types.Member{
		UserID:   req.UserID,
		Nickname: req.Nickname,
	}
	m.members[req.UserID] = member

	return member, nil
}

// UpdateUser 更新成员
func (m *Manager) UpdateUser(req *types.MemberUpdateUserRequest) (*types.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if member, ok := m.members[req.UserID]; ok {
		member.Nickname = req.Nickname
		return member, nil
	}

	member := &types.Member{
		UserID:   req.UserID,
		Nickname: req.Nickname,
	}
	m.members[req.UserID] = member

	return member, nil
}

// DeleteUser 删除成员
func (m *Manager) DeleteUser(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.members, userID)

	return nil
}

// ListUsers 列出所有成员
func (m *Manager) ListUsers(req *types.MemberListUsersRequest) *types.MemberListUsersResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := make([]*types.Member, 0, len(m.members))
	for _, member := range m.members {
		members = append(members, member)
	}

	return &types.MemberListUsersResponse{
		Total:   len(members),
		Members: members,
	}
}

// GetUser 获取成员
func (m *Manager) GetUser(userID string) (*types.Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if member, ok := m.members[userID]; ok {
		return member, nil
	}

	return &types.Member{UserID: userID}, fmt.Errorf("member not found: %s", userID)
}

// Clear 清空成员
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.members = make(map[string]*types.Member)
	m.groups = make(map[string]*types.Group)
}

// AddGroup 添加群
func (m *Manager) AddGroup(req *types.GroupAddRequest) (*types.Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group := &types.Group{
		GroupID: req.GroupID,
		Name:    req.Name,
		Members: make(map[string]*types.Member),
	}
	m.groups[req.GroupID] = group

	return group, nil
}

// AddGroupUser 添加群成员
func (m *Manager) AddGroupUser(req *types.GroupAddUserRequest) (*types.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[req.GroupID]
	if !ok {
		group = &types.Group{
			GroupID: req.GroupID,
			Members: make(map[string]*types.Member),
		}
		m.groups[req.GroupID] = group
	}

	member := &types.Member{
		UserID:   req.UserID,
		Nickname: req.Nickname,
	}
	group.Members[req.UserID] = member

	return member, nil
}

// UpdateGroupUser 更新群成员
func (m *Manager) UpdateGroupUser(req *types.GroupUpdateUserRequest) (*types.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[req.GroupID]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", req.GroupID)
	}

	if member, ok := group.Members[req.UserID]; ok {
		member.Nickname = req.Nickname
		return member, nil
	}

	member := &types.Member{
		UserID:   req.UserID,
		Nickname: req.Nickname,
	}
	group.Members[req.UserID] = member

	return member, nil
}

// DeleteGroupUser 删除群成员
func (m *Manager) DeleteGroupUser(groupID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[groupID]
	if !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}

	delete(group.Members, userID)

	return nil
}

// GetGroup 获取群
func (m *Manager) GetGroup(groupID string) (*types.Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if group, ok := m.groups[groupID]; ok {
		return group, nil
	}

	return &types.Group{GroupID: groupID}, fmt.Errorf("group not found: %s", groupID)
}

// GetGroupUser 获取群成员
func (m *Manager) GetGroupUser(groupID, userID string) (*types.Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[groupID]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}

	if member, ok := group.Members[userID]; ok {
		return member, nil
	}

	return &types.Member{UserID: userID}, fmt.Errorf("member not found: %s/%s", groupID, userID)
}

// ListGroupUsers 列出群成员
func (m *Manager) ListGroupUsers(req *types.GroupListUsersRequest) *types.GroupListUsersResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[req.GroupID]
	if !ok {
		return &types.GroupListUsersResponse{
			GroupID: req.GroupID,
			Total:   0,
			Members: nil,
		}
	}

	members := make([]*types.Member, 0, len(group.Members))
	for _, member := range group.Members {
		members = append(members, member)
	}

	return &types.GroupListUsersResponse{
		GroupID: req.GroupID,
		Total:   len(members),
		Members: members,
	}
}

// DeleteGroup 删除群
func (m *Manager) DeleteGroup(groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.groups, groupID)

	return nil
}
