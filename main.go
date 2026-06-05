package main

import (
	"os"

	"github.com/IKauedev/duck/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
