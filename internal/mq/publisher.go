package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/google/uuid"
)

const (
	ExchangeName = "transfans.events"
	exchangeKind = "topic"

	RoutingKeySubscriptionCreate     = "subscription.create.request"
	RoutingKeySubscriptionDeactivate = "subscription.deactivate.request"

	reconnectDelay = 5 * time.Second
)

type Envelope struct {
	Event            string `json:"event"`
	Timestamp        string `json:"timestamp"`
	MessageID        string `json:"message_id"`
	RequestID        string `json:"request_id"`
	InitiatorService string `json:"initiator_service"`
	Data             any    `json:"data"`
}

type SubscriptionCreateData struct {
	FanID     string `json:"fan_id"`
	CreatorID string `json:"creator_id"`
	TierID    string `json:"tier_id"`
	ExpiresAt string `json:"expires_at"`
}

type SubscriptionDeactivateData struct {
	SubscriptionID string  `json:"subscription_id"`
	Reason         *string `json:"reason"`
}

type Publisher struct {
	url     string
	logger  *slog.Logger
	mu      sync.RWMutex
	channel *amqp.Channel
	conn    *amqp.Connection
	done    chan struct{}
}

func NewPublisher(url string, logger *slog.Logger) *Publisher {
	p := &Publisher{
		url:    url,
		logger: logger,
		done:   make(chan struct{}),
	}

	if err := p.connect(); err != nil {
		logger.Warn("failed to connect to rabbitmq — subscription events will not be published", "error", err)
	} else {
		logger.Info("connected to rabbitmq")
	}

	go p.reconnectLoop()
	return p
}

func (p *Publisher) connect() error {
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("channel: %w", err)
	}

	if err := ch.ExchangeDeclare(ExchangeName, exchangeKind, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("declare exchange: %w", err)
	}

	p.mu.Lock()
	p.conn = conn
	p.channel = ch
	p.mu.Unlock()
	return nil
}

func (p *Publisher) reconnectLoop() {
	for {
		p.mu.RLock()
		conn := p.conn
		p.mu.RUnlock()

		var closed chan *amqp.Error
		if conn != nil {
			closed = conn.NotifyClose(make(chan *amqp.Error, 1))
		} else {
			closed = make(chan *amqp.Error, 1)
			go func() { time.Sleep(reconnectDelay); closed <- nil }()
		}

		select {
		case <-p.done:
			return
		case err := <-closed:
			if err != nil {
				p.logger.Warn("rabbitmq connection lost, reconnecting", "error", err)
			}
			for {
				select {
				case <-p.done:
					return
				case <-time.After(reconnectDelay):
				}
				if connErr := p.connect(); connErr != nil {
					p.logger.Warn("rabbitmq reconnect failed, retrying", "error", connErr)
				} else {
					p.logger.Info("rabbitmq reconnected")
					break
				}
			}
		}
	}
}

func (p *Publisher) Close() {
	close(p.done)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Publisher) Publish(ctx context.Context, routingKey, requestID string, data any) error {
	p.mu.RLock()
	ch := p.channel
	p.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("rabbitmq not connected")
	}

	env := Envelope{
		Event:            routingKey,
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		MessageID:        uuid.New().String(),
		RequestID:        requestID,
		InitiatorService: "payment-service",
		Data:             data,
	}

	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	return ch.PublishWithContext(ctx, ExchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    env.MessageID,
		Body:         body,
	})
}
