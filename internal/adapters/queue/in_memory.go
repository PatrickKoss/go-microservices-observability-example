package queue

import (
	"encoding/json"
	"fmt"
	"sync"
)

// InMemoryQueue is an in-memory implementation of the Queue interface.
type InMemoryQueue struct {
	messages map[string]chan []byte
	mu       sync.RWMutex
}

// NewInMemoryQueue creates a new InMemoryQueue.
func NewInMemoryQueue() Queue {
	return &InMemoryQueue{
		messages: make(map[string]chan []byte),
	}
}

// Publish publishes a message to the queue on a specific topic.
func (q *InMemoryQueue) Publish(topic string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}

	q.mu.RLock()
	topicChan, ok := q.messages[topic]
	q.mu.RUnlock()

	if !ok {
		return fmt.Errorf("topic %s does not exist", topic)
	}

	topicChan <- jsonMessage
	return nil
}

// Consume consumes messages from the queue on a specific topic and passes them to the handler.
func (q *InMemoryQueue) Consume(topic string, handler Handler) error {
	q.mu.Lock()
	topicChan, ok := q.messages[topic]
	if !ok {
		topicChan = make(chan []byte)
		q.messages[topic] = topicChan
	}
	q.mu.Unlock()

	for message := range topicChan {
		err := handler(message)
		if err != nil {
			fmt.Printf("Error processing message: %v\n", err)
		}
	}

	return nil
}
