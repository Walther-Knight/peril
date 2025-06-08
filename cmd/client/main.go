package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

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

	// Pause handler
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, fmt.Sprintf("pause.%s", userName), routing.PauseKey, 1, HandlerPause(gs))
	if err != nil {
		log.Printf("Error subscribing to JSON: %v", err)
		return
	}
	// Move Handler
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, fmt.Sprintf("army_moves.%s", userName), "army_moves.*", 1, func(receivedMove gamelogic.ArmyMove) string {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(receivedMove)
		switch {
		case moveOutcome == gamelogic.MoveOutcomeSamePlayer:
			return "NackDiscard"
		case moveOutcome == gamelogic.MoveOutComeSafe:
			return "Ack"
		case moveOutcome == gamelogic.MoveOutcomeMakeWar:
			err = pubsub.PublishJSON(pubSub, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, userName), gamelogic.RecognitionOfWar{
				Attacker: receivedMove.Player,
				Defender: gs.Player,
			})
			if err != nil {
				return "NackRequeue"
			}
			return "Ack"
		default:
			return "NackDiscard"
		}
	})
	if err != nil {
		log.Printf("Error subscribing to JSON: %v", err)
		return
	}
	// War handler
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix), 0, func(rw gamelogic.RecognitionOfWar) string {
		defer fmt.Print("> ")
		warOutcome, winner, loser := gs.HandleWar(rw)
		switch {
		case warOutcome == gamelogic.WarOutcomeNotInvolved:
			return "NackRequeue"
		case warOutcome == gamelogic.WarOutcomeNoUnits:
			return "NackDiscard"
		case warOutcome == gamelogic.WarOutcomeOpponentWon:
			err = pubsub.PublishGob(pubSub, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.GameLogSlug, userName), routing.GameLog{
				CurrentTime: time.Now(),
				Message:     fmt.Sprintf("%s won a war against %s", winner, loser),
				Username:    userName,
			})
			if err != nil {
				return "NackRequeue"
			}
			return "Ack"
		case warOutcome == gamelogic.WarOutcomeYouWon:
			err = pubsub.PublishGob(pubSub, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.GameLogSlug, userName), routing.GameLog{
				CurrentTime: time.Now(),
				Message:     fmt.Sprintf("%s won a war against %s", winner, loser),
				Username:    userName,
			})
			if err != nil {
				return "NackRequeue"
			}
			return "Ack"
		case warOutcome == gamelogic.WarOutcomeDraw:
			err = pubsub.PublishGob(pubSub, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.GameLogSlug, userName), routing.GameLog{
				CurrentTime: time.Now(),
				Message:     fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser),
				Username:    userName,
			})
			if err != nil {
				return "NackRequeue"
			}
			return "Ack"
		default:
			log.Println("Error resolving war condition. Discarding message.")
			return "NackDiscard"
		}
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
			if len(userInput) <= 1 {
				fmt.Println("Spamming requires an integer in second argument!")
				continue
			}
			spamQuantity, err := strconv.Atoi(userInput[1])
			if err != nil {
				fmt.Printf("Spamming requires and integer. You provided: %s", userInput[1])
			}
			for range spamQuantity {
				msg := gamelogic.GetMaliciousLog()
				err := pubsub.PublishGob(pubSub, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.GameLogSlug, userName), routing.GameLog{
					CurrentTime: time.Now(),
					Message:     msg,
					Username:    userName,
				})
				if err != nil {
					log.Printf("Error spamming Gob to game_logs: %v", err)
					continue
				}
			}
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

func HandlerPause(gs *gamelogic.GameState) func(routing.PlayingState) string {
	return func(ps routing.PlayingState) string { defer fmt.Print("> "); gs.HandlePause(ps); return "Ack" }
}
