package command

import (
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
)

func BindImplantCommands(con *console.Console) {
	bind := makeBind(con)

	bind(consts.ImplantGroup,
		exec.Commands,
		file.Commands,
		filesystem.Commands,
		sys.Commands,
	)

	bind(consts.AliasesGroup)
	bind(consts.ExtensionGroup)
}
