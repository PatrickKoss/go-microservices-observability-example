package notification

import (
	"context"
	"fmt"
	"go-microservices-observability/pkg/tracing"
)

// Service defines the interface for the notification service.
type Service interface {
	Publish(ctx context.Context, userID string) error
}

type service struct {
	tracer tracing.Tracer
}

// NewService creates a new notification service.
func NewService(tracer tracing.Tracer) Service {
	return &service{tracer: tracer}
}

// Publish "publishes" a notification to the given user (in this example, it just logs a message).
func (s *service) Publish(ctx context.Context, userID string) error {
	ctx, span := s.tracer.Start(ctx, "internal.services.notification.Publish")
	defer span.End()

	fmt.Printf("Simulating sending notification to user: %s\n", userID)
	return nil
}
