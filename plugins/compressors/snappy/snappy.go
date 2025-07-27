package snappy

import (
	"github.com/golang/snappy"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Snappy struct{}

func (c *Snappy) Init() error {
	return nil
}

func (c *Snappy) Close() error {
	return nil
}

func (c *Snappy) Compress(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

func init() {
	plugins.AddCompressor("snappy", func() core.Compressor {
		return &Snappy{}
	})
}
