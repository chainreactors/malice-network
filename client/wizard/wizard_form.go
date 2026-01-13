package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// FieldKind represents the type of field in WizardForm
type FieldKind int

const (
	KindSelect FieldKind = iota
	KindMultiSelect
	KindInput
	KindConfirm
	KindNumber
)

// FormField represents a field that can be displayed in WizardForm
type FormField struct {
	Name        string
	Title       string
	Description string
	Kind        FieldKind
	Options     []string     // For Select/MultiSelect
	Selected    int          // For Select: current selection index
	MultiSelect map[int]bool // For MultiSelect: selected indices
	InputValue  string       // For Input/Number
	ConfirmVal  bool         // For Confirm
	Required    bool
	Validate    func(string) error
	Value       interface{} // Pointer to store result
}

// WizardForm is a custom form component with two-layer selection
// Top: field selector (switch between fields)
// Bottom: current field's options/input
type WizardForm struct {
	fields     []*FormField
	fieldIndex int // Current field being edited
	cursor     int // Current cursor within the field (for Select/MultiSelect/Confirm)

	inputMode  bool // Whether we're in text input mode
	inputBuf   string
	inputCurPos int

	width    int
	height   int
	theme    *huh.Theme
	quitting bool
	aborted  bool
}

// NewWizardForm creates a new wizard form
func NewWizardForm(fields []*FormField) *WizardForm {
	return &WizardForm{
		fields:     fields,
		fieldIndex: 0,
		cursor:     0,
		width:      80,
		theme:      huh.ThemeCharm(),
	}
}

// WithTheme sets the theme
func (f *WizardForm) WithTheme(theme *huh.Theme) *WizardForm {
	f.theme = theme
	return f
}

// Init implements tea.Model
func (f *WizardForm) Init() tea.Cmd {
	if len(f.fields) > 0 {
		field := f.fields[0]
		switch field.Kind {
		case KindSelect:
			f.cursor = field.Selected
		case KindConfirm:
			if field.ConfirmVal {
				f.cursor = 0
			} else {
				f.cursor = 1
			}
		}
	}
	return nil
}

// Update implements tea.Model
func (f *WizardForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input mode separately
		if f.inputMode {
			return f.handleInputMode(msg)
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			f.aborted = true
			f.quitting = true
			return f, tea.Quit

		case "up", "k":
			f.saveCurrentField()
			f.prevField()

		case "down", "j":
			f.saveCurrentField()
			f.nextField()

		case "left", "h":
			f.prevOption()

		case "right", "l":
			f.nextOption()

		case " ":
			field := f.currentField()
			if field.Kind == KindMultiSelect {
				f.toggleSelection()
			} else if field.Kind == KindConfirm {
				f.cursor = 1 - f.cursor // Toggle Yes/No
				f.saveCurrentField()
			}

		case "enter":
			field := f.currentField()
			if field.Kind == KindInput || field.Kind == KindNumber {
				// Enter input mode
				f.inputMode = true
				f.inputBuf = field.InputValue
				f.inputCurPos = len(f.inputBuf)
			} else {
				f.saveCurrentField()
				// Submit form
				f.quitting = true
				return f, tea.Quit
			}

		case "a":
			if f.currentField().Kind == KindMultiSelect {
				f.selectAll()
			}

		case "n":
			if f.currentField().Kind == KindMultiSelect {
				f.deselectAll()
			}

		case "y":
			if f.currentField().Kind == KindConfirm {
				f.cursor = 0
				f.saveCurrentField()
			}
		}

	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
	}

	return f, nil
}

// handleInputMode handles key events when in text input mode
func (f *WizardForm) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		// Cancel input, restore original value
		f.inputMode = false
		f.inputBuf = ""

	case "enter":
		// Save input and exit input mode
		field := f.currentField()
		field.InputValue = f.inputBuf
		f.saveCurrentField()
		f.inputMode = false
		f.inputBuf = ""
		// Move to next field
		if f.fieldIndex < len(f.fields)-1 {
			f.nextField()
		}

	case "backspace":
		if len(f.inputBuf) > 0 {
			f.inputBuf = f.inputBuf[:len(f.inputBuf)-1]
		}

	case "left":
		// Cursor movement in input (simplified)

	case "right":
		// Cursor movement in input (simplified)

	default:
		// Add character to input
		if len(msg.String()) == 1 {
			f.inputBuf += msg.String()
		} else if msg.Type == tea.KeySpace {
			f.inputBuf += " "
		}
	}

	return f, nil
}

