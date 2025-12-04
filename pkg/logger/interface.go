package logger

// Logger defines a common interface for logging across the application
type Logger interface {
	// Info logs informational messages
	Info(msg string, keysAndValues ...interface{})

	// Error logs error messages
	Error(msg string, keysAndValues ...interface{})

	// Debug logs debug messages
	Debug(msg string, keysAndValues ...interface{})

	// Warn logs warning messages
	Warn(msg string, keysAndValues ...interface{})
}
