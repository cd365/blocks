// Message queues implemented by RabbitMQ.

package mq

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cd365/blocks/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	amqpUrl            = "amqp://guest:guest@127.0.0.1:5672/vhost_guest"
	amqpExchange       = "exchange_test1"
	amqpRoutingKey1    = "routing_key_test1"
	amqpRoutingKey2    = "routing_key_test2"
	amqpExchangeDirect = "direct"
	amqpExchangeTopic  = "topic"
)

func TestNewPush(t *testing.T) {
	push, err := NewPush(amqpUrl, func(c *amqp.Channel) error {
		return c.ExchangeDeclare(amqpExchange, amqpExchangeTopic, true, false, false, false, nil)
		// return c.ExchangeDeclare(amqpExchange, amqpExchangeDirect, true, false, false, false, nil)
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer func() { _ = push.Close() }()

	push.Logger(log.DefaultLogger)

	for i := 0; i < 10000; i++ {
		err = push.PublishContext(context.Background(), amqpExchange, amqpRoutingKey1, &amqp.Publishing{
			Body: []byte("112233"),
		})
		if err != nil {
			panic(err)
		}
	}

	<-time.After(time.Second * 3)

}

func TestNewPull(t *testing.T) {
	pull, err := NewPull(amqpUrl, func(c *amqp.Channel) (<-chan amqp.Delivery, error) {
		if err := c.ExchangeDeclare(
			amqpExchange,      // name of the exchange
			amqpExchangeTopic, // type
			true,              // durable
			false,             // delete when complete
			false,             // internal
			false,             // noWait
			nil,               // arguments
		); err != nil {
			return nil, fmt.Errorf("exchange declare: %s", err.Error())
		}

		queue, err := c.QueueDeclare(
			"test_consume_queue1", // name of the queue
			true,                  // durable
			false,                 // delete when unused
			false,                 // exclusive
			false,                 // noWait
			nil,                   // arguments
		)
		if err != nil {
			return nil, fmt.Errorf("queue declare: %s", err.Error())
		}

		// binding route key1
		if err = c.QueueBind(
			queue.Name,      // name of the queue
			amqpRoutingKey1, // bindingKey
			amqpExchange,    // sourceExchange
			false,           // noWait
			nil,             // arguments
		); err != nil {
			return nil, fmt.Errorf("queue bind: %s", err.Error())
		}

		// binding route key2
		if err = c.QueueBind(
			queue.Name,      // name of the queue
			amqpRoutingKey2, // bindingKey
			amqpExchange,    // sourceExchange
			false,           // noWait
			nil,             // arguments
		); err != nil {
			return nil, fmt.Errorf("queue bind: %s", err.Error())
		}

		// // binding all route keys
		// if err = c.QueueBind(
		// 	queue.Name,   // name of the queue
		// 	"*",          // bindingKey
		// 	amqpExchange, // sourceExchange
		// 	false,        // noWait
		// 	nil,          // arguments
		// ); err != nil {
		// 	return nil, fmt.Errorf("queue bind: %s", err.Error())
		// }

		err = c.Qos(8, 0, false)
		if err != nil {
			return nil, err
		}

		return c.Consume(
			queue.Name,           // name
			"consumer_tag_test1", // consumerTag,
			false,                // autoAck
			false,                // exclusive
			false,                // noLocal
			false,                // noWait
			nil,                  // arguments
		)
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer func() { _ = pull.Close() }()

	pull.Logger(log.DefaultLogger)

	wg := sync.WaitGroup{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pull.BatchProcess(ctx, time.Second*5, 1000, func(messages []*amqp.Delivery) error {
			length := len(messages)
			fmt.Printf("%d\n", length)
			// msg := make([]string, length)
			// for i := 0; i < length; i++ {
			// 	msg[i] = string(messages[i].Body)
			// }
			// fmt.Printf("%d %#v\n", length, msg)
			return nil
		})
	}()
	<-ctx.Done()
	wg.Wait()
}
