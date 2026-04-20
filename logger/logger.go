package logger

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// sensitiveJsonKeys 敏感字段列表
var sensitiveJsonKeys = []string{"app_secret", "token", "appSecret", "app_key", "appKey"}

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Level int

// String 返回日志级别对应的字符串
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

var (
	globalLevel          atomic.Int32
	globalOutput         = defaultOutput
	globalShowLibrary    = true
	defaultLibraryPrefix = "[yuanbao] "
	mu                   sync.RWMutex
)

type LogEntry struct {
	Time    string
	Level   Level
	Module  string
	Message string
	Fields  map[string]any
}

type Logger struct {
	mu     sync.RWMutex
	module string
	level  Level
	output func(entry LogEntry)
}

func init() {
	globalLevel.Store(int32(LevelInfo))
}

// defaultOutput 默认输出函数
func defaultOutput(entry LogEntry) {
	fields := ""
	if len(entry.Fields) > 0 {
		parts := make([]string, 0, len(entry.Fields))
		for k, v := range entry.Fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fields = strings.Join(parts, " ")
	}

	// 构建消息
	msg := entry.Message
	if fields != "" {
		msg = msg + " " + fields
	}

	// 是否显示前缀
	libraryPrefix := ""
	if globalShowLibrary {
		libraryPrefix = defaultLibraryPrefix
	}

	fmt.Printf("%s%s [%s] [%s] %s\n",
		libraryPrefix,
		entry.Time,
		entry.Level.String(),
		entry.Module,
		msg,
	)
}

// SetLevel 设置全局日志级别
func SetLevel(level Level) {
	globalLevel.Store(int32(level))
}

// SetShowLibrary 设置是否显示库前缀
func SetShowLibrary(show bool) {
	mu.Lock()
	defer mu.Unlock()
	globalShowLibrary = show
}

// SetLevelByName 设置全局日志级别（通过字符串）
func SetLevelByName(name string) error {
	var lvl Level
	switch strings.ToUpper(name) {
	case "DEBUG":
		lvl = LevelDebug
	case "INFO":
		lvl = LevelInfo
	case "WARN":
		lvl = LevelWarn
	case "ERROR":
		lvl = LevelError
	default:
		return fmt.Errorf("unsupported log level: %s", name)
	}
	SetLevel(lvl)
	return nil
}

// SetOutput 设置全局输出函数
func SetOutput(output func(entry LogEntry)) {
	mu.Lock()
	defer mu.Unlock()
	globalOutput = output
}

// New 创建新的 Logger 实例
func New(module string) *Logger {
	return &Logger{
		module: module,
		level:  Level(globalLevel.Load()),
		output: func(entry LogEntry) {
			mu.RLock()
			defer mu.RUnlock()
			globalOutput(entry)
		},
	}
}

// defaultLogger 默认 Logger 实例
var defaultLogger *Logger

// GetLogger 获取 Logger 实例
func GetLogger(module string) *Logger {
	if module == "" {
		if defaultLogger == nil {
			defaultLogger = New("app")
		}
		return defaultLogger
	}
	return New(module)
}

// log 记录日志
func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05.000"),
		Level:   level,
		Module:  l.module,
		Message: msg,
		Fields:  fields,
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.output != nil {
		l.output(entry)
	}
}

// Debug 记录 DEBUG 日志
func (l *Logger) Debug(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelDebug, msg, f)
}

// Info 记录 INFO 日志
func (l *Logger) Info(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelInfo, msg, f)
}

// Warn 记录 WARN 日志
func (l *Logger) Warn(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelWarn, msg, f)
}

// Error 记录 ERROR 日志
func (l *Logger) Error(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelError, msg, f)
}

// mergeFields 合并字段
func mergeFields(fields []map[string]any) map[string]any {
	result := make(map[string]any)
	for _, f := range fields {
		for k, v := range f {
			result[k] = sanitize(v)
		}
	}
	return result
}

func sanitize(v any) any {
	switch val := v.(type) {
	case string:
		return sanitizeString(val)
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = sanitize(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = sanitize(v)
		}
		return result
	default:
		return val
	}
}

func sanitizeString(s string) string {
	return s
}

// Fields 创建日志字段的便捷方法
func Fields() map[string]any {
	return make(map[string]any)
}

// F 添加单个字段
func F(key string, value any) map[string]any {
	return map[string]any{key: value}
}

