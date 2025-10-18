package zlog

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path"
	"runtime"
)

var logger *zap.Logger
var logPath string // 最终会是“目录/文件名”，如 logs/app.log

// 自动调用：初始化日志
func init() {
	logDir := "/Users/xing/Desktop/test/go-ai/gogochat/logs"
	logPath = path.Join(logDir, "app.log") // 拼接日志文件路径：logs/app.log

	// 2. 确保日志目录存在（不存在则创建，避免lumberjack报错）
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("创建日志目录失败：" + err.Error()) // 明确报错，避免隐藏问题
	}

	// 3. 配置日志格式（JSON格式，带ISO时间）
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // 时间格式：2025-10-11T15:58:02+0800
	encoder := zapcore.NewJSONEncoder(encoderConfig)      // 日志输出为JSON格式

	// 4. 获取日志写入器（使用lumberjack，负责创建文件、切割日志）
	fileWriteSyncer := getFileLogWriter() // 关键：调用之前定义的lumberjack逻辑

	// 5. 创建zap核心：同时输出到“控制台”和“文件”，日志级别为Debug
	core := zapcore.NewTee(
		// 输出到控制台（Debug级别，方便开发调试）
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		// 输出到文件（Debug级别，所有日志都存文件）
		zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
	)

	// 6. 初始化zap logger
	logger = zap.New(core)
}

// getFileLogWriter：基于lumberjack实现日志文件管理（切割、备份）
func getFileLogWriter() (writeSyncer zapcore.WriteSyncer) {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   logPath,   // 日志文件路径（已拼接好：logs/app.log）
		MaxSize:    100,       // 单个日志文件最大100MB
		MaxBackups: 60,        // 最多保留60个备份文件（超过则删除旧文件）
		MaxAge:     1,         // 日志文件最多保留1天（超过则删除）
		Compress:   false,     // 不压缩备份文件（开发阶段简化）
	}
	return zapcore.AddSync(lumberJackLogger)
}

// 以下函数（getCallerInfoForLog、Info、Warn等）保持不变，无需修改
func getCallerInfoForLog() (callerFields []zap.Field) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = path.Base(funcName)
	callerFields = append(callerFields, zap.String("func", funcName), zap.String("file", file), zap.Int("line", line))
	return
}

func Info(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Info(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Fatal(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Debug(message, fields...)
}