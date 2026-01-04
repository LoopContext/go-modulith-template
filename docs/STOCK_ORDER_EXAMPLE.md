# Stock and Order System: Complete Step-by-Step Guide

This guide demonstrates how to create a complex e-commerce system with **Stock** and **Order** modules that communicate with each other using the go-modulith-template.

## Overview

We'll build:

-   **Stock Module**: Manages inventory (products, stock levels, reservations)
-   **Order Module**: Manages orders (creation, processing, fulfillment)
-   **Communication**: Order module calls Stock module via gRPC to reserve inventory

## Architecture

```
┌─────────────────────────────────────────────────┐
│           Modulith Process (cmd/server)         │
│                                                 │
│  ┌──────────┐         ┌──────────┐            │
│  │  Stock   │◄────────►│  Order   │            │
│  │  Module  │  gRPC    │  Module  │            │
│  └────┬─────┘          └────┬─────┘            │
│       │                     │                   │
│       └─────────┬───────────┘                   │
│                 │                               │
│          ┌──────▼──────┐                        │
│          │ Event Bus   │                        │
│          │ (in-memory) │                        │
│          └─────────────┘                        │
│                                                 │
│  ┌─────────────────────────────────────────┐    │
│  │  Shared: PostgreSQL, Redis (optional)    │    │
│  └─────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

## Step-by-Step Implementation

### Step 1: Scaffold the Stock Module

```bash
make new-module stock
```

This creates:

-   `modules/stock/` - Module structure
-   `proto/stock/v1/stock.proto` - Protocol definitions
-   `cmd/stock/main.go` - Standalone service entrypoint
-   `configs/stock.yaml` - Module configuration

### Step 2: Define Stock Module Proto Contract

Edit `proto/stock/v1/stock.proto`:

```protobuf
syntax = "proto3";

package stock.v1;

import "google/api/annotations.proto";
import "buf/validate/validate.proto";

option go_package = "github.com/cmelgarejo/go-modulith-template/gen/go/proto/stock/v1;stockv1";

service StockService {
  // Reserve stock for an order
  rpc ReserveStock(ReserveStockRequest) returns (ReserveStockResponse) {
    option (google.api.http) = {
      post: "/v1/stock/reserve"
      body: "*"
    };
  }

  // Release reserved stock (if order cancelled)
  rpc ReleaseStock(ReleaseStockRequest) returns (ReleaseStockResponse) {
    option (google.api.http) = {
      post: "/v1/stock/release"
      body: "*"
    };
  }

  // Confirm stock reservation (order confirmed)
  rpc ConfirmReservation(ConfirmReservationRequest) returns (ConfirmReservationResponse) {
    option (google.api.http) = {
      post: "/v1/stock/confirm"
      body: "*"
    };
  }

  // Get product stock level
  rpc GetStockLevel(GetStockLevelRequest) returns (GetStockLevelResponse) {
    option (google.api.http) = {
      get: "/v1/stock/{product_id}"
    };
  }

  // List all products with stock
  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {
    option (google.api.http) = {
      get: "/v1/stock/products"
    };
  }
}

message ReserveStockRequest {
  string order_id = 1 [(buf.validate.field).string.min_len = 1];
  repeated StockItem items = 2 [(buf.validate.field).repeated.min_items = 1];
}

message StockItem {
  string product_id = 1 [(buf.validate.field).string.min_len = 1];
  int32 quantity = 2 [(buf.validate.field).int32.gt = 0];
}

message ReserveStockResponse {
  string reservation_id = 1;
  bool success = 2;
  repeated StockItem reserved_items = 3;
}

message ReleaseStockRequest {
  string reservation_id = 1 [(buf.validate.field).string.min_len = 1];
}

message ReleaseStockResponse {
  bool success = 1;
}

message ConfirmReservationRequest {
  string reservation_id = 1 [(buf.validate.field).string.min_len = 1];
}

message ConfirmReservationResponse {
  bool success = 1;
}

message GetStockLevelRequest {
  string product_id = 1 [(buf.validate.field).string.min_len = 1];
}

message GetStockLevelResponse {
  string product_id = 1;
  int32 available = 2;
  int32 reserved = 3;
  int32 total = 4;
}

message ListProductsRequest {
  int32 page = 1;
  int32 page_size = 2;
}

message ListProductsResponse {
  repeated Product products = 1;
  int32 total = 2;
}

