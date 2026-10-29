package pricing

type Next int

const (
	Done Next = iota
	Retry
)

// Decide is the core deciding what a status means.
func Decide(status int) Next {
	if status == 429 {
		return Retry
	}
	return Done
}

func Allowed(status int) bool { return status < 400 }

func Label(item string) string { return "item " + item }

// Decision is a sum type the core returns; its methods are the core's.
type Decision interface{ Wait() int }

type Wait struct{ Seconds int }

func (w Wait) Wait() int { return w.Seconds }

func Plan(status int) Decision { return Wait{status} }
