package core

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/chainreactors/logs"
)

type PanicError struct {
	Label     string
	Cause     error
	Recovered any
	Stack     []byte
}

func (e *PanicError) Error() string {
	if e == nil {
		return ""
	}
	msg := fmt.Sprintf("panic: %v", e.Recovered)
	if e.Cause != nil {
		msg = fmt.Sprintf("panic: %v", e.Cause)
	}
	if e.Label == "" {
		return msg
	}
	return fmt.Sprintf("%s: %s", e.Label, msg)
}

func (e *PanicError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type GoErrorHandler func(error)

func RecoverError(label string, recovered any) error {
	var cause error
	if recoveredErr, ok := recovered.(error); ok {
		cause = recoveredErr
	}
	return &PanicError{
		Label:     label,
		Cause:     cause,
		Recovered: recovered,
		Stack:     debug.Stack(),
	}
}

func ErrorText(err error) string {
	if err == nil {
		return ""
	}
	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		if panicErr.Cause != nil {
			return fmt.Sprintf("panic: %v", panicErr.Cause)
		}
		return fmt.Sprintf("panic: %v", panicErr.Recovered)
	}
	return err.Error()
}

func LogGuardedError(label string) GoErrorHandler {
	return func(err error) {
		if err == nil {
			return
		}
		var panicErr *PanicError
		if errors.As(err, &panicErr) {
			fmt.Fprintf(os.Stderr, "[GoGuarded] %s\n%s\n", panicErr.Error(), panicErr.Stack)
			logs.Log.Errorf("%s\n%s", panicErr.Error(), panicErr.Stack)
			return
		}
		if label == "" {
			logs.Log.Errorf("%s", err)
			return
		}
		logs.Log.Errorf("%s: %v", label, err)
	}
}

func CombineErrorHandlers(handlers ...GoErrorHandler) GoErrorHandler {
	return func(err error) {
		if err == nil {
			return
		}
		for _, handler := range handlers {
			if handler != nil {
				handler(err)
			}
		}
	}
}

func RunGuarded(label string, fn func() error, onError GoErrorHandler, cleanups ...func()) (err error) {
	if fn == nil {
		return nil
	}
	handler := onError
	if handler == nil {
		handler = LogGuardedError(label)
	}

	defer func() {
		var allErrs []error
		if err != nil {
			allErrs = append(allErrs, err)
		}
		if recovered := recover(); recovered != nil {
			allErrs = append(allErrs, RecoverError(label, recovered))
		}
		for i := 0; i < len(cleanups); i++ {
			cleanup := cleanups[i]
			if cleanup == nil {
				continue
			}
			if cleanupErr := runCleanup(label, i, cleanup); cleanupErr != nil {
				allErrs = append(allErrs, cleanupErr)
			}
		}
		err = errors.Join(allErrs...)
		if err != nil {
			handler(err)
		}
	}()

	return fn()
}

func runCleanup(label string, index int, cleanup func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = RecoverError(fmt.Sprintf("%s cleanup[%d]", label, index), recovered)
		}
	}()
	cleanup()
	return nil
}

func GoGuarded(label string, fn func() error, onError GoErrorHandler, cleanups ...func()) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		err := RunGuarded(label, fn, onError, cleanups...)
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	return errCh
}