message Product {
  string id = 1;
  string name = 2;
  string sku = 3;
  int32 available_stock = 4;
  int32 reserved_stock = 5;
}
```

### Step 3: Create Stock Database Schema

Edit `modules/stock/resources/db/migration/000001_initial_schema.up.sql`:

```sql
-- Products table
CREATE TABLE products (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  sku VARCHAR(100) NOT NULL UNIQUE,
  description TEXT,
  price DECIMAL(10, 2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Stock levels table
CREATE TABLE stock_levels (
  product_id VARCHAR(64) PRIMARY KEY REFERENCES products(id) ON DELETE CASCADE,
  available INT NOT NULL DEFAULT 0,
  reserved INT NOT NULL DEFAULT 0,
  total INT NOT NULL DEFAULT 0,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Stock reservations table (for order processing)
CREATE TABLE stock_reservations (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL,
  product_id VARCHAR(64) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  quantity INT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'reserved', -- reserved, confirmed, released
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_reservations_order_id ON stock_reservations(order_id);
CREATE INDEX idx_reservations_status ON stock_reservations(status);
CREATE INDEX idx_reservations_expires_at ON stock_reservations(expires_at);
```

Create the down migration: `000001_initial_schema.down.sql`:

```sql
DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS stock_levels;
DROP TABLE IF EXISTS products;
```

### Step 4: Create Stock SQL Queries

Edit `modules/stock/internal/db/query/stock.sql`:

```sql
-- name: CreateProduct :exec
INSERT INTO products (id, name, sku, description, price)
VALUES ($1, $2, $3, $4, $5);

-- name: CreateStockLevel :exec
INSERT INTO stock_levels (product_id, available, reserved, total)
VALUES ($1, $2, $3, $4);

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1 LIMIT 1;

-- name: GetStockLevel :one
SELECT * FROM stock_levels WHERE product_id = $1 LIMIT 1;

-- name: UpdateStockLevel :exec
UPDATE stock_levels
SET available = $2, reserved = $3, total = $4, updated_at = CURRENT_TIMESTAMP
WHERE product_id = $1;

-- name: ReserveStock :exec
INSERT INTO stock_reservations (id, order_id, product_id, quantity, status, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetReservationByID :one
SELECT * FROM stock_reservations WHERE id = $1 LIMIT 1;

-- name: GetReservationsByOrderID :many
SELECT * FROM stock_reservations WHERE order_id = $1;

-- name: UpdateReservationStatus :exec
UPDATE stock_reservations
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ListProducts :many
SELECT p.*, sl.available as available_stock, sl.reserved as reserved_stock
FROM products p
LEFT JOIN stock_levels sl ON p.id = sl.product_id
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountProducts :one
SELECT COUNT(*) FROM products;
```

### Step 5: Generate Code

```bash
# Generate gRPC code from proto
make proto

# Generate SQL code from queries
make sqlc
```

### Step 6: Implement Stock Repository

Edit `modules/stock/internal/repository/repository.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cmelgarejo/go-modulith-template/modules/stock/internal/db/store"
)

type Repository interface {
	CreateProduct(ctx context.Context, id, name, sku, description string, price float64) error
	CreateStockLevel(ctx context.Context, productID string, available, reserved, total int32) error
	GetProduct(ctx context.Context, id string) (*store.Product, error)
	GetStockLevel(ctx context.Context, productID string) (*store.StockLevel, error)
	UpdateStockLevel(ctx context.Context, productID string, available, reserved, total int32) error
	ReserveStock(ctx context.Context, reservationID, orderID, productID string, quantity int32, expiresAt sql.NullTime) error
	GetReservation(ctx context.Context, id string) (*store.StockReservation, error)
	GetReservationsByOrderID(ctx context.Context, orderID string) ([]*store.StockReservation, error)
	UpdateReservationStatus(ctx context.Context, id, status string) error
	ListProducts(ctx context.Context, limit, offset int32) ([]*store.ListProductsRow, error)
	CountProducts(ctx context.Context) (int64, error)
	WithTx(ctx context.Context, fn func(Repository) error) error
}

type SQLRepository struct {
	q  *store.Queries
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{
		q:  store.New(db),
		db: db,
	}
}

func (r *SQLRepository) CreateProduct(ctx context.Context, id, name, sku, description string, price float64) error {
	return r.q.CreateProduct(ctx, store.CreateProductParams{
		ID:          id,
		Name:        name,
		Sku:         sku,
		Description: sql.NullString{String: description, Valid: description != ""},
		Price:       fmt.Sprintf("%.2f", price),
	})
}

func (r *SQLRepository) CreateStockLevel(ctx context.Context, productID string, available, reserved, total int32) error {
	return r.q.CreateStockLevel(ctx, store.CreateStockLevelParams{
		ProductID: productID,
		Available: available,
		Reserved:  reserved,
		Total:     total,
	})
}

func (r *SQLRepository) GetProduct(ctx context.Context, id string) (*store.Product, error) {
	return r.q.GetProductByID(ctx, id)
}

func (r *SQLRepository) GetStockLevel(ctx context.Context, productID string) (*store.StockLevel, error) {
	return r.q.GetStockLevel(ctx, productID)
}

func (r *SQLRepository) UpdateStockLevel(ctx context.Context, productID string, available, reserved, total int32) error {
	return r.q.UpdateStockLevel(ctx, store.UpdateStockLevelParams{
		ProductID: productID,
		Available: available,
		Reserved:  reserved,
		Total:     total,
	})
}

func (r *SQLRepository) ReserveStock(ctx context.Context, reservationID, orderID, productID string, quantity int32, expiresAt sql.NullTime) error {
	return r.q.ReserveStock(ctx, store.ReserveStockParams{
		ID:        reservationID,
		OrderID:   orderID,
		ProductID: productID,
		Quantity:  quantity,
		Status:    "reserved",
		ExpiresAt: expiresAt,
	})
}

func (r *SQLRepository) GetReservation(ctx context.Context, id string) (*store.StockReservation, error) {
	return r.q.GetReservationByID(ctx, id)
}

func (r *SQLRepository) GetReservationsByOrderID(ctx context.Context, orderID string) ([]*store.StockReservation, error) {
	return r.q.GetReservationsByOrderID(ctx, orderID)
}

func (r *SQLRepository) UpdateReservationStatus(ctx context.Context, id, status string) error {
	return r.q.UpdateReservationStatus(ctx, store.UpdateReservationStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *SQLRepository) ListProducts(ctx context.Context, limit, offset int32) ([]*store.ListProductsRow, error) {
	return r.q.ListProducts(ctx, store.ListProductsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *SQLRepository) CountProducts(ctx context.Context) (int64, error) {
	return r.q.CountProducts(ctx)
}

func (r *SQLRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	txRepo := &SQLRepository{
		q:  r.q.WithTx(tx),
		db: r.db,
	}

	err = fn(txRepo)
	return err
}
```

### Step 7: Implement Stock Service

Edit `modules/stock/internal/service/service.go`:

```go
package service

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"go.jetify.com/typeid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	stockv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/stock/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/errors"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/telemetry"
	"github.com/cmelgarejo/go-modulith-template/modules/stock/internal/repository"
)

type StockService struct {
	stockv1.UnimplementedStockServiceServer
	repo repository.Repository
	bus  *events.Bus
}

func NewStockService(repo repository.Repository, bus *events.Bus) *StockService {
	return &StockService{
		repo: repo,
		bus:  bus,
	}
}

func (s *StockService) ReserveStock(ctx context.Context, req *stockv1.ReserveStockRequest) (*stockv1.ReserveStockResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "stock", "ReserveStock")
	defer span.End()

	telemetry.SetAttribute(ctx, "order_id", req.OrderId)
	telemetry.SetAttribute(ctx, "items_count", len(req.Items))

	// Validate stock availability and reserve in transaction
	var reservedItems []*stockv1.StockItem
	var reservationIDs []string

	err := s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		for _, item := range req.Items {
			// Check stock level
			stock, err := txRepo.GetStockLevel(ctx, item.ProductId)
			if err != nil {
				if err == sql.ErrNoRows {
					return errors.NotFound("product %s not found", item.ProductId)
				}
				return errors.Internal("failed to get stock level", errors.WithWrappedError(err))
			}

			// Check availability
			if stock.Available < item.Quantity {
				return errors.Validation("insufficient stock for product %s: available %d, requested %d",
					item.ProductId, stock.Available, item.Quantity)
			}

			// Create reservation
			reservationID, _ := typeid.WithPrefix("resv")
			reservationIDStr := reservationID.String()

			expiresAt := sql.NullTime{
				Time:  time.Now().Add(30 * time.Minute), // 30 min reservation window
				Valid: true,
			}

			if err := txRepo.ReserveStock(ctx, reservationIDStr, req.OrderId, item.ProductId, item.Quantity, expiresAt); err != nil {
				return errors.Internal("failed to reserve stock", errors.WithWrappedError(err))
			}

			// Update stock levels
			newAvailable := stock.Available - item.Quantity
			newReserved := stock.Reserved + item.Quantity

			if err := txRepo.UpdateStockLevel(ctx, item.ProductId, newAvailable, newReserved, stock.Total); err != nil {
				return errors.Internal("failed to update stock level", errors.WithWrappedError(err))
			}

			reservedItems = append(reservedItems, item)
			reservationIDs = append(reservationIDs, reservationIDStr)
		}

		return nil
	})

	if err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(err)
	}

	// Publish event
	s.bus.Publish(ctx, events.Event{
		Name: "stock.reserved",
		Payload: map[string]any{
			"order_id":         req.OrderId,
			"reservation_ids":  reservationIDs,
			"items":            reservedItems,
		},
	})

	return &stockv1.ReserveStockResponse{
		ReservationId: reservationIDs[0], // Return first reservation ID
		Success:        true,
		ReservedItems: reservedItems,
	}, nil
}

func (s *StockService) ReleaseStock(ctx context.Context, req *stockv1.ReleaseStockRequest) (*stockv1.ReleaseStockResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "stock", "ReleaseStock")
	defer span.End()

	// Get reservation
	reservation, err := s.repo.GetReservation(ctx, req.ReservationId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ToGRPC(errors.NotFound("reservation not found"))
		}
		return nil, errors.ToGRPC(errors.Internal("failed to get reservation", errors.WithWrappedError(err)))
	}

	// Release in transaction
	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		// Update reservation status
		if err := txRepo.UpdateReservationStatus(ctx, req.ReservationId, "released"); err != nil {
			return errors.Internal("failed to update reservation", errors.WithWrappedError(err))
		}

		// Get current stock level
		stock, err := txRepo.GetStockLevel(ctx, reservation.ProductID)
		if err != nil {
			return errors.Internal("failed to get stock level", errors.WithWrappedError(err))
		}

		// Restore available stock
		newAvailable := stock.Available + reservation.Quantity
		newReserved := stock.Reserved - reservation.Quantity
		if newReserved < 0 {
			newReserved = 0
		}

		return txRepo.UpdateStockLevel(ctx, reservation.ProductID, newAvailable, newReserved, stock.Total)
	})

	if err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(err)
	}

	return &stockv1.ReleaseStockResponse{Success: true}, nil
}

