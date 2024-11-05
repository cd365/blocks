// Message queues implemented by RabbitMQ.

package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/cd365/blocks/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Push struct {
	url           string
	buildExchange func(c *amqp.Channel) error

	conn    *amqp.Connection
	channel *amqp.Channel

	logger *log.Logger
}

func (s *Push) Close() error {
	if s.channel != nil {
		_ = s.channel.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	return nil
}

func (s *Push) initial() error {
	_ = s.Close()
	conn, err := amqp.Dial(s.url)
	if err != nil {
		return err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}

	if err = s.buildExchange(channel); err != nil {
		return fmt.Errorf("make exchange: %s", err.Error())
	}
	s.conn = conn
	s.channel = channel
	return nil
}

func NewPush(url string, buildExchange func(c *amqp.Channel) error) (*Push, error) {
	push := &Push{
		url:           url,
		buildExchange: buildExchange,
	}
	if err := push.initial(); err != nil {
		return nil, err
	}
	return push, nil
}

func (s *Push) Logger(logger *log.Logger) *Push {
	s.logger = logger
	return s
}

func (s *Push) PublishContext(ctx context.Context, exchangeName, routingKey string, message *amqp.Publishing) error {
	if message == nil {
		return nil
	}
	if s.channel.IsClosed() {
		return fmt.Errorf("channel is closed")
	}
	err := s.channel.PublishWithContext(ctx, exchangeName, routingKey, false, false, *message)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn().Any("content", message).Msg(err.Error())
		}
	}
	return err
}

func (s *Push) PublishContextRetry(ctx context.Context, exchangeName, routingKey string, message *amqp.Publishing, retry int, duration time.Duration) (err error) {
	if retry <= 0 {
		retry = 1
	}
	for i := 0; i < retry; i++ {
		if err = s.PublishContext(ctx, exchangeName, routingKey, message); err == nil {
			break
		}
		if err = s.initial(); err != nil {
			<-time.After(duration)
		}
	}
	return
}

type Pull struct {
	url           string
	buildExchange func(c *amqp.Channel) (<-chan amqp.Delivery, error)

	conn    *amqp.Connection
	channel *amqp.Channel

	logger *log.Logger

	deliveries <-chan amqp.Delivery
}

func (s *Pull) Close() error {
	if s.channel != nil {
		_ = s.channel.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	return nil
}

func (s *Pull) initial() error {
	_ = s.Close()
	conn, err := amqp.Dial(s.url)
	if err != nil {
		return err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if deliveries, rer := s.buildExchange(channel); rer != nil {
		return rer
	} else {
		s.deliveries = deliveries
	}
	s.conn = conn
	s.channel = channel
	return nil
}

func NewPull(url string, buildExchange func(c *amqp.Channel) (<-chan amqp.Delivery, error)) (*Pull, error) {
	pull := &Pull{
		url:           url,
		buildExchange: buildExchange,
	}
	if err := pull.initial(); err != nil {
		return nil, err
	}
	return pull, nil
}

func (s *Pull) Logger(logger *log.Logger) *Pull {
	s.logger = logger
	return s
}

func (s *Pull) BatchProcess(ctx context.Context, timerDuration time.Duration, batch int, handler func(messages []*amqp.Delivery) error) {
	if batch <= 0 || batch > 10000 {
		batch = 1000
	}

	// a list of messages that have been acknowledged.
	lists := make([]*amqp.Delivery, 0, batch)

	// the number of messages that have been acknowledged.
	num := 0

	// the last time when the message batch was processed.
	lastWriteAt := time.Now()

	// batch messages (single-threaded calls).
	write := func() {
		length := len(lists)
		if length == 0 {
			return
		}

		if err := handler(lists); err != nil {
			if s.logger != nil {
				for i := 0; i < length; i++ {
					lists[i].Acknowledger = nil
				}
				s.logger.Warn().Any("content", lists).Msg(err.Error())
			}
		}

		// reinitialization.
		lists = make([]*amqp.Delivery, 0, batch)
		num = 0
		lastWriteAt = time.Now()
	}

	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("panic message queue consumer: %v\n", rec)
		}
		write()
	}()

	timer := time.NewTimer(timerDuration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			write()
			return
		case <-timer.C:
			if time.Now().Unix()-lastWriteAt.Unix() >= 3 {
				write()
			}
			timer.Reset(timerDuration)
		case message := <-s.deliveries:
			if err := message.Ack(false); err != nil {
				write()
				for err = s.initial(); err != nil; {
					<-time.After(time.Millisecond * 500)
				}
				break
			}
			lists = append(lists, &message)
			num++

			if num >= batch {
				write()
			}
		}
	}
}
