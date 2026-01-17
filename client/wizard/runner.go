package wizard

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Runner handles the execution of wizards
type Runner struct {
	wizard *Wizard
	theme  *huh.Theme
}

// NewRunner creates a new runner for the given wizard
func NewRunner(w *Wizard) *Runner {
	return &Runner{
		wizard: w,
		theme:  huh.ThemeCharm(),
	}
}

// WithTheme sets a custom theme
func (r *Runner) WithTheme(theme *huh.Theme) *Runner {
	r.theme = theme
	return r
}

// Run executes the wizard and returns the result
func (r *Runner) Run() (*WizardResult, error) {
	result := NewWizardResult(r.wizard.ID)

	if r.wizard.IsGrouped() {
		return r.runGrouped(result)
	}

	// For non-grouped wizards, create a single group with all fields
	formFields := make([]*FormField, 0, len(r.wizard.Fields))
	for _, f := range r.wizard.Fields {
		ff := r.wizardFieldToFormField(f, result)
		formFields = append(formFields, ff)
	}

	formGroups := []*FormGroup{{
		Name:   "main",
		Title:  r.wizard.Title,
		Fields: formFields,
	}}

	groupedForm := NewGroupedWizardForm(formGroups).WithTheme(r.theme)
	if err := groupedForm.Run(); err != nil {
		return nil, err
	}

	r.finalizeResult(result)
	return result, nil
}

// runGrouped runs the wizard using GroupedWizardForm with Tab navigation
func (r *Runner) runGrouped(result *WizardResult) (*WizardResult, error) {
	formGroups := make([]*FormGroup, 0, len(r.wizard.Groups))

	for _, wg := range r.wizard.Groups {
		formFields := make([]*FormField, 0, len(wg.Fields))

		for _, f := range wg.Fields {
			ff := r.wizardFieldToFormField(f, result)
			formFields = append(formFields, ff)
		}

		formGroups = append(formGroups, &FormGroup{
			Name:        wg.Name,
			Title:       wg.Title,
			Description: wg.Description,
			Fields:      formFields,
			Optional:    wg.Optional,
			Expanded:    wg.Expanded,
		})
	}

	groupedForm := NewGroupedWizardForm(formGroups).WithTheme(r.theme)

	if err := groupedForm.Run(); err != nil {
		return nil, err
	}

	r.finalizeResult(result)
	return result, nil
}

// RunTwoPhase executes the wizard (kept for backward compatibility)
func (r *Runner) RunTwoPhase() (*WizardResult, error) {
	return r.Run()
}

// wizardFieldToFormField converts a WizardField to a FormField
func (r *Runner) wizardFieldToFormField(f *WizardField, result *WizardResult) *FormField {
	ff := &FormField{
		Name:        f.Name,
		Title:       f.Title,
		Description: f.Description,
		Required:    f.Required,
		Validate:    f.Validate,
	}

	switch f.Type {
	case FieldSelect:
		ff.Kind = KindSelect
		ff.Options = f.Options
		val := ""
		if f.Default != nil {
			val = fmt.Sprintf("%v", f.Default)
		}
		// Auto-select first non-empty option if default is empty
		if val == "" && len(f.Options) > 0 {
			for _, opt := range f.Options {
				if opt != "" && opt != "(empty)" {
					val = opt
					break
				}
			}
			// Fallback to first option if all are empty
			if val == "" {
				val = f.Options[0]
			}
		}
		for i, opt := range f.Options {
			if opt == val {
				ff.Selected = i
				break
			}
		}
		result.Values[f.Name] = &val
		ff.Value = &val

	case FieldMultiSelect:
		ff.Kind = KindMultiSelect
		ff.Options = f.Options
		var vals []string
		if f.Default != nil {
			if defaults, ok := f.Default.([]string); ok {
				vals = defaults
			}
		}
		ff.MultiSelect = make(map[int]bool)
		for _, v := range vals {
			for i, opt := range f.Options {
				if opt == v {
					ff.MultiSelect[i] = true
					break
				}
			}
		}
		result.Values[f.Name] = &vals
		ff.Value = &vals

	case FieldConfirm:
		ff.Kind = KindConfirm
		val := false
		if f.Default != nil {
			if b, ok := f.Default.(bool); ok {
				val = b
			}
		}
		ff.ConfirmVal = val
		result.Values[f.Name] = &val
		ff.Value = &val

	case FieldInput, FieldText, FieldFilePath:
		ff.Kind = KindInput
		val := ""
		if f.Default != nil {
			val = fmt.Sprintf("%v", f.Default)
		}
		ff.InputValue = val
		result.Values[f.Name] = &val
		ff.Value = &val

	case FieldNumber:
		ff.Kind = KindNumber
		val := ""
		if f.Default != nil {
			switch v := f.Default.(type) {
			case int:
				val = strconv.Itoa(v)
			case string:
				val = v
			}
		}
		ff.InputValue = val
		result.Values[f.Name] = &val
		ff.Value = &val
	}

	return ff
}

func (r *Runner) finalizeResult(result *WizardResult) {
	for _, f := range r.wizard.Fields {
		if f.Type != FieldNumber {
			continue
		}
		raw, ok := result.Values[f.Name]
		if !ok {
			continue
		}
		switch val := raw.(type) {
		case *string:
			if val == nil {
				result.Values[f.Name] = 0
				continue
			}
			s := strings.TrimSpace(*val)
			if s == "" {
				result.Values[f.Name] = 0
				continue
			}
			if n, err := strconv.Atoi(s); err == nil {
				result.Values[f.Name] = n
			}
		case string:
			s := strings.TrimSpace(val)
			if s == "" {
				result.Values[f.Name] = 0
				continue
			}
			if n, err := strconv.Atoi(s); err == nil {
				result.Values[f.Name] = n
			}
		}
	}
}

// SelectOption represents an option in a select menu
type SelectOption struct {
	Value       string
	Label       string
	Description string
}

// RunSelect displays an interactive select menu and returns the selected value
func RunSelect(title string, options []SelectOption) (string, error) {
	// Prevent lipgloss from sending OSC terminal queries (like \x1b]11;?)
	// which can conflict with readline's input handling and cause garbled output.
	lipglossInitOnce.Do(func() {
		lipgloss.SetHasDarkBackground(true)
	})

	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	selected := options[0].Value

	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		label := opt.Label
		if opt.Description != "" {
			label = fmt.Sprintf("%-12s - %s", opt.Label, opt.Description)
		}
		huhOptions[i] = huh.NewOption(label, opt.Value)
	}

	selectField := huh.NewSelect[string]().
		Title(title).
		Options(huhOptions...).
		Value(&selected)

	form := huh.NewForm(huh.NewGroup(selectField)).WithTheme(huh.ThemeCharm())

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}
