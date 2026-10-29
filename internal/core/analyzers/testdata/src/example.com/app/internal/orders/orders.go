package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"example.com/app/internal/core/pricing"
)

type Store interface {
	Status(ctx context.Context, id string) (int, error)
	Items(ctx context.Context) ([]string, error)
	Raw(ctx context.Context) ([]byte, error)
	Save(ctx context.Context, item string) error
}

type Sleeper interface{ Sleep(ctx context.Context, seconds int) }

var errMissing = errors.New("missing")

func Statuses(ctx context.Context, store Store) error { // want Statuses:"returns a port value"
	status, err := store.Status(ctx, "a")
	if err != nil {
		return fmt.Errorf("reading the status: %w", err)
	}
	if status == 429 { // want `the shell branches on a value read through a port`
		return nil
	}
	switch pricing.Decide(status) {
	case pricing.Retry:
		return store.Save(ctx, "retry")
	case pricing.Done:
	}
	if pricing.Allowed(status) && status > 0 { // want `the shell branches on a value read through a port`
		return nil
	}
	return nil
}

func Errors(ctx context.Context, store Store) error { // want Errors:"returns a port value"
	_, err := store.Status(ctx, "a")
	if errors.Is(err, errMissing) { // want `the shell branches on a value read through a port`
		return nil
	}
	if err == errMissing { // want `the shell branches on a value read through a port`
		return nil
	}
	return err
}

func Ranges(ctx context.Context, store Store) error { // want Ranges:"returns a port value"
	items, err := store.Items(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := store.Save(ctx, pricing.Label(item)); err != nil {
			return err
		}
	}
	for _, item := range items {
		if strings.HasPrefix(item, "x") { // want `the shell branches on a value read through a port`
			return nil
		}
	}
	if len(items) > 3 { // want `the shell branches on a value read through a port`
		return nil
	}
	return nil
}

func Decodes(ctx context.Context, store Store) error { // want Decodes:"returns a port value"
	raw, err := store.Raw(ctx)
	if err != nil {
		return err
	}
	var body struct{ Status int }
	if err := json.Unmarshal(raw, &body); err != nil {
		return err
	}
	if body.Status > 0 { // want `the shell branches on a value read through a port`
		return nil
	}
	return nil
}

func Chooses(ctx context.Context, store Store, handlers map[int]func()) { // want Chooses:"branches on parameter 2"
	status, _ := store.Status(ctx, "a")
	handlers[status]() // want `the shell calls a function chosen by a value read through a port`
	handlers[1]()
}

func Acts(ctx context.Context, store Store, sleeper Sleeper) {
	status, _ := store.Status(ctx, "a")
	plan := pricing.Plan(status)
	if plan.Wait() > 0 {
		sleeper.Sleep(ctx, plan.Wait())
	}
}

func Closures(ctx context.Context, store Store) {
	status, _ := store.Status(ctx, "a")
	func() {
		if status > 0 { // want `the shell branches on a value read through a port`
			return
		}
	}()
}

func Waits(ctx context.Context, results <-chan int) int { // want Waits:"branches on parameter 1; returns parameter 1"
	select {
	case <-ctx.Done():
		return 0
	case r := <-results:
		if r > 0 {
			return r
		}
	}
	return 0
}

func Fetch(ctx context.Context, store Store) int { // want Fetch:"returns a port value"
	status, _ := store.Status(ctx, "a")
	return status
}

func Check(status int) bool { // want Check:"branches on parameter 0"
	if status > 0 {
		return true
	}
	return false
}

func Same(status int) int { return status } // want Same:"returns parameter 0"

func Decided(status int) bool { return pricing.Allowed(status) }

func Helpers(ctx context.Context, store Store) {
	if Fetch(ctx, store) > 0 { // want `the shell branches on a value read through a port`
		return
	}
	Check(Fetch(ctx, store)) // want `Check branches on a value read through a port, passed to it here`
	Check(1)
	if Same(Fetch(ctx, store)) > 0 { // want `the shell branches on a value read through a port`
		return
	}
	if Same(1) > 0 {
		return
	}
	if Decided(Fetch(ctx, store)) {
		return
	}
}

func Parameters(status int) bool { // want Parameters:"returns parameter 0"
	return status > 0
}
