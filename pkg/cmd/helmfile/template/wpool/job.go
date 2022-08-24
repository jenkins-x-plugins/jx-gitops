package wpool

// import (
// 	"context"
// )

// type ExecutionFn func() (string, error)

// type Job struct {
// 	ExecFn ExecutionFn
// }

// func (j Job) execute(ctx context.Context) Result {
// 	value, err := j.ExecFn()
// 	if err != nil {
// 		return Result{
// 			Err: err,
// 		}
// 	}

// 	return Result{
// 		Value: value,
// 	}
// }
