package beats

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	lumberlog "github.com/elastic/go-lumber/log"
	lumber "github.com/elastic/go-lumber/server/v2"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/ider"
	pkgtls "github.com/gekatateam/neptunus/plugins/common/tls"
)

type Beats struct {
	alias          string
	pipe           string
	Address        string            `mapstructure:"address"`
	KeepaliveTimeout      time.Duration `mapstructure:"keepalive_timeout"`
	NetworkTimeout        time.Duration `mapstructure:"network_timeout"`

	*ider.Ider              `mapstructure:",squash"`
	*pkgtls.TLSServerConfig `mapstructure:",squash"`

	server   *lumber.Server
	listener net.Listener

	log    *slog.Logger
	out    chan<- *core.Event
//	parser core.Parser
}

func (i *Beats) Init(config map[string]any, alias string, pipeline string, log *slog.Logger) error {
	if err := mapstructure.Decode(config, i); err != nil {
		return err
	}

	i.alias = alias
	i.pipe = pipeline
	i.log = log

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

	server, err := lumber.NewWithListener(listener,
		lumber.Keepalive(i.KeepaliveTimeout),
		lumber.Timeout(i.NetworkTimeout),
		lumber.TLS(tlsConfig),
	)
	if err != nil {
		return err
	}

	lumberlog.Logger = &lumberLogger{log: log}
	i.server = server

	return nil
}

func (i *Beats) SetChannels(out chan<- *core.Event) {
	i.out = out
}

//func (i *Beats) SetParser(p core.Parser)

func (i *Beats) Close() error {
	defer i.listener.Close()
	return i.server.Close()
}

func (i *Beats) Run() {
	for ljEvent := range i.server.ReceiveChan() {
		for _, v := range ljEvent.Events {
			m, _ := json.Marshal(core.Map(v.(map[string]any)))
			fmt.Println(string(m))
		}
		ljEvent.ACK()
	}
}

func init() {
	plugins.AddInput("beats", func() core.Input {
		return &Beats{
			Address:         ":8800",
			KeepaliveTimeout: 3 * time.Second,
			NetworkTimeout:   30 * time.Second,
			Ider:            &ider.Ider{},
			TLSServerConfig: &pkgtls.TLSServerConfig{},
		}
	})
}
