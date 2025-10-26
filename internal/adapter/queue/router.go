package queue

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Router manages multiple consumers (one per registered queue) on a single AMQP channel.
type Router struct {
	ch            *amqp.Channel
	prefetch      int
	callTimeout   time.Duration
	requeueOnErr  bool
	registrations []registration
}

type registration struct {
	queueName   string
	handler     Handler
	consumerTag string
}

// --- Options ---

type RouterOption func(*Router)

func WithPrefetch(n int) RouterOption          { return func(r *Router) { r.prefetch = n } }
func WithTimeout(d time.Duration) RouterOption { return func(r *Router) { r.callTimeout = d } }
func WithRequeue(b bool) RouterOption          { return func(r *Router) { r.requeueOnErr = b } }

// NewRouter constructs a Router. Defaults: prefetch=50, timeout=10s, requeueOnErr=true.
func NewRouter(ch *amqp.Channel, opts ...RouterOption) *Router {
	r := &Router{
		ch:           ch,
		prefetch:     50,
		callTimeout:  10 * time.Second,
		requeueOnErr: true,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Register associates a queue with a handler. Call multiple times for multiple queues.
func (r *Router) Register(queueName string, h Handler) {
	r.registrations = append(r.registrations, registration{
		queueName:   queueName,
		handler:     h,
		consumerTag: "c_" + queueName,
	})
}

// Start begins consuming; non-blocking (spawns one goroutine per queue).
// QoS (prefetch) is set per-channel and applies to all consumers on this channel.
func (r *Router) Start() error {
	if err := r.ch.Qos(r.prefetch, 0, false); err != nil {
		return err
	}

	for _, reg := range r.registrations {
		deliveries, err := r.ch.Consume(
			reg.queueName,
			reg.consumerTag,
			false, // manual ack
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,
		)
		if err != nil {
			return err
		}

		go func(queueName, tag string, h Handler, msgs <-chan amqp.Delivery) {
			for d := range msgs {
				ctx, cancel := context.WithTimeout(context.Background(), r.callTimeout)
				err := h.Handle(ctx, d)
				cancel()

				if err != nil {
					log.Printf("[rmq-router] handler error queue=%s tag=%s rk=%s err=%v requeue=%v",
						queueName, tag, d.RoutingKey, err, r.requeueOnErr)
					_ = d.Nack(false, r.requeueOnErr)
					continue
				}
				_ = d.Ack(false)
			}
			log.Printf("[rmq-router] consumer stopped queue=%s tag=%s", queueName, tag)
		}(reg.queueName, reg.consumerTag, reg.handler, deliveries)
	}

	return nil
}
