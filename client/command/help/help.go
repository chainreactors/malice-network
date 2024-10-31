package help

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/spf13/cobra"
	"strings"
	"text/template"
)

func SetCustomHelpTemplate() (*template.Template, error) {
	funcMap := TemplateFuncs
	funcMap["RenderOpsec"] = renderOpsec
	funcMap["RenderMarkdown"] = renderMarkdownFunc

	customTemplate := `{{RenderMarkdown "## Description:"}}
{{RenderOpsec (or .Annotations.opsec "0.0") .Name .NamePadding}}

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

	helpTmpl, err := template.New("helpTemplate").Funcs(funcMap).Parse(customTemplate)
	if err != nil {
		return nil, err
	}
	return helpTmpl, nil
}

func HelpFunc(cmd *cobra.Command, ss []string) {
	var s strings.Builder

	helpTmpl, err := SetCustomHelpTemplate()
	if err != nil {
		logs.Log.Errorf("Error creating help template: %s", err)
		return
	}

	err = helpTmpl.Execute(&s, cmd)
	if err != nil {
		logs.Log.Errorf("Error executing help template: %s", err)
		return
	}

	fmt.Fprint(cmd.OutOrStdout(), s.String())
	return
}
