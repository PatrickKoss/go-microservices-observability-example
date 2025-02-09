package queue

// Handler is a function that processes messages.
type Handler func(message []byte) error

// Queue is an interface for a message queue.
type Queue interface {
	Publish(topic string, message interface{}) error
	Consume(topic string, handler Handler) error
}
