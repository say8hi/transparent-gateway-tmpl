package logger

// MockLogger is a mock implementation of Logger for testing
type MockLogger struct{}

// NewMockLogger creates a new mock logger
func NewMockLogger() Logger {
	return &MockLogger{}
}

// Info logs informational messages (no-op in mock)
func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	// No-op for tests
}

// Error logs error messages (no-op in mock)
func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	// No-op for tests
}

// Debug logs debug messages (no-op in mock)
func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	// No-op for tests
}

// Warn logs warning messages (no-op in mock)
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	// No-op for tests
}
