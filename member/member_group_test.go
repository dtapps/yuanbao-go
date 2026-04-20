package member

import (
	"fmt"
	"sync"
	"testing"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.SetLevel(logger.LevelError)
}

// TestGroup_CRUD 验证群的增删改查逻辑
func TestGroup_CRUD(t *testing.T) {
	m := NewManager()
	groupID := "group_123"

	// 1. AddGroup
	_, err := m.AddGroup(&types.GroupAddRequest{GroupID: groupID, Name: "测试群"})
	assert.NoError(t, err)

	// 2. GetGroup
	g, err := m.GetGroup(groupID)
	assert.NoError(t, err)
	assert.Equal(t, "测试群", g.Name)

	// 3. DeleteGroup
	err = m.DeleteGroup(groupID)
	assert.NoError(t, err)

	// 4. GetGroup after delete
	_, err = m.GetGroup(groupID)
	assert.Error(t, err, "删除后应当报错")

	// 5. AddGroup again for other tests
	_, err = m.AddGroup(&types.GroupAddRequest{GroupID: groupID, Name: "测试群"})
	assert.NoError(t, err)
}

// TestGroupUser_CRUD 验证群成员的增删改查逻辑
func TestGroupUser_CRUD(t *testing.T) {
	m := NewManager()
	groupID := "group_123"
	uid := "user_456"

	// Prepare group
	_, _ = m.AddGroup(&types.GroupAddRequest{GroupID: groupID, Name: "测试群"})

	// 1. AddGroupUser
	_, err := m.AddGroupUser(&types.GroupAddUserRequest{
		GroupID:  groupID,
		UserID:   uid,
		Nickname: "Alice",
	})
	assert.NoError(t, err)

	// 2. GetGroupUser
	u, err := m.GetGroupUser(groupID, uid)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", u.Nickname)

	// 3. UpdateGroupUser
	_, err = m.UpdateGroupUser(&types.GroupUpdateUserRequest{
		GroupID:  groupID,
		UserID:   uid,
		Nickname: "Bob",
	})
	assert.NoError(t, err)

	u2, _ := m.GetGroupUser(groupID, uid)
	assert.Equal(t, "Bob", u2.Nickname)

	// 4. ListGroupUsers
	res := m.ListGroupUsers(&types.GroupListUsersRequest{GroupID: groupID})
	assert.Equal(t, 1, res.Total)

	// 5. DeleteGroupUser
	err = m.DeleteGroupUser(groupID, uid)
	assert.NoError(t, err)

	_, err = m.GetGroupUser(groupID, uid)
	assert.Error(t, err, "删除后应当报错")
}

// TestGroup_ThreadSafeOperations 验证群操作的并发安全性
func TestGroup_ThreadSafeOperations(t *testing.T) {
	m := NewManager()
	wg := sync.WaitGroup{}
	count := 100
	groupID := "group_123"

	// Prepare group
	_, _ = m.AddGroup(&types.GroupAddRequest{GroupID: groupID, Name: "测试群"})

	wg.Add(count * 2)

	// 并发：添加/更新群成员
	for i := range count {
		go func(idx int) {
			defer wg.Done()
			uid := fmt.Sprintf("user_%d", idx)
			_, _ = m.AddGroupUser(&types.GroupAddUserRequest{
				GroupID:  groupID,
				UserID:   uid,
				Nickname: "Original",
			})
			_, _ = m.UpdateGroupUser(&types.GroupUpdateUserRequest{
				GroupID:  groupID,
				UserID:   uid,
				Nickname: "Updated",
			})
		}(i)
	}

	// 并发：读取群成员
	for i := range count {
		go func(idx int) {
			defer wg.Done()
			m.ListGroupUsers(&types.GroupListUsersRequest{GroupID: groupID})
			_, _ = m.GetGroupUser(groupID, fmt.Sprintf("user_%d", idx))
		}(i)
	}

	wg.Wait()

	res := m.ListGroupUsers(&types.GroupListUsersRequest{GroupID: groupID})
	assert.Equal(t, count, res.Total, "最终群成员总数应与添加数一致")
}
