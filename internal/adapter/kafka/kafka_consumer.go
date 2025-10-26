package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/IBM/sarama"
	"github.com/aq2208/gorder-api/internal/usecase"
)

// HandlerFunc processes a decoded event.
type HandlerFunc func(ctx context.Context, ev usecase.OrderStatusChangedMsg) error

// Consumer consumes a topic with a single handler.
type Consumer struct {
	Group  sarama.ConsumerGroup
	Topics []string
	Handle HandlerFunc
	Logger *log.Logger // optional
}

func NewConsumer(group sarama.ConsumerGroup, topics []string, h HandlerFunc) *Consumer {
	return &Consumer{
		Group:  group,
		Topics: topics,
		Handle: h,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	handler := &cgHandler{handle: c.Handle, logger: c.Logger}
	for {
		if err := c.Group.Consume(ctx, c.Topics, handler); err != nil {
			return err
		}
		// When Consume returns, it’s because ctx was cancelled or a rebalance happened.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

type cgHandler struct {
	handle HandlerFunc
	logger *log.Logger
}

func (h *cgHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *cgHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *cgHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var ev usecase.OrderStatusChangedMsg
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			if h.logger != nil {
				h.logger.Printf("kafka decode error: %v", err)
			}
			// mark to avoid reprocessing poison
			sess.MarkMessage(msg, "decode-error")
			continue
		}
		if err := h.handle(sess.Context(), ev); err != nil {
			if h.logger != nil {
				h.logger.Printf("handler error: %v (key=%s, off=%d)", err, string(msg.Key), msg.Offset)
			}
			// Do not mark message; let it retry on next poll or route with your DLQ pattern.
			continue
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}
