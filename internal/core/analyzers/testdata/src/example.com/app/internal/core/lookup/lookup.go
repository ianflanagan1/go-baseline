// Package lookup holds the core side of the guide's examples, which the
// analyzers must accept.
package lookup

import (
	"errors"
	"time"
)

type Order struct{ ID int64 }

type Reply struct{ Status int }

func Show(order Order, found bool) Reply {
	if !found {
		return Reply{Status: 404}
	}
	return Reply{Status: 200}
}

type Request struct{ URL string }

type Response struct {
	Status int
	Body   []byte
}

// Step is one of Retry, Done and Fail.
type Step interface{ isStep() }

type Retry struct {
	Request Request
	Wait    time.Duration
}

type Done struct{ Rate float64 }

type Fail struct{ Err error }

func (Retry) isStep() {}
func (Done) isStep()  {}
func (Fail) isStep()  {}

func Next(req Request, resp Response, attempt int) Step {
	if resp.Status == 429 && attempt < 3 {
		return Retry{Request: req, Wait: time.Duration(attempt+1) * time.Second}
	}
	if resp.Status != 200 {
		return Fail{Err: errors.New("no rate")}
	}
	return Done{Rate: float64(len(resp.Body))}
}
