package types

// Group 群
type Group struct {
	GroupID string             // 群ID
	Name    string             // 群名称
	Members map[string]*Member // 成员列表 userID -> Member
}

// GroupAddRequest 添加群 请求
type GroupAddRequest struct {
	GroupID string // 群ID
	Name    string // 群名称
}

// GroupAddUserRequest 添加群成员 请求
type GroupAddUserRequest struct {
	GroupID  string // 群ID
	UserID   string // 用户ID
	Nickname string // 昵称
}

// GroupUpdateUserRequest 更新群成员 请求
type GroupUpdateUserRequest struct {
	GroupID  string // 群ID
	UserID   string // 用户ID
	Nickname string // 昵称
}

// GroupListUsersRequest 列出群成员 请求
type GroupListUsersRequest struct {
	GroupID string // 群ID
}

// GroupListUsersResponse 列出群成员 响应
type GroupListUsersResponse struct {
	GroupID string    // 群ID
	Total   int       // 总数
	Members []*Member // 成员列表
}
