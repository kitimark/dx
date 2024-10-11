package dx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	_, clientDir := newGitTest(t)

	t.Log("develop feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "feature")
	twrite(t, clientDir+"/content", "hello world")
	trun(t, clientDir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	assert.NoError(t, err)

	t.Log("testing sync dev")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	tgitLog(t, clientDir)
	assert.NoError(t, err)
	assert.Equal(t, "feature", tgetHeadBranch(t, clientDir))
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits[0].subCommit, 1, "sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 1, "change ids count is invalid")
	content := tread(t, clientDir+"/content")
	assert.Equal(t, "hello world", content, "merged content is invalid")
	assertNormalTeardown(t, clientDir)
}

func TestSync_FeatureBranchOutdated(t *testing.T) {
	_, clientDir := newGitTest(t)

	t.Log("create feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "feature")

	t.Log("main has new feature during develop feature")
	trun(t, clientDir, "git", "checkout", "main")
	twrite(t, "feature1", "new feature")
	trun(t, clientDir, "git", "add", "feature1")
	trun(t, clientDir, "git", "commit", "-m", "feat: feature 1")

	t.Log("dev branch is reset")
	trun(t, clientDir, "git", "checkout", "dev")
	trun(t, clientDir, "git", "reset", "--hard", "main")

	t.Log("develop feature branch")
	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "hello world")
	trun(t, clientDir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "update")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "main")

	t.Log("testing sync dev")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	assert.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")
	assert.Equal(t, "feature", tgetHeadBranch(t, clientDir))
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits[0].changeIds, 2, "change ids count is invalid")
	assert.Len(t, actualCommits[0].subCommit, 2, "sub commits count is invalid")
	assert.Equal(t, actualCommits[0].subCommit[0].short, "commit message")
	assert.Equal(t, actualCommits[0].subCommit[1].short, "fix: update")
	content := tread(t, clientDir+"/content")
	assert.Equal(t, "update", content, "merged content is invalid")
	assertNormalTeardown(t, clientDir)
}

func TestSync_MultipleSync(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("create feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "feature")

	t.Log("main has new feature during develop feature")
	trun(t, serverDir, "git", "checkout", "main")
	twrite(t, serverDir+"/feature1", "new feature")
	trun(t, serverDir, "git", "add", "feature1")
	trun(t, serverDir, "git", "commit", "-m", "feat: feature 1")

	t.Log("dev branch is reset")
	trun(t, serverDir, "git", "checkout", "dev")
	trun(t, serverDir, "git", "reset", "--hard", "main")
	trun(t, serverDir, "git", "checkout", "main")

	t.Log("develop feature branch")
	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "hello world")
	trun(t, clientDir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "update\n")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "main")

	t.Log("sync dev - 1")
	err = trunMainCommand(t, "sync", "dev")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")

	t.Log("push dev to origin")
	trun(t, clientDir, "git", "push", "origin", "dev")
	tgitLog(t, clientDir, "feature", "dev", "main")

	t.Log("develop feature branch more")
	tappend(t, clientDir+"/content", "fix bug\n")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: fix bug")
	require.NoError(t, err)

	t.Log("sync dev - 2")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")
	assert.Equal(t, "feature", tgetHeadBranch(t, clientDir))
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits[0].subCommit, 1, "new sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 1, "new change ids count is invalid")
	assert.Len(t, actualCommits[1].subCommit, 2, "old sub commits count is invalid")
	assert.Len(t, actualCommits[1].changeIds, 2, "old change ids count is invalid")
	content := tread(t, clientDir+"/content")
	assert.Equal(t, "update\nfix bug\n", content, "merged content is invalid")
	assertNormalTeardown(t, clientDir)
}