func (s *StockService) ConfirmReservation(ctx context.Context, req *stockv1.ConfirmReservationRequest) (*stockv1.ConfirmReservationResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "stock", "ConfirmReservation")
	defer span.End()

	// Get reservation
	reservation, err := s.repo.GetReservation(ctx, req.ReservationId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ToGRPC(errors.NotFound("reservation not found"))
		}
		return nil, errors.ToGRPC(errors.Internal("failed to get reservation", errors.WithWrappedError(err)))
	}

	// Confirm in transaction
	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		// Update reservation status
		if err := txRepo.UpdateReservationStatus(ctx, req.ReservationId, "confirmed"); err != nil {
			return errors.Internal("failed to confirm reservation", errors.WithWrappedError(err))
		}

		// Get current stock level
		stock, err := txRepo.GetStockLevel(ctx, reservation.ProductID)
		if err != nil {
			return errors.Internal("failed to get stock level", errors.WithWrappedError(err))
		}

		// Move from reserved to confirmed (reduce reserved, keep total same)
		newReserved := stock.Reserved - reservation.Quantity
		if newReserved < 0 {
			newReserved = 0
		}

		return txRepo.UpdateStockLevel(ctx, reservation.ProductID, stock.Available, newReserved, stock.Total)
	})

	if err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(err)
	}

	return &stockv1.ConfirmReservationResponse{Success: true}, nil
}

