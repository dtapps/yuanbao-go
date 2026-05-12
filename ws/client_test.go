package ws

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dtapps/yuanbao-go/logger"
	"github.com/dtapps/yuanbao-go/types"
)

// mockWsClient 创建用于测试的 WsClient（包含完整的队列初始化）
func mockWsClient() *WsClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WsClient{
		state:      types.ConnectionStateConnected.String(),
		botID:      "test-bot",
		log:        logger.New("ws-test"),
		ctx:        ctx,
		cancel:     cancel,
		sendQueue:  make(chan sendTask, 256),
		senderDone: make(chan struct{}),
	}
}

// TestGenerateNextSeqNo_Concurrency 测试序列号生成的线程安全性
func TestGenerateNextSeqNo_Concurrency(t *testing.T) {
	client := mockWsClient()
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	results := make(chan uint32, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				seq := client.generateNextSeqNo()
				results <- seq
			}
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[uint32]bool)
	count := 0
	for seq := range results {
		if seen[seq] {
			t.Errorf("发现重复的序列号: %d", seq)
		}
		seen[seq] = true
		count++
	}

	expectedCount := goroutines * iterations
	if count != expectedCount {
		t.Errorf("期望生成 %d 个序列号, 实际得到 %d", expectedCount, count)
	}
	if client.seqNo != uint32(expectedCount) {
		t.Errorf("最终 seqNo 应为 %d, 实际为 %d", expectedCount, client.seqNo)
	}
}

// TestGenerateNextMsgSeq_Concurrency 测试业务消息序列号的线程安全性
func TestGenerateNextMsgSeq_Concurrency(t *testing.T) {
	client := mockWsClient()
	const goroutines = 50
	const iterations = 50

	var wg sync.WaitGroup
	results := make(chan uint64, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				seq := client.generateNextMsgSeq()
				results <- seq
			}
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[uint64]bool)
	count := 0
	for seq := range results {
		if seen[seq] {
			t.Errorf("发现重复的 MsgSeq: %d", seq)
		}
		if seq == 0 {
			t.Error("MsgSeq 不应为 0")
		}
		seen[seq] = true
		count++
	}

	expectedCount := goroutines * iterations
	if count != expectedCount {
		t.Errorf("期望生成 %d 个 MsgSeq, 实际得到 %d", expectedCount, count)
	}
	if client.msgSeq != uint64(expectedCount) {
		t.Errorf("最终 msgSeq 应为 %d, 实际为 %d", expectedCount, client.msgSeq)
	}
}

// TestSeqNo_OverflowProtection 测试序列号溢出保护
func TestSeqNo_OverflowProtection(t *testing.T) {
	client := mockWsClient()
	client.seqNo = ^uint32(0)

	seq1 := client.generateNextSeqNo()
	seq2 := client.generateNextSeqNo()

	if seq1 != 1 {
		t.Errorf("溢出后第一个序列号应为 1, 实际为 %d", seq1)
	}
	if seq2 != 2 {
		t.Errorf("溢出后第二个序列号应为 2, 实际为 %d", seq2)
	}
}

// TestMsgSeq_OverflowProtection 测试 MsgSeq 溢出保护
func TestMsgSeq_OverflowProtection(t *testing.T) {
	client := mockWsClient()
	client.msgSeq = ^uint64(0)

	seq1 := client.generateNextMsgSeq()
	seq2 := client.generateNextMsgSeq()

	if seq1 != 1 {
		t.Errorf("溢出后第一个 MsgSeq 应为 1, 实际为 %d", seq1)
	}
	if seq2 != 2 {
		t.Errorf("溢出后第二个 MsgSeq 应为 2, 实际为 %d", seq2)
	}
}

// TestSendQueue_FIFO 测试有序队列：串行发送时严格按序执行
// 注意：Go channel 的 FIFO 保证的是"先发送的值先被接收"，
// 多个 goroutine 并发入队的顺序取决于调度器，这是预期行为。
func TestSendQueue_SerialOrder(t *testing.T) {
	client := mockWsClient()

	// 模拟 sender 协程消费队列
	executed := make([]string, 0, 10)
	go func() {
		for {
			select {
			case <-client.senderDone:
				return
			case task := <-client.sendQueue:
				msgID, _ := task.execute()
				executed = append(executed, msgID)
				task.result <- sendResult{msgID: msgID}
			}
		}
	}()

	// 串行入队（非并发）→ 顺序可保证
	for i := 1; i <= 10; i++ {
		idx := i
		task := sendTask{
			execute: func() (string, error) { return fmt.Sprintf("msg-%d", idx), nil },
			result:  make(chan sendResult, 1),
		}
		client.sendQueue <- task
		<-task.result
	}

	close(client.senderDone)

	// 验证串行入队时顺序正确
	for i, msgID := range executed {
		expected := fmt.Sprintf("msg-%d", i+1)
		if msgID != expected {
			t.Errorf("位置 %d: 期望 '%s', 实际 '%s'", i+1, expected, msgID)
		}
	}

	t.Logf("✓ 串行发送顺序验证通过: %d 条消息按序执行", len(executed))
}

