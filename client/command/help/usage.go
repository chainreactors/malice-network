package help

import (
	_ "embed"
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

var (
	UsageTemplate = `## Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

## Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

## Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

## Available Commands:{{range .Commands}}{{if (or .IsAvaielableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

## Flags:
{{FlagUsages .LocalFlags}}{{end}}{{if .HasAvailableInheritedFlags}}

## Global Flags:
{{FlagUsages .InheritedFlags}}{{end}}{{if .HasHelpSubCommands}}

## Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

## Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
)

func UsageFunc(cmd *cobra.Command) error {
	var s strings.Builder
	err := tmpl(&s, UsageTemplate, cmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), s.String())
	return err
}

// FormatLongHelp
func FormatLongHelp(long string) string {
	content := removeImages(long)

	return renderMarkdown(content)
}
