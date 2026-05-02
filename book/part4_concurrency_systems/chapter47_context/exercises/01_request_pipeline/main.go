// FILE: book/part4_concurrency_systems/chapter47_context/exercises/01_request_pipeline/main.go
// CHAPTER: 47 — context Package
// EXERCISE: Multi-stage request pipeline that propagates context cancellation,
//           carries request metadata via WithValue, and enforces per-stage timeouts.
//
// Run (from the chapter folder):
//   go run ./exercises/01_request_pipeline

package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEYS
// ─────────────────────────────────────────────────────────────────────────────

type ctxKey string

const (
	keyReqID  ctxKey = "req_id"
	keyUserID ctxKey = "user_id"
	keyRole   ctxKey = "role"
)

func withMeta(ctx context.Context, reqID string, userID int64, role string) context.Context {
	ctx = context.WithValue(ctx, keyReqID, reqID)
	ctx = context.WithValue(ctx, keyUserID, userID)
	ctx = context.WithValue(ctx, keyRole, role)
	return ctx
}

func meta(ctx context.Context) (reqID string, userID int64, role string) {
	reqID, _ = ctx.Value(keyReqID).(string)
	userID, _ = ctx.Value(keyUserID).(int64)
	role, _ = ctx.Value(keyRole).(string)
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// PIPELINE STAGES
// ─────────────────────────────────────────────────────────────────────────────

type OrderRequest struct {
	UserID    int64
	ProductID int
	Qty       int
}

type OrderResponse struct {
	OrderID   string
	Total     float64
	Confirmed bool
}

// authenticate verifies the user has a valid session.
func authenticate(ctx context.Context, req OrderRequest) error {
	rid, uid, role := meta(ctx)
	select {
	case <-time.After(15 * time.Millisecond):
		if uid != req.UserID {
			return fmt.Errorf("user mismatch: ctx=%d req=%d", uid, req.UserID)
		}
		fmt.Printf("  [%s] auth ok: user=%d role=%s\n", rid, uid, role)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("authenticate: %w", ctx.Err())
	}
}

// checkInventory verifies stock is available.
func checkInventory(ctx context.Context, req OrderRequest) error {
	rid, _, _ := meta(ctx)
	select {
	case <-time.After(20 * time.Millisecond):
		if req.ProductID == 999 {
			return errors.New("product 999 out of stock")
		}
		fmt.Printf("  [%s] inventory ok: product=%d qty=%d\n", rid, req.ProductID, req.Qty)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("checkInventory: %w", ctx.Err())
	}
}

// calculatePrice computes the total.
func calculatePrice(ctx context.Context, req OrderRequest) (float64, error) {
	rid, _, _ := meta(ctx)
	select {
	case <-time.After(10 * time.Millisecond):
		price := float64(req.ProductID)*0.99 + float64(req.Qty)*2.5
		fmt.Printf("  [%s] price: %.2f\n", rid, price)
		return price, nil
	case <-ctx.Done():
		return 0, fmt.Errorf("calculatePrice: %w", ctx.Err())
	}
}

// createOrder persists the order.
func createOrder(ctx context.Context, req OrderRequest, total float64) (OrderResponse, error) {
	rid, _, _ := meta(ctx)
	select {
	case <-time.After(25 * time.Millisecond):
		orderID := fmt.Sprintf("ORD-%d-%d", req.UserID, time.Now().UnixMilli()%10000)
		fmt.Printf("  [%s] order created: %s total=%.2f\n", rid, orderID, total)
		return OrderResponse{OrderID: orderID, Total: total, Confirmed: true}, nil
	case <-ctx.Done():
		return OrderResponse{}, fmt.Errorf("createOrder: %w", ctx.Err())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ORCHESTRATOR
// ─────────────────────────────────────────────────────────────────────────────

func placeOrder(ctx context.Context, req OrderRequest) (OrderResponse, error) {
	if err := authenticate(ctx, req); err != nil {
		return OrderResponse{}, err
	}
	if err := checkInventory(ctx, req); err != nil {
		return OrderResponse{}, err
	}
	total, err := calculatePrice(ctx, req)
	if err != nil {
		return OrderResponse{}, err
	}
	return createOrder(ctx, req, total)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	// Scenario 1: normal request — all stages complete within timeout.
	fmt.Println("=== Scenario 1: happy path ===")
	ctx := context.Background()
	ctx = withMeta(ctx, "req-001", 42, "customer")
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	resp, err := placeOrder(ctx, OrderRequest{UserID: 42, ProductID: 10, Qty: 3})
	if err != nil {
		fmt.Printf("  error: %v\n", err)
	} else {
		fmt.Printf("  result: order=%s total=%.2f confirmed=%v\n",
			resp.OrderID, resp.Total, resp.Confirmed)
	}

	// Scenario 2: timeout fires during pipeline.
	fmt.Println()
	fmt.Println("=== Scenario 2: timeout during pipeline ===")
	ctx2 := context.Background()
	ctx2 = withMeta(ctx2, "req-002", 7, "customer")
	ctx2, cancel2 := context.WithTimeout(ctx2, 30*time.Millisecond) // too short
	defer cancel2()

	_, err = placeOrder(ctx2, OrderRequest{UserID: 7, ProductID: 5, Qty: 1})
	fmt.Printf("  error (expected): %v\n", err)

	// Scenario 3: business logic error (out of stock).
	fmt.Println()
	fmt.Println("=== Scenario 3: inventory error ===")
	ctx3 := context.Background()
	ctx3 = withMeta(ctx3, "req-003", 99, "customer")
	ctx3, cancel3 := context.WithTimeout(ctx3, 500*time.Millisecond)
	defer cancel3()

	_, err = placeOrder(ctx3, OrderRequest{UserID: 99, ProductID: 999, Qty: 1})
	fmt.Printf("  error (expected): %v\n", err)

	// Scenario 4: manual cancellation (e.g., user disconnects).
	fmt.Println()
	fmt.Println("=== Scenario 4: manual cancel ===")
	ctx4, cancel4 := context.WithCancel(context.Background())
	ctx4 = withMeta(ctx4, "req-004", 11, "admin")

	go func() {
		time.Sleep(20 * time.Millisecond) // cancel mid-pipeline
		cancel4()
	}()

	_, err = placeOrder(ctx4, OrderRequest{UserID: 11, ProductID: 20, Qty: 2})
	fmt.Printf("  error (expected): %v\n", err)
}
