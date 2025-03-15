package inputrc_test

import (
	"fmt"
	"os/user"
	"strings"

	"github.com/reeflective/readline/inputrc"
)

func Example() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	cfg := inputrc.NewDefaultConfig()
	if err := inputrc.UserDefault(u, cfg, inputrc.WithApp("bash")); err != nil {
		panic(err)
	}
	// Output:
}

func ExampleParse() {
	const example = `
set editing-mode vi
$if Usql
  set keymap vi-insert
  "\r": a-usql-action
  "\d": 'echo test\n'
$endif

`

	cfg := inputrc.NewDefaultConfig()
	if err := inputrc.Parse(strings.NewReader(example), cfg, inputrc.WithApp("usql")); err != nil {
		panic(err)
	}

	fmt.Println("editing mode:", cfg.GetString("editing-mode"))
	fmt.Println("vi-insert:")
	fmt.Printf("  %s: %s\n", inputrc.Escape(string(inputrc.Return)), cfg.Binds["vi-insert"][string(inputrc.Return)].Action)
	fmt.Printf("  %s: '%s'\n", inputrc.Escape(string(inputrc.Delete)), inputrc.EscapeMacro(cfg.Binds["vi-insert"][string(inputrc.Delete)].Action))
	// Output:
	// editing mode: vi
	// vi-insert:
	//   \C-M: a-usql-action
	//   \C-?: 'echo test\n'
}
