package repl

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	mtls2 "github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-tty"
	"github.com/muesli/termenv"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"os"
	"reflect"
)

var (
	Debug           logs.Level = 10
	Warn            logs.Level = 20
	Info            logs.Level = 30
	Error           logs.Level = 40
	Important       logs.Level = 50
	GroupStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	NameStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	DefaultLogStyle            = map[logs.Level]string{
		Debug:     termenv.String(tui.Rocket+"[+]").Bold().Background(tui.Blue).String() + " %s ",
		Warn:      termenv.String(tui.Zap+"[warn]").Bold().Background(tui.Yellow).String() + " %s ",
		Important: termenv.String(tui.Fire+"[*]").Bold().Background(tui.Purple).String() + " %s ",
		Info:      termenv.String(tui.HotSpring+"[i]").Bold().Background(tui.Green).String() + " %s ",
		Error:     termenv.String(tui.Monster+"[-]").Bold().Background(tui.Red).String() + " %s ",
	}
)

type implantFunc func(rpc clientrpc.MaliceRPCClient, sess *core.Session, params ...interface{}) (*clientpb.Task, error)
type ImplantPluginCallback func(content *clientpb.TaskContext) (interface{}, error)

func WrapImplantCallback(callback ImplantPluginCallback) intermediate.ImplantCallback {
	return func(content *clientpb.TaskContext) (string, error) {
		res, err := callback(content)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %v", content.Task.Type, res), nil
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
				param = intermediate.ConvertNumericType(param.(int64), expectedType.Kind())
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

func WrapImplantFunc(con *Console, fun interface{}, callback ImplantPluginCallback) *intermediate.InternalFunc {
	wrappedFunc := wrapImplantFunc(fun)

	interFunc := intermediate.GetInternalFuncSignature(fun)
	interFunc.ArgTypes = interFunc.ArgTypes[2:]
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

		content, err := con.Rpc.WaitTaskFinish(context.Background(), task)
		if err != nil {
			return nil, err
		}

		tui.Down(0)
		con.Log.Importantf(logs.GreenBold(fmt.Sprintf("session: %s task: %d index: %d\n", task.SessionId, task.TaskId, task.Cur)))
		err = handler.HandleMaleficError(content.Spite)
		if err != nil {
			con.Log.Errorf(err.Error())
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

func Login(con *Console, config *mtls2.ClientConfig) error {
	conn, err := mtls2.Connect(config)
	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
		return err
	}
	logs.Log.Importantf("Connected to server %s", config.Address())
	con.ServerStatus, err = core.InitServerStatus(conn, config)
	if err != nil {
		logs.Log.Errorf("init server failed : %v", err)
		return err
	}

	var pipelineCount = 0
	for _, i := range con.Listeners {
		pipelineCount = pipelineCount + len(i.Pipelines.Pipelines)
	}
	var alive = 0
	for _, i := range con.Sessions {
		if i.IsAlive {
			alive++
		}
	}
	logs.Log.Importantf("%d listeners, %d pipelines, %d clients, %d sessions (%d alive)",
		len(con.Listeners), pipelineCount, len(con.Clients), len(con.Sessions), alive)
	return nil
}

func NewConfigLogin(con *Console, yamlFile string) error {
	config, err := mtls2.ReadConfig(yamlFile)
	if err != nil {
		return err
	}
	err = Login(con, config)
	if err != nil {
		return err
	}
	err = assets.MvConfig(yamlFile)
	if err != nil {
		return err
	}
	return nil
}

func AdaptSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("\033[37m%s [%s]> \033[0m", prePrompt, string(runes))
	} else {
		sessionPrompt = fmt.Sprintf("\033[30m%s [%s]> \033[0m", prePrompt, string(runes))
	}
	return sessionPrompt
}

func NewSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", GroupStyle.Render(prePrompt), NameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", GroupStyle.Render(prePrompt), NameStyle.Render(string(runes)))
	}
	return sessionPrompt
}

// From the x/exp source code - gets a slice of keys for a map
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}

	return r
}
