package order

import (
	"context"
	"errors"
	"go-microservices-observability/internal/domain"
	"sync"
	"time"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrOrderAlreadyExists = errors.New("order already exists")

type OutboxMessage struct {
	ID        string
	Topic     string
	Message   []byte
	CreatedAt time.Time
	Status    string
}

type Repository interface {
	List(ctx context.Context) ([]*domain.Order, error)
	Get(ctx context.Context, id string) (*domain.Order, error)
	Create(ctx context.Context, order *domain.Order) error
	Update(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
	StoreOutboxMessage(ctx context.Context, message *OutboxMessage) error
	GetPendingOutboxMessages(ctx context.Context) ([]*OutboxMessage, error)
	MarkOutboxMessageAsProcessed(ctx context.Context, id string) error
}

type repository struct {
	mu     sync.RWMutex
	orders map[string]*domain.Order
	outbox map[string]*OutboxMessage
}

// NewRepository creates a new order repository.
func NewRepository() Repository {
	return &repository{
		orders: make(map[string]*domain.Order),
		outbox: make(map[string]*OutboxMessage),
	}
}

func (r *repository) StoreOutboxMessage(ctx context.Context, message *OutboxMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	message.CreatedAt = time.Now()
	message.Status = "pending"
	r.outbox[message.ID] = message

	return nil
}

func (r *repository) GetPendingOutboxMessages(ctx context.Context) ([]*OutboxMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var messages []*OutboxMessage
	for _, msg := range r.outbox {
		if msg.Status == "pending" {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

func (r *repository) MarkOutboxMessageAsProcessed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if msg, exists := r.outbox[id]; exists {
		msg.Status = "processed"
		return nil
	}

	return errors.New("outbox message not found")
}

func (r *repository) Create(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[order.ID]; exists {
		return ErrOrderAlreadyExists
	}

	r.orders[order.ID] = order

	return nil
}

func (r *repository) Get(ctx context.Context, id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.orders[id]
	if !exists {
		return nil, ErrOrderNotFound
	}

	return order, nil
}

func (r *repository) Update(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[order.ID]; !exists {
		return ErrOrderNotFound
	}

	r.orders[order.ID] = order

	return nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orders[id]; !exists {
		return ErrOrderNotFound
	}

	delete(r.orders, id)

	return nil
}

func (r *repository) List(ctx context.Context) ([]*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]*domain.Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}

	return orders, nil
}
