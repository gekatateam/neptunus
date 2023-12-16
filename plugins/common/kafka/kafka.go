package kafka

import (
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

var _ kafka.Logger = (*kafkaLogger)(nil)

func NewLogger(log *slog.Logger) *kafkaLogger {
	return &kafkaLogger{
		log:   log,
		isErr: false,
	}
}

func NewErrorLogger(log *slog.Logger) *kafkaLogger {
	return &kafkaLogger{
		log:   log,
		isErr: true,
	}
}

type kafkaLogger struct {
	log   *slog.Logger
	isErr bool
}

func (l *kafkaLogger) Printf(msg string, args ...any) {
	if l.isErr {
		l.log.Error(fmt.Sprintf(msg, args...))
	} else {
		l.log.Debug(fmt.Sprintf(msg, args...))
	}
}
