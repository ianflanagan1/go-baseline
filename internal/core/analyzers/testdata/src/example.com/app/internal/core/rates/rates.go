package rates

import "os"

type Clock interface{ Now() int }

type Shape interface{ Area() int }

type Square struct{ Side int }

func (s Square) Area() int { return s.Side * s.Side }

type Service struct{ Clock Clock }

func Late(c Clock) bool { return c.Now() > 10 }

func Area(s Shape) int { return s.Area() }

func Total(shapes ...Shape) int {
	total := 0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

func Apply(f func(int) int, x int) int { return f(x) }

func Double(x int) int { return x * 2 }

func Describe(err error) string { return err.Error() }

func Show(v any) string { _ = v; return "" }

func Larger[T Shape](a, b T) T {
	if a.Area() > b.Area() {
		return a
	}
	return b
}

// depguard also bans os here.
func home() string { return os.Getenv("HOME") } // want `os.Getenv reads configuration`