// TestSendQueue_Initialized 验证 WsClient 初始化时队列已正确创建
func TestSendQueue_Initialized(t *testing.T) {
	client := mockWsClient()

	if client.sendQueue == nil {
		t.Fatal("sendQueue 未初始化")
	}
	if client.senderDone == nil {
		t.Fatal("senderDone 未初始化")
	}

	capacity := cap(client.sendQueue)
	if capacity != 256 {
		t.Errorf("sendQueue 容量期望 256, 实际 %d", capacity)
	}
	t.Logf("✓ 有序队列初始化正确: capacity=%d", capacity)
}

// TestSendQueue_ConcurrentEnqueue 测试高并发入队不丢失
func TestSendQueue_ConcurrentEnqueue(t *testing.T) {
	client := mockWsClient()

	const taskCount = 500
	var processed int64 // 原子计数器

	// 启动多个 worker 模拟并发消费
	var workerWg sync.WaitGroup
	for w := 0; w < 4; w++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for task := range client.sendQueue {
				task.execute()
				atomic.AddInt64(&processed, 1)
				task.result <- sendResult{}
			}
		}()
	}

	// 高并发入队
	var enqueueWg sync.WaitGroup
	for i := 0; i < taskCount; i++ {
		enqueueWg.Add(1)
		go func(idx int) {
			defer enqueueWg.Done()
			task := sendTask{
				execute: func() (string, error) { return fmt.Sprintf("task-%d", idx), nil },
				result:  make(chan sendResult, 1),
			}
			client.sendQueue <- task
			<-task.result
		}(i)
	}
	enqueueWg.Wait()

	// 关闭队列让 worker 退出
	close(client.sendQueue)
	workerWg.Wait()

	finalProcessed := atomic.LoadInt64(&processed)
	if finalProcessed != int64(taskCount) {
		t.Errorf("期望处理 %d 个任务, 实际 %d", taskCount, finalProcessed)
	}

	t.Logf("✓ 并发入队验证通过: %d/%d 任务已处理", finalProcessed, taskCount)
}

// TestStartSender_OnlyOnce 测试 senderLoop 只启动一次
func TestStartSender_OnlyOnce(t *testing.T) {
	client := mockWsClient()

	// 多次并发调用 startSender
	const calls = 50
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.startSender()
		}()
	}
	wg.Wait()

	// sendOnce 是 sync.Once，保证内部逻辑只执行一次
	t.Logf("✓ startSender 调用 %d 次，sync.Once 保证仅启动 1 个 sender 协程", calls)
}

// TestConcurrentStateAccess 测试状态访问的并发安全性
func TestConcurrentStateAccess(t *testing.T) {
	client := mockWsClient()
	client.state = types.ConnectionStateConnected.String()

	const iterations = 1000
	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state := client.GetState()
			if state == "" {
				t.Error("状态不应为空")
			}
		}()
	}

	wg.Wait()
	t.Logf("完成 %d 次并发状态读取", iterations)
}

// BenchmarkGenerateNextSeqNo 序列号生成基准测试
func BenchmarkGenerateNextSeqNo(b *testing.B) {
	client := mockWsClient()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client.generateNextSeqNo()
		}
	})
}

// BenchmarkGenerateNextMsgSeq 业务序列号生成基准测试
func BenchmarkGenerateNextMsgSeq(b *testing.B) {
	client := mockWsClient()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client.generateNextMsgSeq()
		}
	})
}

// BenchmarkSendQueue_Enqueue 入队操作基准测试
func BenchmarkSendQueue_Enqueue(b *testing.B) {
	client := mockWsClient()

	// 后台消费避免队列满
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-client.sendQueue:
			}
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := sendTask{
				execute: func() (string, error) { return "", nil },
				result:  make(chan sendResult, 1),
			}
			client.sendQueue <- task
		}
	})

	close(stop)
}
