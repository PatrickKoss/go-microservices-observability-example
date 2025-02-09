package order

import (
	"context"
	"go-microservices-observability/internal/adapters/queue"
	"go-microservices-observability/internal/adapters/repository/order"
	"go-microservices-observability/internal/domain"
	"go-microservices-observability/internal/services/inventory"
	"go-microservices-observability/internal/services/notification"
	"go-microservices-observability/pkg/tracing"
)

type Service interface {
	Create(ctx context.Context, order *domain.Order) error
	Get(ctx context.Context, id string) (*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.Order, error)
}

type service struct {
	repo        order.Repository
	tracer      tracing.Tracer
	queueClient queue.Queue
}

func NewService(repo order.Repository, tracer tracing.Tracer, queueClient queue.Queue) Service {
	return &service{repo: repo, tracer: tracer, queueClient: queueClient}
}

func (s *service) Create(ctx context.Context, order *domain.Order) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.order.Create")
	defer span.End()

	err := s.repo.Create(ctx, order)
	if err != nil {
		return err
	}

	err = s.queueClient.Publish(inventory.DeductItemsTopic, inventory.DeductItemsMessage{
		ProductIDs:  order.ProductIDs,
		SpanContext: tracing.NewSpanContext(span.SpanContext()),
	})
	if err != nil {
		return nil
	}

	err = s.queueClient.Publish(notification.SendNotificationTopic, notification.SendNotificationMessage{
		UserID:      "test",
		SpanContext: tracing.NewSpanContext(span.SpanContext()),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) Get(ctx context.Context, id string) (*domain.Order, error) {
	ctx, span := s.tracer.Start(ctx, "internal.services.order.Get")
	defer span.End()

	return s.repo.Get(ctx, id)
}

func (s *service) Update(ctx context.Context, order *domain.Order) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.order.Update")
	defer span.End()

	return s.repo.Update(ctx, order)
}

func (s *service) Delete(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.order.Delete")
	defer span.End()

	return s.repo.Delete(ctx, id)
}

func (s *service) List(ctx context.Context) ([]*domain.Order, error) {
	ctx, span := s.tracer.Start(ctx, "internal.services.order.List")
	defer span.End()

	return s.repo.List(ctx)
}