// FS 添加脱敏字段
func FS(key string, value any) map[string]any {
	if value == nil {
		return map[string]any{key: nil}
	}

	// 如果是 Debug 级别，不进行脱敏
	if Level(globalLevel.Load()) == LevelDebug {
		return map[string]any{key: value}
	}

	switch v := value.(type) {
	case string:
		// 如果是 JSON 字符串
		if len(v) > 2 && v[0] == '{' {
			return map[string]any{key: string(maskJsonBytes([]byte(v), sensitiveJsonKeys...))}
		}

		// 如果是 URL (识别 http/https/ws/wss)
		if strings.Contains(v, "://") {
			return map[string]any{key: maskURL(v)}
		}

		// 普通字符串直接遮掩
		return map[string]any{key: maskValue(v)}

	case []byte:
		// 字节流直接扫描脱敏
		return map[string]any{key: string(maskJsonBytes(v, sensitiveJsonKeys...))}

	default:
		return map[string]any{key: value}
	}
}

// FieldsFromMap 从 map 创建日志字段
func FieldsFromMap(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		result[k] = sanitize(v)
	}
	return result
}

// CapturePanic 捕获 panic 并记录
func CapturePanic(l *Logger) {
	if r := recover(); r != nil {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		l.Error("Panic recovered", map[string]any{
			"panic": fmt.Sprintf("%v", r),
			"stack": string(buf[:n]),
		})
	}
}

// maskValue 隐藏敏感值
func maskValue(value string) string {
	l := len(value)
	if l == 0 {
		return ""
	}

	// 将字符串转为 rune 切片（处理 UTF-8 字符更安全）或 byte 切片（追求极致性能）
	// 这里使用 byte 演示，逻辑与 maskRawRange 一致
	b := []byte(value)

	if l <= 8 {
		for i := range b {
			b[i] = '*'
		}
	} else {
		// 保留前 3 位和后 3 位，中间全部替换为 *
		for i := 3; i < l-3; i++ {
			b[i] = '*'
		}
	}
	return string(b)
}

// maskJsonBytes 高性能字节流脱敏：不反序列化，直接指针扫描，支持自定义 Key
func maskJsonBytes(data []byte, keys ...string) []byte {
	if len(data) == 0 {
		return data
	}

	// 浅拷贝一份，避免修改原始业务数据
	res := make([]byte, len(data))
	copy(res, data)

	for _, k := range keys {
		keyTag := []byte("\"" + k + "\"")
		cursor := 0
		for {
			// 1. 定位 Key
			idx := bytes.Index(res[cursor:], keyTag)
			if idx == -1 {
				break
			}

			startSearch := cursor + idx + len(keyTag)
			quoteStart := -1
			// 2. 寻找 Value 的起始引号
			for i := startSearch; i < len(res); i++ {
				if res[i] == '"' {
					quoteStart = i + 1
					break
				}
				// 容错：如果遇到对象结束还没找到引号，说明不是预期的 JSON 字符串格式
				if res[i] == '}' || res[i] == ',' {
					break
				}
			}

			if quoteStart != -1 {
				// 3. 寻找 Value 的结束引号
				for i := quoteStart; i < len(res); i++ {
					// 处理转义引号 \"，确保不会打码过头
					if res[i] == '"' && res[i-1] != '\\' {
						maskRawRange(res[quoteStart:i])
						cursor = i + 1
						break
					}
				}
			} else {
				cursor = startSearch
			}
			if cursor >= len(res) {
				break
			}
		}
	}
	return res
}

// maskRawRange 基础内存遮掩：保留头尾各3位
func maskRawRange(b []byte) {
	l := len(b)
	if l <= 8 {
		for i := range b {
			b[i] = '*'
		}
	} else {
		for i := 3; i < l-3; i++ {
			b[i] = '*'
		}
	}
}

// maskURL 精准脱敏 URL 中的参数 (高性能版)
func maskURL(u string) string {

	keys := []string{}
	for _, key := range sensitiveJsonKeys {
		keys = append(keys, key+"=")
	}

	result := u
	for _, key := range keys {
		start := strings.Index(result, key)
		if start == -1 {
			continue
		}

		valStart := start + len(key)
		// 寻找参数的结束符 (& 或 字符串末尾)
		valEnd := strings.Index(result[valStart:], "&")

		if valEnd == -1 {
			// 参数在最后: token=xxxxxx
			rawVal := result[valStart:]
			result = result[:valStart] + maskValue(rawVal)
		} else {
			// 参数在中间: token=xxxxxx&version=1.0
			rawVal := result[valStart : valStart+valEnd]
			result = result[:valStart] + maskValue(rawVal) + result[valStart+valEnd:]
		}
	}
	return result
}
