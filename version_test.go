package dx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVersion(t *testing.T) {
	err := trunMainCommand(t, "version")
	assert.NoError(t, err)
}
