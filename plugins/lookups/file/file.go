package file

import (
	"errors"
	"os"
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/core/lookup"
)

type File struct {
	*core.BaseLookup `mapstructure:"-"`
	File             string `mapstructure:"file"`

	parser core.Parser
}

func (l *File) Init() error {
	return nil
}

func (l *File) Close() error {
	return l.parser.Close()
}

func (l *File) SetParser(p core.Parser) {
	l.parser = p
}

func (l *File) Update() (any, error) {
	content, err := os.ReadFile(l.File)
	if err != nil {
		return nil, err
	}

	event, err := l.parser.Parse(content, "")
	if err != nil {
		return nil, err
	}

	if len(event) == 0 {
		return nil, errors.New("parser returns zero events, nothing to store")
	}

	if len(event) > 1 {
		l.Log.Warn("parser returns more than one event, only first event will be used for lookup data")
	}

	return event[0].Data, nil
}

func init() {
	plugins.AddLookup("file", func() core.Lookup {
		return &lookup.Lookup{
			LazyLookup: &File{},
			Interval:   30 * time.Second,
		}
	})
}
