package template

import (
	"context"
	"testing"
	"time"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
)

const (
	workerCount = 3
)

func TestWorkerPool(t *testing.T) {
	cr := NewCommandRunners(workerCount)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	commands := []*cmdrunner.Command{}

	commands = append(commands, &cmdrunner.Command{
		Name: "echo",
		Args: []string{"hello"},
	})
	go cr.GenerateFrom(commands)

	go cr.Run(ctx)

	for {
		select {
		case r, ok := <-cr.Results():
			if !ok {
				continue
			}

			val := r.Value
			if val != "hello" {
				t.Fatalf("wrong value %v; expected %v", val, "hello")
			}
		case <-cr.Done:
			return
		default:
		}
	}
}

func BenchmarkWorkerPool(b *testing.B) {
	cr := NewCommandRunners(workerCount)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	commands := []*cmdrunner.Command{}

	commands = append(commands, &cmdrunner.Command{
		Name: "sleep",
		Args: []string{"5"},
	})
	commands = append(commands, &cmdrunner.Command{
		Name: "sleep",
		Args: []string{"10"},
	})
	commands = append(commands, &cmdrunner.Command{
		Name: "sleep",
		Args: []string{"15"},
	})
	go cr.GenerateFrom(commands)

	go cr.Run(ctx)
	b.ResetTimer()
	for {
		select {
		case _, ok := <-cr.Results():
			if !ok {
				continue
			}

		case <-cr.Done:
			return
		default:
		}
	}
}

 func TestWorkerPool_TimeOut(t *testing.T) {
	cr := NewCommandRunners(workerCount)

	
 	ctx, cancel := context.WithTimeout(context.TODO(), time.Nanosecond*10)
	defer cancel()

	commands := []*cmdrunner.Command{}
	commands = append(commands, &cmdrunner.Command{
		Name: "sleep",
		Args: []string{"5"},
	})

	go cr.GenerateFrom(commands)

	go cr.Run(ctx)
 	for {
 		select {
 		case r := <-cr.Results():
 			if r.Err != nil && r.Err != context.DeadlineExceeded {
 				t.Fatalf("expected error: %v; got: %v", context.DeadlineExceeded, r.Err)
 			}
 		case <-cr.Done:
 			return
 		default:
 		}
 	}
 }

func TestWorkerPool_Cancel(t *testing.T) {
 	cr := NewCommandRunners(workerCount)

 	ctx, cancel := context.WithCancel(context.TODO())
	

 	commands := []*cmdrunner.Command{}
	commands = append(commands, &cmdrunner.Command{
		Name: "sleep",
		Args: []string{"5"},
	})
	 
	go cr.GenerateFrom(commands)

	go cr.Run(ctx)
	cancel()
 	for {
 		select {
 		case r := <-cr.Results():
 			if r.Err != nil && r.Err != context.Canceled {
 				t.Fatalf("expected error: %v; got: %v", context.Canceled, r.Err)
 			}
 		case <-cr.Done:
 			return
 		default:
 		}
 	}

}
