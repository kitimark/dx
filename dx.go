package dx

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var Main = &cobra.Command{
	Use:               "dx [flags] [command]",
	PersistentPreRunE: cmdMainPreRun,
}

func init() {
	Main.PersistentFlags().BoolP("debug", "d", false, "print debug info")
	Main.AddCommand(CmdCommit)
	Main.AddCommand(CmdSync)
}

func cmdMainPreRun(cmd *cobra.Command, _ []string) error {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return err
	}
	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	err = gitInit()
	if err != nil {
		return err
	}
	return nil
}