// View implements tea.Model
func (f *WizardForm) View() string {
	var sb strings.Builder

	// Styles
	selectedFieldStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("212")).
		Padding(0, 1)
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	indexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedOptionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("212")).
		Padding(0, 1)
	unselectedOptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Padding(0, 1)
	selectedCountStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(0, 1)
	inputBlurStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	currentField := f.currentField()

	// Top row: Current field name only (compact display)
	sb.WriteString(arrowStyle.Render("◀ "))
	sb.WriteString(selectedFieldStyle.Render(currentField.Title))
	sb.WriteString(arrowStyle.Render(" ▶"))
	sb.WriteString(indexStyle.Render(fmt.Sprintf("  (%d/%d)", f.fieldIndex+1, len(f.fields))))
	sb.WriteString("\n\n")

	switch currentField.Kind {
	case KindSelect:
		var optionParts []string
		for i, opt := range currentField.Options {
			if i == f.cursor {
				optionParts = append(optionParts, selectedOptionStyle.Render(opt))
			} else {
				optionParts = append(optionParts, unselectedOptionStyle.Render(opt))
			}
		}
		sb.WriteString(strings.Join(optionParts, " "))
		sb.WriteString(indexStyle.Render(fmt.Sprintf("  (%d/%d)", f.cursor+1, len(currentField.Options))))

	case KindMultiSelect:
		var optionParts []string
		for i, opt := range currentField.Options {
			marker := "○"
			if currentField.MultiSelect[i] {
				marker = "●"
			}
			label := fmt.Sprintf("%s %s", marker, opt)
			if i == f.cursor {
				optionParts = append(optionParts, selectedOptionStyle.Render(label))
			} else {
				optionParts = append(optionParts, unselectedOptionStyle.Render(label))
			}
		}
		sb.WriteString(strings.Join(optionParts, " "))
		sb.WriteString(indexStyle.Render(fmt.Sprintf("  (%d/%d)", f.cursor+1, len(currentField.Options))))

		count := 0
		for _, selected := range currentField.MultiSelect {
			if selected {
				count++
			}
		}
		if count > 0 {
			sb.WriteString(selectedCountStyle.Render(fmt.Sprintf("  已选: %d 项", count)))
		}

	case KindConfirm:
		yesLabel := "Yes"
		noLabel := "No"
		if f.cursor == 0 {
			sb.WriteString(selectedOptionStyle.Render(yesLabel))
			sb.WriteString(" ")
			sb.WriteString(unselectedOptionStyle.Render(noLabel))
		} else {
			sb.WriteString(unselectedOptionStyle.Render(yesLabel))
			sb.WriteString(" ")
			sb.WriteString(selectedOptionStyle.Render(noLabel))
		}

	case KindInput, KindNumber:
		if f.inputMode {
			// Show input with cursor
			display := f.inputBuf + "█"
			sb.WriteString(inputStyle.Render(display))
		} else {
			// Show current value
			display := currentField.InputValue
			if display == "" {
				display = "(空)"
			}
			sb.WriteString(inputBlurStyle.Render(display))
			sb.WriteString(indexStyle.Render("  按 Enter 编辑"))
		}
	}

	// Help text
	sb.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	switch currentField.Kind {
	case KindMultiSelect:
		sb.WriteString(helpStyle.Render("↑/↓: 切换字段  ←/→: 移动  Space: 选择/取消  a: 全选  n: 全不选  Enter: 提交"))
	case KindConfirm:
		sb.WriteString(helpStyle.Render("↑/↓: 切换字段  ←/→: 切换  y: Yes  n: No  Enter: 提交"))
	case KindInput, KindNumber:
		if f.inputMode {
			sb.WriteString(helpStyle.Render("Enter: 保存  Esc: 取消"))
		} else {
			sb.WriteString(helpStyle.Render("↑/↓: 切换字段  Enter: 编辑  ←/→: 切换字段"))
		}
	default:
		sb.WriteString(helpStyle.Render("↑/↓: 切换字段  ←/→: 选择  Enter: 提交"))
	}

	return sb.String()
}

// Helper methods

func (f *WizardForm) currentField() *FormField {
	if f.fieldIndex >= 0 && f.fieldIndex < len(f.fields) {
		return f.fields[f.fieldIndex]
	}
	return nil
}

