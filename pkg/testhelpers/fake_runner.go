package testhelpers

import (
	"sort"
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FakeRunner for testing command runners
type FakeRunner struct {
	Commands     []*util.Command
	ResultOutput string
	ResultError  error
}

// FakeResult the expected results
type FakeResult struct {
	CLI string
	Dir string
}

// Run the default implementation
func (f *FakeRunner) Run(c *util.Command) (string, error) {
	f.Commands = append(f.Commands, c)
	return f.ResultOutput, f.ResultError
}

// Expects expects the given results
func (f *FakeRunner) ExpectResults(t *testing.T, results ...FakeResult) {
	commands := f.Commands
	for _, c := range commands {
		t.Logf("got command %s\n", c.String())
	}

	require.Equal(t, len(results), len(commands), "expected command invocations")

	sort.Slice(commands, func(i, j int) bool {
		c1 := commands[i].String()
		c2 := commands[j].String()
		return c1 < c2
	})

	sort.Slice(results, func(i, j int) bool {
		c1 := results[i].CLI
		c2 := results[j].CLI
		return c1 < c2
	})

	for i, r := range results {
		c := commands[i]
		assert.Equal(t, r.CLI, c.String(), "command line for command %d", i+1)
		if r.Dir != "" {
			assert.Equal(t, r.Dir, c.Dir, "directory line for command %d", i+1)
		}
	}
}
