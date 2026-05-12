package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// LogBuffer 是一个内存环形日志缓冲区，同时写入 stdout
type LogBuffer struct {
	mu      sync.RWMutex
	entries []string
	maxSize int
	file    *os.File
}

var logBuffer *LogBuffer

// InitLogBuffer 初始化日志缓冲区，同时写入文件和内存
func InitLogBuffer(maxSize int) {
	logBuffer = &LogBuffer{
		entries: make([]string, 0, maxSize),
		maxSize: maxSize,
	}

	// 打开日志文件
	logFile, err := os.OpenFile("../db/datas/bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("无法打开日志文件:", err)
		return
	}
	logBuffer.file = logFile
}

// LogWriter 实现 io.Writer 接口，将输出同时写入内存缓冲和 stdout
type LogWriter struct{}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	if logBuffer != nil {
		logBuffer.append(string(p))
	}
	// 同时写入 stdout
	return os.Stdout.Write(p)
}

// FileLogWriter 同时写入内存缓冲、文件和 stdout
type FileLogWriter struct{}

func (flw *FileLogWriter) Write(p []byte) (n int, err error) {
	if logBuffer != nil {
		logBuffer.append(string(p))
		if logBuffer.file != nil {
			logBuffer.file.Write(p)
		}
	}
	return os.Stdout.Write(p)
}

func (lb *LogBuffer) append(s string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), s)
	lb.entries = append(lb.entries, entry)
	if len(lb.entries) > lb.maxSize {
		lb.entries = lb.entries[len(lb.entries)-lb.maxSize:]
	}
}

// GetLogs 返回最近的日志条目
func (lb *LogBuffer) GetLines(n int) []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	total := len(lb.entries)
	if n <= 0 || n > total {
		n = total
	}
	start := total - n
	result := make([]string, n)
	copy(result, lb.entries[start:])
	return result
}

// SetupLogOutput 将日志系统初始化完成的标记
func SetupLogOutput() {
	if logBuffer != nil {
		fmt.Println("日志系统已初始化")
	}
}
