package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/horizoonn/shortener/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger

	file *os.File
}

type contextKey string

const loggerContextKey = contextKey("logger")

func ToContext(ctx context.Context, log *Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, log)
}

func FromContext(ctx context.Context) *Logger {
	log, ok := ctx.Value(loggerContextKey).(*Logger)
	if !ok {
		return &Logger{Logger: zap.NewNop()}
	}

	return log
}

func New(cfg config.LoggerConfig) (*Logger, error) {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("unmarshal log level: %w", err)
	}

	if err := os.MkdirAll(cfg.Folder, 0750); err != nil {
		return nil, fmt.Errorf("mkdir log folder: %w", err)
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")
	logFilePath := filepath.Join(cfg.Folder, fmt.Sprintf("%s.log", timestamp))

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0600) // #nosec G304 -- log path is process configuration.
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000000")
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
		zapcore.NewCore(encoder, zapcore.AddSync(logFile), level),
	)

	return &Logger{
		Logger: zap.New(core, zap.AddCaller()),
		file:   logFile,
	}, nil
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	if l == nil || l.Logger == nil {
		return &Logger{Logger: zap.NewNop()}
	}

	return &Logger{
		Logger: l.Logger.With(fields...),
		file:   l.file,
	}
}

func (l *Logger) Close() {
	if l == nil {
		return
	}

	_ = l.Sync()

	if l.file != nil {
		if err := l.file.Close(); err != nil {
			fmt.Printf("failed to close application logger: %v\n", err)
		}
	}
}
