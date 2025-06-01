package beats

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	lumber "github.com/elastic/go-lumber/server/v2"
	"github.com/gekatateam/mappath"
	"github.com/goccy/go-json"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

type Beats struct {
	*core.BaseInput  `mapstructure:"-"`
	Address          string            `mapstructure:"address"`
	KeepaliveTimeout time.Duration     `mapstructure:"keepalive_timeout"`
	NetworkTimeout   time.Duration     `mapstructure:"network_timeout"`
	NumWorkers       int               `mapstructure:"num_workers"`
	AckOnDelivery    bool              `mapstructure:"ack_on_delivery"`
	KeepTimestamp    bool              `mapstructure:"keep_timestamp"`
	LabelMetadata    map[string]string `mapstructure:"labelmetadata"`

	*ider.Ider              `mapstructure:",squash"`
	*pkgtls.TLSServerConfig `mapstructure:",squash"`

	server    *lumber.Server
	listener  net.Listener
	tlsConfig *tls.Config
}

func (i *Beats) Init() error {
	if len(i.Address) == 0 {
		return errors.New("address required")
	}

	if err := i.Ider.Init(); err != nil {
		return err
	}

	tlsConfig, err := i.TLSServerConfig.Config()
	if err != nil {
		return err
	}
	i.tlsConfig = tlsConfig

	var listener net.Listener
	if i.TLSServerConfig.Enable {
		l, err := tls.Listen("tcp", i.Address, tlsConfig)
		if err != nil {
			return fmt.Errorf("error creating TLS listener: %v", err)
		}
		listener = l
	} else {
		l, err := net.Listen("tcp", i.Address)
		if err != nil {
			return fmt.Errorf("error creating listener: %v", err)
		}
		listener = l
	}
	i.listener = listener

	return nil
}

func (i *Beats) Close() error {
	if i.server != nil {
		if err := i.server.Close(); err != nil {
			i.Log.Error("beats server graceful shutdown ended with error",
				"error", err.Error(),
			)
		}
	}

	return i.listener.Close()
}

func (i *Beats) Run() {
	i.Log.Info(fmt.Sprintf("starting lumberjack server on %v", i.Address))
	if server, err := lumber.NewWithListener(i.listener,
		lumber.Keepalive(i.KeepaliveTimeout),
		lumber.Timeout(i.NetworkTimeout),
		lumber.TLS(i.tlsConfig),
		lumber.JSONDecoder(json.Unmarshal),
	); err != nil {
		i.Log.Error("lumberjack server startup failed",
			"error", err.Error(),
		)
		return
	} else {
		i.Log.Info("lumberjack server started")
		i.server = server
	}

	wg := &sync.WaitGroup{}
	for j := 0; j < i.NumWorkers; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			batchWg := &sync.WaitGroup{}
			for ljBatch := range i.server.ReceiveChan() {
				for _, v := range ljBatch.Events {
					now := time.Now()
					event, err := i.toEvent(v)
					if err != nil {
						i.Log.Error("beat event reading error",
							"error", err,
						)
						i.Observe(metrics.EventFailed, time.Since(now))
						continue
					}

					if i.AckOnDelivery {
						batchWg.Add(1)
						event.AddHook(batchWg.Done)
					}

					if i.KeepTimestamp {
						if rawTimestamp, err := event.GetField("@timestamp"); err == nil {
							if timestamp, err := time.Parse(time.RFC3339Nano, rawTimestamp.(string)); err == nil {
								event.Timestamp = timestamp
							}
						}
					}

					for label, metadata := range i.LabelMetadata {
						if m, err := event.GetField("@metadata." + metadata); err == nil {
							event.SetLabel(label, m.(string))
						}
					}

					i.Ider.Apply(event)

					i.Out <- event
					i.Log.Debug("event accepted",
						elog.EventGroup(event),
					)
					i.Observe(metrics.EventAccepted, time.Since(now))
				}

				batchWg.Wait()
				ljBatch.ACK()
			}
		}()
	}

	wg.Wait()
}

func (i *Beats) toEvent(beatEvent any) (*core.Event, error) {
	rawData, ok := beatEvent.(map[string]any)
	if !ok {
		return nil, errors.New("received event is not representable as core.Event")
	}

	beat, err := mappath.Get(rawData, "@metadata.beat")
	if err != nil {
		return nil, errors.New("received event has no @metadata.beat field")
	}

	return core.NewEventWithData("beats."+beat.(string), rawData), nil
}

func init() {
	plugins.AddInput("beats", func() core.Input {
		return &Beats{
			Address:          ":5044",
			KeepaliveTimeout: 3 * time.Second,
			NetworkTimeout:   30 * time.Second,
			NumWorkers:       runtime.NumCPU(),
			AckOnDelivery:    true,
			KeepTimestamp:    true,
			Ider:             &ider.Ider{},
			TLSServerConfig:  &pkgtls.TLSServerConfig{},
		}
	})
}
