package self

import (
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/core/self"
)

func init() {
	plugins.AddKeykeeper("self", func() core.Keykeeper {
		return &self.Self{}
	})
}
