package repl

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/mattn/go-tty"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"os"
	"reflect"
)

type implantFunc func(rpc clientrpc.MaliceRPCClient, sess *Session, params ...interface{}) (*clientpb.Task, error)
type ImplantCallback func(*clientpb.TaskContext) (interface{}, error)

func wrapImplantFunc(fun interface{}) implantFunc {
	return func(rpc clientrpc.MaliceRPCClient, sess *Session, params ...interface{}) (*clientpb.Task, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params)+2 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn()-1, len(params))
		}

		// 构建参数切片
		in := make([]reflect.Value, len(params)+2)
		in[0] = reflect.ValueOf(rpc)
		in[1] = reflect.ValueOf(sess)
		for i, param := range params {
			if reflect.TypeOf(param) != funcType.In(i+2) {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, funcType.In(i+1), reflect.TypeOf(param))
			}
			in[i+2] = reflect.ValueOf(param)
		}

		// 调用函数并返回结果
		results := funcValue.Call(in)

		// 处理返回值并转换为 (*clientpb.Task, error)
		task, _ := results[0].Interface().(*clientpb.Task)
		var err error
		if results[1].Interface() != nil {
			err = results[1].Interface().(error)
		}

		return task, err
	}
}

func WrapImplantFunc(con *Console, fun interface{}, callback ImplantCallback) *intermediate.InternalFunc {
	wrappedFunc := wrapImplantFunc(fun)
	interFunc := intermediate.GetInternalFuncSignature(fun)
	interFunc.ArgTypes = interFunc.ArgTypes[1:]
	interFunc.Func = func(args ...interface{}) (interface{}, error) {
		task, err := wrappedFunc(con.Rpc, con.GetInteractive(), args...)
		if err != nil {
			return nil, err
		}

		taskContext, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		if callback != nil {
			return callback(taskContext)
		} else {
			return taskContext, nil
		}
	}
	return interFunc
}

func WrapActiveFunc(con *Console, fun interface{}, callback ImplantCallback) *intermediate.InternalFunc {
	wrappedFunc := wrapImplantFunc(fun)
	internalFunc := intermediate.GetInternalFuncSignature(fun)
	internalFunc.ArgTypes = internalFunc.ArgTypes[2:]
	internalFunc.Func = func(args ...interface{}) (interface{}, error) {
		var sess *Session
		if len(args) == 0 {
			return nil, fmt.Errorf("implant func first args must be session")
		} else {
			var ok bool
			sess, ok = args[0].(*Session)
			if !ok {
				return nil, fmt.Errorf("implant func first args must be session")
			}
			args = args[1:]
		}

		task, err := wrappedFunc(con.Rpc, sess, args...)
		if err != nil {
			return nil, err
		}

		taskContext, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		if callback != nil {
			return callback(taskContext)
		} else {
			return taskContext, nil
		}
	}
	return internalFunc
}

func WrapServerFunc(con *Console, fun interface{}) *intermediate.InternalFunc {
	wrappedFunc := func(con *Console, params ...interface{}) (interface{}, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// 检查函数的参数数量是否匹配
		if funcType.NumIn() != len(params)+1 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn()-1, len(params))
		}

		// 构建参数切片
		in := make([]reflect.Value, len(params)+1)
		in[0] = reflect.ValueOf(con)
		for i, param := range params {
			if reflect.TypeOf(param) != funcType.In(i+1) {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, funcType.In(i+1), reflect.TypeOf(param))
			}
			in[i+1] = reflect.ValueOf(param)
		}

		// 调用函数并返回结果
		results := funcValue.Call(in)

		// 假设函数有两个返回值，第一个是返回值，第二个是错误
		var err error
		if len(results) == 2 && results[1].Interface() != nil {
			err = results[1].Interface().(error)
		}

		return results[0].Interface(), err
	}
	internalFunc := intermediate.GetInternalFuncSignature(fun)
	internalFunc.ArgTypes = internalFunc.ArgTypes[1:]
	internalFunc.Func = func(args ...interface{}) (interface{}, error) {
		return wrappedFunc(con, args...)
	}

	return internalFunc
}

func exitConsole(c *console.Console) {
	open, err := tty.Open()
	if err != nil {
		panic(err)
	}
	defer open.Close()
	var isExit = false
	fmt.Print("Press 'Y/y'  or 'Ctrl+D' to confirm exit: ")

	for {
		readRune, err := open.ReadRune()
		if err != nil {
			panic(err)
		}
		if readRune == 0 {
			continue
		}
		switch readRune {
		case 'Y', 'y':
			os.Exit(0)
		case 4: // ASCII code for Ctrl+C
			os.Exit(0)
		default:
			isExit = true
		}
		if isExit {
			break
		}
	}
}

// exitImplantMenu uses the background command to detach from the implant menu.
func exitImplantMenu(c *console.Console) {
	root := c.Menu(consts.ImplantMenu).Command
	root.SetArgs([]string{consts.CommandBackground})
	root.Execute()
}

func CmdExists(name string, cmd *cobra.Command) bool {
	for _, c := range cmd.Commands() {
		if name == c.Name() {
			return true
		}
	}
	return false
}
