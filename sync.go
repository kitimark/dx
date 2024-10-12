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

func NewSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "sync [flags] [--continue | branch]",
		Args: cmdSyncArgs,
		RunE: cmdSyncRun,
	}

	cmd.PersistentFlags().Bool("continue", false, "continue sync commits")

	return cmd
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
	var s *sync
	flags := cmd.Flags()
	con, err := flags.GetBool("continue")
	if con {
		s, err = prepareContinueSync()
		if err != nil {
			return err
		}
	} else {
		s, err = prepareSync(args)
		if err != nil {
			s.cleanup()
			return err
		}
	}
	defer s.cleanup()

	pendingCommitIndex := -1
	if len(s.syncedCommits) == 0 {
		pendingCommitIndex = len(s.currentCommits) - 1
	}
	var appliedChangeIds []string
	for _, c := range s.syncedCommits {
		appliedChangeIds = append(appliedChangeIds, c.ChangeIDs...)
	}
	for i, c := range s.currentCommits {
		if !slices.Contains(appliedChangeIds, c.ChangeIDs[0]) {
			pendingCommitIndex = i
		}
	}
	if !con && pendingCommitIndex == -1 {
		slog.Info("no pending commits to sync")
		return nil
	}

	slog.Info("pending commit", "first", pendingCommitIndex, "last", 0)

	for i := pendingCommitIndex; i >= 0; i-- {
		out, err := execOutputErr("git", "cherry-pick", s.currentCommits[i].Hash)
		if err != nil {
			if isCodeConflict(out) {
				fmt.Printf(`CONFLICT: syncing commit to %s
hint: After resolving the conflicts, mark them with
hint: "git add/rm <pathspec>"
hint: "dx sync --continue"
`, s.syncBranch)
				s.tdOpts.ignoreSwitchBranchBack = true
				s.tmpSyncBranch.ignoreCleanup = true
				cmd.SilenceUsage = true
				return errors.New("code conflict")
			}
			return err
		}
	}

	_, err = execOutputErr("git", "checkout", s.syncBranch)
	if err != nil {
		return err
	}
	_, err = execOutputErr("git", "merge", "--squash", s.tmpSyncBranch.name)
	if err != nil {
		return err
	}
	commitLogs := "#commits\n"
	for i := len(s.tmpSyncedCommits) - 1; i >= 0; i-- {
		commitLogs += s.tmpSyncedCommits[i].Message
		commitLogs += "---\n"
	}
	for i := pendingCommitIndex; i >= 0; i-- {
		commitLogs += s.currentCommits[i].Message
		commitLogs += "---\n"
	}
	_, err = execOutputErr("git", "commit", "-m", "sync from "+s.currentBranch, "-m", commitLogs)
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

	tmpSyncBranch    *tmpSyncBranch
	tmpSyncedCommits []*Commit

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

func prepareSync(args []string) (s *sync, err error) {
	s = &sync{
		tdOpts: &teardownOpts{},
	}

	s.syncBranch = args[0]
	s.currentBranch, err = getCurrentBranchName()
	if err != nil {
		return
	}
	s.registerCleanup(func() {
		if s.tdOpts.ignoreSwitchBranchBack {
			return
		}
		out, err := execOutputErr("git", "checkout", s.currentBranch)
		if err != nil {
			slog.Warn("cannot checkout branch", "branch", s.currentBranch, "output", out, "error", err)
		}
	})

	if s.currentBranch == s.syncBranch {
		slog.Error("cannot sync branch with same branch", "current_branch", s.currentBranch,
			"sync branch", s.syncBranch)
		err = errors.New("cannot sync branch with same branch")
		return
	}

	err = resetBranchFromOrigin(s.syncBranch)
	if err != nil {
		return
	}

	slog.Info("syncing branch", "branch_to", s.syncBranch, "branch_from", s.currentBranch)
	s.currentCommits, err = getCommitsFromMainToBranchName(s.currentBranch)
	if err != nil {
		return
	}
	s.syncedCommits, err = getCommitsFromMainToBranchName(s.syncBranch)
	if err != nil {
		return
	}

	s.tmpSyncBranch, err = newTempSyncBranch(s.currentBranch, s.syncBranch)
	if err != nil {
		return
	}
	s.registerCleanup(s.tmpSyncBranch.cleanup)
	return
}

func prepareContinueSync() (s *sync, err error) {
	tmpSyncBranchName, err := getCurrentBranchName()
	if err != nil {
		return
	}
	tmpSyncBranch, err := parseTempSyncBranch(tmpSyncBranchName)
	if err != nil {
		return
	}
	s = &sync{
		currentBranch: tmpSyncBranch.from,
		syncBranch:    tmpSyncBranch.to,
		tdOpts:        &teardownOpts{},
		tmpSyncBranch: tmpSyncBranch,
	}
	s.registerCleanup(func() {
		if s.tdOpts.ignoreSwitchBranchBack {
			return
		}
		out, err := execOutputErr("git", "checkout", s.currentBranch)
		if err != nil {
			slog.Warn("cannot checkout branch", "branch", s.currentBranch, "output", out, "error", err)
		}
	})

	slog.Info("continue syncing branch", "branch_to", s.syncBranch, "branch_from", s.currentBranch)
	_, err = execOutputErr("git", "-c", "core.editor=true", "cherry-pick", "--continue")
	if err != nil {
		return
	}
	s.currentCommits, err = getCommitsFromMainToBranchName(s.currentBranch)
	if err != nil {
		return
	}
	s.syncedCommits, err = getCommitsFromMainToBranchName(s.tmpSyncBranch.name)
	if err != nil {
		return
	}
	s.tmpSyncedCommits, err = getCommits(s.tmpSyncBranch.name, s.syncBranch)
	if err != nil {
		return
	}

	s.registerCleanup(s.tmpSyncBranch.cleanup)
	return
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
	commits, err := getCommits(branchName, mainBranchName)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func getCommits(headBranch, baseBranch string) ([]*Commit, error) {
	out, err := execOutputErr("git", "log", "--format=format:%H%x00%B%x00",
		baseBranch+".."+headBranch)
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

func parseTempSyncBranch(tmpBranch string) (*tmpSyncBranch, error) {
	if !strings.HasPrefix(tmpBranch, "tmp-sync-") {
		return nil, errors.New("invalid temp branch")
	}
	data := strings.Split(tmpBranch[len("tmp-sync-"):], "-")
	to, err := b32de(data[0])
	if err != nil {
		return nil, err
	}
	from, err := b32de(data[1])
	if err != nil {
		return nil, err
	}
	return &tmpSyncBranch{
		name: tmpBranch,
		from: from,
		to:   to,
	}, nil
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
