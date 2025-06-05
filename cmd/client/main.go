package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	connString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)

	if err != nil {
		log.Printf("Error connecting to amqp: %v", err)
		return
	}
	defer conn.Close()
	fmt.Println("Successful connection to server..")

	_, err = conn.Channel()
	if err != nil {
		log.Printf("Error starting pubSub channel: %v", err)
		return
	}

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	gamelogic.PrintClientHelp()

	//userInput := gamelogic.GetInput()

	// durable = 0 transient = 1
	pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, fmt.Sprintf("pause.%s", userName), routing.PauseKey, 1)

	closePeril := make(chan os.Signal, 1)
	signal.Notify(closePeril, os.Interrupt)
	<-closePeril
	fmt.Println("Shutting down Peril client...")

}
