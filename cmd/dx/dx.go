package main

import (
	"log/slog"
	"os"

	"github.com/kitimark/dx"
)

func main() {
	err := dx.Main.Execute()
	if err != nil {
		slog.Info(err.Error())
		os.Exit(2)
	}
}
