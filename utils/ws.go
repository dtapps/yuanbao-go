package utils

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
)

// ParseDelays 解析延迟时间字符串为 time.Duration 列表
func ParseDelays(s string) []time.Duration {
	parts := strings.Split(s, ",")
	delays := make([]time.Duration, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		d, err := time.ParseDuration(p)
		if err == nil {
			delays = append(delays, d)
		}
	}

	if len(delays) == 0 {
		return []time.Duration{1 * time.Second}
	}

	return delays
}

// EncodeBizPB 编码业务 Protobuf 消息
func EncodeBizPB(msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal proto: %w", err)
	}
	return data, nil
}

// DecodeBizPB 解码业务 Protobuf 消息
func DecodeBizPB[T proto.Message](data []byte) (T, error) {
	var msg T
	if err := proto.Unmarshal(data, msg); err != nil {
		return msg, fmt.Errorf("unmarshal proto: %w", err)
	}
	return msg, nil
}
