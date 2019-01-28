package main

import (
	"fmt"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initLogger(logpath string, loglevel string) *zap.Logger {

	hook := lumberjack.Logger{
		Filename:   logpath, // 日志文件路径
		MaxSize:    128,     // megabytes
		MaxBackups: 30,      // 最多保留30个备份
		MaxAge:     7,       // days
		Compress:   false,
	}

	w := zapcore.AddSync(&hook)

	// 设置日志级别
	var level zapcore.Level
	switch loglevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		w,
		level,
	)

	logger := zap.New(core)
	logger.Info("DefaultLogger init success")

	return logger
}

func main() {
	// 服务重新启动, 日志会追加, 不会删除
	logger := initLogger("./test.log", "debug")
	for i := 0; i < 6; i++ {
		logger.Info(fmt.Sprint("Info log ", i), zap.Int("line", 47))
		logger.Debug(fmt.Sprint("Debug log ", i), zap.ByteString("level", []byte("xxx")))
		logger.Info(fmt.Sprint("Info log ", i), zap.String("level", `{"a":"4","b":"5"}`))
		logger.Warn(fmt.Sprint("Warn log ", i), zap.String("level", `{"a":"7","b":"8"}`))
	}
	logger.Info("==============================")
	logger.Info("failed to fetch URL",
		// Structured context as strongly typed Field values.
		zap.String("url", "www.google.com"),
		zap.Int("attempt", 3),
		zap.Duration("backoff", time.Second),
	)
}
