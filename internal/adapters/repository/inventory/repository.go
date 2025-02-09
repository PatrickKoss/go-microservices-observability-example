package inventory

import (
	"context"
	"errors"
	"go-microservices-observability/internal/domain"
	"sync"
)

var ErrProductNotFound = errors.New("product not found")

type Repository interface {
	Get(ctx context.Context, id string) (*domain.Product, error)
	List(ctx context.Context) ([]*domain.Product, error)
	Create(ctx context.Context, product *domain.Product) error
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	mu       sync.RWMutex
	products map[string]*domain.Product
}

func NewRepository() Repository {
	return &repository{
		products: make(map[string]*domain.Product),
	}
}

func (r *repository) Create(ctx context.Context, product *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[product.ID]; exists {
		return ErrProductNotFound
	}

	r.products[product.ID] = product

	return nil
}

func (r *repository) Get(ctx context.Context, id string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, exists := r.products[id]
	if !exists {
		return nil, ErrProductNotFound
	}

	return product, nil
}

func (r *repository) Update(ctx context.Context, product *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[product.ID]; !exists {
		return ErrProductNotFound
	}

	r.products[product.ID] = product

	return nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[id]; !exists {
		return ErrProductNotFound
	}

	delete(r.products, id)

	return nil
}

func (r *repository) List(ctx context.Context) ([]*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]*domain.Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}

	return products, nil
}
