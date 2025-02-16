package order

import (
	"context"
	"encoding/json"
	"go-microservices-observability/internal/adapters/queue"
	orderRepo "go-microservices-observability/internal/adapters/repository/order"
	"go-microservices-observability/internal/domain"
	"go-microservices-observability/internal/services/inventory"
	"go-microservices-observability/internal/services/notification"
	"go-microservices-observability/pkg/tracing"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, order *domain.Order) error
	Get(ctx context.Context, id string) (*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.Order, error)
	Shutdown()
}

type service struct {
	repo        orderRepo.Repository
	tracer      tracing.Tracer
	queueClient queue.Queue
	worker      *orderRepo.OutboxWorker
}

func NewService(repo orderRepo.Repository, tracer tracing.Tracer, queueClient queue.Queue) Service {
	worker := orderRepo.NewOutboxWorker(repo, queueClient, 1*time.Second)
	worker.Start()

	return &service{
		repo:        repo,
		tracer:      tracer,
		queueClient: queueClient,
		worker:      worker,
	}
}

func (s *service) Create(ctx context.Context, order *domain.Order) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.order.Create")
	defer span.End()

	err := s.repo.Create(ctx, order)
	if err != nil {
		return err
	}

	// Create inventory deduction message
	deductItemsMsg := inventory.DeductItemsMessage{
		ProductIDs:  order.ProductIDs,
		SpanContext: tracing.NewSpanContext(span.SpanContext()),
	}
	deductItemsBytes, err := json.Marshal(deductItemsMsg)
	if err != nil {
		return err
	}

	// Store inventory message in outbox
	err = s.repo.StoreOutboxMessage(ctx, &orderRepo.OutboxMessage{
		ID:      uuid.New().String(),
		Topic:   inventory.DeductItemsTopic,
		Message: deductItemsBytes,
	})
	if err != nil {
		return err
	}

	// Create notification message
	notificationMsg := notification.SendNotificationMessage{
		UserID:      "test",
		SpanContext: tracing.NewSpanContext(span.SpanContext()),
	}
	notificationBytes, err := json.Marshal(notificationMsg)
	if err != nil {
		return err
	}

	// Store notification message in outbox
	err = s.repo.StoreOutboxMessage(ctx, &orderRepo.OutboxMessage{
		ID:      uuid.New().String(),
		Topic:   notification.SendNotificationTopic,
		Message: notificationBytes,
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

func (s *service) Shutdown() {
	if s.worker != nil {
		s.worker.Stop()
	}
}
