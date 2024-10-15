package dx

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func NewMainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "dx [flags] [command]",
		PersistentPreRunE: cmdMainPreRun,
	}

	cmd.PersistentFlags().BoolP("debug", "d", false, "print debug info")

	cmd.AddCommand(NewCommitCmd())
	cmd.AddCommand(NewSyncCmd())
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewResolveConflictCmd())

	return cmd
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
