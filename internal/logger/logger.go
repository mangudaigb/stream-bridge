package logger

import (
	"fmt"

	"github.com/jibitesh/request-response-manager/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger   *zap.Logger
	sugar    *zap.SugaredLogger
	useSugar bool
)

func Init() error {
	mode := config.AppConfig.Logger.Env
	levelStr := config.AppConfig.Logger.Level
	encoding := config.AppConfig.Logger.Encoding
	outputPaths := config.AppConfig.Logger.OutputPaths
	errorOutputPaths := config.AppConfig.Logger.ErrorOutputPaths

	if len(outputPaths) == 0 {
		outputPaths = []string{"stdout"}
	}
	if len(errorOutputPaths) == 0 {
		errorOutputPaths = []string{"stderr"}
	}
	if encoding == "" {
		if mode == "dev" {
			encoding = "console"
		} else {
			encoding = "json"
		}
	}

	atomicLevel := zap.NewAtomicLevel()
	lvl, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		if mode == "dev" {
			lvl = zapcore.DebugLevel
		} else {
			lvl = zapcore.InfoLevel
		}
	}
	atomicLevel.SetLevel(lvl)

	var cfg zap.Config
	if mode == "dev" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.Level = atomicLevel
	cfg.OutputPaths = outputPaths
	cfg.ErrorOutputPaths = errorOutputPaths
	cfg.Encoding = encoding

	encCfg := cfg.EncoderConfig
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncoderConfig = encCfg

	zapLogger, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("cannot build zap logger: %w", err)
	}
	logger = zapLogger
	sugar = logger.Sugar()

	if mode == "dev" {
		useSugar = true
	} else {
		useSugar = false
	}
	return nil
}

func Error(msg string, fields ...interface{}) {
	if useSugar {
		sugar.Fatalw(msg, fields...)
	} else {
		logger.Fatal(msg, toZapFields(fields...)...)
	}
}

func Info(msg string, fields ...interface{}) {
	if useSugar {
		sugar.Infow(msg, fields...)
	} else {
		logger.Info(msg, toZapFields(fields...)...)
	}
}

func Debug(msg string, fields ...interface{}) {
	if useSugar {
		sugar.Debugw(msg, fields...)
	} else {
		logger.Debug(msg, toZapFields(fields...)...)
	}
}

func Sync() {
	_ = logger.Sync()
}

func toZapFields(args ...interface{}) []zap.Field {
	var fields []zap.Field

	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			// skip malformed pair
			continue
		}
		value := args[i+1]
		fields = append(fields, zap.Any(key, value))
	}
	return fields
}
