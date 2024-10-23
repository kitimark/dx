package dx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

func TestResolveConflict_GoModConflict(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("server - init go project")
	trun(t, serverDir, "git", "checkout", "dev")
	trun(t, serverDir, "go", "mod", "init", "test/go-mod-conflict")
	twrite(t, serverDir+"/.gitignore", "go-mod-conflict")
	twrite(t, serverDir+"/main.go", `package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
`)
	trun(t, serverDir, "go", "mod", "tidy")
	trun(t, serverDir, "git", "add", ".")
	trun(t, serverDir, "go", "build", "./...")
	trun(t, serverDir, "git", "commit", "-m", "init go project")

	t.Log("client - implement main")
	trun(t, clientDir, "git", "fetch")
	trun(t, clientDir, "git", "checkout", "dev")
	trun(t, clientDir, "git", "pull", "origin", "dev")
	twrite(t, clientDir+"/main.go", `package main

import "nmyk.io/cowsay"

func main() {
	cowsay.Cowsay("hello world")
}
`)
	trun(t, clientDir, "go", "mod", "tidy")
	trun(t, clientDir, "git", "add", ".")
	trun(t, clientDir, "go", "build", "./...")
	trun(t, clientDir, "git", "commit", "-m", "print with cowsay")

	t.Log("server - implement main")
	twrite(t, serverDir+"/main.go", `package main

import (
	"fmt"
	"github.com/fatih/color"
)

func main() {
	fmt.Println("hello world")
	color.Cyan("wow, it prints a color")
}
`)
	trun(t, serverDir, "go", "mod", "tidy")
	trun(t, serverDir, "git", "add", ".")
	trun(t, serverDir, "go", "build", "./...")
	trun(t, serverDir, "git", "commit", "-m", "print color message")

	t.Log("client - try to pull rebase dev")
	trun(t, clientDir, "git", "fetch")
	tgitLog(t, clientDir, "main", "dev", "origin/dev")
	out, err := trunErr(t, clientDir, "git", "pull", "-r", "origin", "dev")
	require.Error(t, err)
	t.Log("got conflict error:\n", out)
	tgitStatus(t, clientDir)

	t.Log("client - run dx resolve-conflict")
	err = trunMainCommand(t, "--debug", "resolve-conflict")
	assert.Error(t, err)
	twrite(t, clientDir+"/main.go", `package main

import (
	"github.com/fatih/color"
	"nmyk.io/cowsay"
)

func main() {
	cowsay.Cowsay("hello world")
	color.Cyan("wow, it prints a color")
}
`)
	err = trunMainCommand(t, "--debug", "resolve-conflict")
	assert.NoError(t, err)
	trun(t, clientDir, "go", "build", "./...")
	out = tread(t, clientDir+"/go.mod")
	gomod := treadGoMod(t, clientDir+"/go.mod")
	assert.Equal(t, gomod.Syntax.Stmt[0].(*modfile.Line).Token, []string{"module", "test/go-mod-conflict"})

	requireLineBlock := gomod.Syntax.Stmt[len(gomod.Syntax.Stmt)-2].(*modfile.LineBlock)
	assert.Equal(t, requireLineBlock.Line[0].Token[0], "github.com/fatih/color")
	assert.Equal(t, requireLineBlock.Line[1].Token[0], "nmyk.io/cowsay")

	indirectRequireLineBlock := gomod.Syntax.Stmt[len(gomod.Syntax.Stmt)-1].(*modfile.LineBlock)
	for _, l := range indirectRequireLineBlock.Line {
		assert.Equal(t, l.Suffix[0].Token, "// indirect")
	}
}

func TestResolveConflict_YarnLockConflict(t *testing.T) {
	serverDir, clientDir := newGitTest(t)

	t.Log("server - init nodejs project with yarn")
	trun(t, serverDir, "git", "checkout", "dev")
	trun(t, serverDir, "yarn", "init", "-y")
	twrite(t, serverDir+"/.gitignore", "node_modules")
	trun(t, serverDir, "git", "add", ".")
	trun(t, serverDir, "git", "commit", "-m", "init go project")
	trun(t, serverDir, "git", "checkout", "main")

	t.Log("client - fetch dev branch")
	trun(t, clientDir, "git", "fetch")
	trun(t, clientDir, "git", "checkout", "dev")
	trun(t, clientDir, "git", "pull", "origin", "dev")

	t.Log("client - add dependencies")
	trun(t, clientDir, "yarn", "add", "express@4.21.1")
	trun(t, clientDir, "git", "add", ".")
	trun(t, clientDir, "git", "commit", "-m", "add express")

	t.Log("server - add more dependencies")
	trun(t, serverDir, "git", "checkout", "dev")
	trun(t, serverDir, "yarn", "add", "--dev", "jest@29.7.0")
	trun(t, serverDir, "git", "add", ".")
	trun(t, serverDir, "git", "commit", "-m", "add jest")
	trun(t, serverDir, "git", "checkout", "main")

	t.Log("client - pull rebase dev branch")
	trun(t, clientDir, "git", "fetch")
	tgitLog(t, clientDir, "main", "dev", "origin/dev")
	_, err := trunErr(t, clientDir, "git", "pull", "-r", "origin", "dev")
	require.Error(t, err)
	packagejson := tread(t, clientDir+"/package.json")
	yarnlock := tread(t, clientDir+"/yarn.lock")
	assertConflictContent(t, packagejson)
	assertConflictContent(t, yarnlock)

	t.Log("client - run dx resolve-conflict")
	err = trunMainCommand(t, "--debug", "resolve-conflict")
	require.Error(t, err, "package.json must still conflicted")

	twrite(t, clientDir+"/package.json", `{
  "name": "server",
  "version": "1.0.0",
  "main": "index.js",
  "author": "tester <tester@example.com>",
  "license": "MIT",
  "devDependencies": {
    "jest": "29.7.0"
  },
  "dependencies": {
    "express": "4.21.1"
  }
}
`)

	err = trunMainCommand(t, "--debug", "resolve-conflict")
	assert.NoError(t, err)
	yarnlock = tread(t, clientDir+"/yarn.lock")
	assertNoConflictContent(t, yarnlock)
}
