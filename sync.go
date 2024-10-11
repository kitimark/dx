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
	Args:    cmdSyncArgs,
	RunE:    cmdSyncRun,
}

func init() {
	CmdSync.PersistentFlags().Bool("continue", false, "continue sync commits")
}

func cmdSyncArgs(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	cont, err := flags.GetBool("continue")
	if err != nil {
		return err
	}
	if cont {
		if len(args) != 0 {
			return errors.New("no required arguments")
		}
	} else {
		if len(args) != 1 {
			return errors.New("require only one arguments")
		}
	}
	return nil
}

func cmdSyncRun(cmd *cobra.Command, args []string) error {
	tdOpts := &teardownOpts{}

	syncBranch := args[0]
	currentBranchName, err := getCurrentBranchName()
	if err != nil {
		return err
	}
	defer func() {
		if tdOpts.ignoreSwitchBranchBack {
			return
		}
		out, err := execOutputErr("git", "checkout", currentBranchName)
		if err != nil {
			slog.Warn("cannot checkout branch", "branch", currentBranchName, "output", out, "error", err)
		}
	}()

	if currentBranchName == syncBranch {
		slog.Error("cannot sync branch with same branch", "current_branch", currentBranchName,
			"sync branch", syncBranch)
		return errors.New("cannot sync branch with same branch")
	}

	err = resetBranchFromOrigin(syncBranch)
	if err != nil {
		return err
	}

	slog.Info("syncing branch", "branch_to", syncBranch, "branch_from", currentBranchName)
	currentCommits, err := getCommitsFromMainToBranchName(currentBranchName)
	if err != nil {
		return err
	}
	syncedCommits, err := getCommitsFromMainToBranchName(syncBranch)
	if err != nil {
		return err
	}

	tmpBranch, err := newTempSyncBranch(currentBranchName, syncBranch)
	if err != nil {
		return err
	}
	defer tmpBranch.cleanup()

	pendingCommitIndex := -1
	if len(syncedCommits) == 0 {
		pendingCommitIndex = len(currentCommits) - 1
	}
	var appliedChangeIds []string
	for _, c := range syncedCommits {
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

	for i := pendingCommitIndex; i >= 0; i-- {
		out, err := execOutputErr("git", "cherry-pick", currentCommits[i].Hash)
		if err != nil {
			if isCodeConflict(out) {
				fmt.Printf(fmt.Sprintf(`CONFLICT: syncing commit to %s
hint: After resolving the conflicts, mark them with
hint: "git add/rm <pathspec>"
hint: "dx sync --continue"
`, syncBranch))
				tdOpts.ignoreSwitchBranchBack = true
				tmpBranch.ignoreCleanup = true
				cmd.SilenceUsage = true
				return errors.New("code conflict")
			}
			return err
		}
	}

	_, err = execOutputErr("git", "checkout", syncBranch)
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "merge", "--squash", tmpBranch.name)
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

type sync struct {
	syncBranch    string
	syncedCommits []*Commit

	currentBranch  string
	currentCommits []*Commit

	tmpSyncBranch *tmpSyncBranch

	tdOpts     *teardownOpts
	cleanupFns []func()
}

func (s *sync) registerCleanup(fn func()) {
	s.cleanupFns = append(s.cleanupFns, fn)
}

func (s *sync) cleanup() {
	for i := len(s.cleanupFns) - 1; i >= 0; i-- {
		s.cleanupFns[i]()
	}
}

func getCurrentBranchName() (string, error) {
	currentBranchName, err := execOutputErr("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(currentBranchName, "\n"), nil
}

func resetBranchFromOrigin(syncBranch string) error {
	slog.Info("try to reset the sync branch", "branch", syncBranch)
	currentBranch, err := getCurrentBranchName()
	if err != nil {
		return nil
	}
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
	_, err = execOutputErr("git", "checkout", currentBranch)
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

// isCodeConflict returns true if error message is code conflict pattern
//
// ### Example Message
//
//	Auto-merging main
//	CONFLICT (add/add): Merge conflict in main
//	error: could not apply 7efd1d7... feat: client feature 1
//	hint: After resolving the conflicts, mark them with
//	hint: "git add/rm <pathspec>", then run
//	hint: "git cherry-pick --continue".
//	hint: You can instead skip this commit with "git cherry-pick --skip".
//	hint: To abort and get back to the state before "git cherry-pick",
//	hint: run "git cherry-pick --abort".
func isCodeConflict(msg string) bool {
	return strings.Contains(msg, "CONFLICT")
}

type teardownOpts struct {
	ignoreSwitchBranchBack bool
}

type tmpSyncBranch struct {
	name          string
	from          string
	to            string
	ignoreCleanup bool
}

func newTempSyncBranch(from, to string) (*tmpSyncBranch, error) {
	b := &tmpSyncBranch{
		name: fmt.Sprintf("tmp-sync-%s-%s-%s", b32en(to), b32en(from), bson.NewObjectID().Hex()),
		from: from,
		to:   to,
	}
	_, err := execOutputErr("git", "checkout", "-b", b.name, to)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (b *tmpSyncBranch) cleanup() {
	if b.ignoreCleanup {
		return
	}
	_, err := execOutputErr("git", "branch", "-D", b.name)
	if err != nil {
		slog.Warn("error during remove sync branch", "err", err)
	}
}
