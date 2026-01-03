// Package examples demonstrates saga pattern usage for multi-step operations with compensation.
//
// This example shows how to use the saga pattern to orchestrate a multi-step order creation
// process that spans multiple modules (order → inventory → payment). If any step fails,
// compensation is executed to roll back completed steps.
//
// This is a simplified example. In production, consider using Temporal for durable,
// distributed saga orchestration.
package examples

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cmelgarejo/go-modulith-template/internal/saga"
)

// Mock services for demonstration
type orderService struct {
	orders map[string]bool
}

func newOrderService() *orderService {
	return &orderService{
		orders: make(map[string]bool),
	}
}

func (s *orderService) CreateOrder(_ context.Context, orderID string) error {
	s.orders[orderID] = true
	return nil
}

func (s *orderService) CancelOrder(_ context.Context, orderID string) error {
	delete(s.orders, orderID)
	return nil
}

func (s *orderService) HasOrder(orderID string) bool {
	return s.orders[orderID]
}

type inventoryService struct {
	stock map[string]int
}

func newInventoryService() *inventoryService {
	return &inventoryService{
		stock: map[string]int{
			"item1": 10,
			"item2": 5,
		},
	}
}

func (s *inventoryService) ReserveItem(_ context.Context, itemID string, quantity int) error {
	currentStock := s.stock[itemID]
	if currentStock < quantity {
		return errors.New("insufficient stock")
	}

	s.stock[itemID] = currentStock - quantity

	return nil
}

func (s *inventoryService) ReleaseReservation(_ context.Context, itemID string, quantity int) error {
	s.stock[itemID] += quantity
	return nil
}

func (s *inventoryService) GetStock(itemID string) int {
	return s.stock[itemID]
}

type paymentService struct {
	payments map[string]bool
}

func newPaymentService() *paymentService {
	return &paymentService{
		payments: make(map[string]bool),
	}
}

func (s *paymentService) ProcessPayment(_ context.Context, orderID string, _ float64) error {
	s.payments[orderID] = true
	return nil
}

func (s *paymentService) RefundPayment(_ context.Context, orderID string) error {
	delete(s.payments, orderID)
	return nil
}

func (s *paymentService) HasPayment(orderID string) bool {
	return s.payments[orderID]
}

// TestSagaOrderCreation_Success demonstrates a successful multi-step saga.
func TestSagaOrderCreation_Success(t *testing.T) {
	ctx := context.Background()

	orderSvc := newOrderService()
	inventorySvc := newInventoryService()
	paymentSvc := newPaymentService()

	orderID := "order-123"
	itemID := "item1"
	quantity := 2
	amount := 99.99

	// Create saga for order creation
	saga := saga.New()

	// Step 1: Create order
	saga.AddStep("create_order",
		func(ctx context.Context) error {
			return orderSvc.CreateOrder(ctx, orderID)
		},
		func(ctx context.Context) error {
			return orderSvc.CancelOrder(ctx, orderID)
		},
	)

	// Step 2: Reserve inventory
	saga.AddStep("reserve_inventory",
		func(ctx context.Context) error {
			return inventorySvc.ReserveItem(ctx, itemID, quantity)
		},
		func(ctx context.Context) error {
			return inventorySvc.ReleaseReservation(ctx, itemID, quantity)
		},
	)

	// Step 3: Process payment
	saga.AddStep("process_payment",
		func(ctx context.Context) error {
			return paymentSvc.ProcessPayment(ctx, orderID, amount)
		},
		func(ctx context.Context) error {
			return paymentSvc.RefundPayment(ctx, orderID)
		},
	)

	// Execute saga
	err := saga.Execute(ctx)
	require.NoError(t, err)

	// Verify all steps completed successfully
	assert.True(t, orderSvc.HasOrder(orderID))
	assert.Equal(t, 8, inventorySvc.GetStock(itemID)) // 10 - 2 = 8
	assert.True(t, paymentSvc.HasPayment(orderID))
}

// TestSagaOrderCreation_WithCompensation demonstrates compensation when a step fails.
func TestSagaOrderCreation_WithCompensation(t *testing.T) {
	ctx := context.Background()

	orderSvc := newOrderService()
	inventorySvc := newInventoryService()
	paymentSvc := newPaymentService()

	orderID := "order-456"
	itemID := "item2"
	quantity := 10 // More than available stock (5)
	amount := 199.99

	// Create saga for order creation
	saga := saga.New()

	// Step 1: Create order (will succeed)
	saga.AddStep("create_order",
		func(ctx context.Context) error {
			return orderSvc.CreateOrder(ctx, orderID)
		},
		func(ctx context.Context) error {
			return orderSvc.CancelOrder(ctx, orderID)
		},
	)

	// Step 2: Reserve inventory (will fail - insufficient stock)
	saga.AddStep("reserve_inventory",
		func(ctx context.Context) error {
			return inventorySvc.ReserveItem(ctx, itemID, quantity)
		},
		func(ctx context.Context) error {
			return inventorySvc.ReleaseReservation(ctx, itemID, quantity)
		},
	)

	// Step 3: Process payment (won't execute due to step 2 failure)
	saga.AddStep("process_payment",
		func(ctx context.Context) error {
			return paymentSvc.ProcessPayment(ctx, orderID, amount)
		},
		func(ctx context.Context) error {
			return paymentSvc.RefundPayment(ctx, orderID)
		},
	)

	// Execute saga - should fail at step 2
	err := saga.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserve_inventory")

	// Verify compensation: order should be cancelled
	assert.False(t, orderSvc.HasOrder(orderID))

	// Inventory should not have changed (reservation failed)
	assert.Equal(t, 5, inventorySvc.GetStock(itemID))

	// Payment should not have been processed
	assert.False(t, paymentSvc.HasPayment(orderID))
}

// TestSagaOrderCreation_PaymentFailure demonstrates compensation when payment fails.
func TestSagaOrderCreation_PaymentFailure(t *testing.T) {
	ctx := context.Background()

	orderSvc := newOrderService()
	inventorySvc := newInventoryService()
	paymentSvc := newPaymentService()

	orderID := "order-789"
	itemID := "item1"
	quantity := 1

	// Create saga for order creation
	saga := saga.New()

	// Step 1: Create order (will succeed)
	saga.AddStep("create_order",
		func(ctx context.Context) error {
			return orderSvc.CreateOrder(ctx, orderID)
		},
		func(ctx context.Context) error {
			return orderSvc.CancelOrder(ctx, orderID)
		},
	)

	// Step 2: Reserve inventory (will succeed)
	saga.AddStep("reserve_inventory",
		func(ctx context.Context) error {
			return inventorySvc.ReserveItem(ctx, itemID, quantity)
		},
		func(ctx context.Context) error {
			return inventorySvc.ReleaseReservation(ctx, itemID, quantity)
		},
	)

	// Step 3: Process payment (will fail)
	saga.AddStep("process_payment",
		func(_ context.Context) error {
			return errors.New("payment declined")
		},
		func(ctx context.Context) error {
			return paymentSvc.RefundPayment(ctx, orderID)
		},
	)

	// Execute saga - should fail at step 3
	err := saga.Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "process_payment")

	// Verify compensation: order should be cancelled and inventory released
	assert.False(t, orderSvc.HasOrder(orderID))
	assert.Equal(t, 10, inventorySvc.GetStock(itemID)) // Stock should be restored
	assert.False(t, paymentSvc.HasPayment(orderID))
}