func TestSync_SyncWithAnotherCommitWithoutDX(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("client - create feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "feature")

	t.Log("server - main has new feature during develop feature")
	trun(t, serverDir, "git", "checkout", "main")
	twrite(t, serverDir+"/feature1", "new feature")
	trun(t, serverDir, "git", "add", "feature1")
	trun(t, serverDir, "git", "commit", "-m", "feat: feature 1")

	t.Log("server - dev branch is reset")
	trun(t, serverDir, "git", "checkout", "dev")
	trun(t, serverDir, "git", "reset", "--hard", "main")
	trun(t, serverDir, "git", "checkout", "main")

	t.Log("client - develop feature branch")
	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "hello world")
	trun(t, clientDir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "update\n")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "main")

	t.Log("client - sync dev - 1")
	err = trunMainCommand(t, "sync", "dev")
	require.NoError(t, err)

	t.Log("client - push dev to origin")
	trun(t, clientDir, "git", "push", "origin", "dev")

	t.Log("server - another dev guys push dev without using dx")
	trun(t, serverDir, "git", "checkout", "dev")
	twrite(t, serverDir+"/another_feature", "another feature without using dx")
	trun(t, serverDir, "git", "add", "another_feature")
	trun(t, serverDir, "git", "commit", "-m", "feat: another feature without using dx")
	trun(t, serverDir, "git", "checkout", "main")

	t.Log("client - develop feature branch more")
	trun(t, clientDir, "git", "checkout", "feature")
	tappend(t, clientDir+"/content", "fix bug\n")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: fix bug")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")

	t.Log("client - sync dev - 2")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")
	assert.Equal(t, "feature", tgetHeadBranch(t, clientDir))
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits[0].subCommit, 1, "new sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 1, "new change ids count is invalid")
	assert.Len(t, actualCommits[1].subCommit, 0, "this is commit without using dx")
	assert.Len(t, actualCommits[1].changeIds, 0, "this is commit without using dx")
	assert.Len(t, actualCommits[2].subCommit, 2, "old sub commits count is invalid")
	assert.Len(t, actualCommits[2].changeIds, 2, "old change ids count is invalid")
	content := tread(t, clientDir+"/content")
	assert.Equal(t, "update\nfix bug\n", content, "merged content is invalid")
	assertNormalTeardown(t, clientDir)
}

func TestSync_CompactNonPushCommits(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("create feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "feature")

	t.Log("main has new feature during develop feature")
	trun(t, serverDir, "git", "checkout", "main")
	twrite(t, serverDir+"/feature1", "new feature")
	trun(t, serverDir, "git", "add", "feature1")
	trun(t, serverDir, "git", "commit", "-m", "feat: feature 1")

	t.Log("dev branch is reset")
	trun(t, serverDir, "git", "checkout", "dev")
	trun(t, serverDir, "git", "reset", "--hard", "main")
	trun(t, serverDir, "git", "checkout", "main")

	t.Log("develop feature branch")
	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "hello world")
	trun(t, clientDir, "git", "add", "content")
	err := trunMainCommand(t, "commit", "-m", "commit message")
	require.NoError(t, err)

	trun(t, clientDir, "git", "checkout", "feature")
	twrite(t, clientDir+"/content", "update\n")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: update")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "main")

	t.Log("sync dev - 1 without push to origin")
	err = trunMainCommand(t, "sync", "dev")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")

	t.Log("develop feature branch more")
	tappend(t, clientDir+"/content", "fix bug\n")
	trun(t, clientDir, "git", "add", "content")
	err = trunMainCommand(t, "commit", "-m", "fix: fix bug")
	require.NoError(t, err)

	t.Log("sync dev - 2 ")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.NoError(t, err)
	tgitLog(t, clientDir, "feature", "dev", "main")
	assert.Equal(t, "feature", tgetHeadBranch(t, clientDir))
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Equal(t, actualCommits[0].short, "sync from feature")
	assert.Len(t, actualCommits[0].subCommit, 3, "new sub commits count is invalid")
	assert.Len(t, actualCommits[0].changeIds, 3, "new change ids count is invalid")
	content := tread(t, clientDir+"/content")
	assert.Equal(t, "update\nfix bug\n", content, "merged content is invalid")
	assertNormalTeardown(t, clientDir)
}

func TestSync_PullSyncBranchBeforeSync(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("server: make commit is git server")
	trun(t, serverDir, "git", "checkout", "dev")
	twrite(t, serverDir+"/srv_feature1", "srv_feature1")
	trun(t, serverDir, "git", "add", "srv_feature1")
	trun(t, serverDir, "git", "commit", "-m", "feat: server feature 1")

	t.Log("client: develop feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "client_feature1")
	twrite(t, clientDir+"/client_feature1", "client_feature1")
	trun(t, clientDir, "git", "add", "client_feature1")
	err := trunMainCommand(t, "commit", "-m", "feat: client feature 1")
	require.NoError(t, err)

	trun(t, clientDir, "git", "fetch")
	tgitLog(t, clientDir, "client_feature1", "origin/dev", "dev", "main")

	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.NoError(t, err)
	tgitLog(t, clientDir, "client_feature1", "origin/dev", "dev", "main")
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits, 3)
	assert.Equal(t, actualCommits[0].short, "sync from client_feature1")
	assert.Equal(t, actualCommits[1].short, "feat: server feature 1")
	assertNormalTeardown(t, clientDir)
}

