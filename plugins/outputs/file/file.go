package file

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
)

type File struct {
	*core.BaseOutput `mapstructure:"-"`
	Path             string `mapstructure:"path"`
	Append           bool   `mapstructure:"append"`

	file *os.File
	ser  core.Serializer
}

func (o *File) Init() error {
	if o.Path == "" {
		return fmt.Errorf("file path is not set")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(o.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file with appropriate flags
	flag := os.O_WRONLY | os.O_CREATE
	if o.Append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(o.Path, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	o.file = file
	return nil
}

func (o *File) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *File) Run() {
	for e := range o.In {
		now := time.Now()
		event, err := o.ser.Serialize(e)
		if err != nil {
			o.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		// Write to file
		_, err = o.file.Write(append(event, '\n'))
		if err != nil {
			o.Log.Error("failed to write to file",
				"error", err,
				"path", o.Path,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		o.Done <- e
		o.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (o *File) Close() error {
	if o.file != nil {
		return o.file.Close()
	}
	return nil
}

func init() {
	plugins.AddOutput("file", func() core.Output {
		return &File{
			Append: true,
		}
	})
}
