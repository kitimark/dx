package dx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBranchName(t *testing.T) {
	_, clientDir := newGitTest(t)
	trun(t, clientDir, "git", "checkout", "-b", "feature")

	err := gitInit()
	assert.NoError(t, err)
	assert.Equal(t, "main", mainBranchName)
}

func TestParseCommits(t *testing.T) {
	testcases := []struct {
		name     string
		lmsg     string
		expected []*Commit
	}{{
		name:     "empty commit log",
		lmsg:     "",
		expected: []*Commit{},
	}, {
		name: "dx commit logs",
		lmsg: "07eec49d7c27d88936e84724200a568d5143b84f\x00fix: update\n\n" +
			"change-id: 6700e097b63149da786409f7\n\x00" +
			"\n0becbfe5b066fa153d7b253be6bdd9b211d7918b\x00commit message\n\n" +
			"change-id: 6700e097b63149da786409f6\n\x00",
		expected: []*Commit{{
			Hash:      "07eec49d7c27d88936e84724200a568d5143b84f",
			ChangeIDs: []string{"6700e097b63149da786409f7"},
		}, {
			Hash:      "0becbfe5b066fa153d7b253be6bdd9b211d7918b",
			ChangeIDs: []string{"6700e097b63149da786409f6"},
		}},
	}, {
		name: "sub commit log",
		lmsg: "8330adca6b1cc2873d50ebba590e031bec6b909f\x00sync from feature\n\n" +
			"#commits\n" +
			"fix: update\n\n" +
			"change-id: 6700db6126743cdea25c9964\n" +
			"---\n" +
			"commit message\n\n" +
			"change-id: 6700db6126743cdea25c9963\n" +
			"---\n\x00",
		expected: []*Commit{{
			Hash:      "8330adca6b1cc2873d50ebba590e031bec6b909f",
			ChangeIDs: []string{"6700db6126743cdea25c9964", "6700db6126743cdea25c9963"},
		}},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseCommits(tc.lmsg)
			assert.Len(t, actual, len(tc.expected))
			for i, ex := range tc.expected {
				assert.Equal(t, ex.Hash, actual[i].Hash)
				assert.Equal(t, ex.ChangeIDs, actual[i].ChangeIDs)
			}
		})
	}
}
