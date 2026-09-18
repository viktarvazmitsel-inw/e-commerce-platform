package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	// 1. Establish RabbitMQ connection at startup
	// "rabbitmq" matches the hostname set in Docker networking
	conn, err := amqp.Dial("amqp://guest:guest@broker:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// 2. Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// 3. Declare the queue
	// Must match the exact settings/name used in the sender
	q, err := ch.QueueDeclare(
		"test", // queue name
		true,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		amqp.Table{
            amqp.QueueTypeArg: amqp.QueueTypeQuorum,
        },
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// 4. Register a consumer
	msgs, err := ch.Consume(
		q.Name, // queue name
		"",     // consumer tag (empty string generates a unique tag)
		true,   // auto-ack (automatically acknowledges message receipt)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// 5. Block main goroutine and continuously print received messages
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages in queue '%s'. To exit press CTRL+C", q.Name)
	<-forever
}
