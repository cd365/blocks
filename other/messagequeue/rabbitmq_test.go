package messagequeue

import (
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"sync"
	"testing"
	"time"
)

func TestNewPush(t *testing.T) {
	push, err := NewPush("", func(c *amqp.Channel) error {
		return c.ExchangeDeclare("exchange1", "direct", true, false, false, false, nil)
		// return c.ExchangeDeclare("exchange2", "topic", true, false, false, false, nil)
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = push.PushOnce(nil, "exchange1", "key1")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestNewPull(t *testing.T) {
	pull, err := NewPull("", func(c *amqp.Channel) (<-chan amqp.Delivery, error) {
		if err := c.ExchangeDeclare(
			"exchange1", // name of the exchange
			"topic",     // type
			true,        // durable
			false,       // delete when complete
			false,       // internal
			false,       // noWait
			nil,         // arguments
		); err != nil {
			return nil, fmt.Errorf("exchange declare: %s", err.Error())
		}

		queue, err := c.QueueDeclare(
			"queueName", // name of the queue
			true,        // durable
			false,       // delete when unused
			false,       // exclusive
			false,       // noWait
			nil,         // arguments
		)
		if err != nil {
			return nil, fmt.Errorf("queue declare: %s", err.Error())
		}

		// binding route key1
		if err = c.QueueBind(
			queue.Name,                 // name of the queue
			zerolog.WarnLevel.String(), // bindingKey
			"exchange1",                // sourceExchange
			false,                      // noWait
			nil,                        // arguments
		); err != nil {
			return nil, fmt.Errorf("queue bind: %s", err.Error())
		}

		// binding route key2
		if err = c.QueueBind(
			queue.Name,                  // name of the queue
			zerolog.ErrorLevel.String(), // bindingKey
			"exchange1",                 // sourceExchange
			false,                       // noWait
			nil,                         // arguments
		); err != nil {
			return nil, fmt.Errorf("queue bind: %s", err.Error())
		}

		err = c.Qos(8, 0, false)
		if err != nil {
			return nil, err
		}

		return c.Consume(
			queue.Name,     // name
			"consumerTag1", // consumerTag,
			false,          // autoAck
			false,          // exclusive
			false,          // noLocal
			false,          // noWait
			nil,            // arguments
		)
	})
	if err != nil {
		t.Error(err)
		return
	}
	wg := sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*9)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pull.PullBatch(ctx, time.Second*3, 1000, func(messages [][]byte) error {
			fmt.Printf("%v\n", messages)
			return nil
		})
	}()

}
