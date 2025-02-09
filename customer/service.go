package customer

import (
	"context"

	"go.opentelemetry.io/otel"
)

type Service interface {
	CreateCustomer(ctx context.Context, customer *Customer) error
	GetCustomer(ctx context.Context, id string) (*Customer, error)
	UpdateCustomer(ctx context.Context, customer *Customer) error
	DeleteCustomer(ctx context.Context, id string) error
	ListCustomers(ctx context.Context) ([]*Customer, error)
}

type Customer struct {
	ID string
	// Add customer fields
}

type service struct {
	repo CustomerRepository
}

func NewService(repo CustomerRepository) Service {
	return &service{repo: repo}
}

func (s *service) CreateCustomer(ctx context.Context, customer *Customer) error {
	ctx, span := otel.Tracer("customer-service").Start(ctx, "CreateCustomer")
	defer span.End()

	return s.repo.CreateCustomer(ctx, customer)
}

func (s *service) GetCustomer(ctx context.Context, id string) (*Customer, error) {
	ctx, span := otel.Tracer("customer-service").Start(ctx, "GetCustomer")
	defer span.End()

	return s.repo.GetCustomer(ctx, id)
}

func (s *service) UpdateCustomer(ctx context.Context, customer *Customer) error {
	ctx, span := otel.Tracer("customer-service").Start(ctx, "UpdateCustomer")
	defer span.End()

	return s.repo.UpdateCustomer(ctx, customer)
}

func (s *service) DeleteCustomer(ctx context.Context, id string) error {
	ctx, span := otel.Tracer("customer-service").Start(ctx, "DeleteCustomer")
	defer span.End()

	return s.repo.DeleteCustomer(ctx, id)
}

func (s *service) ListCustomers(ctx context.Context) ([]*Customer, error) {
	ctx, span := otel.Tracer("customer-service").Start(ctx, "ListCustomers")
	defer span.End()

	return s.repo.ListCustomers(ctx)
}
