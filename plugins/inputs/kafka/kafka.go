package kafka

import (
	"log/slog"

	"github.com/segmentio/kafka-go"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Kafka struct {
	alias         string
	pipe          string
	id            uint64
	EnableMetrics bool     `mapstructure:"enable_metrics"`
	Brokers       []string `mapstructure:"brokers"`
	ClientId      string   `mapstructure:"client_id"`

	reader *kafka.Reader

	log    *slog.Logger
	out    chan<- *core.Event
	parser core.Parser
}

func (i *Kafka) Init(config map[string]any, alias, pipeline string, log *slog.Logger) error {
	return nil
}

func (i *Kafka) Prepare(out chan<- *core.Event) {
	i.out = out
}

func (i *Kafka) SetParser(p core.Parser) {
	i.parser = p
}

func (i *Kafka) Run() {

}

func (i *Kafka) Close() error {
	return nil
}

func (i *Kafka) Alias() string {
	return i.alias
}

func init() {
	plugins.AddInput("kafka", func() core.Input {
		return &Kafka{

		}
	})
}
