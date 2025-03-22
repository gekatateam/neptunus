package telegram

import (
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"log/slog"
	"time"
)

type Telegram struct {
	*core.BaseOutput `mapstructure:"-"`
	Token            string  `mapstructure:"api_token"`
	ChatIDs          []int64 `mapstructure:"chat_ids"`
	FromField        string  `mapstructure:"from_field"`
	*retryer.Retryer `mapstructure:",squash"`
	//requestersPool   *pool.Pool[*core.Event]
	bot *tgbotapi.BotAPI
	ser core.Serializer
}

func (o *Telegram) Init() error {
	if len(o.Token) == 0 {
		return errors.New("token required")
	}

	if len(o.ChatIDs) == 0 {
		return errors.New("chat_ids required")
	}

	api, err := tgbotapi.NewBotAPI(o.Token)
	if err != nil {
		return err
	}
	o.bot = api

	return nil
}

func (o *Telegram) SetSerializer(s core.Serializer) {
	o.ser = s
}

func (o *Telegram) Close() error {
	return nil
}

func (o *Telegram) Run() {
	for e := range o.In {
		now := time.Now()
		event, err := o.ser.Serialize(e)

		if err != nil {
			o.Log.Error("serialization failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		if len(event) == 0 {
			o.Log.Info("event is empty",
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			o.Done <- e
			o.Observe(metrics.EventRejected, time.Since(now))
			continue
		}

		for _, chatID := range o.ChatIDs {
			msg := tgbotapi.NewMessage(chatID, string(event))
			err := o.Retryer.Do("send telegram message", o.Log, func() error {
				_, err := o.bot.Send(msg)
				return err
			})

			if err != nil {
				o.Log.Error("send message failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				continue
			}
		}

		o.Done <- e
		o.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddOutput("telegram", func() core.Output {
		return &Telegram{
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
