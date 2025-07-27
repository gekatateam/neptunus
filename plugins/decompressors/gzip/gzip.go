package gzip

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Gzip struct{}

func (c *Gzip) Init() error {
	return nil
}

func (c *Gzip) Close() error {
	return nil
}

func (c *Gzip) Decompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	buf.Write(data)

	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	data, err = io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func init() {
	plugins.AddDecompressor("gzip", func() core.Decompressor {
		return &Gzip{}
	})
}
