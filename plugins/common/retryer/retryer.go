package retryer

import (
	"fmt"
	"log/slog"
	"time"
)

// Retryer retries passed tryFunc until RetryAttempts reached or func returns nil error.
type Retryer struct {
	RetryAttempts int           `mapstructure:"retry_attempts"`
	RetryAfter    time.Duration `mapstructure:"retry_after"`
}

func (r *Retryer) Do(action string, log *slog.Logger, tryFunc func() error) error {
	var attempts int = 1
	for {
		var err error
		if err = tryFunc(); err == nil {
			log.Debug(fmt.Sprintf("%v succeeded on %v of %v attempt", action, attempts, r.RetryAttempts))
			return nil
		}

		switch {
		case r.RetryAttempts > 0 && attempts < r.RetryAttempts:
			log.Warn(fmt.Sprintf("%v attempt %v of %v failed", action, attempts, r.RetryAttempts),
				"error", err,
			)
			attempts++
			time.Sleep(r.RetryAfter)
		case r.RetryAttempts > 0 && attempts >= r.RetryAttempts:
			log.Error(fmt.Sprintf("%v failed after %v attempts", action, attempts),
				"error", err,
			)
			return err
		default:
			log.Error(fmt.Sprintf("%v failed", action),
				"error", err,
			)
			time.Sleep(r.RetryAfter)
		}
	}
}
