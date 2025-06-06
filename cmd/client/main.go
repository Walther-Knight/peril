package main

import (
	"fmt"
	"log"
	"os"

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

	pubSub, err := conn.Channel()
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
	// binding for queues
	pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, fmt.Sprintf("pause.%s", userName), routing.PauseKey, 1)
	pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, fmt.Sprintf("army_moves.%s", userName), "army_moves.*", 1)

	// instantiate new game
	gs := gamelogic.NewGameState(userName)

	// subscribe handlers for commands
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, fmt.Sprintf("pause.%s", userName), routing.PauseKey, 1, HandlerPause(gs))
	if err != nil {
		log.Printf("Error subscribing to JSON: %v", err)
		return
	}

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, fmt.Sprintf("army_moves.%s", userName), "army_moves.*", 1, func(receivedMove gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(receivedMove)
	})
	if err != nil {
		log.Printf("Error subscribing to JSON: %v", err)
		return
	}

	for {
		userInput := gamelogic.GetInput()
		switch {
		case userInput == nil:
			continue

		case len(userInput) == 0:
			continue

		case userInput[0] == "spawn":
			err := gs.CommandSpawn(userInput)
			if err != nil {
				fmt.Println(err)
				continue
			}
			continue

		case userInput[0] == "move":
			move, err := gs.CommandMove(userInput)
			if err != nil {
				fmt.Println(err)
				continue
			}
			pubsub.PublishJSON(pubSub, routing.ExchangePerilTopic, fmt.Sprintf("army_moves.%s", userName), move)
			continue

		case userInput[0] == "status":
			gs.CommandStatus()
			continue

		case userInput[0] == "help":
			gamelogic.PrintClientHelp()
			continue

		case userInput[0] == "spam":
			fmt.Println("Spamming not allowed yet!")
			continue

		case userInput[0] == "quit":
			gamelogic.PrintQuit()
			return

		default:
			fmt.Println("Invalid command. Try again.")
			continue
		}
	}
}

func HandlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) { defer fmt.Print("> "); gs.HandlePause(ps) }
}
