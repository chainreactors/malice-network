package wizard

import (
	"fmt"
	"strconv"
	"strings"

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

	r.finalizeResult(result)
	return result, nil
}

// RunCompact executes the wizard using the compact two-layer selector UI
// Supports all field types
func (r *Runner) RunCompact() (*WizardResult, error) {
	result := NewWizardResult(r.wizard.ID)

	// Convert all wizard fields to FormField
	formFields := make([]*FormField, 0, len(r.wizard.Fields))

	for _, f := range r.wizard.Fields {
		ff := &FormField{
			Name:     f.Name,
			Title:    f.Title,
			Required: f.Required,
			Validate: f.Validate,
		}

		switch f.Type {
		case FieldSelect:
			ff.Kind = KindSelect
			ff.Options = f.Options
			val := ""
			if f.Default != nil {
				val = fmt.Sprintf("%v", f.Default)
			}
			if val == "" && len(f.Options) > 0 {
				val = f.Options[0]
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

		default:
			return nil, fmt.Errorf("unsupported field type: %d", f.Type)
		}

		formFields = append(formFields, ff)
	}

	// Create and run the compact form
	wizardForm := NewWizardForm(formFields).WithTheme(r.theme)

	if err := wizardForm.Run(); err != nil {
		return nil, err
	}

	r.finalizeResult(result)
	return result, nil
}

// RunTwoPhase executes the wizard in two phases:
// Phase 1: Use compact UI for Select/MultiSelect fields
// Phase 2: Use standard form for other fields
func (r *Runner) RunTwoPhase() (*WizardResult, error) {
	// Now RunCompact supports all field types, so just use it directly
	return r.RunCompact()
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

func requiredStringValidator(label string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			if label != "" {
				return fmt.Errorf("%s is required", label)
			}
			return fmt.Errorf("value is required")
		}
		return nil
	}
}

func chainStringValidators(validators ...func(string) error) func(string) error {
	return func(s string) error {
		for _, v := range validators {
			if v == nil {
				continue
			}
			if err := v(s); err != nil {
				return err
			}
		}
		return nil
	}
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

	if f.Required || f.Validate != nil {
		validate := f.Validate
		if f.Required {
			label := f.Title
			if label == "" {
				label = f.Name
			}
			validate = chainStringValidators(requiredStringValidator(label), f.Validate)
		}
		input = input.Validate(validate)
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

	if f.Required || f.Validate != nil {
		validate := f.Validate
		if f.Required {
			label := f.Title
			if label == "" {
				label = f.Name
			}
			validate = chainStringValidators(requiredStringValidator(label), f.Validate)
		}
		text = text.Validate(validate)
	}

	return text
}

func (r *Runner) buildSelectField(f *WizardField, result *WizardResult) huh.Field {
	val := ""
	if f.Default != nil {
		val = fmt.Sprintf("%v", f.Default)
	}
	if f.Default == nil && val == "" && len(f.Options) > 0 {
		val = f.Options[0]
	}

	result.Values[f.Name] = &val

	// Use horizontal select for inline display (similar to Claude Code plan mode)
	selectField := NewHorizontalSelect(f.Options).
		Title(f.Title).
		Value(&val).
		Key(f.Name)

	if f.Description != "" {
		selectField = selectField.Description(f.Description)
	}

	if f.Required || f.Validate != nil {
		selectField = selectField.Validate(func(s string) error {
			var required func(string) error
			if f.Required {
				label := f.Title
				if label == "" {
					label = f.Name
				}
				required = requiredStringValidator(label)
			}
			return chainStringValidators(required, f.Validate)(s)
		})
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

	// Use horizontal multi-select for inline display
	multiSelect := NewHorizontalMultiSelect(f.Options).
		Title(f.Title).
		Value(&vals).
		Key(f.Name)

	if f.Description != "" {
		multiSelect = multiSelect.Description(f.Description)
	}

	if f.Required {
		multiSelect = multiSelect.Validate(func(values []string) error {
			if len(values) == 0 {
				label := f.Title
				if label == "" {
					label = f.Name
				}
				return fmt.Errorf("%s is required", label)
			}
			return nil
		})
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

	// Required has no clear meaning for a boolean confirm; leave it to callers via custom validation.
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
			s = strings.TrimSpace(s)
			if s == "" {
				if f.Required {
					label := f.Title
					if label == "" {
						label = f.Name
					}
					return fmt.Errorf("%s is required", label)
				}
				return nil
			}
			_, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("please enter a valid number")
			}
			if f.Validate != nil {
				return f.Validate(s)
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

	if f.Required || f.Validate != nil {
		validate := f.Validate
		if f.Required {
			label := f.Title
			if label == "" {
				label = f.Name
			}
			validate = chainStringValidators(requiredStringValidator(label), f.Validate)
		}
		input = input.Validate(validate)
	}

	return input
}

// SelectOption represents an option in a select menu
type SelectOption struct {
	Value       string
	Label       string
	Description string
}

// RunSelect displays an interactive select menu and returns the selected value
func RunSelect(title string, options []SelectOption) (string, error) {
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
		Value(&selected).
		Inline(true)

	form := huh.NewForm(huh.NewGroup(selectField)).WithTheme(huh.ThemeCharm())

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}
