package dx

import (
	"slices"
	"strings"

	"github.com/kitimark/dx/pkg/conflictresolver"
	"github.com/kitimark/dx/pkg/exec"
	"github.com/spf13/cobra"
)

func NewResolveConflictCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resolve-conflict",
		Aliases: []string{"resolve"},
		RunE:    cmdResolveConflictRun,
	}
	return cmd
}

func cmdResolveConflictRun(cmd *cobra.Command, _ []string) error {
	conflictedFiles, err := getConflictedFiles()
	if err != nil {
		return err
	}

	for _, r := range conflictresolver.ConflictResolvers {
		if r.Detect(conflictedFiles) {
			err = r.Resolve(conflictedFiles)
			if err != nil {
				cmd.SilenceUsage = true
				return err
			}
		}
	}

	return nil
}

var gitXYConflictedStatuses = []string{"AA", "UU"}

// getConflictedFiles return list of conflict files that parsed from `git status --short`
//
// ### Example output of `git status --short`
//
//	UU go.mod
//	AA go.sum
//	UU main.go
//
// ### Output notation
//
// ref: https://git-scm.com/docs/git-status#_short_format
func getConflictedFiles() ([]string, error) {
	out, err := exec.OutputErr("git", "status", "--short")
	if err != nil {
		return nil, err
	}
	var conflictedFiles []string
	for _, line := range strings.Split(out, "\n") {
		content := strings.Split(line, " ")
		if slices.Contains(gitXYConflictedStatuses, content[0]) {
			conflictedFiles = append(conflictedFiles, content[1])
		}
	}
	return conflictedFiles, nil
}
