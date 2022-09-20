package template

import (
	"context"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
)

const (
	jobsCount   = 10
	workerCount = 2
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
	commands = append(commands, &cmdrunner.Command{
		Name: "echo",
		Args: []string{"world"},
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

// func TestWorkerPool_TimeOut(t *testing.T) {
// 	wp := New(workerCount)

// 	ctx, cancel := context.WithTimeout(context.TODO(), time.Nanosecond*10)
// 	defer cancel()

// 	go wp.Run(ctx)

// 	for {
// 		select {
// 		case r := <-wp.Results():
// 			if r.Err != nil && r.Err != context.DeadlineExceeded {
// 				t.Fatalf("expected error: %v; got: %v", context.DeadlineExceeded, r.Err)
// 			}
// 		case <-wp.Done:
// 			return
// 		default:
// 		}
// 	}
// }

// func TestWorkerPool_Cancel(t *testing.T) {
// 	wp := New(workerCount)

// 	ctx, cancel := context.WithCancel(context.TODO())

// 	go wp.Run(ctx)
// 	cancel()

// 	for {
// 		select {
// 		case r := <-wp.Results():
// 			if r.Err != nil && r.Err != context.Canceled {
// 				t.Fatalf("expected error: %v; got: %v", context.Canceled, r.Err)
// 			}
// 		case <-wp.Done:
// 			return
// 		default:
// 		}
// 	}
// }

// func testJobs() []Job {
// 	jobs := make([]Job, jobsCount)
// 	for i := 0; i < jobsCount; i++ {
// 		jobs[i] = Job{
// 			Descriptor: JobDescriptor{
// 				ID:       JobID(fmt.Sprintf("%v", i)),
// 				JType:    "anyType",
// 				Metadata: nil,
// 			},
// 			ExecFn: execFn,
// 			Args:   i,
// 		}
// 	}
// 	return jobs
// }
