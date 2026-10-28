package app

import (
	"strings"

	"example.com/app/internal/core/rates"
	"example.com/ratelib"
	"example.com/sdk"
)

type Clock interface{ Now() int }

type shellShape struct{}

func (shellShape) Area() int { return 1 }

func Run(clock Clock, client sdk.Client) {
	rates.Late(clock)  // want `concrete type isn't known`
	rates.Late(client) // want `example.com/sdk.Client may do I/O`
	rates.Area(rates.Square{Side: 2})
	rates.Area(shellShape{}) // want `shellShape may do I/O`
	rates.Apply(rates.Double, 1)
	rates.Apply(func(x int) int { return x }, 1) // want `declared outside the core`
	n := 2
	rates.Apply(func(x int) int { return x * n }, 1) // want `closure made outside the core`
	rates.Describe(nil)
	rates.Show(3)
	rates.Show(&strings.Builder{})
	rates.Larger(rates.Square{}, rates.Square{})
	rates.Larger(shellShape{}, shellShape{}) // want `type argument .*shellShape`
}

func Variadic(clock Clock, shapes []rates.Shape) {
	rates.Total(rates.Square{}, rates.Square{})
	rates.Total(rates.Square{}, shellShape{}) // want `shellShape may do I/O`
	rates.Total(shapes...)                    // want `slice of .*Shape made outside the core`
	rates.Total()
	ratelib.Late(clock) // want `concrete type isn't known`
}

func NewService(clock Clock) rates.Service {
	return rates.Service{Clock: clock} // want `concrete type isn't known`
}
