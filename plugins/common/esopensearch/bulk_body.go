package esopensearch

import (
	"bytes"
	"fmt"

	"github.com/goccy/go-json"
)

type BulkBody struct {
	*bytes.Buffer
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
	if err := json.NewEncoder(b.Buffer).Encode(CreateOp{Create: params}); err != nil {
		return fmt.Errorf("bulk.CreateOp: %w", err)
	}

	//b.WriteByte('\n')
	b.Write(doc)
	b.WriteByte('\n')
	return nil
}

func (b *BulkBody) IndexOp(doc []byte, params BulkOp) error {
	if err := json.NewEncoder(b.Buffer).Encode(IndexOp{Index: params}); err != nil {
		return fmt.Errorf("bulk.IndexOp: %w", err)
	}

	//b.WriteByte('\n')
	b.Write(doc)
	b.WriteByte('\n')
	return nil
}