func (s *StockService) GetStockLevel(ctx context.Context, req *stockv1.GetStockLevelRequest) (*stockv1.GetStockLevelResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "stock", "GetStockLevel")
	defer span.End()

	stock, err := s.repo.GetStockLevel(ctx, req.ProductId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ToGRPC(errors.NotFound("product not found"))
		}
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to get stock level", errors.WithWrappedError(err)))
	}

	return &stockv1.GetStockLevelResponse{
		ProductId: req.ProductId,
		Available: stock.Available,
		Reserved:  stock.Reserved,
		Total:     stock.Total,
	}, nil
}

func (s *StockService) ListProducts(ctx context.Context, req *stockv1.ListProductsRequest) (*stockv1.ListProductsResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "stock", "ListProducts")
	defer span.End()

	pageSize := int32(20)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}

	page := int32(0)
	if req.Page > 0 {
		page = req.Page - 1
	}

	offset := page * pageSize

	products, err := s.repo.ListProducts(ctx, pageSize, offset)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to list products", errors.WithWrappedError(err)))
	}

	total, err := s.repo.CountProducts(ctx)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to count products", errors.WithWrappedError(err)))
	}

	var protoProducts []*stockv1.Product
	for _, p := range products {
		available := int32(0)
		reserved := int32(0)
		if p.AvailableStock.Valid {
			available = p.AvailableStock.Int32
		}
		if p.ReservedStock.Valid {
			reserved = p.ReservedStock.Int32
		}

		protoProducts = append(protoProducts, &stockv1.Product{
			Id:             p.ID,
			Name:           p.Name,
			Sku:            p.Sku,
			AvailableStock: available,
			ReservedStock:  reserved,
		})
	}

	return &stockv1.ListProductsResponse{
		Products: protoProducts,
		Total:     int32(total),
	}, nil
}
```

### Step 8: Scaffold the Order Module

```bash
make new-module order
```

### Step 9: Define Order Module Proto Contract

Edit `proto/order/v1/order.proto`:

```protobuf
syntax = "proto3";

