package logger

import (
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"time"
)

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// WithField 创建字段
func WithField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// WithError 创建错误字段
func WithError(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Level 日志级别
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// Logger JSON 格式日志
type Logger struct {
	mu      sync.Mutex
	level   Level
	output  *os.File
	encoder *json.Encoder
}

// 全局日志
var DefaultLogger *Logger

// Init 初始化
func Init(debug bool) error {
	DefaultLogger = &Logger{
		level:  InfoLevel,
		output: os.Stderr,
	}
	DefaultLogger.encoder = json.NewEncoder(DefaultLogger.output)
	return nil
}

// InitDefault 初始化默认日志
func InitDefault() {
	Init(false)
}

// Sync 同步缓冲
func Sync() {}

// Close 关闭
func Close() {
	if DefaultLogger != nil && DefaultLogger.output != os.Stderr {
		DefaultLogger.output.Sync()
	}
}

// Info 记录信息日志
func Info(msg string, fields ...Field) {
	DefaultLogger.log(InfoLevel, msg, fields)
}

// Error 记录错误日志
func Error(msg string, fields ...Field) {
	DefaultLogger.log(ErrorLevel, msg, fields)
}

// Debug 记录调试日志
func Debug(msg string, fields ...Field) {
	DefaultLogger.log(DebugLevel, msg, fields)
}

// Warn 记录警告日志
func Warn(msg string, fields ...Field) {
	DefaultLogger.log(WarnLevel, msg, fields)
}

// Fatal 记录致命错误并退出
func Fatal(msg string, fields ...Field) {
	DefaultLogger.log(FatalLevel, msg, fields)
	os.Exit(1)
}

// FatalError 便捷方法
func FatalError(err error, msg string) {
	if err != nil {
		Error(msg, WithError(err))
		os.Exit(1)
	}
}

// IfError 便捷方法
func IfError(err error, msg string) {
	if err != nil {
		Error(msg, WithError(err))
	}
}

func (l *Logger) log(level Level, msg string, fields []Field) {
	if level < l.level {
		return
	}

	entry := map[string]interface{}{
		"time":   time.Now().Format(time.RFC3339),
		"level":  []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}[level],
		"msg":    msg,
		"caller": caller(),
	}

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	l.mu.Lock()
	l.encoder.Encode(entry)
	l.mu.Unlock()
}

func caller() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "?"
	}
	return file + ":" + itoa(line)
}

// itoa 简单的数字转字符串
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s []byte
	for n > 0 {
		s = append(s, byte('0'+n%10))
		n /= 10
	}
	// 反转
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return string(s)
}