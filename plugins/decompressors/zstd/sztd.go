package zstd

import (
	"github.com/klauspost/compress/zstd"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
)

var readersStorage = sharedstorage.New[*zstd.Decoder, int]()

type Zstd struct {
	decoder *zstd.Decoder
}

func (c *Zstd) Init() error {
	r, err := zstd.NewReader(nil)
	if err != nil {
		return err
	}

	c.decoder = readersStorage.CompareAndStore(1, r)

	return nil
}

func (c *Zstd) Close() error {
	if readersStorage.Leave(1) {
		c.decoder.Close()
	}
	return nil
}

func (c *Zstd) Decompress(data []byte) ([]byte, error) {
	return c.decoder.DecodeAll(data, nil)
}

func init() {
	plugins.AddDecompressor("zstd", func() core.Decompressor {
		return &Zstd{}
	})
}
