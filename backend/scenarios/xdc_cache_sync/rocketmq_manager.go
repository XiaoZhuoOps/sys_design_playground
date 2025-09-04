package xdccachesync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

func NewRocketMQManager(nameservers []string, topic, group string) (*RocketMQManager, error) {
	// Create producer
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(nameservers),
		producer.WithRetry(2),
		producer.WithGroupName("cache_invalidation_producer"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// Create consumer
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer(nameservers),
		consumer.WithConsumerModel(consumer.BroadCasting), // Broadcast mode for cache invalidation
		consumer.WithGroupName(group),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return &RocketMQManager{
		producer: p,
		consumer: c,
		topic:    topic,
		group:    group,
	}, nil
}

func (rmq *RocketMQManager) StartProducer() error {
	return rmq.producer.Start()
}

func (rmq *RocketMQManager) SendInvalidationMessage(msg *InvalidationMessage) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal invalidation message: %w", err)
	}

	rocketMsg := &primitive.Message{
		Topic: rmq.topic,
		Body:  msgBytes,
	}

	// Set message tag for filtering
	rocketMsg.WithTag("cache_invalidation")

	// Send async message
	_, err = rmq.producer.SendSync(context.Background(), rocketMsg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (rmq *RocketMQManager) StartConsuming(handler func(*InvalidationMessage) error) error {
	// Subscribe to topic
	err := rmq.consumer.Subscribe(rmq.topic, consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: "cache_invalidation",
	}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		for _, msg := range msgs {
			var invalidationMsg InvalidationMessage
			if err := json.Unmarshal(msg.Body, &invalidationMsg); err != nil {
				log.Printf("Failed to unmarshal invalidation message: %v", err)
				continue
			}

			if err := handler(&invalidationMsg); err != nil {
				log.Printf("Failed to handle invalidation message: %v", err)
				// Continue processing other messages even if one fails
				continue
			}
		}
		return consumer.ConsumeSuccess, nil
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start consumer
	return rmq.consumer.Start()
}

func (rmq *RocketMQManager) Stop() error {
	if rmq.producer != nil {
		if err := rmq.producer.Shutdown(); err != nil {
			log.Printf("Failed to shutdown producer: %v", err)
		}
	}

	if rmq.consumer != nil {
		if err := rmq.consumer.Shutdown(); err != nil {
			log.Printf("Failed to shutdown consumer: %v", err)
		}
	}

	return nil
}