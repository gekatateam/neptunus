package mock

type mockLogger struct{}

func NewLogger() *mockLogger {
	return &mockLogger{}
}

func (l *mockLogger) Tracef(format string, args ...interface{}) {}

func (l *mockLogger) Debugf(format string, args ...interface{}) {}

func (l *mockLogger) Infof(format string, args ...interface{}) {}

func (l *mockLogger) Warnf(format string, args ...interface{}) {}

func (l *mockLogger) Errorf(format string, args ...interface{}) {}

func (l *mockLogger) Fatalf(format string, args ...interface{}) {}

func (l *mockLogger) Trace(args ...interface{}) {}

func (l *mockLogger) Debug(args ...interface{}) {}

func (l *mockLogger) Info(args ...interface{}) {}

func (l *mockLogger) Warn(args ...interface{}) {}

func (l *mockLogger) Error(args ...interface{}) {}

func (l *mockLogger) Fatal(args ...interface{}) {}
