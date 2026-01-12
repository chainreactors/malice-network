package wizard

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
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

	// Build huh form fields
	fields := make([]huh.Field, 0, len(r.wizard.Fields))

	for _, f := range r.wizard.Fields {
		field, err := r.buildField(f, result)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	// Create the form with a single group
	group := huh.NewGroup(fields...)

	form := huh.NewForm(group).WithTheme(r.theme)

	// Run the form
	err := form.Run()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// buildField creates a huh field from a WizardField
func (r *Runner) buildField(f *WizardField, result *WizardResult) (huh.Field, error) {
	switch f.Type {
	case FieldInput:
		return r.buildInputField(f, result), nil

	case FieldText:
		return r.buildTextField(f, result), nil

	case FieldSelect:
		return r.buildSelectField(f, result), nil

	case FieldMultiSelect:
		return r.buildMultiSelectField(f, result), nil

	case FieldConfirm:
		return r.buildConfirmField(f, result), nil

	case FieldNumber:
		return r.buildNumberField(f, result), nil

	case FieldFilePath:
		return r.buildFilePathField(f, result), nil

	default:
		return nil, fmt.Errorf("unsupported field type: %d", f.Type)
	}
}

func (r *Runner) buildInputField(f *WizardField, result *WizardResult) huh.Field {
	val := ""
	if f.Default != nil {
		val = fmt.Sprintf("%v", f.Default)
	}

	result.Values[f.Name] = &val

	input := huh.NewInput().
		Title(f.Title).
		Value(&val)

	if f.Description != "" {
		input = input.Description(f.Description)
	}

	if f.Validate != nil {
		input = input.Validate(f.Validate)
	}

	return input
}

func (r *Runner) buildTextField(f *WizardField, result *WizardResult) huh.Field {
	val := ""
	if f.Default != nil {
		val = fmt.Sprintf("%v", f.Default)
	}

	result.Values[f.Name] = &val

	text := huh.NewText().
		Title(f.Title).
		Value(&val)

	if f.Description != "" {
		text = text.Description(f.Description)
	}

	if f.Validate != nil {
		text = text.Validate(f.Validate)
	}

	return text
}

func (r *Runner) buildSelectField(f *WizardField, result *WizardResult) huh.Field {
	val := ""
	if f.Default != nil {
		val = fmt.Sprintf("%v", f.Default)
	}

	result.Values[f.Name] = &val

	options := make([]huh.Option[string], len(f.Options))
	for i, opt := range f.Options {
		options[i] = huh.NewOption(opt, opt)
	}

	selectField := huh.NewSelect[string]().
		Title(f.Title).
		Options(options...).
		Value(&val)

	if f.Description != "" {
		selectField = selectField.Description(f.Description)
	}

	return selectField
}

func (r *Runner) buildMultiSelectField(f *WizardField, result *WizardResult) huh.Field {
	var vals []string
	if f.Default != nil {
		if defaults, ok := f.Default.([]string); ok {
			vals = defaults
		}
	}

	result.Values[f.Name] = &vals

	options := make([]huh.Option[string], len(f.Options))
	for i, opt := range f.Options {
		options[i] = huh.NewOption(opt, opt)
	}

	multiSelect := huh.NewMultiSelect[string]().
		Title(f.Title).
		Options(options...).
		Value(&vals)

	if f.Description != "" {
		multiSelect = multiSelect.Description(f.Description)
	}

	return multiSelect
}

func (r *Runner) buildConfirmField(f *WizardField, result *WizardResult) huh.Field {
	val := false
	if f.Default != nil {
		if b, ok := f.Default.(bool); ok {
			val = b
		}
	}

	result.Values[f.Name] = &val

	confirm := huh.NewConfirm().
		Title(f.Title).
		Value(&val)

	if f.Description != "" {
		confirm = confirm.Description(f.Description)
	}

	return confirm
}

func (r *Runner) buildNumberField(f *WizardField, result *WizardResult) huh.Field {
	val := ""
	if f.Default != nil {
		switch v := f.Default.(type) {
		case int:
			val = strconv.Itoa(v)
		case string:
			val = v
		}
	}

	result.Values[f.Name] = &val

	input := huh.NewInput().
		Title(f.Title).
		Value(&val).
		Validate(func(s string) error {
			if s == "" {
				return nil
			}
			_, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("please enter a valid number")
			}
			return nil
		})

	if f.Description != "" {
		input = input.Description(f.Description)
	}

	return input
}

func (r *Runner) buildFilePathField(f *WizardField, result *WizardResult) huh.Field {
	val := ""
	if f.Default != nil {
		val = fmt.Sprintf("%v", f.Default)
	}

	result.Values[f.Name] = &val

	// Use Input for file path since FilePicker might not be available in all versions
	input := huh.NewInput().
		Title(f.Title).
		Value(&val).
		Placeholder("Enter file path...")

	if f.Description != "" {
		input = input.Description(f.Description)
	}

	return input
}
