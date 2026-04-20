package message

import (
	"crypto/rand"
	"fmt"
	"time"
)

// GenerateMsgID 生成消息ID
func GenerateMsgID() string {
	// 生成32位UUID
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x%x%x%x%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// GenerateRandom 生成随机数
func GenerateRandom() string {
	return fmt.Sprintf("%d", GenerateMsgRandom())
}

// GenerateMsgRandom 生成消息随机数
func GenerateMsgRandom() uint32 {
	return uint32(time.Now().UnixNano() / 1e6)
}
