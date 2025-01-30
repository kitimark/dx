package conflictresolver

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// issue from fleet-bff reslove go.mod conflict files
func TestFormatGoMod(t *testing.T) {
	f, err := os.ReadFile("./fixtures/issue_conflict_same_mods/go.mod")
	require.NoError(t, err)
	f = removeConflictAnnotation(f)

	actual, err := formatGoMod("go.mod", f)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	// expect the conflict annotation is removed, it's not resolve mods yet.
	assert.Equal(t, []byte(`module issueconflictsamemods

go 1.23.0

require (
	github.com/fatih/color v1.18.0
	nmyk.io/cowsay v1.0.2
	nmyk.io/cowsay v1.0.1
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
`), actual)
}
