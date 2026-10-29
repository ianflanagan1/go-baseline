// Package guide is the shell side of the guide's examples, which the
// analyzers must accept.
package guide

import (
	"context"
	"fmt"
	"time"

	"example.com/app/internal/core/lookup"
)

type Orders interface {
	ByID(ctx context.Context, id int64) (order lookup.Order, found bool, err error)
}

func Show(ctx context.Context, store Orders, id int64) (lookup.Reply, error) {
	order, found, err := store.ByID(ctx, id)
	if err != nil {
		return lookup.Reply{}, fmt.Errorf("showing order %d: %w", id, err)
	}
	return lookup.Show(order, found), nil
}

type Sender interface {
	Send(ctx context.Context, req lookup.Request) (lookup.Response, error)
}

type Sleeper interface {
	Sleep(ctx context.Context, d time.Duration)
}

func Latest(ctx context.Context, sender Sender, sleeper Sleeper, req lookup.Request) (float64, error) {
	for attempt := 0; ; attempt++ {
		resp, err := sender.Send(ctx, req)
		if err != nil {
			return 0, fmt.Errorf("fetching the rate: %w", err)
		}
		switch step := lookup.Next(req, resp, attempt).(type) {
		case lookup.Retry:
			sleeper.Sleep(ctx, step.Wait)
			req = step.Request
		case lookup.Done:
			return step.Rate, nil
		case lookup.Fail:
			return 0, fmt.Errorf("fetching the rate: %w", step.Err)
		}
	}
}
