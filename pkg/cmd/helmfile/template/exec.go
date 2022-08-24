package template

import (
	"context"
	"fmt"
	"sync"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
)

type Result struct {
	Attempts int
	Value    string
	Err      error
}

type CommandRunners struct {
	commandRunner cmdrunner.CommandRunner
	runnersCount  int
	commands      chan *cmdrunner.Command
	results       chan Result
	Done          chan struct{}
}

func NewCommandRunners(count int, commandRunner cmdrunner.CommandRunner) CommandRunners {
	return CommandRunners{
		commandRunner: commandRunner,
		runnersCount:  count,
		commands:      make(chan *cmdrunner.Command, count),
		results:       make(chan Result, count),
		Done:          make(chan struct{}),
	}
}

func (cr CommandRunners) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < cr.runnersCount; i++ {
		wg.Add(1)
		// fan out worker goroutines
		//reading from jobs channel and
		//pushing calcs into results channel
		go cr.worker(ctx, &wg, cr.commands, cr.results)
	}

	wg.Wait()
	close(cr.Done)
	close(cr.results)
}

func (cr CommandRunners) worker(ctx context.Context, wg *sync.WaitGroup, commands <-chan *cmdrunner.Command, results chan<- Result) {
	defer wg.Done()
	for {
		select {
		case command, ok := <-commands:
			if !ok {
				return
			}
			// fan-in job execution multiplexing results into the results channel
			result, err := cr.commandRunner(command)
			results <- Result{command.Attempts(), result, err}
		case <-ctx.Done():
			fmt.Printf("cancelled worker. Error detail: %v\n", ctx.Err())
			results <- Result{
				Err: ctx.Err(),
			}
			return
		}
	}
}
func (cr CommandRunners) Results() <-chan Result {
	return cr.results
}

func (cr CommandRunners) GenerateFrom(commands []*cmdrunner.Command) {
	for i, _ := range commands {
		cr.commands <- commands[i]
	}
	close(cr.commands)
}
