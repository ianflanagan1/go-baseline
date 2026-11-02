package ratelib

type Clock interface{ Now() int }

func Late(c Clock) bool { return c.Now() > 10 }
