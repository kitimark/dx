package dx

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var CmdSync = &cobra.Command{
	Use:     "sync [flags] <branch>",
	Example: "sync dev",
	Args:    cobra.ExactArgs(1),
	RunE:    cmdSyncRun,
}

func cmdSyncRun(_ *cobra.Command, args []string) error {
	syncBranch := args[0]
	currentBranchName, err := execOutputErr("git", "rev-parse", "--abbrev-ref", "HEAD")
	defer func() {
		out, err := execOutputErr("git", "checkout", currentBranchName)
		if err != nil {
			slog.Warn("cannot checkout branch", "branch", currentBranchName, "output", out, "error", err)
		}
	}()
	if err != nil {
		return err
	}
	currentBranchName = strings.TrimRight(currentBranchName, "\n")

	if currentBranchName == syncBranch {
		slog.Error("cannot sync branch with same branch", "current_branch", currentBranchName,
			"sync branch", syncBranch)
		return errors.New("cannot sync branch with same branch")
	}

	slog.Info("try to pull the sync branch", "branch", syncBranch)
	_, err = execOutputErr("git", "fetch")
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "checkout", syncBranch)
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "reset", "--hard", "origin/"+syncBranch)
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "checkout", currentBranchName)
	if err != nil {
		return err
	}

	slog.Info("syncing branch", "branch_to", syncBranch, "branch_from", currentBranchName)
	currentCommits, err := getCommitsFromMainToBranchName(currentBranchName)
	if err != nil {
		return err
	}
	syncCommits, err := getCommitsFromMainToBranchName(syncBranch)
	if err != nil {
		return err
	}

	pendingCommitIndex := -1
	if len(syncCommits) == 0 {
		pendingCommitIndex = len(currentCommits) - 1
	}
	var appliedChangeIds []string
	for _, c := range syncCommits {
		appliedChangeIds = append(appliedChangeIds, c.ChangeIDs...)
	}
	for i, c := range currentCommits {
		if !slices.Contains(appliedChangeIds, c.ChangeIDs[0]) {
			pendingCommitIndex = i
		}
	}
	if pendingCommitIndex == -1 {
		slog.Info("no pending commits to sync")
		return nil
	}

	slog.Info("pending commit", "first", pendingCommitIndex, "last", 0)

	tmpSyncBranch := "tmp-sync-" + bson.NewObjectID().Hex()
	_, err = execOutputErr("git", "checkout", "-b", tmpSyncBranch, syncBranch)
	if err != nil {
		return err
	}
	defer func() {
		_, err = execOutputErr("git", "branch", "-D", tmpSyncBranch)
		if err != nil {
			slog.Error("error during remove sync branch", "err", err)
		}
	}()

	for i := pendingCommitIndex; i >= 0; i-- {
		_, err = execOutputErr("git", "cherry-pick", currentCommits[i].Hash)
		if err != nil {
			return err
		}
	}

	syncCommits, err = getCommitsFromMainToBranchName(syncBranch)
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "checkout", syncBranch)
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "merge", "--squash", tmpSyncBranch)
	if err != nil {
		return err
	}
	commitLogs := "#commits\n"
	for i := pendingCommitIndex; i >= 0; i-- {
		commitLogs += currentCommits[i].Message
		commitLogs += "---\n"
	}
	_, err = execOutputErr("git", "commit", "-m", "sync from "+currentBranchName, "-m", commitLogs)
	if err != nil {
		return err
	}

	return nil
}

func getCommitsFromMainToBranchName(branchName string) ([]*Commit, error) {
	out, err := execOutputErr("git", "log", "--format=format:%H%x00%B%x00",
		mainBranchName+".."+branchName)
	if err != nil {
		return nil, fmt.Errorf("got error during execute: %s: %w", out, err)
	}
	return parseCommits(out), nil
}
