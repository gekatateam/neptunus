package kafka

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/segmentio/kafka-go"
)

type labelBalancer struct {
	label string
	log   *slog.Logger
}

func (o *labelBalancer) Balance(msg kafka.Message, partitions ...int) (partition int) {
	for _, h := range msg.Headers {
		if h.Key == o.label {
			partition, err := strconv.Atoi(string(h.Value))
			if err != nil {
				o.log.Warn("partition calculation error",
					"error", err,
				)
				return partitions[0]
			}

			if !slices.Contains(partitions, partition) {
				o.log.Warn("partition calculation error",
					"error", fmt.Sprintf("partition %v not in partitions list", partition),
				)
				return partitions[0]
			}

			return partition
		}
	}
	o.log.Warn("partition calculation error",
		"error", fmt.Sprintf("message has no %v header", o.label),
	)
	return partitions[0]
}
