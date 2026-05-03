# Chapter 80 Exercises — Event-Driven Architecture

## Exercise 1 — Order Saga (`exercises/01_order_saga`)

Build a choreography-based saga for an order fulfillment flow using an in-process event bus.

### Services

**`InventoryService`**:
- Listens for `order.placed` → reserves stock → publishes `inventory.reserved` or `inventory.failed`
- Listens for `order.cancelled` → releases reserved stock (compensation)

**`PaymentService`**:
- Listens for `inventory.reserved` → charges card → publishes `payment.charged` or `payment.failed`

**`OrderService`**:
- `PlaceOrder(ctx, orderID, customerID, items, total)` publishes `order.placed`
- Listens for `payment.charged` → marks order `confirmed`, publishes `order.confirmed`
- Listens for `inventory.failed` or `payment.failed` → marks order `cancelled`, publishes `order.cancelled`

**`NotificationService`**:
- Listens for `order.confirmed` → sends confirmation notification
- Listens for `order.cancelled` → sends cancellation notification

### Scenarios to demonstrate

1. **Happy path**: inventory has stock, payment succeeds → order confirmed, notification sent
2. **Payment failure**: inventory reserved, but payment card declined → order cancelled, inventory released (compensation)
3. **Inventory failure**: stock insufficient → order cancelled immediately (no compensation needed)

### Behaviour rules

- Services must not hold a mutex while calling `bus.Publish` — release the lock first
- Compensation only applies if the step completed (e.g. no inventory to release if reservation never happened)
- `OrderService.State(orderID)` returns the current `*OrderState`

### Hints

- Use a `map[string]bool` of `failOrders` in `PaymentService` for configurable failure injection
- The event payload type determines which failure handler fires — use a type switch
- Track `reserved map[string]int` in `InventoryService` so compensation knows how much to restore
