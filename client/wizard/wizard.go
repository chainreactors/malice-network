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

	// OptionsProvider is called to dynamically populate Options before running
	// The ctx parameter is typically *core.Console for accessing RPC
	OptionsProvider func(ctx interface{}) []string

	// parent is a reference to the wizard for chaining
	parent *Wizard
	// groupParent is a reference to the group for chaining (if field belongs to a group)
	groupParent *WizardGroup
}

// SetRequired marks this field as required and returns the wizard for chaining
func (f *WizardField) SetRequired() *Wizard {
	f.Required = true
	return f.parent
}

// Require marks this field as required and returns the field for chaining
func (f *WizardField) Require() *WizardField {
	f.Required = true
	return f
}

// SetValidate sets a validation function and returns the wizard for chaining
func (f *WizardField) SetValidate(fn func(string) error) *Wizard {
	f.Validate = fn
	return f.parent
}

// Validate sets a validation function and returns the field for chaining
func (f *WizardField) WithValidate(fn func(string) error) *WizardField {
	f.Validate = fn
	return f
}

// SetRequiredWithValidate marks field as required and sets validation
func (f *WizardField) SetRequiredWithValidate(fn func(string) error) *Wizard {
	f.Required = true
	f.Validate = fn
	return f.parent
}

// SetDescription sets the field description and returns the wizard for chaining
func (f *WizardField) SetDescription(desc string) *Wizard {
	f.Description = desc
	return f.parent
}

// Desc sets the field description and returns the field for chaining
func (f *WizardField) Desc(desc string) *WizardField {
	f.Description = desc
	return f
}

// SetOptionsProvider sets a function to dynamically populate options
func (f *WizardField) SetOptionsProvider(provider func(ctx interface{}) []string) *Wizard {
	f.OptionsProvider = provider
	return f.parent
}

// End returns the parent wizard for chaining after field configuration
func (f *WizardField) End() *Wizard {
	return f.parent
}

// EndGroup returns the parent group for chaining after field configuration
// Use this when the field was added to a group
func (f *WizardField) EndGroup() *WizardGroup {
	return f.groupParent
}

// WizardGroup represents a logical group of fields
type WizardGroup struct {
	Name        string         // Group identifier (e.g., "basic", "network")
	Title       string         // Display title (e.g., "基础配置")
	Description string         // Group description
	Fields      []*WizardField // Fields in this group
	parent      *Wizard        // Reference to parent wizard
}

// Wizard is the main wizard structure
type Wizard struct {
	ID          string
	Title       string
	Description string
	Fields      []*WizardField   // Flat list (for backward compatibility)
	Groups      []*WizardGroup   // Grouped fields for pagination
}

// NewWizard creates a new wizard instance
func NewWizard(id, title string) *Wizard {
	return &Wizard{
		ID:     id,
		Title:  title,
		Fields: make([]*WizardField, 0),
		Groups: make([]*WizardGroup, 0),
	}
}

// WithDescription sets the wizard description
func (w *Wizard) WithDescription(desc string) *Wizard {
	w.Description = desc
	return w
}

// AddField adds a field to the wizard
func (w *Wizard) AddField(field *WizardField) *Wizard {
	field.parent = w
	w.Fields = append(w.Fields, field)
	return w
}

// Field returns the last added field for configuration chaining
func (w *Wizard) Field() *WizardField {
	if len(w.Fields) == 0 {
		return nil
	}
	return w.Fields[len(w.Fields)-1]
}

// IsGrouped returns true if the wizard uses grouped fields
func (w *Wizard) IsGrouped() bool {
	return len(w.Groups) > 0
}

// NewGroup creates a new group and adds it to the wizard
func (w *Wizard) NewGroup(name, title string) *WizardGroup {
	group := &WizardGroup{
		Name:   name,
		Title:  title,
		Fields: make([]*WizardField, 0),
		parent: w,
	}
	w.Groups = append(w.Groups, group)
	return group
}

// Group returns a group by name
func (w *Wizard) Group(name string) *WizardGroup {
	for _, g := range w.Groups {
		if g.Name == name {
			return g
		}
	}
	return nil
}

// WithDescription sets the group description and returns the group for chaining
func (g *WizardGroup) WithDescription(desc string) *WizardGroup {
	g.Description = desc
	return g
}

// End returns the parent wizard for switching to another group
func (g *WizardGroup) End() *Wizard {
	return g.parent
}

