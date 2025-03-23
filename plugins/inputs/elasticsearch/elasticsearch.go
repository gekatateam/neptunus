package elasticsearch

import (
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/robfig/cron/v3"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Elasticsearch struct {
	*core.BaseInput `mapstructure:"-"`
	Location        string   `mapstructure:"location"`
	Address         []string `mapstructure:"address"`
	Username        string   `mapstructure:"username"`
	Password        string   `mapstructure:"password"`
	APIKey          string   `mapstructure:"api_key"`
	CloudID         string   `mapstructure:"cloud_id"`
	Queries         []*Query `mapstructure:"queries"`

	client *elasticsearch.Client
	cron   *cron.Cron
}

func (i *Elasticsearch) Init() error {
	// Create Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: i.Address,
	}

	// Configure authentication
	if i.Username != "" && i.Password != "" {
		cfg.Username = i.Username
		cfg.Password = i.Password
	} else if i.APIKey != "" {
		cfg.APIKey = i.APIKey
	}

	if i.CloudID != "" {
		cfg.CloudID = i.CloudID
	}

	var err error
	i.client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Verify connection
	_, err = i.client.Info()
	if err != nil {
		return fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}

	// Initialize scheduler
	loc, err := time.LoadLocation(i.Location)
	if err != nil {
		return err
	}

	i.cron = cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(),
		cron.WithLogger(cron.DiscardLogger),
	)

	// Set up each query as a scheduled job
	for _, q := range i.Queries {
		q.BaseInput = i.BaseInput
		q.client = i.client
		if _, err := i.cron.AddJob(q.Schedule, q); err != nil {
			return fmt.Errorf("query %v scheduling failed: %w", q.Name, err)
		}
	}
	return nil
}

func (i *Elasticsearch) Close() error {
	i.cron.Stop()
	return nil
}

func (i *Elasticsearch) Run() {
	i.cron.Run()
}

func init() {
	plugins.AddInput("elasticsearch", func() core.Input {
		return &Elasticsearch{
			Location: "UTC",
			Address:  []string{"http://localhost:9200"},
			Queries:  []*Query{},
		}
	})
}
