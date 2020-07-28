package main

import (
	"fmt"
	"os"

	"github.com/roman-kulish/spatio-temporal-deduplication-example/cmd/example/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
