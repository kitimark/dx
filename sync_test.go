package dx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	tmpdir := newGitTest(t)

	t.Log("develop feature branch")
	trun(t, tmpdir, "git", "checkout", "-b", "feature")
	twrite(t, tmpdir+"/content", "hello world")
	trun(t, tmpdir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	assert.NoError(t, err)

	t.Log("testing sync dev")
	err = trunMainCommand(t, "sync", "dev")
	assert.NoError(t, err)
	assert.Equal(t, "feature", tgetHeadBranch(t, tmpdir))
	actualCommits := tgetCommits(t, tmpdir, "dev")
	assert.Len(t, actualCommits[0].subCommit, 1, "sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 1, "change ids count is invalid")
	content := tread(t, tmpdir+"/content")
	assert.Equal(t, "hello world", content, "merged content is invalid")
}

func TestSync_FeatureBranchOutdated(t *testing.T) {
	tmpdir := newGitTest(t)

	t.Log("create feature branch")
	trun(t, tmpdir, "git", "checkout", "-b", "feature")

	t.Log("main has new feature during develop feature")
	trun(t, tmpdir, "git", "checkout", "main")
	twrite(t, "feature1", "new feature")
	trun(t, tmpdir, "git", "add", "feature1")
	trun(t, tmpdir, "git", "commit", "-m", "feat: feature 1")

	t.Log("dev branch is reset")
	trun(t, tmpdir, "git", "checkout", "dev")
	trun(t, tmpdir, "git", "reset", "--hard", "main")

	t.Log("develop feature branch")
	trun(t, tmpdir, "git", "checkout", "feature")
	twrite(t, tmpdir+"/content", "hello world")
	trun(t, tmpdir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, tmpdir, "git", "checkout", "feature")
	twrite(t, tmpdir+"/content", "update")
	trun(t, tmpdir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)

	out := trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "main")
	t.Log("log:\n", out)

	t.Log("testing sync dev")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	assert.NoError(t, err)
	out = trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "dev", "main")
	t.Log("log:\n", out)
	assert.Equal(t, "feature", tgetHeadBranch(t, tmpdir))
	actualCommits := tgetCommits(t, tmpdir, "dev")
	assert.Len(t, actualCommits[0].changeIds, 2, "change ids count is invalid")
	assert.Len(t, actualCommits[0].subCommit, 2, "sub commits count is invalid")
	assert.Equal(t, actualCommits[0].subCommit[0].short, "commit message")
	assert.Equal(t, actualCommits[0].subCommit[1].short, "fix: update")
	content := tread(t, tmpdir+"/content")
	assert.Equal(t, "update", content, "merged content is invalid")
}

func TestSync_MultipleSync(t *testing.T) {
	tmpdir := newGitTest(t)

	t.Log("create feature branch")
	trun(t, tmpdir, "git", "checkout", "-b", "feature")

	t.Log("main has new feature during develop feature")
	trun(t, tmpdir, "git", "checkout", "main")
	twrite(t, "feature1", "new feature")
	trun(t, tmpdir, "git", "add", "feature1")
	trun(t, tmpdir, "git", "commit", "-m", "feat: feature 1")

	t.Log("dev branch is reset")
	trun(t, tmpdir, "git", "checkout", "dev")
	trun(t, tmpdir, "git", "reset", "--hard", "main")

	t.Log("develop feature branch")
	trun(t, tmpdir, "git", "checkout", "feature")
	twrite(t, tmpdir+"/content", "hello world")
	trun(t, tmpdir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, tmpdir, "git", "checkout", "feature")
	twrite(t, tmpdir+"/content", "update\n")
	trun(t, tmpdir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)

	out := trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "main")
	t.Log("log:\n", out)

	t.Log("sync dev - 1")
	err = trunMainCommand(t, "sync", "dev")
	require.NoError(t, err)

	out = trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "dev", "main")
	t.Log("log:\n", out)

	t.Log("develop feature branch more")
	tappend(t, tmpdir+"/content", "fix bug\n")
	trun(t, tmpdir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: fix bug")
	require.NoError(t, err)

	t.Log("sync dev - 2")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.NoError(t, err)
	out = trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "dev", "main")
	t.Log("log:\n", out)
	assert.Equal(t, "feature", tgetHeadBranch(t, tmpdir))
	actualCommits := tgetCommits(t, tmpdir, "dev")
	assert.Len(t, actualCommits[0].subCommit, 1, "new sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 1, "new change ids count is invalid")
	assert.Len(t, actualCommits[1].subCommit, 2, "old sub commits count is invalid")
	assert.Len(t, actualCommits[1].changeIds, 2, "old change ids count is invalid")
	content := tread(t, tmpdir+"/content")
	assert.Equal(t, "update\nfix bug\n", content, "merged content is invalid")
}

func TestSync_SyncWithAnotherCommitWithoutDX(t *testing.T) {
	tmpdir := newGitTest(t)

	t.Log("create feature branch")
	trun(t, tmpdir, "git", "checkout", "-b", "feature")

	t.Log("main has new feature during develop feature")
	trun(t, tmpdir, "git", "checkout", "main")
	twrite(t, "feature1", "new feature")
	trun(t, tmpdir, "git", "add", "feature1")
	trun(t, tmpdir, "git", "commit", "-m", "feat: feature 1")

	t.Log("dev branch is reset")
	trun(t, tmpdir, "git", "checkout", "dev")
	trun(t, tmpdir, "git", "reset", "--hard", "main")

	t.Log("develop feature branch")
	trun(t, tmpdir, "git", "checkout", "feature")
	twrite(t, tmpdir+"/content", "hello world")
	trun(t, tmpdir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, tmpdir, "git", "checkout", "feature")
	twrite(t, tmpdir+"/content", "update\n")
	trun(t, tmpdir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)

	out := trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "main")
	t.Log("log:\n", out)

	t.Log("sync dev - 1")
	err = trunMainCommand(t, "sync", "dev")
	require.NoError(t, err)

	t.Log("another dev guys push dev without using dx")
	trun(t, tmpdir, "git", "checkout", "dev")
	twrite(t, tmpdir+"/another_feature", "another feature without using dx")
	trun(t, tmpdir, "git", "add", "another_feature")
	trun(t, tmpdir, "git", "commit", "-m", "feat: another feature without using dx")

	t.Log("develop feature branch more")
	trun(t, tmpdir, "git", "checkout", "feature")
	tappend(t, tmpdir+"/content", "fix bug\n")
	trun(t, tmpdir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: fix bug")
	require.NoError(t, err)

	out = trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "dev", "main")
	t.Log("log:\n", out)

	t.Log("sync dev - 2")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.NoError(t, err)
	out = trun(t, tmpdir, "git", "log", "--graph", "--decorate", "feature", "dev", "main")
	t.Log("log:\n", out)
	assert.Equal(t, "feature", tgetHeadBranch(t, tmpdir))
	actualCommits := tgetCommits(t, tmpdir, "dev")
	assert.Len(t, actualCommits[0].subCommit, 1, "new sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 1, "new change ids count is invalid")
	assert.Len(t, actualCommits[1].subCommit, 0, "this is commit without using dx")
	assert.Len(t, actualCommits[1].changeIds, 0, "this is commit without using dx")
	assert.Len(t, actualCommits[2].subCommit, 2, "old sub commits count is invalid")
	assert.Len(t, actualCommits[2].changeIds, 2, "old change ids count is invalid")
	content := tread(t, tmpdir+"/content")
	assert.Equal(t, "update\nfix bug\n", content, "merged content is invalid")
}
