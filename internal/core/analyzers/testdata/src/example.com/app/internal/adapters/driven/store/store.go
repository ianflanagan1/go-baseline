package store

import "os"

func Path() string { return os.Getenv("STORE_PATH") } // want `os.Getenv reads configuration`
