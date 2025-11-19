// Package logger 提供了日志记录功能
// 支持不同级别的日志（INFO、DEBUG、WARN、ERROR、TEST）
// 每个节点有独立的日志文件，包含高精度时间戳
package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger 日志记录器
// 包含不同级别的日志记录器，所有日志都写入同一个文件
type Logger struct {
	infoLogger      *log.Logger  // 信息级别日志记录器
	debugLogger     *log.Logger  // 调试级别日志记录器
	warnLogger      *log.Logger  // 警告级别日志记录器
	errorLogger     *log.Logger  // 错误级别日志记录器
	testLogger      *log.Logger  // 测试级别日志记录器
	timestampFormat string       // 时间戳格式字符串
}

// NewLogger 初始化日志系统，为每个节点或角色创建日志文件
// 功能：
//  1. 创建 logs 目录（如果不存在）
//  2. 根据节点 ID 和角色生成日志文件名
//  3. 打开日志文件（追加模式）
//  4. 创建不同级别的日志记录器
//  5. 设置高精度时间戳格式
// 参数:
//   - nodeID: 节点 ID（用于生成日志文件名）
//   - role: 角色类型（"node"、"client"、"blockchain"、"result" 等）
// 返回:
//   - *Logger: 初始化完成的日志记录器
func NewLogger(nodeID int64, role string) *Logger {
	// 创建 logs 目录（如果不存在），权限为 0755
	os.MkdirAll("logs", 0755)
	logFile := ""

	// 根据角色生成日志文件名
	switch role {
	case "node":
		// 节点日志：logs/node_{nodeID}.log
		logFile = fmt.Sprintf("logs/node_%d.log", nodeID)
	case "client":
		// 客户端日志：logs/client.log
		logFile = "logs/client.log"
	case "blockchain":
		// 区块链日志：logs/blockchain.log
		logFile = "logs/blockchain.log"
	case "result":
		// 结果日志：logs/result.log
		logFile = "logs/result.log"
	default:
		// 其他日志：logs/others.log
		logFile = "logs/others.log"
	}

	// 打开日志文件（创建、只写、追加模式），权限为 0666
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	// 创建自定义时间戳格式，包含微秒精度
	// 格式：2006-01-02 15:04:05.000000
	timestampFormat := "2006-01-02 15:04:05.000000"

	// 创建不同级别的日志记录器
	// 使用 0 标志表示不添加默认前缀（我们自己添加时间戳）
	l := &Logger{
		infoLogger:  log.New(file, "[INFO] ", 0),
		debugLogger: log.New(file, "[DEBUG] ", 0),
		warnLogger:  log.New(file, "[WARN] ", 0),
		errorLogger: log.New(file, "[ERROR] ", 0),
		testLogger:  log.New(file, "[TEST] ", 0),
	}

	// 设置自定义时间戳格式
	l.setTimestampFormat(timestampFormat)
	return l
}

// setTimestampFormat 设置时间戳格式
// 参数:
//   - format: 时间戳格式字符串（Go 时间格式）
func (l *Logger) setTimestampFormat(format string) {
	l.timestampFormat = format
}

// formatLogMessage 格式化日志消息，添加高精度时间戳
// 功能：
//  1. 获取当前时间并格式化为指定格式
//  2. 格式化消息内容
//  3. 组合时间戳、级别和消息内容
// 参数:
//   - level: 日志级别（如 "[INFO]"）
//   - format: 消息格式字符串
//   - args: 消息参数
// 返回:
//   - string: 格式化后的完整日志消息
func (l *Logger) formatLogMessage(level, format string, args ...interface{}) string {
	// 获取当前时间并格式化为指定格式
	timestamp := time.Now().Format(l.timestampFormat)
	// 格式化消息内容
	message := fmt.Sprintf(format, args...)
	// 组合时间戳、级别和消息内容
	return fmt.Sprintf("%s %s %s", timestamp, level, message)
}

// Info 记录信息级别的日志
// 用于记录一般性的信息，如节点启动、状态变化等
// 参数:
//   - format: 消息格式字符串
//   - args: 消息参数
func (l *Logger) Info(format string, args ...interface{}) {
	if l.infoLogger != nil {
		message := l.formatLogMessage("[INFO]", format, args...)
		l.infoLogger.Print(message)
	}
}

// Debug 记录调试级别的日志
// 用于记录详细的调试信息，如连接建立、消息发送等
// 参数:
//   - format: 消息格式字符串
//   - args: 消息参数
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debugLogger != nil {
		message := l.formatLogMessage("[DEBUG]", format, args...)
		l.debugLogger.Print(message)
	}
}

// Warn 记录警告级别的日志
// 用于记录警告信息，如非致命性错误、异常情况等
// 参数:
//   - format: 消息格式字符串
//   - args: 消息参数
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.warnLogger != nil {
		message := l.formatLogMessage("[WARN]", format, args...)
		l.warnLogger.Print(message)
	}
}

// Error 记录错误级别的日志
// 用于记录错误信息，如连接失败、消息处理错误等
// 参数:
//   - format: 消息格式字符串
//   - args: 消息参数
func (l *Logger) Error(format string, args ...interface{}) {
	if l.errorLogger != nil {
		message := l.formatLogMessage("[ERROR]", format, args...)
		l.errorLogger.Print(message)
	}
}

// Test 记录测试级别的日志
// 用于记录测试相关的信息
// 参数:
//   - format: 消息格式字符串
//   - args: 消息参数
func (l *Logger) Test(format string, args ...interface{}) {
	if l.testLogger != nil {
		message := l.formatLogMessage("[TEST]", format, args...)
		l.testLogger.Print(message)
	}
}
