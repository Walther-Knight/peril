package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		return err
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         jsonData,
		DeliveryMode: 2,
	})
	if err != nil {
		return err
	}
	return nil
}
