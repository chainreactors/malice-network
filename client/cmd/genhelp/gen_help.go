package main

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/pipe"
	"github.com/chainreactors/malice-network/client/command/privilege"
	"github.com/chainreactors/malice-network/client/command/reg"
	"github.com/chainreactors/malice-network/client/command/service"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/taskschd"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"os"
)

func GenImplantHelp(con *repl.Console) {
	implantMd, err := os.Create("implant_template.md")
	if err != nil {
		panic(err)
	}
	help.GenGroupHelp(implantMd, con, consts.ImplantGroup,
		tasks.Commands,
		modules.Commands,
		explorer.Commands,
		addon.Commands,
	)

	help.GenGroupHelp(implantMd, con, consts.ExecuteGroup,
		exec.Commands)

	help.GenGroupHelp(implantMd, con, consts.SysGroup,
		sys.Commands,
		service.Commands,
		reg.Commands,
		taskschd.Commands,
		privilege.Commands,
	)

	help.GenGroupHelp(implantMd, con, consts.FileGroup,
		file.Commands,
		filesystem.Commands,
		pipe.Commands)
}

func GenClientHelp(con *repl.Console) {
	clientMd, err := os.Create("client_template.md")
	if err != nil {
		panic(err)
	}
	help.GenGroupHelp(clientMd, con, consts.GenericGroup,
		generic.Commands)

	help.GenGroupHelp(clientMd, con, consts.ManageGroup,
		sessions.Commands,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		mal.Commands,
	)

	help.GenGroupHelp(clientMd, con, consts.ListenerGroup,
		listener.Commands,
	)

	help.GenGroupHelp(clientMd, con, consts.GeneratorGroup,
		build.Commands)

}

func main() {
	con, err := repl.NewConsole()
	if err != nil {
		fmt.Println(err)
		return
	}

	GenClientHelp(con)
	GenImplantHelp(con)
}
