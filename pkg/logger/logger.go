package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds the configuration for the logger
type Config struct {
	Level         string // debug, info, warn, error
	ComponentName string // component name for structured logging
	EnableStdout  bool   // enable stdout logging
	Development   bool   // enable development mode (pretty printing)
}

// DefaultConfig returns a default configuration
func DefaultConfig(componentName string) *Config {
	return &Config{
		Level:         "info",
		ComponentName: componentName,
		EnableStdout:  true,
		Development:   false,
	}
}

// ZapLogger wraps zap.Logger to provide structured logging
type ZapLogger struct {
	logger    *zap.Logger
	component string
}

// NewZapLogger creates a new zap-based logger
func NewZapLogger(config *Config) (*ZapLogger, error) {
	if config == nil {
		config = DefaultConfig("default")
	}

	// parse log level
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", config.Level, err)
	}

	// create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// use different encoder for development
	var encoder zapcore.Encoder
	if config.Development {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	// create logger with caller skip
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	// add component field if provided
	if config.ComponentName != "" {
		logger = logger.With(zap.String("component", config.ComponentName))
	}

	return &ZapLogger{
		logger:    logger,
		component: config.ComponentName,
	}, nil
}

// NewProductionLogger creates a production-ready logger
func NewProductionLogger(componentName string) (*ZapLogger, error) {
	config := &Config{
		Level:         "info",
		ComponentName: componentName,
		EnableStdout:  true,
		Development:   false,
	}
	return NewZapLogger(config)
}

// NewDevelopmentLogger creates a development logger with pretty printing
func NewDevelopmentLogger(componentName string) (*ZapLogger, error) {
	config := &Config{
		Level:         "debug",
		ComponentName: componentName,
		EnableStdout:  true,
		Development:   true,
	}
	return NewZapLogger(config)
}

// Info logs informational messages
func (l *ZapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, convertToFields(keysAndValues)...)
}

// Error logs error messages
func (l *ZapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, convertToFields(keysAndValues)...)
}

// Debug logs debug messages
func (l *ZapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, convertToFields(keysAndValues)...)
}

// Warn logs warning messages
func (l *ZapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, convertToFields(keysAndValues)...)
}

// Fatal logs a fatal message and exits
func (l *ZapLogger) Fatal(msg string, keysAndValues ...interface{}) {
	l.logger.Fatal(msg, convertToFields(keysAndValues)...)
}

// With returns a new logger with additional fields
func (l *ZapLogger) With(keysAndValues ...interface{}) Logger {
	return &ZapLogger{
		logger:    l.logger.With(convertToFields(keysAndValues)...),
		component: l.component,
	}
}

// Sync flushes any buffered log entries
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// convertToFields converts key-value pairs to zap fields
func convertToFields(keysAndValues []interface{}) []zap.Field {
	if len(keysAndValues) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}
			value := keysAndValues[i+1]

			// handle different types
			switch v := value.(type) {
			case error:
				fields = append(fields, zap.Error(v))
			case string:
				fields = append(fields, zap.String(key, v))
			case int:
				fields = append(fields, zap.Int(key, v))
			case int64:
				fields = append(fields, zap.Int64(key, v))
			case float64:
				fields = append(fields, zap.Float64(key, v))
			case bool:
				fields = append(fields, zap.Bool(key, v))
			default:
				fields = append(fields, zap.Any(key, v))
			}
		}
	}
	return fields
}
