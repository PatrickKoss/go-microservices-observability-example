package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"go-microservices-observability/pkg/tracing"
)

const DeductItemsTopic = "deduct-items"

// DeductItemsMessage defines the structure of the message for deducting items.
type DeductItemsMessage struct {
	ProductIDs  []string            `json:"productIds"`
	SpanContext tracing.SpanContext `json:"spanContext"`
}

// NewDeductItemsHandler creates a new handler for deducting items from inventory.
func NewDeductItemsHandler(service Service, tracer tracing.Tracer) func(message []byte) error {
	return func(message []byte) error {
		var deductItemsMessage DeductItemsMessage
		if err := json.Unmarshal(message, &deductItemsMessage); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		ctx := context.Background()

		ctx, span := tracer.StartSpanWithContext(
			ctx,
			"internal.services.inventory.consumer.DeductItems",
			deductItemsMessage.SpanContext.SpanContext,
		)
		defer span.End()

		println("DeductItemsMessage: ", fmt.Sprintf("%+v", deductItemsMessage))

		// Deduct each product.
		for _, productID := range deductItemsMessage.ProductIDs {
			product, err := service.Get(ctx, productID)
			if err != nil {
				fmt.Printf("Error getting product %s: %v\n", productID, err)
				continue // Continue to the next product.
			}

			// For now, just delete the product.  In a real system, you'd likely
			// update a quantity or status field.
			err = service.Delete(ctx, productID)
			if err != nil {
				fmt.Printf("Error deleting product %s: %v\n", productID, err)
				// Consider whether to continue or return an error based on your requirements.
				return err
			}
			fmt.Printf("Product %s deducted (deleted) from inventory.\n", product.Name)
		}

		return nil
	}
}
