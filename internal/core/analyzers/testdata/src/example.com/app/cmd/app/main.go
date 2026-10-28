package main

import (
	"os"

	"example.com/app/internal/app"
	"example.com/app/internal/core/rates"
	"example.com/sdk"
)

func main() {
	_ = os.Getenv("TOKEN")
	app.Run(nil, sdk.Client{})
	rates.Late(sdk.Client{}) // want `example.com/sdk.Client may do I/O`
}