package order.v1;

import "google/api/annotations.proto";
import "buf/validate/validate.proto";

option go_package = "github.com/cmelgarejo/go-modulith-template/gen/go/proto/order/v1;orderv1";

service OrderService {
  // Create a new order
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse) {
    option (google.api.http) = {
      post: "/v1/orders"
      body: "*"
    };
  }

  // Get order details
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse) {
    option (google.api.http) = {
      get: "/v1/orders/{order_id}"
    };
  }

  // Confirm order (after payment)
  rpc ConfirmOrder(ConfirmOrderRequest) returns (ConfirmOrderResponse) {
    option (google.api.http) = {
      post: "/v1/orders/{order_id}/confirm"
      body: "*"
    };
  }

  // Cancel order
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse) {
    option (google.api.http) = {
      post: "/v1/orders/{order_id}/cancel"
      body: "*"
    };
  }

  // List user orders
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse) {
    option (google.api.http) = {
      get: "/v1/orders"
    };
  }
}

message CreateOrderRequest {
  string user_id = 1 [(buf.validate.field).string.min_len = 1];
  repeated OrderItem items = 2 [(buf.validate.field).repeated.min_items = 1];
  string shipping_address = 3 [(buf.validate.field).string.min_len = 1];
}

message OrderItem {
  string product_id = 1 [(buf.validate.field).string.min_len = 1];
  int32 quantity = 2 [(buf.validate.field).int32.gt = 0];
  double price = 3 [(buf.validate.field).double.gt = 0];
}

message CreateOrderResponse {
  string order_id = 1;
  string status = 2;
  double total_amount = 3;
  string reservation_id = 4; // Stock reservation ID
}

message GetOrderRequest {
  string order_id = 1 [(buf.validate.field).string.min_len = 1];
}

message GetOrderResponse {
  Order order = 1;
}

message Order {
  string id = 1;
  string user_id = 2;
  string status = 3; // pending, confirmed, cancelled, fulfilled
  double total_amount = 4;
  string shipping_address = 5;
  string reservation_id = 6;
  string created_at = 7;
  repeated OrderItem items = 8;
}

message ConfirmOrderRequest {
  string order_id = 1 [(buf.validate.field).string.min_len = 1];
}

message ConfirmOrderResponse {
  bool success = 1;
}

message CancelOrderRequest {
  string order_id = 1 [(buf.validate.field).string.min_len = 1];
  string reason = 2;
}

message CancelOrderResponse {
  bool success = 1;
}

message ListOrdersRequest {
  string user_id = 1 [(buf.validate.field).string.min_len = 1];
  int32 page = 2;
  int32 page_size = 3;
}

