package messagequeue

import (
	"context"
	"fmt"
	"github.com/cd365/blocks/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type Push struct {
	url           string
	buildExchange func(c *amqp.Channel) error

	conn    *amqp.Connection
	channel *amqp.Channel

	logger *log.Logger
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

func (s *Push) Close() error {
	if s.channel != nil {
		_ = s.channel.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
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

func (s *Push) PushOnce(message []byte, exchangeName, routingKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if s.channel.IsClosed() {
		return fmt.Errorf("channel is closed")
	}
	err := s.channel.PublishWithContext(
		ctx,
		exchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn().Bytes("content", message).Str("error", err.Error()).Msg("message push failed")
		}
		return fmt.Errorf("failed to publish a message: %v", err.Error())
	}
	return nil
}

func (s *Push) PushRetry(message []byte, exchangeName, routingKey string, retry int) (err error) {
	if retry <= 0 {
		retry = 1
	}
	for i := 0; i < retry; i++ {
		if err = s.PushOnce(message, exchangeName, routingKey); err == nil {
			break
		}
		if s.logger != nil {
			s.logger.Warn().Msg("will retry push message")
		}
		if err = s.initial(); err != nil {
			<-time.After(time.Millisecond * 200)
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

func (s *Pull) PullBatch(ctx context.Context, duration time.Duration, batch int, handler func(messages [][]byte) error) {
	var message amqp.Delivery
	if batch <= 0 || batch > 10000 {
		batch = 1000
	}
	// 已被确认的消息列表
	lists := make([][]byte, 0, batch)
	// 已被确认的消息的数量
	num := 0
	// 消息批处理的最后时间
	lastWriteAt := time.Now()

	// 批处理消息(单线程调用)
	write := func() {
		length := len(lists)
		if length == 0 {
			return
		}
		writes := make([][]byte, length)
		for i := 0; i < length; i++ {
			writes[i] = lists[num]
		}

		if err := handler(writes); err != nil {
			if s.logger != nil {
				content := make([]string, length)
				for i := 0; i < length; i++ {
					content[i] = string(writes[i])
				}
				s.logger.Warn().Strs("content", content).Err(err).Msg("batch message failed")
			}
		}

		// 重新初始化
		lists = make([][]byte, 0, batch)
		num = 0
		lastWriteAt = time.Now()
	}

	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("panic logger.consumer: %v\n", rec)
		}
		write()
	}()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			write()
			return
		case <-timer.C:
			if time.Now().Sub(lastWriteAt) >= 3 {
				write()
			}
			timer.Reset(duration)
		case message = <-s.deliveries:
			if err := message.Ack(false); err != nil {
				write()
				for err = s.initial(); err != nil; {
					<-time.After(time.Millisecond * 200)
				}
				break
			}
			lists = append(lists, message.Body)
			num++

			if num >= batch {
				write()
			}
		}
	}
}
