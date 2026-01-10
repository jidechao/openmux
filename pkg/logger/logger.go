package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel = LevelInfo
	mu           sync.RWMutex
	std          = log.New(os.Stderr, "", log.LstdFlags)
)

// SetLevel 设置日志级别
func SetLevel(levelStr string) {
	mu.Lock()
	defer mu.Unlock()

	switch strings.ToLower(levelStr) {
	case "debug":
		currentLevel = LevelDebug
	case "info":
		currentLevel = LevelInfo
	case "warn", "warning":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}
}

// SetOutput 设置日志输出
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	std.SetOutput(w)
}

// Debugf 打印调试日志
func Debugf(format string, v ...interface{}) {
	if shouldLog(LevelDebug) {
		output("DEBUG", fmt.Sprintf(format, v...))
	}
}

// Infof 打印信息日志
func Infof(format string, v ...interface{}) {
	if shouldLog(LevelInfo) {
		output("INFO", fmt.Sprintf(format, v...))
	}
}

// Warnf 打印警告日志
func Warnf(format string, v ...interface{}) {
	if shouldLog(LevelWarn) {
		output("WARN", fmt.Sprintf(format, v...))
	}
}

// Errorf 打印错误日志
func Errorf(format string, v ...interface{}) {
	if shouldLog(LevelError) {
		output("ERROR", fmt.Sprintf(format, v...))
	}
}

// Fatalf 打印致命错误并退出
func Fatalf(format string, v ...interface{}) {
	output("FATAL", fmt.Sprintf(format, v...))
	os.Exit(1)
}

func shouldLog(level Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level >= currentLevel
}

func output(levelStr, msg string) {
	std.Output(3, fmt.Sprintf("[%s] %s", levelStr, msg))
}

// Printf 兼容标准库，默认使用 Info 级别
func Printf(format string, v ...interface{}) {
	Infof(format, v...)
}

// Println 兼容标准库
func Println(v ...interface{}) {
	if shouldLog(LevelInfo) {
		output("INFO", fmt.Sprint(v...))
	}
}
