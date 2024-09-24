package help

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

var HelpTemplate = `## Description:
{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

func HelpFunc(cmd *cobra.Command, ss []string) {
	var s strings.Builder
	err := tmpl(&s, HelpTemplate, cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Fprint(cmd.OutOrStdout(), renderMarkdown(s.String()))
}
