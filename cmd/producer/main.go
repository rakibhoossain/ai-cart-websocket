package main

import (
	"encoding/json"
	"flag"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rakib/go-websocket-service/config"
)

type MqMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

func main() {
	cfg := config.Load()

	channelName := flag.String("channel", "", "Channel (e.g. user:123 or shop:456)")
	userID := flag.String("user", "", "UserID (numeric ID usually)")
	entityType := flag.String("type", "customer", "Entity Type (customer/user)")
	message := flag.String("msg", "hello world", "Message content")
	flag.Parse()

	targetChannel := *channelName
	if targetChannel == "" && *userID != "" {
		targetChannel = *entityType + ":" + *userID
	}

	if targetChannel != "" {
		log.Printf("Sending to channel: %s", targetChannel)
	} else {
		log.Fatal("Error: channel or user ID must be specified")
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
		Channel: targetChannel,
		Data:    dataBytes,
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
