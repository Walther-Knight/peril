package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
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

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	chanName, err := conn.Channel()
	if err != nil {
		log.Printf("Error starting pubSub channel: %v", err)
		return nil, amqp.Queue{}, err
	}
	var durable bool
	var autoDelete bool
	var exclusive bool
	if simpleQueueType == 1 {
		durable = false
		autoDelete = true
		exclusive = true
	} else {
		durable = true
		autoDelete = false
		exclusive = false
	}

	deadLetter := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}

	var queue amqp.Queue

	if queueName == "peril_dlq" {
		queue, err = chanName.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)
		if err != nil {
			log.Printf("Error declaring pubsub queue: %v", err)
			return nil, amqp.Queue{}, err
		}
	} else {
		queue, err = chanName.QueueDeclare(queueName, durable, autoDelete, exclusive, false, deadLetter)
		if err != nil {
			log.Printf("Error declaring pubsub queue: %v", err)
			return nil, amqp.Queue{}, err
		}
	}

	err = chanName.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		log.Printf("Error binding pubsub queue: %v", err)
		return nil, amqp.Queue{}, err
	}

	return chanName, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) string,
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	deliveryChan, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("Error getting delivery channel: %v", err)
		return err
	}
	go func(D <-chan amqp.Delivery) {
		for delivery := range D {
			var data T
			json.Unmarshal(delivery.Body, &data)
			acktype := handler(data)
			switch {
			case acktype == "Ack":
				log.Printf("Ack for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Ack(false)
			case acktype == "NackRequeue":
				log.Printf("NackRequeue for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Nack(false, true)
			case acktype == "NackDiscard":
				log.Printf("NackDiscard for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Nack(false, false)
			default:
				log.Printf("Default for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Nack(false, false)
			}
		}
	}(deliveryChan)

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
	if err != nil {
		return err
	}

	err = ch.Publish(exchange, key, false, false, amqp.Publishing{
		ContentType:  "application/gob",
		Body:         buffer.Bytes(),
		DeliveryMode: 2,
	})
	if err != nil {
		return err
	}
	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) string,
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	err = channel.Qos(10, 0, true)
	if err != nil {
		log.Printf("Error setting channel QoS: %v", err)
	}
	deliveryChan, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("Error getting delivery channel: %v", err)
		return err
	}
	go func(D <-chan amqp.Delivery) {
		for delivery := range D {
			var data T
			buffer := bytes.NewBuffer(delivery.Body)
			decoder := gob.NewDecoder(buffer)
			err = decoder.Decode(&data)
			if err != nil {
				log.Println(data)
				log.Printf("Error decoding gob: %v", err)
				return
			}
			acktype := handler(data)
			switch {
			case acktype == "Ack":
				log.Printf("Ack for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Ack(false)
			case acktype == "NackRequeue":
				log.Printf("NackRequeue for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Nack(false, true)
			case acktype == "NackDiscard":
				log.Printf("NackDiscard for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Nack(false, false)
			default:
				log.Printf("Default for key: %v Message Body: %v\n", delivery.RoutingKey, data)
				delivery.Nack(false, false)
			}
		}
	}(deliveryChan)

	return nil
}
