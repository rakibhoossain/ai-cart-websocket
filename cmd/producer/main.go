package main

import (
	"encoding/json"
	"flag"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rakib/go-websocket-service/config"
	"github.com/rakib/go-websocket-service/mq"
)

func main() {
	cfg := config.Load()

	channelName := flag.String("channel", "", "Target channel (e.g. user:123 or shop:456)")
	message := flag.String("msg", "hello world", "Message content (plain text or JSON)")
	flag.Parse()

	if *channelName == "" {
		log.Fatal("Error: -channel must be specified (e.g. -channel=user:123)")
	}

	if cfg.RabbitMQURL == "" {
		log.Fatal("Error: RABBITMQ_URL is not configured in environment or .env")
	}

	log.Printf("Sending to channel: %s", *channelName)

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

	// If message is valid JSON, pass as raw JSON; otherwise, marshal as JSON string
	var dataBytes json.RawMessage
	if json.Valid([]byte(*message)) {
		dataBytes = json.RawMessage(*message)
	} else {
		marshaled, _ := json.Marshal(*message)
		dataBytes = marshaled
	}

	payload := mq.MqMessage{
		Channel: *channelName,
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
