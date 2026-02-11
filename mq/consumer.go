package mq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rakib/go-websocket-service/ws"
)

type MqMessage struct {
	UserID string          `json:"user_id"`
	Data   json.RawMessage `json:"data"`
}

func StartConsumer(url, queueName string, hub *ws.Hub) {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var msg MqMessage
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}

			log.Printf("Received message: %v", msg)

			if msg.UserID != "" {
				hub.SendToUser(msg.UserID, msg.Data)
			} else {
				log.Printf("Warning: Received message without UserID, ignoring: %s", msg.Data)
			}
		}
	}()

	<-forever
}
