package mixer

import (
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/core/mixer"
)

func init() {
	plugins.AddProcessor("mixer", func() core.Processor {
		return &mixer.Mixer{}
	})
}
