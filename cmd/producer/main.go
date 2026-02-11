package main

import (
	"encoding/json"
	"flag"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rakib/go-websocket-service/config"
)

type MqMessage struct {
	UserID string          `json:"user_id"`
	Data   json.RawMessage `json:"data"`
}

func main() {
	cfg := config.Load()

	userID := flag.String("user", "", "UserID (numeric ID usually)")
	entityType := flag.String("type", "customer", "Entity Type (customer/user)")
	message := flag.String("msg", "hello world", "Message content")
	flag.Parse()

	targetID := ""
	if *userID != "" {
		targetID = *entityType + ":" + *userID
		log.Printf("Sending to target: %s", targetID)
	} else {
		log.Println("Note: No user ID specified, broadcasting message")
	}

	conn, err := amqp.Dial(cfg.RabbitMQURL)
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
		cfg.RabbitMQQueue, // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	dataBytes, _ := json.Marshal(*message)
	payload := MqMessage{
		UserID: targetID,
		Data:   dataBytes,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Fatalf("Failed to publish a message: %v", err)
	}
	log.Printf(" [x] Sent %s", body)
}
