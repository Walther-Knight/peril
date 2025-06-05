package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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
	gamelogic.PrintServerHelp()

	for {
		userInput := gamelogic.GetInput()
		switch {
		case userInput == nil:
			continue

		case len(userInput) == 0:
			continue

		case strings.ToLower(userInput[0]) == "pause":
			log.Println("Sending pause message")
			err = pubsub.PublishJSON(pubSub, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
			if err != nil {
				log.Printf("Error publishing: %s", err)
				return
			}
			continue

		case strings.ToLower(userInput[0]) == "resume":
			log.Println("Sending resume message")
			err = pubsub.PublishJSON(pubSub, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
			if err != nil {
				log.Printf("Error publishing: %s", err)
				return
			}
			continue

		case strings.ToLower(userInput[0]) == "pause":
			log.Println("Sending pause message")
			err = pubsub.PublishJSON(pubSub, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
			if err != nil {
				log.Printf("Error publishing: %s", err)
				return
			}
			continue

		case strings.ToLower(userInput[0]) == "quit":
			fmt.Println("Shutting down Peril server...")
			return

		default:
			fmt.Println("Invalid command. Try again.")
			continue
		}
	}

}
