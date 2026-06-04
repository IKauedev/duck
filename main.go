package main

import (
	"os"

	"duck/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
