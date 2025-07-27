package gzip

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Gzip struct {
	CompressionLevel string `mapstructure:"gzip_level"`
	level            int
}

func (c *Gzip) Init() error {
	switch c.CompressionLevel {
	case "NoCompression":
		c.level = flate.NoCompression
	case "BestSpeed":
		c.level = flate.BestSpeed
	case "BestCompression":
		c.level = flate.BestCompression
	case "DefaultCompression":
		c.level = flate.DefaultCompression
	case "HuffmanOnly":
		c.level = flate.HuffmanOnly
	default:
		return fmt.Errorf("unknown compression level: %v", c.CompressionLevel)
	}

	return nil
}

func (c *Gzip) Close() error {
	return nil
}

func (c *Gzip) Compress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	w, err := gzip.NewWriterLevel(buf, c.level)
	if err != nil {
		return nil, err
	}
	w.Write(data)

	if err := w.Flush(); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func init() {
	plugins.AddCompressor("gzip", func() core.Compressor {
		return &Gzip{
			CompressionLevel: "DefaultCompression",
		}
	})
}
