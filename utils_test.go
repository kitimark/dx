package dx

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func trunMainCommand(t *testing.T, args ...string) error {
	t.Helper()

	os.Args = append([]string{Main.CommandPath()}, args...)
	return Main.Execute()
}

func newGitTest(t *testing.T) string {
	t.Helper()
	_, err := exec.LookPath("git")
	require.NoError(t, err)
	tmpdir, err := os.MkdirTemp("", "dx-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.RemoveAll(tmpdir)
		if err != nil {
			panic(err)
		}
	})
	trun(t, tmpdir, "git", "init")
	trun(t, tmpdir, "git", "config", "--local", "user.name", "tester")
	trun(t, tmpdir, "git", "config", "--local", "user.email", "tester@example.com")
	trun(t, tmpdir, "git", "branch", "-M", "main")
	twrite(t, tmpdir+"/file", "this is main")
	trun(t, tmpdir, "git", "add", "file")
	trun(t, tmpdir, "git", "commit", "-m", "initial commit")

	trun(t, tmpdir, "git", "checkout", "-b", "dev")
	trun(t, tmpdir, "git", "checkout", "main")

	wd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpdir)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Chdir(wd)
		if err != nil {
			panic(err)
		}
	})
	return tmpdir
}

func trun(t *testing.T, dir, command string, args ...string) string {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "got error: "+string(out))
	return string(out)
}

func twrite(t *testing.T, file, data string) {
	t.Helper()
	err := os.WriteFile(file, []byte(data), 0644)
	require.NoError(t, err)
}

func tappend(t *testing.T, file string, data string) {
	t.Helper()
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(data)
	require.NoError(t, err)
}

func tread(t *testing.T, file string) string {
	t.Helper()
	out, err := os.ReadFile(file)
	require.NoError(t, err)
	return string(out)
}

type tcommit struct {
	message   string
	short     string
	changeIds []string
	subCommit []*tcommit
}

func tgetCommits(t *testing.T, dir, branch string) []*tcommit {
	t.Helper()
	out := trun(t, dir, "git", "log", "--format=format:%B%x00", branch)
	return tparseCommits(t, out)
}

func tparseCommits(t *testing.T, raw string) []*tcommit {
	t.Helper()
	var commits []*tcommit
	for _, cmsg := range strings.Split(raw, "\x00") {
		cmsg = strings.TrimLeft(cmsg, "\n")
		if cmsg == "" {
			continue
		}
		if !strings.HasPrefix(cmsg, "sync from ") {
			commits = append(commits, tparseLeafCommit(t, cmsg))
			continue
		}
		cmsg = regexp.MustCompile("sync from (.*)\n\n#commits\n").
			ReplaceAllString(cmsg, "")
		var sc []*tcommit
		for _, c := range strings.Split(cmsg, "\n---\n") {
			if c == "" {
				continue
			}
			sc = append(sc, tparseLeafCommit(t, c))
		}
		var cIds []string
		for _, c := range sc {
			cIds = append(cIds, c.changeIds...)
		}
		commits = append(commits, &tcommit{
			message:   cmsg,
			short:     strings.Split(cmsg, "\n")[0],
			changeIds: cIds,
			subCommit: sc,
		})
	}
	return commits
}

func tparseLeafCommit(t *testing.T, msg string) *tcommit {
	t.Helper()
	c := new(tcommit)
	c.message = msg
	for i, l := range strings.Split(msg, "\n") {
		if i == 0 {
			c.short = l
		}
		if strings.HasPrefix(l, "change-id: ") {
			changeId := l[len("change-id: "):]
			c.changeIds = []string{changeId}
		}
	}
	return c
}

func tgetHeadBranch(t *testing.T, dir string) string {
	t.Helper()
	out := trun(t, dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out)
}
