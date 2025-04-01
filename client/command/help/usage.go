package help

import (
	_ "embed"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/tui"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"strconv"
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

	customTemplate := `
{{RenderMarkdown "## Usage:"}}{{if .Runnable}}
{{RenderMarkdown (TrimParentCommand .UseLine .)}}{{end}}{{if .HasAvailableSubCommands}}
{{RenderMarkdown (printf "%s %s" (TrimParentCommand .CommandPath .) "[command]")}}{{end}}{{if gt (len .Aliases) 0}}

{{RenderMarkdown "## Aliases:"}}
{{RenderMarkdown .NameAndAliases}}{{end}}{{if .HasExample}}

{{RenderMarkdown "## Examples:"}}
{{RenderMarkdown .Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{RenderMarkdown "## Available Commands:"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
    {{RenderHelp .}} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{RenderMarkdown (printf "### %s" .Title)}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
    {{RenderHelp .}} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{RenderMarkdown "## Additional Commands:"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
    {{RenderHelp .}} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{RenderMarkdown "## Flags:"}}
{{RenderMarkdown (.LocalFlags | FlagUsages)}}{{end}}{{if .HasAvailableInheritedFlags}}

{{RenderMarkdown "## Global Flags:"}}
{{RenderMarkdown (.InheritedFlags | FlagUsages)}}{{end}}{{if .HasHelpSubCommands}}

{{RenderMarkdown "## Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{RenderMarkdown (printf "%s %s" (rpad .CommandPath .CommandPathPadding) .Short)}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{RenderMarkdown (printf "## Use \"%s [command] --help\" for more information about a command." .Name)}}{{end}}
`

	usageTmpl, err := template.New("usageTemplate").Funcs(funcMap).Parse(customTemplate)
	if err != nil {
		return nil, err
	}
	return usageTmpl, nil
}

func RenderHelp(cmd *cobra.Command) string {
	const (
		nameWidth  = 20 // Name 列宽度
		ttpWidth   = 10 // TTP 列宽度
		opsecWidth = 15 // OPSEC 列宽度
	)

	// Name 部分
	name := cmd.Name()
	if len(name) > nameWidth {
		name = name[:nameWidth-3] + "..." // 截断超长名称
	}
	nameStr := fmt.Sprintf("%-*s", nameWidth, name)

	// TTP 部分
	ttp := ""
	if val, ok := cmd.Annotations["ttp"]; ok && val != "" {
		ttp = fmt.Sprintf("(%s)", val)
	}
	ttpStr := fmt.Sprintf("%-*s", ttpWidth, ttp)

	// OPSEC 部分
	opsecStr := ""
	var opsec float64
	if val, ok := cmd.Annotations["opsec"]; ok {
		var err error
		opsec, err = strconv.ParseFloat(val, 64)
		if err == nil && opsec != 0.0 {
			opsecStr = fmt.Sprintf("[opsec %.1f]", opsec)
		}
	}
	opsecStr = fmt.Sprintf("%-*s", opsecWidth, opsecStr)

	fullDescription := nameStr + ttpStr + opsecStr

	switch {
	case opsec > 0 && opsec <= 3.9:
		return tui.RedFg.Render(fullDescription)
	case opsec >= 4.0 && opsec <= 6.9:
		return tui.OrangeFg.Render(fullDescription)
	case opsec >= 7.0 && opsec <= 8.9:
		return tui.YellowFg.Render(fullDescription)
	case opsec >= 9.0 && opsec <= 10.0:
		return tui.GreenFg.Render(fullDescription)
	default:
		if termenv.HasDarkBackground() {
			return tui.WhiteFg.Render(fullDescription)
		}
		return tui.BlackFg.Render(fullDescription)
	}
}
