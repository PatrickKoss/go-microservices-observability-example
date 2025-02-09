package order

import (
	"context"
	"errors"
	"go-microservices-observability/internal/domain"
	"sync"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrOrderAlreadyExists = errors.New("order already exists")

type Repository interface {
	List(ctx context.Context) ([]*domain.Order, error)
	Get(ctx context.Context, id string) (*domain.Order, error)
	Create(ctx context.Context, order *domain.Order) error
	Update(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	mu     sync.RWMutex
	orders map[string]*domain.Order
}

// NewRepository creates a new order repository.
func NewRepository() Repository {
	return &repository{
		orders: make(map[string]*domain.Order),
	}
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
