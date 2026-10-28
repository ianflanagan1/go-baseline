package main

import (
	"flag"
	"os"

	"github.com/ianflanagan1/go-adapters/env"
)

func main() {
	flag.Parse()
	_ = os.Getenv("TOKEN")
	_ = env.System{}
}
