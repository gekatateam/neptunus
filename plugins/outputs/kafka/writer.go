package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/protocol"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	kafkastats "github.com/gekatateam/neptunus/plugins/common/metrics"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

type topicWriter struct {
	*core.BaseOutput
	clientId string

	enableMetrics     bool
	keepTimestamp     bool
	partitionBalancer string
	partitionLabel    string
	keyLabel          string
	headerLabels      map[string]string

	input  chan *core.Event
	writer *kafka.Writer
	ser    core.Serializer

	*batcher.Batcher[*core.Event]
	*retryer.Retryer
}

func (w *topicWriter) Close() error {
	close(w.input)
	return nil
}

func (w *topicWriter) Push(e *core.Event) {
	w.input <- e
}

func (w *topicWriter) Run() {
	if w.enableMetrics {
		kafkastats.RegisterKafkaWriter(w.Pipeline, w.Alias, w.writer.Topic, w.clientId, w.writer.Stats)
		defer kafkastats.UnregisterKafkaWriter(w.Pipeline, w.Alias, w.writer.Topic, w.clientId)
	}
	w.Log.Info(fmt.Sprintf("producer for topic %v spawned", w.writer.Topic))

	w.Batcher.Run(w.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		messages := []kafka.Message{}
		readyEvents := make(map[uuid.UUID]*eventMsgStatus)

		for _, e := range buf {
			now := time.Now()
			event, err := w.ser.Serialize(e)
			if err != nil {
				w.Log.Error("serialization failed, event skipped",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				w.Done <- e
				w.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			msg := kafka.Message{
				Value:      event,
				WriterData: e.UUID,
			}

			if w.keepTimestamp {
				msg.Time = e.Timestamp
			}

			for header, label := range w.headerLabels {
				if v, ok := e.GetLabel(label); ok {
					msg.Headers = append(msg.Headers, protocol.Header{
						Key:   header,
						Value: []byte(v),
					})
				}
			}

			if w.partitionBalancer == "label" {
				if label, ok := e.GetLabel(w.partitionLabel); ok {
					msg.Headers = append(msg.Headers, protocol.Header{
						Key:   w.partitionLabel,
						Value: []byte(label),
					})
				}
			}

			if len(w.keyLabel) > 0 {
				if label, ok := e.GetLabel(w.keyLabel); ok {
					msg.Key = []byte(label)
				}
			}

			messages = append(messages, msg)
			readyEvents[e.UUID] = &eventMsgStatus{
				Event:     e,
				spentTime: time.Since(now),
				error:     errors.New("not delivered yet"),
			}
		}

		if len(readyEvents) == 0 {
			return
		}

		eventsStat := w.write(messages, readyEvents)

		// mark as done only after successful write
		// or when the maximum number of attempts has been reached
		for _, e := range eventsStat {
			w.Done <- e.Event
			if e.error != nil {
				w.Log.Error("event produce failed",
					"error", e.error,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				w.Observe(metrics.EventFailed, e.spentTime)
			} else {
				w.Log.Debug("event produced",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				w.Observe(metrics.EventAccepted, e.spentTime)
			}
		}
	})

	if err := w.writer.Close(); err != nil {
		w.Log.Warn(fmt.Sprintf("producer for topic %v closed with error", w.writer.Topic),
			"error", err,
		)
	} else {
		w.Log.Info(fmt.Sprintf("producer for topic %v closed", w.writer.Topic))
	}
}

func (w *topicWriter) write(messages []kafka.Message, eventsStatus map[uuid.UUID]*eventMsgStatus) map[uuid.UUID]*eventMsgStatus {
	w.Retryer.Do("write to topic "+w.writer.Topic, w.Log, func() error {
		if len(messages) == 0 {
			return nil // if no messages left, break cycle
		}

		now := time.Now()
		err := w.writer.WriteMessages(context.Background(), messages...)
		timePerEvent := durationPerEvent(time.Since(now), len(messages))

		switch writeErr := err.(type) {
		case nil: // all messages delivered successfully
			for _, m := range messages {
				successMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				successMsg.spentTime += timePerEvent
				successMsg.error = nil
			}
			return nil // all messages delivered, break cycle
		case kafka.WriteErrors:
			// retriable messages goes to next round
			var retriable []kafka.Message = nil
			for i, m := range messages {
				msgErr := writeErr[i]
				if msgErr == nil { // this message delivred successfully
					successMsg := eventsStatus[m.WriterData.(uuid.UUID)]
					successMsg.spentTime += timePerEvent
					successMsg.error = nil
					continue
				}

				if kafkaErr, ok := msgErr.(kafka.Error); ok && kafkaErr.Temporary() { // timeout and temporary errors are retriable
					retriable = append(retriable, m)
					retriableMsg := eventsStatus[m.WriterData.(uuid.UUID)]
					retriableMsg.spentTime += timePerEvent
					retriableMsg.error = kafkaErr
					continue
				}

				var timeoutError interface{ Timeout() bool }
				if errors.As(msgErr, &timeoutError) && timeoutError.Timeout() { // typically it is a network io timeout
					retriable = append(retriable, m)
					retriableMsg := eventsStatus[m.WriterData.(uuid.UUID)]
					retriableMsg.spentTime += timePerEvent
					retriableMsg.error = msgErr
					continue
				}

				if errors.Is(msgErr, io.ErrUnexpectedEOF) { // this error means that broker is down
					retriable = append(retriable, m)
					retriableMsg := eventsStatus[m.WriterData.(uuid.UUID)]
					retriableMsg.spentTime += timePerEvent
					retriableMsg.error = msgErr
					continue
				}

				// any other errors means than message cannot be produced
				failedMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				failedMsg.spentTime += timePerEvent
				failedMsg.error = msgErr
			}
			messages = retriable
		case kafka.MessageTooLargeError: // exclude too large message and send others
			tooLargeMsg := eventsStatus[writeErr.Message.WriterData.(uuid.UUID)]
			tooLargeMsg.spentTime += timePerEvent
			tooLargeMsg.error = writeErr
			messages = writeErr.Remaining
		case kafka.Error:
			for _, m := range messages {
				kafkaMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				kafkaMsg.spentTime += timePerEvent
				kafkaMsg.error = writeErr
			}

			if !writeErr.Temporary() { // not temporary errors are not retriable
				return nil // nothing to send again, break cycle
			}
		default: // any other errors are unretriable
			for _, m := range messages {
				failedMsg := eventsStatus[m.WriterData.(uuid.UUID)]
				failedMsg.spentTime += timePerEvent
				failedMsg.error = writeErr
			}
			return nil // nothing to send again, break cycle
		}

		return err // retriable messages goes to next round
	})

	return eventsStatus
}

type eventMsgStatus struct {
	*core.Event
	spentTime time.Duration
	error     error
}

func durationPerEvent(totalTime time.Duration, batchSize int) time.Duration {
	return time.Duration(int64(totalTime) / int64(batchSize))
}
