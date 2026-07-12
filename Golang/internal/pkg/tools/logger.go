// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/tools/logger.go
// GitHub: https://github.com/NanmiCoder
// Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
//
// 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
// 1. 不得用于任何商业用途。
// 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
// 3. 不得进行大规模爬取或对平台造成运营干扰。
// 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
// 5. 不得用于任何非法或不当的用途。
//
// 详细许可条款请参阅项目根目录下的LICENSE文件。
// 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	logger   *log.Logger
	logLevel int
}

// 日志级别
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

// GetLogger 获取一个Logger实例
func GetLogger() *Logger {
	return NewLogger(io.Writer(os.Stdout), "MediaCrawlerPro-Golang", 0, DEBUG)
}

// NewLogger 创建一个新的Logger
func NewLogger(out io.Writer, prefix string, flag int, level int) *Logger {
	// 使用空前缀初始化logger,防止重复前缀
	return &Logger{
		logger:   log.New(out, "", 0),
		logLevel: level,
	}
}

func (l *Logger) SetLogLevel(level int) {
	l.logLevel = level
}

func (l *Logger) logMessage(level string, v ...interface{}) {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		shortFile := file[strings.LastIndex(file, "/")+1:]
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		message := fmt.Sprintf("%s MediaCrawlerPro-Golang %s (%s:%d) - %s",
			timestamp,
			level,
			shortFile,
			line,
			fmt.Sprint(v...))
		l.logger.Output(2, message)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	if l.logLevel <= DEBUG {
		l.logMessage("DEBUG", v...)
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.logLevel <= INFO {
		l.logMessage("INFO", v...)
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.logLevel <= WARN {
		l.logMessage("WARN", v...)
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.logLevel <= ERROR {
		l.logMessage("ERROR", v...)
	}
}

func (l *Logger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

func (l *Logger) SetFlags(flag int) {
	l.logger.SetFlags(flag)
}
