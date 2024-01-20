package elasticsearch

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
)

var _ elastictransport.Logger = (*TransportLogger)(nil)

type TransportLogger struct {
	log *slog.Logger
}

func (l *TransportLogger) LogRoundTrip(_ *http.Request, _ *http.Response, e error, _ time.Time, d time.Duration) error {
	if e != nil {
		l.log.Debug(e.Error())
	}
	l.log.Debug(fmt.Sprintf("request took %v microseconds", d.Microseconds()))
	return nil
}

func (l *TransportLogger) RequestBodyEnabled() bool {
	return false
}

func (l *TransportLogger) ResponseBodyEnabled() bool {
	return false
}
