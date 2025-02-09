package inventory

import (
	"context"
	"go-microservices-observability/internal/adapters/repository/inventory"
	"go-microservices-observability/internal/domain"
	"go-microservices-observability/pkg/tracing"
)

type Service interface {
	Get(ctx context.Context, id string) (*domain.Product, error)
	List(ctx context.Context) ([]*domain.Product, error)
	Create(ctx context.Context, product *domain.Product) error
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo   inventory.Repository
	tracer tracing.Tracer
}

func NewService(repo inventory.Repository, tracer tracing.Tracer) Service {
	return &service{repo: repo, tracer: tracer}
}

func (s *service) Get(ctx context.Context, id string) (*domain.Product, error) {
	ctx, span := s.tracer.Start(ctx, "internal.services.inventory.Get")
	defer span.End()

	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context) ([]*domain.Product, error) {
	ctx, span := s.tracer.Start(ctx, "internal.services.inventory.List")
	defer span.End()

	return s.repo.List(ctx)
}

func (s *service) Create(ctx context.Context, product *domain.Product) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.inventory.Create")
	defer span.End()

	return s.repo.Create(ctx, product)
}

func (s *service) Update(ctx context.Context, product *domain.Product) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.inventory.Update")
	defer span.End()

	return s.repo.Update(ctx, product)
}

func (s *service) Delete(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.inventory.Delete")
	defer span.End()

	return s.repo.Delete(ctx, id)
}
