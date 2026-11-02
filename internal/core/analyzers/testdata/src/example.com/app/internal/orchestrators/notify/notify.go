package notify

import (
	"context"

	"example.com/app/internal/orchestrators/orders"
)

func Notify(ctx context.Context, store orders.Store) {
	orders.Check(orders.Fetch(ctx, store)) // want `Check branches on a value read through a port, passed to it here`
	if orders.Fetch(ctx, store) > 0 { // want `the orchestrator branches on a value read through a port`
		return
	}
	if orders.Decided(orders.Fetch(ctx, store)) {
		return
	}
}
