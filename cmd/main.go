package main

import (
	"context"
	"fmt"
	"go-microservices-observability/internal/adapters/queue"
	inventory2 "go-microservices-observability/internal/adapters/repository/inventory"
	"go-microservices-observability/internal/adapters/repository/order"
	order_rest "go-microservices-observability/internal/adapters/rest/order"
	user_rest "go-microservices-observability/internal/adapters/rest/user"
	"go-microservices-observability/internal/adapters/user"
	"go-microservices-observability/internal/services/inventory"
	"go-microservices-observability/internal/services/notification"
	order_service "go-microservices-observability/internal/services/order"
	"go-microservices-observability/pkg/diagnostics"
	"go-microservices-observability/pkg/tracing"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	orderServiceExporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(
			fmt.Sprintf("%s:%s", "localhost", "4317"),
		),
		otlptracegrpc.WithReconnectionPeriod(5*time.Second),
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	queueClient := queue.NewInMemoryQueue()

	orderServiceTracer := tracing.NewTracer("order-service", orderServiceExporter)
	orderRepository := order.NewRepository()
	orderService := order_service.NewService(orderRepository, orderServiceTracer, queueClient)

	orderRestAPITracer := tracing.NewTracer("order-rest-api", orderServiceExporter)

	userClientTracer := tracing.NewTracer("user-client", orderServiceExporter)

	userRestAPITracer := tracing.NewTracer("user-rest-api", orderServiceExporter)
	userRestAPI := user_rest.NewServer(userRestAPITracer)

	inventoryServiceTracer := tracing.NewTracer("inventory-service", orderServiceExporter)
	inventoryRepository := inventory2.NewRepository()
	inventoryService := inventory.NewService(inventoryRepository, inventoryServiceTracer)
	deductItemTracer := tracing.NewTracer("deduct-item-handler", orderServiceExporter)
	deductItemsHandler := inventory.NewDeductItemsHandler(inventoryService, deductItemTracer)

	go func() {
		err = queueClient.Consume(inventory.DeductItemsTopic, deductItemsHandler)
		if err != nil {
			panic(err)
		}
	}()

	notificationServiceTracer := tracing.NewTracer("notification-service", orderServiceExporter)
	notificationService := notification.NewService(notificationServiceTracer)
	notificationTracer := tracing.NewTracer("send-notification-handler", orderServiceExporter)
	sendNotificationHandler := notification.NewSendNotificationHandler(
		notificationService,
		notificationTracer,
	)

	go func() {
		err = queueClient.Consume(notification.SendNotificationTopic, sendNotificationHandler)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := userRestAPI.ListenAndServe(8081); err != nil {
			panic(err)
		}
	}()

	diagnosticsServer := diagnostics.NewServer(9000)
	go func() {
		if err := diagnosticsServer.Start(); err != nil {
			log.Println(err)
		}
	}()

	userClient := user.NewClient(&user.Config{
		Address:    "http://localhost:8081",
		HTTPClient: &http.Client{},
		Tracer:     userClientTracer,
	})

	orderRestAPIServer := order_rest.NewServer(orderService, orderRestAPITracer, userClient)
	go func() {
		err := orderRestAPIServer.ListenAndServe(8080)
		if err != nil {
			log.Println(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Println("Shutting down servers...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = diagnosticsServer.Shutdown(ctx)
	if err != nil {
		log.Println(err)
	}

	err = orderRestAPIServer.Shutdown(ctx)
	if err != nil {
		log.Println(err)
	}

	err = userRestAPI.Shutdown(ctx)
	if err != nil {
		log.Println(err)
	}

	// Shutdown order service and its worker
	orderService.Shutdown()

	cancel()
}
