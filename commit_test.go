package dx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommit(t *testing.T) {
	tmpdir := newGitTest(t)

	trun(t, tmpdir, "git", "checkout", "-b", "feature")

	twrite(t, tmpdir+"/content", "hello world")
	trun(t, tmpdir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	assert.NoError(t, err)

	out := trun(t, tmpdir, "git", "log")
	t.Log(out)
	assert.Contains(t, out, "change-id", "after commit, missing change-id")
}
