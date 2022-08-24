package wpool

// import (
// 	"context"
// 	"errors"
// 	"reflect"
// 	"testing"
// )

// var (
// 	errDefault = errors.New("wrong argument type")
// )

// type TestCommand struct {
// 	value string
// }

// func (t TestCommand) ExecFn() (string, error) {
// 	return "", nil
// }

// func Test_job_Execute(t *testing.T) {
// 	ctx := context.TODO()

// 	test := TestCommand{
// 		value: "a",
// 	}
// 	type fields struct {
// 		execFn ExecutionFn
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		want   Result
// 	}{
// 		{
// 			name: "job execution success",
// 			fields: fields{
// 				execFn: test.ExecFn,
// 			},
// 			want: Result{
// 				Value: "",
// 			},
// 		},
// 		{
// 			name: "job execution failure",
// 			fields: fields{
// 				execFn: test.ExecFn,
// 			},
// 			want: Result{
// 				Err: errDefault,
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			j := Job{
// 				ExecFn: tt.fields.execFn,
// 			}

// 			got := j.execute(ctx)
// 			if tt.want.Err != nil {
// 				if !reflect.DeepEqual(got.Err, tt.want.Err) {
// 					t.Errorf("execute() = %v, wantError %v", got.Err, tt.want.Err)
// 				}
// 				return
// 			}

// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("execute() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
