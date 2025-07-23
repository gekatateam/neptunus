package baiscpools

import (
	"bytes"
	"sync"
)

var BytesBuffer = &sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}
