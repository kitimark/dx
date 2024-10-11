package dx

import (
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func trunMainCommand(t *testing.T, args ...string) error {
	t.Helper()

	t.Log("run dx:", "dx " + strings.Join(args, " "))
	cmd := NewMainCmd()
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newGitTest(t *testing.T) (string, string) {
	t.Helper()
	_, err := exec.LookPath("git")
	require.NoError(t, err)
	tmpdir, err := os.MkdirTemp("", "dx-test")
	require.NoError(t, err)
	err = os.Setenv("GIT_CONFIG_GLOBAL", tmpdir+"/git/config")
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.RemoveAll(tmpdir)
		if err != nil {
			panic(err)
		}
	})

	serverDir := tmpdir + "/server"
	tmkdir(t, serverDir)
	trun(t, serverDir, "git", "init")
	trun(t, serverDir, "git", "config", "--local", "user.name", "tester")
	trun(t, serverDir, "git", "config", "--local", "user.email", "tester@example.com")
	trun(t, serverDir, "git", "branch", "-M", "main")
	twrite(t, serverDir+"/file", "this is main")
	trun(t, serverDir, "git", "add", "file")
	trun(t, serverDir, "git", "commit", "-m", "initial commit")

	trun(t, serverDir, "git", "checkout", "-b", "dev")
	trun(t, serverDir, "git", "checkout", "main")

	clientDir := tmpdir + "/client"
	tmkdir(t, clientDir)
	trun(t, clientDir, "git", "clone", serverDir+"/.git", ".")
	trun(t, clientDir, "git", "fetch", "origin", "dev:dev")
	trun(t, clientDir, "git", "config", "--local", "user.name", "tester")
	trun(t, clientDir, "git", "config", "--local", "user.email", "tester@example.com")

	wd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(clientDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Chdir(wd)
		if err != nil {
			panic(err)
		}
	})
	return serverDir, clientDir
}

func tmkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(dir, 0700); err != nil {
		require.NoError(t, err)
	}
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
		msg := regexp.MustCompile("sync from (.*)\n\n#commits\n").
			ReplaceAllString(cmsg, "")
		var sc []*tcommit
		for _, c := range strings.Split(msg, "\n---\n") {
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

func tgitLog(t *testing.T, dir string, branches ...string) {
	t.Helper()
	args := []string{"log", "--graph", "--decorate"}
	if len(branches) == 0 {
		args = append(args, "--all")
	} else {
		args = append(args, branches...)
		args = append(args, "--", ".")
	}
	out := trun(t, dir, "git", args...)
	t.Log("git log:\n", out)
}

func assertNoBranch(t *testing.T, dir, pattern string) {
	t.Helper()
	out := tgetBranchList(t, dir, pattern)
	assert.Empty(t, out)
}

func assertBranchExist(t *testing.T, dir, pattern string) {
	t.Helper()
	out := tgetBranchList(t, dir, pattern)
	assert.NotEmpty(t, out)
}

func tgetBranchList(t *testing.T, dir, pattern string) string {
	out := trun(t, dir, "git", "branch", "--list", pattern)
	return out
}

func assertNormalTeardown(t *testing.T, dir string) {
	assertNoBranch(t, dir, "tmp-sync*")
}

func removeConflictAnnotate(t *testing.T, content string) string {
	t.Helper()
	re := regexp.MustCompile("(<<<<<<<|=======|>>>>>>>)(.*)")
	var result []string
	for _, line := range strings.Split(content, "\n") {
		if !re.MatchString(line) {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}
