package core

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/logs"
)

// SafeGo launches a goroutine with panic recovery and optional cleanup functions.
// Cleanups execute in the order they are passed. Recovery always runs last,
// catching panics from both fn and cleanup functions.
func SafeGo(fn func(), cleanups ...func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				fmt.Fprintf(os.Stderr, "[SafeGo] panic recovered: %v\n%s\n", r, stack)
				logs.Log.Errorf("panic recovered: %v\n%s", r, stack)
			}
		}()
		for i := len(cleanups) - 1; i >= 0; i-- {
			defer cleanups[i]()
		}
		fn()
	}()
}

// SafeGoWithInfo is like SafeGo but includes a descriptive label in panic logs.
func SafeGoWithInfo(info string, fn func(), cleanups ...func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				fmt.Fprintf(os.Stderr, "[SafeGo] panic in %s: %v\n%s\n", info, r, stack)
				logs.Log.Errorf("panic in %s: %v\n%s", info, r, stack)
			}
		}()
		for i := len(cleanups) - 1; i >= 0; i-- {
			defer cleanups[i]()
		}
		fn()
	}()
}

// SafeGoWithTask is like SafeGo but publishes a task error event on panic,
// so the client can see the failure.
func SafeGoWithTask(task *Task, fn func(), cleanups ...func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				fmt.Fprintf(os.Stderr, "[SafeGo] panic in task %s: %v\n%s\n", task.Name(), r, stack)
				logs.Log.Errorf("panic in task %s: %v\n%s", task.Name(), r, stack)
				EventBroker.Publish(Event{
					EventType: consts.EventTask,
					Op:        consts.CtrlTaskError,
					Task:      task.ToProtobuf(),
					Err:       fmt.Sprintf("panic: %v", r),
				})
			}
		}()
		for i := len(cleanups) - 1; i >= 0; i-- {
			defer cleanups[i]()
		}
		fn()
	}()
}
