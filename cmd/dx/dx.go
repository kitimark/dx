package main

import (
	"log/slog"
	"os"

	"github.com/kitimark/dx"
)

func main() {
	cmd := dx.NewMainCmd()
	err := cmd.Execute()
	if err != nil {
		slog.Info(err.Error())
		os.Exit(2)
	}
}
