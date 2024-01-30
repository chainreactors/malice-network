package styles

import (
	"fmt"
	"github.com/erikgeiser/promptkit/confirmation"
	"os"
)

type ConfirmModel struct {
	model *confirmation.Confirmation
}

func (c *ConfirmModel) New(prompt string) {
	c.model = confirmation.New(prompt,
		confirmation.NewValue(true))
}

// Example:
// resultTemplateArrow is the ResultTemplate that matches TemplateArrow.
const resultTemplateArrow = `
{{- print .Prompt " " -}}
{{- if .FinalValue -}}
	{{- Foreground "32" "Yes" -}}
{{- else -}}
	{{- Foreground "32" "No" -}}
{{- end }}
`

// Example:
// templateYN is a classic template with [yn] indicator where the current
// value is capitalized and bold.
const templateYN = `
{{- Bold .Prompt -}}
{{ if .YesSelected -}}
	{{- print " [" (Bold "Y") "/n]" -}}
{{- else if .NoSelected -}}
	{{- print " [y/" (Bold "N") "]" -}}
{{- else -}}
	{{- " [y/n]" -}}
{{- end }}
`

func (c *ConfirmModel) SetStyles(ResultTemplateArrow, TemplateYN string) {
	if ResultTemplateArrow == "" {
		c.model.ResultTemplate = resultTemplateArrow
	} else {
		c.model.ResultTemplate = ResultTemplateArrow
	}
	if ResultTemplateArrow == "" {
		c.model.Template = templateYN
	} else {
		c.model.Template = TemplateYN
	}
}

func (c *ConfirmModel) Run() bool {
	ready, err := c.model.RunPrompt()
	if err != nil {
		fmt.Printf("Confirm error: %v\n", err)
		os.Exit(1)
	}
	return ready
}
