package order

import (
	"context"
	"encoding/json"
	"go-microservices-observability/internal/adapters/queue"
	"log"
	"time"
)

type OutboxWorker struct {
	repository Repository
	queue      queue.Queue
	interval   time.Duration
	done       chan struct{}
}

func NewOutboxWorker(repository Repository, queue queue.Queue, interval time.Duration) *OutboxWorker {
	return &OutboxWorker{
		repository: repository,
		queue:      queue,
		interval:   interval,
		done:       make(chan struct{}),
	}
}

func (w *OutboxWorker) Start() {
	go w.processOutbox()
}

func (w *OutboxWorker) Stop() {
	close(w.done)
}

func (w *OutboxWorker) processOutbox() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			if err := w.processMessages(); err != nil {
				log.Printf("Error processing outbox messages: %v", err)
			}
		}
	}
}

func (w *OutboxWorker) processMessages() error {
	ctx := context.Background()
	messages, err := w.repository.GetPendingOutboxMessages(ctx)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		// Create a map to hold the raw JSON message
		var rawMessage map[string]interface{}
		if err := json.Unmarshal(msg.Message, &rawMessage); err != nil {
			log.Printf("Error unmarshaling message %s: %v", msg.ID, err)
			continue
		}

		// Re-marshal the message to ensure it's properly formatted JSON
		messageBytes, err := json.Marshal(rawMessage)
		if err != nil {
			log.Printf("Error re-marshaling message %s: %v", msg.ID, err)
			continue
		}

		if err := w.queue.Publish(msg.Topic, messageBytes); err != nil {
			log.Printf("Error publishing message %s: %v", msg.ID, err)
			continue
		}

		if err := w.repository.MarkOutboxMessageAsProcessed(ctx, msg.ID); err != nil {
			log.Printf("Error marking message %s as processed: %v", msg.ID, err)
		}
	}

	return nil
}
