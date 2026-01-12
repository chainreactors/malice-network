package wizard

// FieldType represents the type of a wizard field
type FieldType int

const (
	FieldInput FieldType = iota
	FieldText
	FieldSelect
	FieldMultiSelect
	FieldConfirm
	FieldNumber
	FieldFilePath
)

// WizardField represents a single field in the wizard
type WizardField struct {
	Name        string
	Title       string
	Description string
	Type        FieldType
	Default     interface{}
	Options     []string
	Required    bool
	Validate    func(string) error
}

// Wizard is the main wizard structure
type Wizard struct {
	ID          string
	Title       string
	Description string
	Fields      []*WizardField
}

// NewWizard creates a new wizard instance
func NewWizard(id, title string) *Wizard {
	return &Wizard{
		ID:     id,
		Title:  title,
		Fields: make([]*WizardField, 0),
	}
}

// WithDescription sets the wizard description
func (w *Wizard) WithDescription(desc string) *Wizard {
	w.Description = desc
	return w
}

// AddField adds a field to the wizard
func (w *Wizard) AddField(field *WizardField) *Wizard {
	w.Fields = append(w.Fields, field)
	return w
}

// Input adds an input field
func (w *Wizard) Input(name, title string, defaultVal string) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldInput,
		Default: defaultVal,
	})
}

// InputWithDesc adds an input field with description
func (w *Wizard) InputWithDesc(name, title, desc string, defaultVal string) *Wizard {
	return w.AddField(&WizardField{
		Name:        name,
		Title:       title,
		Description: desc,
		Type:        FieldInput,
		Default:     defaultVal,
	})
}

// Text adds a multi-line text field
func (w *Wizard) Text(name, title string, defaultVal string) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldText,
		Default: defaultVal,
	})
}

// Select adds a select field
func (w *Wizard) Select(name, title string, options []string) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldSelect,
		Options: options,
	})
}

// SelectWithDefault adds a select field with default value
func (w *Wizard) SelectWithDefault(name, title string, options []string, defaultVal string) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldSelect,
		Options: options,
		Default: defaultVal,
	})
}

// MultiSelect adds a multi-select field
func (w *Wizard) MultiSelect(name, title string, options []string) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldMultiSelect,
		Options: options,
	})
}

// MultiSelectWithDefault adds a multi-select field with default values
func (w *Wizard) MultiSelectWithDefault(name, title string, options []string, defaults []string) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldMultiSelect,
		Options: options,
		Default: defaults,
	})
}

// Confirm adds a confirm field
func (w *Wizard) Confirm(name, title string, defaultVal bool) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldConfirm,
		Default: defaultVal,
	})
}

// Number adds a number input field
func (w *Wizard) Number(name, title string, defaultVal int) *Wizard {
	return w.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldNumber,
		Default: defaultVal,
	})
}

// FilePath adds a file path picker field
func (w *Wizard) FilePath(name, title string) *Wizard {
	return w.AddField(&WizardField{
		Name:  name,
		Title: title,
		Type:  FieldFilePath,
	})
}

// Clone creates a copy of the wizard
func (w *Wizard) Clone() *Wizard {
	clone := &Wizard{
		ID:          w.ID,
		Title:       w.Title,
		Description: w.Description,
		Fields:      make([]*WizardField, len(w.Fields)),
	}
	for i, f := range w.Fields {
		fieldCopy := *f
		clone.Fields[i] = &fieldCopy
	}
	return clone
}
