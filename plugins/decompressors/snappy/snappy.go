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

func (c *Snappy) Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

func init() {
	plugins.AddDecompressor("snappy", func() core.Decompressor {
		return &Snappy{}
	})
}
