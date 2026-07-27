package not

import (
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/core/not"
)

func init() {
	plugins.AddFilter("not", func() core.Filter {
		return &not.Not{}
	})
}
