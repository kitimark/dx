package dx

import (
	"fmt"
	"slices"
	"strings"
)

var mainBranchName string
var defaultMainBranchName = []string{"master", "main"}

func gitInit() error {
	args := append([]string{"branch", "--list"}, defaultMainBranchName...)
	args = append(args, "--format", "%(refname:short)")
	branches, err := execOutputErr("git", args...)
	if err != nil {
		return fmt.Errorf("error during get branch: %w", err)
	}

	for _, b := range strings.Split(branches, "\n") {
		if slices.Contains(defaultMainBranchName, b) {
			mainBranchName = b
			break
		}
	}
	if mainBranchName == "" {
		return fmt.Errorf("main (%s) branch is not found", strings.Join(defaultMainBranchName, ","))
	}
	return nil
}

type Commit struct {
	Hash      string
	Message   string
	ChangeIDs []string
	SubCommit []*Commit
}

// parseCommits only support with '%H%x00%B%x00' format
func parseCommits(lmsg string) []*Commit {
	fields := strings.Split(lmsg, "\x00")
	for i, f := range fields {
		fields[i] = strings.TrimLeft(f, "\n")
	}
	var commits []*Commit
	const numFields = 2
	for i := 0; i+numFields < len(fields); i += numFields {
		c := &Commit{
			Hash:    fields[i],
			Message: fields[i+1],
		}
		if strings.HasPrefix(c.Message, "sync from ") {
			c.SubCommit = parseSubCommit(c.Message)
			for _, sc := range c.SubCommit {
				c.ChangeIDs = append(c.ChangeIDs, sc.ChangeIDs...)
			}
		} else {
			for _, line := range strings.Split(c.Message, "\n") {
				parseCommitId(c, line)
			}
		}
		commits = append(commits, c)
	}
	return commits
}

func parseSubCommit(msg string) []*Commit {
	subMsg := strings.Split(msg, "---\n")
	subCommits := make([]*Commit, len(subMsg))
	for i, msg := range subMsg {
		c := &Commit{
			Message: msg,
		}
		for _, line := range strings.Split(c.Message, "\n") {
			if line == "#commit" {
				continue
			}
			parseCommitId(c, line)
		}
		subCommits[i] = c
	}
	return subCommits
}

func parseCommitId(c *Commit, line string) {
	if strings.HasPrefix(line, "change-id: ") {
		changeId := line[len("change-id: "):]
		c.ChangeIDs = []string{changeId}
	}
}
