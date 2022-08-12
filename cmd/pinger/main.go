package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/assembla/cony"
	"github.com/streadway/amqp"
)

var (
	url    = os.Getenv("RABBIT_URL")
	prefix = os.Getenv("MESSAGE_PREFIX")
)

func main() {
	if url == "" {
		log.Fatal("RABBIT_URL must be set")
	}

	if prefix != "ping" && prefix != "pong" {
		log.Fatalf("prefix flag should be either 'ping' or 'pong' not '%v'", prefix)
	}

	cli := cony.NewClient(
		cony.URL(url),
		cony.Backoff(cony.DefaultBackoff),
	)

	// Declarations
	queue := &cony.Queue{
		AutoDelete: true,
		Name:       prefix,
	}

	exchange := cony.Exchange{
		Name:       "pingpong",
		Kind:       amqp.ExchangeDirect,
		AutoDelete: true,
	}

	binding := cony.Binding{
		Queue:    queue,
		Exchange: exchange,
		Key:      prefix,
	}

	cli.Declare([]cony.Declaration{
		cony.DeclareQueue(queue),
		cony.DeclareExchange(exchange),
		cony.DeclareBinding(binding),
	})

	cns := cony.NewConsumer(
		queue,
		cony.AutoAck(),
	)

	cli.Consume(cns)

	var other string
	if prefix == "ping" {
		other = "pong"
	} else {
		other = "ping"
	}

	pbl := cony.NewPublisher(
		exchange.Name,
		other,
	)

	cli.Publish(pbl)

	if prefix == "ping" {
		go func() {
			// Send the first message!
			log.Printf("%v: %v", prefix, 0)
			err := pbl.Publish(amqp.Publishing{
				Body:         []byte(fmt.Sprint(0)),
				DeliveryMode: amqp.Persistent,
			})
			if err != nil {
				log.Fatalf("Failed to send first message: %v", err)
			}
		}()
	}

	for cli.Loop() {
		select {
		case msg := <-cns.Deliveries():
			num, err := strconv.ParseInt(string(msg.Body), 0, 64)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}
			res := num + 1
			log.Printf("%v: %v", prefix, res)
			<-time.After(time.Second)
			err = pbl.Publish(amqp.Publishing{
				Body:         []byte(fmt.Sprint(res)),
				DeliveryMode: amqp.Persistent,
			})
			if err != nil {
				log.Printf("Client publish error: %v\n", err)
			}
		case err := <-cns.Errors():
			log.Printf("Consumer error: %v\n", err)
		case err := <-cli.Errors():
			log.Printf("Client error: %v\n", err)
		case blocked := <-cli.Blocking():
			log.Printf("Client is blocked %v\n", blocked)
		}
	}
}
