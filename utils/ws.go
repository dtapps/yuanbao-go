package utils

import (
	"strings"
	"time"
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