message ListOrdersResponse {
  repeated Order orders = 1;
  int32 total = 2;
}
```

### Step 10: Create Order Database Schema

Edit `modules/order/resources/db/migration/000001_initial_schema.up.sql`:

```sql
-- Orders table
CREATE TABLE orders (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, confirmed, cancelled, fulfilled
  total_amount DECIMAL(10, 2) NOT NULL,
  shipping_address TEXT NOT NULL,
  reservation_id VARCHAR(64), -- Stock reservation ID
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Order items table
CREATE TABLE order_items (
  id VARCHAR(64) PRIMARY KEY,
  order_id VARCHAR(64) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id VARCHAR(64) NOT NULL,
  quantity INT NOT NULL,
  price DECIMAL(10, 2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
```

Create the down migration: `000001_initial_schema.down.sql`:

```sql
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
```

### Step 11: Create Order SQL Queries

Edit `modules/order/internal/db/query/order.sql`:

```sql
-- name: CreateOrder :exec
INSERT INTO orders (id, user_id, status, total_amount, shipping_address, reservation_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CreateOrderItem :exec
INSERT INTO order_items (id, order_id, product_id, quantity, price)
VALUES ($1, $2, $3, $4, $5);

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1 LIMIT 1;

-- name: GetOrderItems :many
SELECT * FROM order_items WHERE order_id = $1;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ListOrdersByUserID :many
SELECT * FROM orders WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOrdersByUserID :one
SELECT COUNT(*) FROM orders WHERE user_id = $1;
```

### Step 12: Generate Code Again

```bash
make generate-all
```

### Step 13: Implement Order Repository

Edit `modules/order/internal/repository/repository.go` (similar structure to stock repository).

### Step 14: Implement Order Service with Stock Communication

Edit `modules/order/internal/service/service.go`:

```go
package service

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"go.jetify.com/typeid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/order/v1"
	stockv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/stock/v1"
	"github.com/cmelgarejo/go-modulith-template/internal/errors"
	"github.com/cmelgarejo/go-modulith-template/internal/events"
	"github.com/cmelgarejo/go-modulith-template/internal/telemetry"
	"github.com/cmelgarejo/go-modulith-template/modules/order/internal/repository"
)

type OrderService struct {
	orderv1.UnimplementedOrderServiceServer
	repo       repository.Repository
	bus        *events.Bus
	stockClient stockv1.StockServiceClient
	grpcConn   *grpc.ClientConn
}

func NewOrderService(repo repository.Repository, bus *events.Bus, grpcAddr string) (*OrderService, error) {
	// Create gRPC client for Stock module
	// In modulith: connects to in-process server (127.0.0.1:9000 by default)
	// In microservices: connects to stock-service:9000
	// Note: gRPC port is configurable via GRPC_PORT or configs/server.yaml
	conn, err := grpc.NewClient(
		grpcAddr, // "127.0.0.1:9000" for modulith (default, configurable)
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stock client: %w", err)
	}

	return &OrderService{
		repo:        repo,
		bus:         bus,
		stockClient: stockv1.NewStockServiceClient(conn),
		grpcConn:    conn,
	}, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "order", "CreateOrder")
	defer span.End()

	telemetry.SetAttribute(ctx, "user_id", req.UserId)
	telemetry.SetAttribute(ctx, "items_count", len(req.Items))

	// Generate order ID
	orderID, _ := typeid.WithPrefix("order")
	orderIDStr := orderID.String()

	// Calculate total
	var totalAmount float64
	var stockItems []*stockv1.StockItem
	for _, item := range req.Items {
		totalAmount += item.Price * float64(item.Quantity)
		stockItems = append(stockItems, &stockv1.StockItem{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		})
	}

	// Reserve stock via gRPC call to Stock module
	reserveReq := &stockv1.ReserveStockRequest{
		OrderId: orderIDStr,
		Items:   stockItems,
	}

	reserveResp, err := s.stockClient.ReserveStock(ctx, reserveReq)
	if err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to reserve stock", errors.WithWrappedError(err)))
	}

	if !reserveResp.Success {
		return nil, errors.ToGRPC(errors.Validation("stock reservation failed"))
	}

	// Create order in transaction
	err = s.repo.WithTx(ctx, func(txRepo repository.Repository) error {
		// Create order
		if err := txRepo.CreateOrder(ctx, orderIDStr, req.UserId, "pending", totalAmount, req.ShippingAddress, reserveResp.ReservationId); err != nil {
			return errors.Internal("failed to create order", errors.WithWrappedError(err))
		}

		// Create order items
		for _, item := range req.Items {
			itemID, _ := typeid.WithPrefix("item")
			if err := txRepo.CreateOrderItem(ctx, itemID.String(), orderIDStr, item.ProductId, item.Quantity, item.Price); err != nil {
				return errors.Internal("failed to create order item", errors.WithWrappedError(err))
			}
		}

		return nil
	})

	if err != nil {
		// Release stock reservation on failure
		_, _ = s.stockClient.ReleaseStock(ctx, &stockv1.ReleaseStockRequest{
			ReservationId: reserveResp.ReservationId,
		})

		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(err)
	}

	// Publish event
	s.bus.Publish(ctx, events.Event{
		Name: "order.created",
		Payload: map[string]any{
			"order_id":        orderIDStr,
			"user_id":         req.UserId,
			"total_amount":    totalAmount,
			"reservation_id":  reserveResp.ReservationId,
		},
	})

	return &orderv1.CreateOrderResponse{
		OrderId:       orderIDStr,
		Status:        "pending",
		TotalAmount:   totalAmount,
		ReservationId: reserveResp.ReservationId,
	}, nil
}

func (s *OrderService) ConfirmOrder(ctx context.Context, req *orderv1.ConfirmOrderRequest) (*orderv1.ConfirmOrderResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "order", "ConfirmOrder")
	defer span.End()

	// Get order
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ToGRPC(errors.NotFound("order not found"))
		}
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to get order", errors.WithWrappedError(err)))
	}

	if order.Status != "pending" {
		return nil, errors.ToGRPC(errors.Validation("order is not in pending status"))
	}

	// Confirm stock reservation
	if order.ReservationID.Valid {
		_, err = s.stockClient.ConfirmReservation(ctx, &stockv1.ConfirmReservationRequest{
			ReservationId: order.ReservationID.String,
		})
		if err != nil {
			telemetry.RecordError(span, err)
			return nil, errors.ToGRPC(errors.Internal("failed to confirm stock reservation", errors.WithWrappedError(err)))
		}
	}

	// Update order status
	if err := s.repo.UpdateOrderStatus(ctx, req.OrderId, "confirmed"); err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to update order status", errors.WithWrappedError(err)))
	}

	// Publish event
	s.bus.Publish(ctx, events.Event{
		Name: "order.confirmed",
		Payload: map[string]any{
			"order_id": req.OrderId,
			"user_id":  order.UserID,
		},
	})

	return &orderv1.ConfirmOrderResponse{Success: true}, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	ctx, span := telemetry.ServiceSpan(ctx, "order", "CancelOrder")
	defer span.End()

	// Get order
	order, err := s.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ToGRPC(errors.NotFound("order not found"))
		}
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to get order", errors.WithWrappedError(err)))
	}

	if order.Status == "cancelled" {
		return &orderv1.CancelOrderResponse{Success: true}, nil
	}

	// Release stock reservation if order is pending
	if order.Status == "pending" && order.ReservationID.Valid {
		_, err = s.stockClient.ReleaseStock(ctx, &stockv1.ReleaseStockRequest{
			ReservationId: order.ReservationID.String,
		})
		if err != nil {
			slog.WarnContext(ctx, "failed to release stock on cancel", "error", err)
			// Continue with cancellation even if stock release fails
		}
	}

	// Update order status
	if err := s.repo.UpdateOrderStatus(ctx, req.OrderId, "cancelled"); err != nil {
		telemetry.RecordError(span, err)
		return nil, errors.ToGRPC(errors.Internal("failed to cancel order", errors.WithWrappedError(err)))
	}

	// Publish event
	s.bus.Publish(ctx, events.Event{
		Name: "order.cancelled",
		Payload: map[string]any{
			"order_id": req.OrderId,
			"user_id":  order.UserID,
			"reason":   req.Reason,
		},
	})

	return &orderv1.CancelOrderResponse{Success: true}, nil
}

