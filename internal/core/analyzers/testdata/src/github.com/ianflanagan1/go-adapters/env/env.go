package env

import "os"

type System struct{}

func Lookup() string { return os.Getenv("X") }
