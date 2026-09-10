package main

import (
	"fmt"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

type App struct {
	ch *amqp.Channel
}

func main() {
	// 1. Establish RabbitMQ connection ONCE at startup
	conn, err := amqp.Dial("amqp://guest:guest@broker:5672/")
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

	app := &App{ch: ch}

	http.HandleFunc("/", app.handleMain(q.Name))

	log.Println("Sender running on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (app *App) handleMain(queueName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	    if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
	    }

		body := "Hello from Go Sender!"

		err := app.ch.Publish(
			"",        // exchange
			queueName, // routing key (queue name)
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			},
		)

		if err != nil {
			log.Printf("Failed to publish message: %v", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "Message successfully sent to RabbitMQ!")
	}
}
