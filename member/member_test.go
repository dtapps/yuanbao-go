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

// TestManager_ThreadSafeOperations 验证成员管理器在并发读写下的安全性
func TestManager_ThreadSafeOperations(t *testing.T) {
	m := NewManager()
	wg := sync.WaitGroup{}
	count := 100

	wg.Add(count * 2)

	// 并发：添加/更新用户
	for i := range count {
		go func(idx int) {
			defer wg.Done()
			uid := fmt.Sprintf("user_%d", idx)
			_, _ = m.AddUser(&types.MemberAddUserRequest{
				UserID:   uid,
				Nickname: "Original",
			})
			// 紧接着尝试更新
			_, _ = m.UpdateUser(&types.MemberUpdateUserRequest{
				UserID:   uid,
				Nickname: "Updated",
			})
		}(i)
	}

	// 并发：读取用户
	for i := range count {
		go func(idx int) {
			defer wg.Done()
			m.ListUsers(&types.MemberListUsersRequest{})
			_, _ = m.GetUser(fmt.Sprintf("user_%d", idx))
		}(i)
	}

	wg.Wait()

	res := m.ListUsers(&types.MemberListUsersRequest{})
	assert.Equal(t, count, res.Total, "最终用户总数应与添加数一致")
}

// TestGetManager_ThreadSafeInitialization 验证跨账号获取管理器时的并发安全性
func TestGetManager_ThreadSafeInitialization(t *testing.T) {
	wg := sync.WaitGroup{}
	accountIDs := []string{"acc_1", "acc_2", "acc_3"}
	iterations := 50

	wg.Add(len(accountIDs) * iterations)

	for _, accID := range accountIDs {
		for range iterations {
			go func(id string) {
				defer wg.Done()
				mgr := GetManager(id)
				assert.NotNil(t, mgr)
			}(accID)
		}
	}

	wg.Wait()
}

// TestManager_CRUD_Logic 验证基础增删改查逻辑
func TestManager_CRUD_Logic(t *testing.T) {
	m := NewManager()
	uid := "12345"

	// 1. Add
	_, _ = m.AddUser(&types.MemberAddUserRequest{UserID: uid, Nickname: "Alice"})

	// 2. Get
	u, err := m.GetUser(uid)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", u.Nickname)

	// 3. Update
	_, _ = m.UpdateUser(&types.MemberUpdateUserRequest{UserID: uid, Nickname: "Bob"})
	u2, _ := m.GetUser(uid)
	assert.Equal(t, "Bob", u2.Nickname)

	// 4. Delete
	_ = m.DeleteUser(uid)
	_, err = m.GetUser(uid)
	assert.Error(t, err, "删除后应当报错")

	// 5. Clear
	_, _ = m.AddUser(&types.MemberAddUserRequest{UserID: "any", Nickname: "any"})
	m.Clear()
	assert.Equal(t, 0, m.ListUsers(&types.MemberListUsersRequest{}).Total)
}
