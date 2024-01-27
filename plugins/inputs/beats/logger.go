package beats

import (
	"fmt"
	"log/slog"

	lumberlog "github.com/elastic/go-lumber/log"
)

var _ lumberlog.Logging = (*lumberLogger)(nil)

type lumberLogger struct {
	log *slog.Logger
}

func (l *lumberLogger) Printf(format string, a ...any) {
	l.log.Debug(fmt.Sprintf(format, a...))
}
func (l *lumberLogger) Println(a ...any) {
	l.log.Debug(fmt.Sprint(a...))
}
func (l *lumberLogger) Print(a ...any) {
	l.log.Debug(fmt.Sprint(a...))
}

type lumberSilentLogger struct{}

func (l *lumberSilentLogger) Printf(format string, a ...any) {}

func (l *lumberSilentLogger) Println(a ...any) {}

func (l *lumberSilentLogger) Print(a ...any) {}

func init() {
	lumberlog.Logger = &lumberSilentLogger{}
}
