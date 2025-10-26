package queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aq2208/gorder-api/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConsumer struct {
	Ch           *amqp.Channel
	QueueName    string
	Handler      Handler
	Prefetch     int
	CallTimeout  time.Duration
	RequeueOnErr bool
}

func NewConsumer(ch *amqp.Channel, queueName string, h Handler) *RabbitConsumer {
	return &RabbitConsumer{
		Ch:           ch,
		QueueName:    queueName,
		Handler:      h,
		Prefetch:     50,
		CallTimeout:  10 * time.Second,
		RequeueOnErr: true,
	}
}

// Start begins consuming; non-blocking (spawns a goroutine).
func (c *RabbitConsumer) Start() error {
	// fair dispatch
	if err := c.Ch.Qos(c.Prefetch, 0, false); err != nil {
		return err
	}

	msgs, err := c.Ch.Consume(
		c.QueueName,
		"",    // consumer tag
		false, // manual ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var ev usecase.CreatedMsg
			if err := json.Unmarshal(d.Body, &ev); err != nil {
				log.Printf("[rmq-consumer] bad message: %v body=%q", err, string(d.Body))
				_ = d.Nack(false, false) // drop poison
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), c.CallTimeout)
			err := c.Handler.Handle(ctx, d)
			cancel()

			if err != nil {
				log.Printf("[rmq-consumer] handler error: %v, requeue=%v", err, c.RequeueOnErr)
				_ = d.Nack(false, c.RequeueOnErr)
				continue
			}
			_ = d.Ack(false)
		}
	}()
	return nil
}
