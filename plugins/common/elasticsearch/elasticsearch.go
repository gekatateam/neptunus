package elasticsearch

import (
	"encoding/json"
	"errors"
)

type bulkAction string

const (
	Index  bulkAction = "index"
	Create bulkAction = "create"
)

type IndexId struct {
	Index string `json:"_index"`
	Id    string `json:"_id,omitempty"`
}

type BulkOperation struct {
	Action bulkAction
	IndexId
}

func (bo BulkOperation) MarshalJSON() ([]byte, error) {
	if len(bo.Index) == 0 {
		return nil, errors.New("_index required")
	}

	if len(bo.Action) == 0 {
		return nil, errors.New("action required")
	}

	ii, err := json.Marshal(bo.IndexId)
	if err != nil {
		return nil, err
	}

	var op []byte
	op = append(op, `{ "`...)
	op = append(op, bo.Action...)
	op = append(op, `": `...)
	op = append(op, ii...)
	op = append(op, ` }`...)

	return op, nil
}
