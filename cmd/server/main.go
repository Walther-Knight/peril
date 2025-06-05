package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)

	if err != nil {
		log.Printf("Error connecting to amqp: %v", err)
		return
	}
	defer conn.Close()
	fmt.Println("Successful connection to server..")

	pubSub, err := conn.Channel()
	if err != nil {
		log.Printf("Error starting pubSub channel: %v", err)
		return
	}

	err = pubsub.PublishJSON(pubSub, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})
	if err != nil {
		log.Printf("Error publishing: %s", err)
		return
	}

	closePeril := make(chan os.Signal, 1)
	signal.Notify(closePeril, os.Interrupt)
	<-closePeril
	fmt.Println("Shutting down Peril...")

}
