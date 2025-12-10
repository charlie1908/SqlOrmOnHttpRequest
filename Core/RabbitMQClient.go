package Core

import (
	"fmt"
	"github.com/streadway/amqp"
	shared "httpRequestName/Shared"
	"log"
)

type RabbitMQClient struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

func NewRabbitMQClient() (*RabbitMQClient, error) {
	rabbitmqHost, err := Decrypt(shared.Config.RABBITMQHOST, shared.Config.SECRETKEY)
	rabbitmqPassword, err := Decrypt(shared.Config.RABBITMQPASSWORD, shared.Config.SECRETKEY)
	rabbitmqPort := shared.Config.RABBITMQPORT
	rabbitmqUser, err := Decrypt(shared.Config.RABBITMQUSER, shared.Config.SECRETKEY)
	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		rabbitmqUser,
		rabbitmqPassword,
		rabbitmqHost,
		rabbitmqPort,
	)
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return &RabbitMQClient{
		Connection: conn,
		Channel:    ch,
	}, nil
}

func (r *RabbitMQClient) Publish(queueName string, body string) error {
	_, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = r.Channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published to %s: %s", queueName, body)
	return nil
}

func (r *RabbitMQClient) Consume(queueName string, handler func(string)) error {
	_, err := r.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			handler(string(d.Body))
		}
	}()

	log.Printf("Consuming from %s...", queueName)
	return nil
}

func (r *RabbitMQClient) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Connection != nil {
		r.Connection.Close()
	}
}