// ... (implement GetOrder and ListOrders similarly)
```

### Step 15: Update Order Module to Accept gRPC Address

Edit `modules/order/module.go`:

```go
// Initialize sets up the order module
func (m *Module) Initialize(r *registry.Registry) error {
	cfg, ok := r.Config().(*config.AppConfig)
	if !ok {
		return fmt.Errorf("invalid config type")
	}

	// Get gRPC address from config (defaults to localhost for modulith)
	grpcAddr := cfg.GRPCPort
	if grpcAddr == "" {
		grpcAddr = "127.0.0.1:9000"  // Default port (configurable via GRPC_PORT or configs/server.yaml)
	} else {
		grpcAddr = "127.0.0.1:" + grpcAddr
	}

	repo := repository.NewSQLRepository(r.DB())
	svc, err := service.NewOrderService(repo, r.EventBus(), grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to create order service: %w", err)
	}

	m.svc = svc
	return nil
}
```

### Step 16: Register Both Modules

Edit `cmd/server/setup/registry.go`:

```go
func RegisterModules(reg *registry.Registry) {
	reg.Register(auth.NewModule())
	reg.Register(stock.NewModule())
	reg.Register(order.NewModule())
}
```

### Step 17: Run Migrations and Start Server

```bash
# Start database
make docker-up-minimal

