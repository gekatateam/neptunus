package logrus

import (
	"fmt"
	"os"

	"github.com/gekatateam/pipeline/config"

	"github.com/sirupsen/logrus"
)

var extraFields map[string]any

func InitializeLogger(cfg config.Common) error {
	switch l := cfg.LogLevel; l {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		return fmt.Errorf("unknown log level: %v", l)
	}

	switch f := cfg.LogFormat; f {
	case "logfmt": // default formatter
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
		})
	default:
		return fmt.Errorf("unknown log format: %v", f)
	}

	extraFields = cfg.LogFields
	return nil
}

type logrusLogger struct {
	Fields logrus.Fields
}

func NewLogger(f map[string]any) *logrusLogger {
	var loggerFields = make(map[string]any, len(extraFields))
	for k, v := range extraFields {
		loggerFields[k] = v
	}

	for k, v := range f {
		loggerFields[k] = v
	}

	return &logrusLogger{Fields: loggerFields}
}

func (l *logrusLogger) Tracef(format string, args ...interface{}) {
	logrus.WithFields(l.Fields).Tracef(format, args...)
}

func (l *logrusLogger) Debugf(format string, args ...interface{}) {
	logrus.WithFields(l.Fields).Debugf(format, args...)
}

func (l *logrusLogger) Infof(format string, args ...interface{}) {
	logrus.WithFields(l.Fields).Infof(format, args...)
}

func (l *logrusLogger) Warnf(format string, args ...interface{}) {
	logrus.WithFields(l.Fields).Warnf(format, args...)
}

func (l *logrusLogger) Errorf(format string, args ...interface{}) {
	logrus.WithFields(l.Fields).Errorf(format, args...)
}

func (l *logrusLogger) Fatalf(format string, args ...interface{}) {
	logrus.WithFields(l.Fields).Fatalf(format, args...)
}

func (l *logrusLogger) Trace(args ...interface{}) {
	logrus.WithFields(l.Fields).Trace(args...)
}

func (l *logrusLogger) Debug(args ...interface{}) {
	logrus.WithFields(l.Fields).Debug(args...)
}

func (l *logrusLogger) Info(args ...interface{}) {
	logrus.WithFields(l.Fields).Info(args...)
}

func (l *logrusLogger) Warn(args ...interface{}) {
	logrus.WithFields(l.Fields).Warn(args...)
}

func (l *logrusLogger) Error(args ...interface{}) {
	logrus.WithFields(l.Fields).Error(args...)
}

func (l *logrusLogger) Fatal(args ...interface{}) {
	logrus.WithFields(l.Fields).Fatal(args...)
}

func init() {
	// default params
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
	})
}
