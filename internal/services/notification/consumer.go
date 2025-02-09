package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"go-microservices-observability/pkg/tracing"
)

const SendNotificationTopic = "send-notification"

// SendNotificationMessage defines the structure of the message for sending notifications.
type SendNotificationMessage struct {
	UserID      string              `json:"userId"`
	SpanContext tracing.SpanContext `json:"spanContext"`
}

// NewSendNotificationHandler creates a new handler for sending notifications.
func NewSendNotificationHandler(service Service, tracer tracing.Tracer) func(message []byte) error {
	return func(message []byte) error {
		var sendNotificationMessage SendNotificationMessage
		if err := json.Unmarshal(message, &sendNotificationMessage); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		ctx := context.Background()

		ctx, span := tracer.StartSpanWithContext(
			ctx,
			"internal.services.notification.consumer.SendNotification",
			sendNotificationMessage.SpanContext.SpanContext,
		)
		defer span.End()

		println("SendNotificationMessage: ", fmt.Sprintf("%+v", sendNotificationMessage))

		// "Publish" the notification using the notification service.
		if err := service.Publish(ctx, sendNotificationMessage.UserID); err != nil {
			fmt.Printf("Error publishing notification to user %s: %v\n", sendNotificationMessage.UserID, err)
			return err
		}

		fmt.Printf("Notification sent (simulated) to user %s\n", sendNotificationMessage.UserID)
		return nil
	}
}
