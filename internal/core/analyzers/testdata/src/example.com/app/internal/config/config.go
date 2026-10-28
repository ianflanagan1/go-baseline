package config

import (
	"flag"
	"os"

	"github.com/ianflanagan1/go-adapters/env"
)

func Read() (string, bool) {
	flag.Parse() // want `flag.Parse reads configuration`
	_ = flag.NewFlagSet("x", flag.ContinueOnError)
	_ = os.Args                  // want `os.Args reads configuration`
	_ = env.System{}             // want `the env adapter reads configuration`
	return os.LookupEnv("TOKEN") // want `os.LookupEnv reads configuration`
}
