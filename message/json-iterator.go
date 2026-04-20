package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// snakeCaseRegex 匹配 snake_case 的 JSON key（在双引号内）
var snakeCaseRegex = regexp.MustCompile(`"([a-z]+(?:_[a-z0-9]+)+)"`)

func convertSnakeCaseToCamelCase(data []byte) []byte {
	return snakeCaseRegex.ReplaceAllFunc(data, func(match []byte) []byte {
		// 提取匹配的部分，例如: "msg_type"
		s := string(match)
		// 去掉引号
		s = strings.Trim(s, "\"")
		// 转换为 camelCase
		parts := strings.Split(s, "_")
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) > 0 {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		result := strings.Join(parts, "")
		return []byte("\"" + result + "\"")
	})
}

// UnmarshalAny 是一个泛型函数，支持自动识别 JSON 和 Protobuf
// T 必须是指向结构体的指针，且满足 proto.Message 接口（如果是 PB 的话）
func UnmarshalAny[T any](data []byte, v T) error {
	if len(data) == 0 {
		return fmt.Errorf("数据为空")
	}

	data = bytes.TrimSpace(data)

	// 如果是双引号开头
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("解开字符串外壳失败: %w", err)
		}
		data = bytes.TrimSpace([]byte(s))
	}

	// 再次检查，确保现在是 JSON 格式
	if len(data) > 0 && data[0] == '{' {
		// 如果是 Protobuf 类型，先将 snake_case 转换为 camelCase，再用 protojson 解析
		if m, ok := any(v).(proto.Message); ok {
			camelCaseData := convertSnakeCaseToCamelCase(data)
			options := protojson.UnmarshalOptions{
				DiscardUnknown: true,
			}
			if err := options.Unmarshal(camelCaseData, m); err == nil {
				// logger.GetLogger("message").
				// 	Debug("protojson 解析 JSON 成功", logger.F("data", string(data)))
				return nil
			}
		}

		// 如果不是 Protobuf 类型，使用标准 JSON 解析
		if err := json.Unmarshal(data, v); err == nil {
			// logger.GetLogger("message").
			// 	Debug("json 解析 JSON 成功", logger.F("data", string(data)))
			return nil
		}
	}

	// Protobuf 二进制解析
	if m, ok := any(v).(proto.Message); ok {
		if err := proto.Unmarshal(data, m); err == nil {
			// logger.GetLogger("message").
			// 	Debug("proto 解析 Protobuf 成功", logger.F("data", string(data)))
			return nil
		}
	}

	return fmt.Errorf("该类型不支持 Protobuf / JSON 解析 或 数据格式错误")
}
