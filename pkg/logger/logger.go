package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Initialize(logLevel string) error {
	zLevel, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	config := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			LevelKey:     "level",
			TimeKey:      "time",
			CallerKey:    "caller",
			EncodeLevel:  zapcore.LowercaseLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	log, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

func Logger() *zap.Logger {
	return log
}

func Sync() error {
	return log.Sync()
}