# Run migrations
make migrate-up

# Start server
make dev
```

### Step 18: Test the System

```bash
# Create a product (via Stock API)
curl -X POST http://localhost:8080/v1/stock/products \
  -H "Content-Type: application/json" \
  -d '{
    "id": "prod_01h...",
    "name": "Laptop",
    "sku": "LAP-001",
    "price": 999.99
  }'

# Add stock
curl -X POST http://localhost:8080/v1/stock/levels \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "prod_01h...",
    "available": 10,
    "total": 10
  }'

# Create order (calls Stock module internally)
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_01h...",
    "items": [
      {
        "product_id": "prod_01h...",
        "quantity": 2,
        "price": 999.99
      }
    ],
    "shipping_address": "123 Main St"
  }'
```

## Communication Flow

### Order Creation Flow

```
1. Client → Order Service (CreateOrder)
   │
   ├─→ Calculate total amount
   │
   ├─→ gRPC Call → Stock Service (ReserveStock)
   │   │
   │   ├─→ Check stock availability
   │   ├─→ Create reservation
   │   ├─→ Update stock levels (available - quantity, reserved + quantity)
   │   └─→ Return reservation_id
   │
   ├─→ Create order in database
   ├─→ Create order items
   │
   └─→ Publish "order.created" event
```

### Order Confirmation Flow

```
1. Client → Order Service (ConfirmOrder)
   │
   ├─→ Get order from database
   │
   ├─→ gRPC Call → Stock Service (ConfirmReservation)
   │   │
   │   ├─→ Update reservation status to "confirmed"
   │   └─→ Update stock levels (reserved - quantity)
   │
   ├─→ Update order status to "confirmed"
   │
   └─→ Publish "order.confirmed" event
```

### Order Cancellation Flow

```
1. Client → Order Service (CancelOrder)
   │
   ├─→ Get order from database
   │
   ├─→ gRPC Call → Stock Service (ReleaseStock)
   │   │
   │   ├─→ Update reservation status to "released"
   │   └─→ Restore stock (available + quantity, reserved - quantity)
   │
   ├─→ Update order status to "cancelled"
   │
   └─→ Publish "order.cancelled" event
```

## Key Concepts Demonstrated

1. **Module Isolation**: Each module has its own database schema, proto definitions, and business logic
2. **gRPC Communication**: Order module calls Stock module via gRPC (in-process in modulith, network in microservices)
3. **Transaction Management**: Stock reservations use database transactions for consistency
4. **Event-Driven**: Events published for order lifecycle (created, confirmed, cancelled)
5. **Error Handling**: Proper error mapping using template's error system
6. **Telemetry**: Automatic tracing and metrics via OpenTelemetry

## Next Steps

1. Add more business logic (payment processing, shipping, etc.)
2. Implement event handlers for cross-module reactions
3. Add integration tests
4. Deploy as microservices (update gRPC addresses)
5. Add distributed event bus (Kafka, RabbitMQ) for microservices

## Troubleshooting

### Module not found

-   Ensure modules are registered in `cmd/server/setup/registry.go`
-   Run `make generate-all` after proto changes

### gRPC connection errors

-   Check gRPC port in config (`GRPC_PORT`)
-   In modulith: use `127.0.0.1:9000` (default, configurable via `GRPC_PORT` or `configs/server.yaml`)
-   In microservices: use service discovery address

### Migration errors

-   Ensure migrations are in correct order
-   Check database connection string

### Stock reservation fails

-   Verify product exists and has sufficient stock
-   Check reservation expiration (30 minutes default)