func TestSync_CodeConflict(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("server: make commit is git server")
	trun(t, serverDir, "git", "checkout", "dev")
	twrite(t, serverDir+"/main", "srv_feature1\n")
	trun(t, serverDir, "git", "add", "main")
	trun(t, serverDir, "git", "commit", "-m", "feat: server feature 1")

	t.Log("client: develop feature branch")
	trun(t, clientDir, "git", "checkout", "-b", "client_feature1")
	twrite(t, clientDir+"/main", "client_feature1\n")
	trun(t, clientDir, "git", "add", "main")
	err := trunMainCommand(t, "commit", "-m", "feat: client feature 1")
	require.NoError(t, err)

	t.Log("client: try to sync")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.ErrorContains(t, err, "code conflict")
	tgitLog(t, clientDir)
	assertBranchExist(t, clientDir, "tmp-sync*")
	out := tread(t, clientDir+"/main")
	t.Log("conflict content:\n", out)

	t.Log("client: resolve conflict")
	out = removeConflictAnnotate(t, out)
	t.Log("resolved content:\n", out)
	twrite(t, clientDir+"/main", out)
	trun(t, clientDir, "git", "add", "main")
	err = trunMainCommand(t, "--debug", "sync", "--continue")
	tgitLog(t, clientDir)
	require.NoError(t, err)
	content := tread(t, clientDir+"/main")
	assert.Equal(t, "client_feature1\n", content, "original content is invalid")
	trun(t, clientDir, "git", "checkout", "dev")
	content = tread(t, clientDir+"/main")
	assert.Equal(t, "srv_feature1\nclient_feature1\n", content, "merged content is invalid")
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits[0].changeIds, 1)
	assert.Len(t, actualCommits[0].subCommit, 1)
	assert.Equal(t, actualCommits[0].subCommit[0].short, "feat: client feature 1")
}

func TestSync_CodeMultipleConflict(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("server: make commit is git server")
	trun(t, serverDir, "git", "checkout", "dev")
	twrite(t, serverDir+"/main", "srv_feature1\n")
	twrite(t, serverDir+"/lib", "lib_feature1\n")
	trun(t, serverDir, "git", "add", "main", "lib")
	trun(t, serverDir, "git", "commit", "-m", "feat: server feature 1")

	t.Log("client: develop feature branch - 1")
	trun(t, clientDir, "git", "checkout", "-b", "client_feature1")
	twrite(t, clientDir+"/main", "client_feature1\n")
	trun(t, clientDir, "git", "add", "main")
	err := trunMainCommand(t, "commit", "-m", "feat: client feature 1")
	require.NoError(t, err)

	t.Log("client: develop feature branch - 2")
	twrite(t, clientDir+"/lib", "lib_client_feature1\n")
	trun(t, clientDir, "git", "add", "lib")
	err = trunMainCommand(t, "commit", "-m", "feat(lib): update lib")
	require.NoError(t, err)

	t.Log("client: try to sync")
	err = trunMainCommand(t, "--debug", "sync", "dev")
	require.ErrorContains(t, err, "code conflict")
	assertBranchExist(t, clientDir, "tmp-sync*")
	out := tread(t, clientDir+"/main")
	t.Log("conflict content:\n", out)

	t.Log("client: resolve conflict - 1")
	out = removeConflictAnnotate(t, out)
	t.Log("resolved content:\n", out)
	twrite(t, clientDir+"/main", out)
	trun(t, clientDir, "git", "add", "main")
	err = trunMainCommand(t, "--debug", "sync", "--continue")
	require.ErrorContains(t, err, "code conflict")
	tgitLog(t, clientDir)

	out = tread(t, clientDir+"/lib")
	t.Log("conflict content:\n", out)

	t.Log("client: resolve conflict - 2")
	out = removeConflictAnnotate(t, out)
	t.Log("resolved content:\n", out)
	twrite(t, clientDir+"/lib", out)
	trun(t, clientDir, "git", "add", "lib")
	err = trunMainCommand(t, "--debug", "sync", "--continue")
	tgitLog(t, clientDir)
	require.NoError(t, err)
	content := tread(t, clientDir+"/main")
	assert.Equal(t, "client_feature1\n", content, "original main content is invalid")
	content = tread(t, clientDir+"/lib")
	assert.Equal(t, "lib_client_feature1\n", content, "original lib content is invalid")
	trun(t, clientDir, "git", "checkout", "dev")
	content = tread(t, clientDir+"/main")
	assert.Equal(t, "srv_feature1\nclient_feature1\n", content, "merged main content is invalid")
	content = tread(t, clientDir+"/lib")
	assert.Equal(t, "lib_feature1\nlib_client_feature1\n", content, "merged lib content is invalid")
	actualCommits := tgetCommits(t, clientDir, "dev")
	assert.Len(t, actualCommits[0].changeIds, 2)
	assert.Len(t, actualCommits[0].subCommit, 2)
	assert.Equal(t, actualCommits[0].subCommit[0].short, "feat: client feature 1")
	assert.Equal(t, actualCommits[0].subCommit[1].short, "feat(lib): update lib")
}
