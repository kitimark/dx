package dx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommit(t *testing.T) {
	_, clientDir := newGitTest(t)

	trun(t, clientDir, "git", "checkout", "-b", "feature")

	twrite(t, clientDir+"/content", "hello world")
	trun(t, clientDir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	assert.NoError(t, err)

	out := trun(t, clientDir, "git", "log")
	t.Log(out)
	assert.Contains(t, out, "change-id", "after commit, missing change-id")
}