func (f *WizardForm) nextField() {
	f.fieldIndex++
	if f.fieldIndex >= len(f.fields) {
		f.fieldIndex = 0 // Cycle back to first
	}
	f.initCursorForField()
}

func (f *WizardForm) prevField() {
	f.fieldIndex--
	if f.fieldIndex < 0 {
		f.fieldIndex = len(f.fields) - 1 // Cycle to last
	}
	f.initCursorForField()
}

func (f *WizardForm) initCursorForField() {
	field := f.currentField()
	switch field.Kind {
	case KindSelect:
		f.cursor = field.Selected
	case KindMultiSelect:
		f.cursor = 0
	case KindConfirm:
		if field.ConfirmVal {
			f.cursor = 0
		} else {
			f.cursor = 1
		}
	default:
		f.cursor = 0
	}
}

func (f *WizardForm) nextOption() {
	field := f.currentField()
	switch field.Kind {
	case KindSelect:
		if f.cursor < len(field.Options)-1 {
			f.cursor++
		} else {
			f.cursor = 0 // Cycle
		}
		field.Selected = f.cursor
		f.saveCurrentField()
	case KindMultiSelect:
		if f.cursor < len(field.Options)-1 {
			f.cursor++
		} else {
			f.cursor = 0 // Cycle
		}
	case KindConfirm:
		f.cursor = 1 - f.cursor
		f.saveCurrentField()
	case KindInput, KindNumber:
		// In non-input mode, switch to next field
		if !f.inputMode {
			f.saveCurrentField()
			f.nextField()
		}
	}
}

func (f *WizardForm) prevOption() {
	field := f.currentField()
	switch field.Kind {
	case KindSelect:
		if f.cursor > 0 {
			f.cursor--
		} else {
			f.cursor = len(field.Options) - 1 // Cycle
		}
		field.Selected = f.cursor
		f.saveCurrentField()
	case KindMultiSelect:
		if f.cursor > 0 {
			f.cursor--
		} else {
			f.cursor = len(field.Options) - 1 // Cycle
		}
	case KindConfirm:
		f.cursor = 1 - f.cursor
		f.saveCurrentField()
	case KindInput, KindNumber:
		// In non-input mode, switch to prev field
		if !f.inputMode {
			f.saveCurrentField()
			f.prevField()
		}
	}
}

func (f *WizardForm) toggleSelection() {
	field := f.currentField()
	if field.MultiSelect == nil {
		field.MultiSelect = make(map[int]bool)
	}
	field.MultiSelect[f.cursor] = !field.MultiSelect[f.cursor]
	f.saveCurrentField()
}

func (f *WizardForm) selectAll() {
	field := f.currentField()
	if field.MultiSelect == nil {
		field.MultiSelect = make(map[int]bool)
	}
	for i := range field.Options {
		field.MultiSelect[i] = true
	}
	f.saveCurrentField()
}

func (f *WizardForm) deselectAll() {
	field := f.currentField()
	field.MultiSelect = make(map[int]bool)
	f.saveCurrentField()
}

func (f *WizardForm) saveCurrentField() {
	field := f.currentField()
	if field == nil {
		return
	}

	switch field.Kind {
	case KindSelect:
		if ptr, ok := field.Value.(*string); ok && ptr != nil {
			if field.Selected >= 0 && field.Selected < len(field.Options) {
				*ptr = field.Options[field.Selected]
			}
		}
	case KindMultiSelect:
		if ptr, ok := field.Value.(*[]string); ok && ptr != nil {
			var selected []string
			for i, opt := range field.Options {
				if field.MultiSelect[i] {
					selected = append(selected, opt)
				}
			}
			*ptr = selected
		}
	case KindConfirm:
		if ptr, ok := field.Value.(*bool); ok && ptr != nil {
			*ptr = (f.cursor == 0)
		}
	case KindInput, KindNumber:
		if ptr, ok := field.Value.(*string); ok && ptr != nil {
			*ptr = field.InputValue
		}
	}
}

// Run executes the form
func (f *WizardForm) Run() error {
	p := tea.NewProgram(f)
	_, err := p.Run()
	if err != nil {
		return err
	}
	if f.aborted {
		return fmt.Errorf("wizard aborted")
	}
	// Final save of all fields
	for i := range f.fields {
		f.fieldIndex = i
		f.saveCurrentField()
	}
	return nil
}

// Aborted returns true if the user cancelled
func (f *WizardForm) Aborted() bool {
	return f.aborted
}

// Legacy types for backward compatibility
type SelectableField = FormField
