package help

import (
	_ "embed"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/spf13/cobra"
	"strings"
	"text/template"
)

func UsageFunc(cmd *cobra.Command) error {
	var s strings.Builder
	useageTmpl, err := SetCustomUsageTemplate()

	if err != nil {
		logs.Log.Errorf("Error creating usage template: %s", err)
		return err
	}
	err = useageTmpl.Execute(&s, cmd)
	if err != nil {
		logs.Log.Errorf("Error executing usage template: %s", err)
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), s.String())
	return err
}

func FormatLongHelp(long string) string {
	content := removeImages(long)

	return renderMarkdown(content)
}

func SetCustomUsageTemplate() (*template.Template, error) {
	funcMap := TemplateFuncs

	customTemplate := `{{RenderMarkdown "## Usage:"}}{{if .Runnable}}
  {{RenderMarkdown .UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{RenderMarkdown (printf "%s %s" .CommandPath "[command]")}}{{end}}{{if gt (len .Aliases) 0}}

{{RenderMarkdown "## Aliases:"}}
  {{RenderMarkdown .NameAndAliases}}{{end}}{{if .HasExample}}

{{RenderMarkdown "## Examples:"}}
{{RenderMarkdown .Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{RenderMarkdown "## Available Commands:"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
{{RenderOpsec (or .Annotations.opsec "0.0") .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{RenderMarkdown (printf "### %s" .Title)}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
    {{RenderOpsec (or .Annotations.opsec "0.0") .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{RenderMarkdown "## Additional Commands:"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
{RenderOpsec (or .Annotations.opsec "0.0") .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{RenderMarkdown "## Flags:"}}
{{RenderMarkdown (.LocalFlags.FlagUsages | trimTrailingWhitespaces)}}{{end}}{{if .HasAvailableInheritedFlags}}

{{RenderMarkdown "## Global Flags:"}}
{{RenderMarkdown (.InheritedFlags.FlagUsages | trimTrailingWhitespaces)}}{{end}}{{if .HasHelpSubCommands}}

{{RenderMarkdown "## Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{RenderMarkdown (printf "%s %s" (rpad .CommandPath .CommandPathPadding) .Short)}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{RenderMarkdown (printf "## Use \"%s [command] --help\" for more information about a command." .CommandPath)}}{{end}}
`

	usageTmpl, err := template.New("usageTemplate").Funcs(funcMap).Parse(customTemplate)
	if err != nil {
		return nil, err
	}
	return usageTmpl, nil
}
