//go:build unit
// +build unit

package merge_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/git/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePullRefs(t *testing.T) {
	pullRefs := "master:ef08a6cd194c2687d4bc12df6bb8a86f53c348ba,2739:5b351f4eae3c4afbb90dd7787f8bf2f8c454723f,2822:bac2a1f34fd54811fb767f69543f59eb3949b2a5"
	shas, err := merge.ParsePullRefs(pullRefs)
	require.NoError(t, err)

	expected := &merge.PullRefs{
		BaseBranch: "master",
		BaseSha:    "ef08a6cd194c2687d4bc12df6bb8a86f53c348ba",
		ToMerge: []merge.Pair{
			{
				Key: "2739",
				SHA: "5b351f4eae3c4afbb90dd7787f8bf2f8c454723f",
			},
			{
				Key: "2822",
				SHA: "bac2a1f34fd54811fb767f69543f59eb3949b2a5",
			},
		},
	}
	assert.Equal(t, expected, shas)
}

func TestParsePullRefWithAdditionalRefExpression(t *testing.T) {
	pullRefs := "master:0ec6b33a1bf37b3f06ecea6687763df4a528da9c,18:ee1ddf30f6508546b5570508ffeda303b2b794d9:refs/pull/18/head"
	shas, err := merge.ParsePullRefs(pullRefs)
	require.NoError(t, err)

	expected := &merge.PullRefs{
		BaseBranch: "master",
		BaseSha:    "0ec6b33a1bf37b3f06ecea6687763df4a528da9c",
		ToMerge: []merge.Pair{
			{
				Key: "18",
				SHA: "ee1ddf30f6508546b5570508ffeda303b2b794d9",
			},
		},
	}
	assert.Equal(t, expected, shas)
}

func TestPullRefToString(t *testing.T) {
	expectedRefs := "master:ef08a,2739:5b351,2822:bac2a"

	pr, err := merge.ParsePullRefs(expectedRefs)
	assert.NoError(t, err)

	assert.Equal(t, expectedRefs, pr.String())
}
