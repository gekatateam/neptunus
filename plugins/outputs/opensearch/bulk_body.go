package opensearch

import (
	"bytes"
	"fmt"

	"github.com/goccy/go-json"
)

type BulkBody struct {
	buf *bytes.Buffer
}

type CreateOp struct {
	Create BulkOp `json:"create"`
}

type IndexOp struct {
	Index BulkOp `json:"index"`
}

type BulkOp struct {
	Index   string `json:"_index"`
	Id      string `json:"_id"`
	Routing string `json:"routing,omitempty"`
}

func (b *BulkBody) CreateOp(doc []byte, params BulkOp) error {
	if err := json.NewEncoder(b.buf).Encode(CreateOp{Create: params}); err != nil {
		return fmt.Errorf("bulk.CreateOp: %w", err)
	}

	b.buf.WriteByte('\n')
	b.buf.Write(doc)
	b.buf.WriteByte('\n')
	return nil
}

func (b *BulkBody) IndexOp(doc []byte, params BulkOp) error {
	if err := json.NewEncoder(b.buf).Encode(IndexOp{Index: params}); err != nil {
		return fmt.Errorf("bulk.IndexOp: %w", err)
	}

	b.buf.WriteByte('\n')
	b.buf.Write(doc)
	b.buf.WriteByte('\n')
	return nil
}

func (b *BulkBody) Read(p []byte) (n int, err error) {
	return b.Read(p)
}