// AddField adds a field to this group (and also to wizard's flat list)
func (g *WizardGroup) AddField(field *WizardField) *WizardGroup {
	field.parent = g.parent
	field.groupParent = g
	g.Fields = append(g.Fields, field)
	g.parent.Fields = append(g.parent.Fields, field)
	return g
}

// Field returns the last added field in this group for configuration chaining
func (g *WizardGroup) Field() *WizardField {
	if len(g.Fields) == 0 {
		return nil
	}
	return g.Fields[len(g.Fields)-1]
}

// Input adds an input field to the group
func (g *WizardGroup) Input(name, title string, defaultVal string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldInput,
		Default: defaultVal,
	})
}

// InputWithDesc adds an input field with description to the group
func (g *WizardGroup) InputWithDesc(name, title, desc string, defaultVal string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:        name,
		Title:       title,
		Description: desc,
		Type:        FieldInput,
		Default:     defaultVal,
	})
}

// Text adds a multi-line text field to the group
func (g *WizardGroup) Text(name, title string, defaultVal string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldText,
		Default: defaultVal,
	})
}

// Select adds a select field to the group
func (g *WizardGroup) Select(name, title string, options []string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldSelect,
		Options: options,
	})
}

// SelectWithDefault adds a select field with default value to the group
func (g *WizardGroup) SelectWithDefault(name, title string, options []string, defaultVal string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldSelect,
		Options: options,
		Default: defaultVal,
	})
}

// MultiSelect adds a multi-select field to the group
func (g *WizardGroup) MultiSelect(name, title string, options []string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldMultiSelect,
		Options: options,
	})
}

// MultiSelectWithDefault adds a multi-select field with default values to the group
func (g *WizardGroup) MultiSelectWithDefault(name, title string, options []string, defaults []string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldMultiSelect,
		Options: options,
		Default: defaults,
	})
}

// Confirm adds a confirm field to the group
func (g *WizardGroup) Confirm(name, title string, defaultVal bool) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldConfirm,
		Default: defaultVal,
	})
}

// Number adds a number input field to the group
func (g *WizardGroup) Number(name, title string, defaultVal int) *WizardGroup {
	return g.AddField(&WizardField{
		Name:    name,
		Title:   title,
		Type:    FieldNumber,
		Default: defaultVal,
	})
}

// FilePath adds a file path picker field to the group
func (g *WizardGroup) FilePath(name, title string) *WizardGroup {
	return g.AddField(&WizardField{
		Name:  name,
		Title: title,
		Type:  FieldFilePath,
	})
}

// GetField returns a field by name
func (w *Wizard) GetField(name string) *WizardField {
	for _, f := range w.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
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
		Fields:      make([]*WizardField, 0, len(w.Fields)),
		Groups:      make([]*WizardGroup, 0, len(w.Groups)),
	}

	// Build a map from original field to cloned field for group referencing
	fieldMap := make(map[*WizardField]*WizardField)

	// Clone all fields
	for _, f := range w.Fields {
		fieldCopy := *f
		fieldCopy.parent = clone
		if f.Options != nil {
			fieldCopy.Options = append([]string(nil), f.Options...)
		}
		if defaults, ok := f.Default.([]string); ok {
			fieldCopy.Default = append([]string(nil), defaults...)
		}
		clone.Fields = append(clone.Fields, &fieldCopy)
		fieldMap[f] = &fieldCopy
	}

	// Clone groups and reference cloned fields
	for _, g := range w.Groups {
		groupCopy := &WizardGroup{
			Name:        g.Name,
			Title:       g.Title,
			Description: g.Description,
			Fields:      make([]*WizardField, 0, len(g.Fields)),
			parent:      clone,
		}
		for _, f := range g.Fields {
			if clonedField, ok := fieldMap[f]; ok {
				groupCopy.Fields = append(groupCopy.Fields, clonedField)
			}
		}
		clone.Groups = append(clone.Groups, groupCopy)
	}

	return clone
}

// PrepareOptions calls OptionsProvider for all fields that have one,
// populating their Options dynamically. The ctx parameter is passed to providers.
func (w *Wizard) PrepareOptions(ctx interface{}) {
	for _, f := range w.Fields {
		if f.OptionsProvider != nil {
			opts := f.OptionsProvider(ctx)
			if len(opts) > 0 {
				f.Options = opts
			}
		}
	}
}
