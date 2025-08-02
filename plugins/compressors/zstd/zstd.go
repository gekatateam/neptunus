package zstd

import (
	"fmt"

	"github.com/klauspost/compress/zstd"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
)

var writersStorage = sharedstorage.New[*zstd.Encoder, zstd.EncoderLevel]()

type Zstd struct {
	CompressionLevel string `mapstructure:"zstd_level"`

	level   zstd.EncoderLevel
	encoder *zstd.Encoder
}

func (c *Zstd) Init() error {
	switch c.CompressionLevel {
	case "Fastest":
		c.level = zstd.SpeedFastest
	case "Default":
		c.level = zstd.SpeedDefault
	case "BetterCompression":
		c.level = zstd.SpeedBetterCompression
	case "BestCompression":
		c.level = zstd.SpeedBestCompression
	default:
		return fmt.Errorf("unknown compression level: %v", c.CompressionLevel)
	}

	w, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(c.level))
	if err != nil {
		return err
	}

	c.encoder = writersStorage.CompareAndStore(c.level, w)

	return nil
}

func (c *Zstd) Close() error {
	if writersStorage.Leave(c.level) {
		return c.encoder.Close()
	}
	return nil
}

func (c *Zstd) Compress(data []byte) ([]byte, error) {
	return c.encoder.EncodeAll(data, nil), nil
}

func init() {
	plugins.AddCompressor("zstd", func() core.Compressor {
		return &Zstd{
			CompressionLevel: "Default",
		}
	})
}
