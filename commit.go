package dx

import (
	"fmt"
	"log/slog"

	"github.com/kitimark/dx/pkg/exec"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func NewCommitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "commit [flags]",
		Example: "commit -m \"commit message\"",
		Args:    cobra.NoArgs,
		RunE:    cmdCommitRun,
	}

	cmd.PersistentFlags().StringP("message", "m", "", "message")
	err := cmd.MarkPersistentFlagRequired("message")
	if err != nil {
		panic(err)
	}

	return cmd
}

func cmdCommitRun(cmd *cobra.Command, _ []string) error {
	flags := cmd.Flags()
	message, err := flags.GetString("message")
	if err != nil {
		return err
	}
	slog.Info("args", slog.String("message", message))
	changeId := bson.NewObjectID().Hex()
	changeIdMessage := fmt.Sprintf("change-id: %s", changeId)
	args := []string{"commit", "-m", message, "-m", changeIdMessage}
	_, err = exec.OutputErr("git", args...)
	return err
}
