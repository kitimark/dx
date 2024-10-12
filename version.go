package dx

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"runtime/debug"
)

func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "version",
		RunE: cmdVersionRun,
	}
	return cmd
}

func cmdVersionRun(_ *cobra.Command, args []string) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return errors.New("error during get version")
	}
	fmt.Println(info.Main.Version)
	return nil
}
