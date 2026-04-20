package types

// Member 成员
type Member struct {
	UserID   string // 用户ID
	Nickname string // 昵称
}

// MemberAddUserRequest 添加成员 请求
type MemberAddUserRequest struct {
	UserID   string // 用户ID
	Nickname string // 昵称
}

// MemberUpdateUserRequest 更新成员 请求
type MemberUpdateUserRequest struct {
	UserID   string // 用户ID
	Nickname string // 昵称
}

// MemberListUsersRequest 列出所有成员 请求
type MemberListUsersRequest struct{}

// MemberListUsersResponse 列出所有成员 响应
type MemberListUsersResponse struct {
	Total   int       // 总数
	Members []*Member // 成员列表
}
