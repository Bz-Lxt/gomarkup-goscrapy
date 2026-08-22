package logger

import (
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"goscrapy/internal/timeutil"
)

var (
	mu     sync.RWMutex
	global *zap.Logger
)

func Init(level string) *zap.Logger {
	lvl := parseLevel(level)
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = beijingTimeEncoder
	encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	encCfg.EncodeDuration = zapcore.StringDurationEncoder
	encCfg.CallerKey = "caller"

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(os.Stdout),
		lvl,
	)
	opts := []zap.Option{zap.AddCaller()}
	if lvl == zapcore.DebugLevel {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}
	lg := zap.New(core, opts...)
	mu.Lock()
	global = lg
	mu.Unlock()
	zap.ReplaceGlobals(lg)
	return lg
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic", "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func beijingTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(timeutil.Format(t))
}

func L() *zap.Logger {
	mu.RLock()
	lg := global
	mu.RUnlock()
	if lg != nil {
		return lg
	}
	return Init("info")
}

func Named(name string) *zap.Logger {
	return L().Named(name)
}

func Sync() {
	if lg := L(); lg != nil {
		_ = lg.Sync()
	}
}
