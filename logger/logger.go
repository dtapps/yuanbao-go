package logger

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

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

var (
	globalLevel          = LevelInfo
	globalOutput         = defaultOutput
	globalShowLibrary    = true
	defaultLibraryPrefix = "[yuanbao] "
	mu                   sync.RWMutex
)

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

func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	globalLevel = level
}

func SetShowLibrary(show bool) {
	mu.Lock()
	defer mu.Unlock()
	globalShowLibrary = show
}

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

func SetOutput(output func(entry LogEntry)) {
	mu.Lock()
	defer mu.Unlock()
	globalOutput = output
}

func New(module string) *Logger {
	return &Logger{
		module: module,
		level:  globalLevel,
		output: func(entry LogEntry) {
			mu.RLock()
			defer mu.RUnlock()
			globalOutput(entry)
		},
	}
}

var defaultLogger *Logger

func GetLogger(module string) *Logger {
	if module == "" {
		if defaultLogger == nil {
			defaultLogger = New("app")
		}
		return defaultLogger
	}
	return New(module)
}

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

func (l *Logger) Debug(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelDebug, msg, f)
}

func (l *Logger) Info(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelInfo, msg, f)
}

func (l *Logger) Warn(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelWarn, msg, f)
}

func (l *Logger) Error(msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(LevelError, msg, f)
}

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

var sensitiveKeys = map[string]bool{
	"token":      true,
	"signature":  true,
	"app_key":    true,
	"appkey":     true,
	"appsecret":  true,
	"app_secret": true,
	"secret":     true,
	"password":   true,
	"x-token":    true,
}

func sanitizeString(s string) string {
	return s
}

func MaskValue(value string) string {
	if len(value) < 8 {
		return "***"
	}
	return value[:3] + "****" + value[len(value)-3:]
}

// Fields 创建日志字段的便捷方法
func Fields() map[string]any {
	return make(map[string]any)
}

// F 添加单个字段
func F(key string, value any) map[string]any {
	return map[string]any{key: value}
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
