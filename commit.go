package dx

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var CmdCommit = &cobra.Command{
	Use:     "commit [flags]",
	Example: "commit -m \"commit message\"",
	Args:    cobra.NoArgs,
	RunE:    cmdCommitRun,
}

func init() {
	CmdCommit.PersistentFlags().StringP("message", "m", "", "message")
	err := CmdCommit.MarkPersistentFlagRequired("message")
	if err != nil {
		panic(err)
	}
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
	_, err = execOutputErr("git", args...)
	return err
}
