package repl

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/mals"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"reflect"
)

var (
	ErrorAlreadyScriptName = errors.New("already exist script name")
)

func NewPlugins() *Plugins {
	plugins := &Plugins{
		Plugins: make(map[string]*plugin.Plugin),
	}
	return plugins
}

type Plugins struct {
	Plugins map[string]*plugin.Plugin
}

func (plugins *Plugins) LoadPlugin(manifest *plugin.MalManiFest, con *Console, rootCmd *cobra.Command) (plugin.Plugin, error) {
	if _, ok := plugins.Plugins[manifest.Name]; ok {
		return nil, ErrorAlreadyScriptName
	}

	var plug plugin.Plugin
	var err error
	switch manifest.Type {
	case plugin.LuaScript:
		plug, err = plugin.NewLuaMalPlugin(manifest)
	case plugin.GoPlugin:
		plug, err = plugin.NewGoMalPlugin(manifest)
	default:
		return nil, fmt.Errorf("not found valid script type: %s", manifest.Type)
	}
	if err != nil {
		return nil, err
	}

	err = plug.Run()
	if err != nil {
		return nil, err
	}
	for _, cmd := range plug.Commands() {
		cmd.CMD.GroupID = consts.MalGroup
		rootCmd.AddCommand(cmd.CMD)
	}
	return plug, nil
}

type implantFunc func(rpc clientrpc.MaliceRPCClient, sess *core.Session, params ...interface{}) (*clientpb.Task, error)

// ImplantFuncCallback, function internal callback func, retrun golang struct
type ImplantFuncCallback func(content *clientpb.TaskContext) (interface{}, error)

func WrapClientCallback(callback ImplantFuncCallback) intermediate.ImplantCallback {
	return func(content *clientpb.TaskContext) (string, error) {
		res, err := callback(content)
		if err != nil {
			return "", err
		}
		switch res.(type) {
		case string:
			output := res.(string)
			if output == "" {
				return "not output", nil
			} else {
				return output, nil
			}
		case bool:
			if res.(bool) {
				return fmt.Sprintf("%s ok", content.Task.Type), nil
			} else {
				return fmt.Sprintf("%s failed", content.Task.Type), nil
			}
		default:
			return fmt.Sprintf("%v", res), nil
		}
	}
}

func wrapImplantFunc(fun interface{}) implantFunc {
	return func(rpc clientrpc.MaliceRPCClient, sess *core.Session, params ...interface{}) (*clientpb.Task, error) {
		funcValue := reflect.ValueOf(fun)
		funcType := funcValue.Type()

		// debug
		//fmt.Println(runtime.FuncForPC(reflect.ValueOf(fun).Pointer()).Name())
		//for i := 0; i < funcType.NumIn(); i++ {
		//	fmt.Println(funcType.In(i).String())
		//}
		//fmt.Printf("%v\n", params)

		// 检查函数的参数数量是否匹配, rpc与session是强制要求的默认值, 自动+2
		if funcType.NumIn() != len(params)+2 {
			return nil, fmt.Errorf("expected %d arguments, got %d", funcType.NumIn(), len(params))
		}

		in := make([]reflect.Value, len(params)+2)
		in[0] = reflect.ValueOf(rpc)
		in[1] = reflect.ValueOf(sess)
		for i, param := range params {
			expectedType := funcType.In(i + 2)
			paramType := reflect.TypeOf(param)
			if paramType.Kind() == reflect.Int64 {
				param = mals.ConvertNumericType(param.(int64), expectedType.Kind())
			}
			if reflect.TypeOf(param) != expectedType {
				return nil, fmt.Errorf("argument %d should be %v, got %v", i+1, funcType.In(i+3), reflect.TypeOf(param))
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

func WrapImplantFunc(con *Console, fun interface{}, callback ImplantFuncCallback) *mals.MalFunction {
	wrappedFunc := wrapImplantFunc(fun)

	interFunc := mals.GetInternalFuncSignature(fun)
	interFunc.ArgTypes = interFunc.ArgTypes[1:]
	interFunc.HasLuaCallback = true
	interFunc.Func = func(args ...interface{}) (interface{}, error) {
		var sess *core.Session
		if len(args) == 0 {
			return nil, fmt.Errorf("implant func first args must be session")
		} else {
			var ok bool
			sess, ok = args[0].(*core.Session)
			if !ok {
				return nil, fmt.Errorf("implant func first args must be session")
			}
			args = args[1:]
		}

		task, err := wrappedFunc(con.Rpc, sess, args...)
		if err != nil {
			return nil, err
		}

		out := fmt.Sprintf("args %v", args)
		if len(out) > 512 {
			sess.Console(task, "args too long")
		} else {
			sess.Console(task, out)
		}
		content, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		tui.Down(1)
		err = handler.HandleMaleficError(content.Spite)
		if err != nil {
			con.Log.Errorf(err.Error() + "\n")
			return nil, err
		}

		if callback != nil {
			return callback(content)
		} else {
			return content, nil
		}
	}
	return interFunc
}

func WrapServerFunc(con *Console, fun interface{}) *mals.MalFunction {
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
	internalFunc := mals.GetInternalFuncSignature(fun)
	internalFunc.ArgTypes = internalFunc.ArgTypes[1:]
	internalFunc.Func = func(args ...interface{}) (interface{}, error) {
		return wrappedFunc(con, args...)
	}

	return internalFunc
}
