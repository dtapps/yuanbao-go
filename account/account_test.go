package account

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

// TestManager_ThreadSafety 验证 Manager 在高并发下的读写安全性
func TestManager_ThreadSafety(t *testing.T) {
	m := NewManager()
	wg := sync.WaitGroup{}
	workerCount := 100
	iterations := 100

	wg.Add(workerCount * 2) // 一半写，一半读

	// 并发写入测试
	for i := range workerCount {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				accountID := fmt.Sprintf("acc-%d-%d", id, j)
				acc := &types.Account{AccountID: accountID}
				_, _ = m.AddAccount(accountID, acc)
			}
		}(i)
	}

	// 并发读取测试
	for i := range workerCount {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				m.ListAccounts()
				_, _ = m.GetAccount(fmt.Sprintf("acc-%d-%d", id, j))
			}
		}(i)
	}

	wg.Wait()

	// 最终验证数量
	res := m.ListAccounts()
	assert.Equal(t, workerCount*iterations, res.Total, "账号总数应符合预期")
}

// TestGetManager_Singleton 验证全局单例在并发初始化时是否能保持唯一指针
func TestGetManager_Singleton(t *testing.T) {
	var instances []*Manager
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	spawnCount := 50
	wg.Add(spawnCount)

	for range spawnCount {
		go func() {
			defer wg.Done()
			inst := GetManager()

			mu.Lock()
			instances = append(instances, inst)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 验证所有获取到的实例内存地址是否一致
	firstInstance := instances[0]
	assert.NotNil(t, firstInstance)

	for _, inst := range instances {
		assert.True(t, inst == firstInstance, "所有获取到的 Manager 实例应当是同一个内存地址")
	}
}
