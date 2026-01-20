package wizard

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	// lipglossInitOnce ensures we only initialize lipgloss background detection once
	// to avoid OSC terminal queries that can conflict with readline input handling.
	lipglossInitOnce sync.Once
)

// FieldKind represents the type of field in the form
type FieldKind int

const (
	KindSelect FieldKind = iota
	KindMultiSelect
	KindInput
	KindConfirm
	KindNumber
)

// FormTheme defines styles for the grouped wizard form
type FormTheme struct {
	TabActive           lipgloss.Style
	TabInactive         lipgloss.Style
	TabCompleted        lipgloss.Style
	Separator           lipgloss.Style
	Error               lipgloss.Style
	Help                lipgloss.Style
	FocusedTitle        lipgloss.Style
	NormalTitle         lipgloss.Style
	Description         lipgloss.Style
	SelectedOption      lipgloss.Style
	UnselectedOption    lipgloss.Style
	FocusedUnselected   lipgloss.Style
	MultiSelectChecked  lipgloss.Style
	InputFocused        lipgloss.Style
	InputBlurred        lipgloss.Style
}

// DefaultFormTheme returns the default theme for grouped wizard forms
func DefaultFormTheme() *FormTheme {
	return &FormTheme{
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("212")).
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Padding(0, 1),
		TabCompleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Padding(0, 1),
		Separator:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Error:        lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
		Help:         lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		FocusedTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		NormalTitle:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		Description:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
		SelectedOption: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("212")).
			Padding(0, 1),
		UnselectedOption:   lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1),
		FocusedUnselected:  lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Padding(0, 1),
		MultiSelectChecked: lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Padding(0, 1),
		InputFocused:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1),
		InputBlurred:       lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1),
	}
}

// Package-level default theme instance
var defaultFormTheme = DefaultFormTheme()

// defaultTerminalWidth is the fallback width when terminal size cannot be determined
const defaultTerminalWidth = 80

// getTerminalWidth returns the current terminal width or a default value
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return defaultTerminalWidth
	}
	return width
}

// FormField represents a field that can be displayed in the form
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

// GroupedWizardForm is a wizard form with Tab navigation for groups
type GroupedWizardForm struct {
	groups     []*FormGroup
	groupIndex int // Current group being edited

	// Current field within group
	fieldIndex int
	cursor     int // Cursor within field options

	inputMode   bool
	inputBuf    string
	inputCurPos int

	width     int
	height    int
	theme     *huh.Theme
	formTheme *FormTheme
	quitting  bool
	aborted   bool

	errMsg string
}

// FormGroup represents a group of fields
type FormGroup struct {
	Name        string
	Title       string
	Description string
	Fields      []*FormField
	Optional    bool // If true, this group can be collapsed
	Expanded    bool // If true and Optional, show fields; otherwise collapsed
}

// NewGroupedWizardForm creates a new grouped wizard form
func NewGroupedWizardForm(groups []*FormGroup) *GroupedWizardForm {
	return &GroupedWizardForm{
		groups:     groups,
		groupIndex: 0,
		fieldIndex: 0,
		cursor:     0,
		width:      getTerminalWidth(),
		theme:      huh.ThemeCharm(),
		formTheme:  defaultFormTheme,
	}
}

// WithTheme sets the huh theme
func (f *GroupedWizardForm) WithTheme(theme *huh.Theme) *GroupedWizardForm {
	f.theme = theme
	return f
}

// WithFormTheme sets the form theme for styling
func (f *GroupedWizardForm) WithFormTheme(theme *FormTheme) *GroupedWizardForm {
	f.formTheme = theme
	return f
}

// Init implements tea.Model
func (f *GroupedWizardForm) Init() tea.Cmd {
	f.initCursorForField()
	return nil
}

// Update implements tea.Model
func (f *GroupedWizardForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input mode separately
		if f.inputMode {
			return f.handleInputMode(msg)
		}

		key := msg.String()

		// Check if current group is a collapsed optional group
		group := f.currentGroup()
		isCollapsedOptional := group != nil && group.Optional && !group.Expanded

		// Number keys 1-9 for group navigation
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			groupNum := int(key[0] - '1')
			if groupNum < len(f.groups) {
				f.errMsg = ""
				f.saveCurrentField()
				f.groupIndex = groupNum
				f.fieldIndex = 0
				f.initCursorForField()
				return f, nil
			}
		}

		switch key {
		case "ctrl+c", "esc":
			f.aborted = true
			f.quitting = true
			return f, tea.Quit

		case "tab":
			// Next group
			f.errMsg = ""
			f.saveCurrentField()
			f.nextGroup()

		case "shift+tab":
			// Previous group
			f.errMsg = ""
			f.saveCurrentField()
			f.prevGroup()

		case "up", "k":
			if isCollapsedOptional {
				break // No field navigation in collapsed group
			}
			f.errMsg = ""
			f.saveCurrentField()
			f.prevField()

		case "down", "j":
			if isCollapsedOptional {
				break // No field navigation in collapsed group
			}
			f.errMsg = ""
			f.saveCurrentField()
			f.nextField()

		case "left", "h":
			if isCollapsedOptional {
				break
			}
			f.errMsg = ""
			f.prevOption()

		case "right", "l":
			if isCollapsedOptional {
				break
			}
			f.errMsg = ""
			f.nextOption()

		case " ":
			f.errMsg = ""
			// Handle collapsed optional group - expand it
			if isCollapsedOptional {
				group.Expanded = true
				f.fieldIndex = 0
				f.initCursorForField()
				break
			}
			field := f.currentField()
			if field == nil {
				break
			}
			if field.Kind == KindMultiSelect {
				f.toggleSelection()
			} else if field.Kind == KindConfirm {
				f.cursor = 1 - f.cursor
				f.saveCurrentField()
			}

		case "ctrl+d":
			return f.trySubmit()

		case "enter":
			// Handle collapsed optional group - expand it
			if isCollapsedOptional {
				f.errMsg = ""
				group.Expanded = true
				f.fieldIndex = 0
				f.initCursorForField()
				break
			}
			field := f.currentField()
			if field == nil {
				return f.trySubmit()
			}
			if field.Kind == KindInput || field.Kind == KindNumber {
				f.errMsg = ""
				f.inputMode = true
				f.inputBuf = field.InputValue
				f.inputCurPos = len(f.inputBuf)
			} else {
				return f.trySubmit()
			}

		case "c":
			// Collapse current optional group if expanded
			if group != nil && group.Optional && group.Expanded {
				f.errMsg = ""
				group.Expanded = false
				f.fieldIndex = 0
			}

		case "a":
			if isCollapsedOptional {
				break
			}
			if f.currentField() != nil && f.currentField().Kind == KindMultiSelect {
				f.errMsg = ""
				f.selectAll()
			}

		case "n":
			if isCollapsedOptional {
				break
			}
			field := f.currentField()
			if field != nil {
				if field.Kind == KindMultiSelect {
					f.errMsg = ""
					f.deselectAll()
				} else if field.Kind == KindConfirm {
					f.errMsg = ""
					f.cursor = 1
					f.saveCurrentField()
				}
			}

		case "y":
			if isCollapsedOptional {
				break
			}
			if f.currentField() != nil && f.currentField().Kind == KindConfirm {
				f.errMsg = ""
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
func (f *GroupedWizardForm) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		f.inputMode = false
		f.inputBuf = ""
		f.errMsg = ""

	case "enter":
		field := f.currentField()
		if field == nil {
			f.inputMode = false
			return f, nil
		}
		candidate := f.inputBuf
		old := field.InputValue
		field.InputValue = candidate
		if err := f.validateField(field); err != nil {
			field.InputValue = old
			f.errMsg = err.Error()
			return f, nil
		}
		f.saveCurrentField()
		f.inputMode = false
		f.inputBuf = ""
		f.errMsg = ""
		// Move to next field
		if f.fieldIndex < len(f.currentGroup().Fields)-1 {
			f.nextField()
		}

	case "ctrl+d":
		field := f.currentField()
		if field == nil {
			f.inputMode = false
			return f.trySubmit()
		}
		candidate := f.inputBuf
		old := field.InputValue
		field.InputValue = candidate
		if err := f.validateField(field); err != nil {
			field.InputValue = old
			f.errMsg = err.Error()
			return f, nil
		}
		f.saveCurrentField()
		f.inputMode = false
		f.inputBuf = ""
		f.errMsg = ""
		return f.trySubmit()

	case "backspace":
		f.errMsg = ""
		if len(f.inputBuf) > 0 {
			f.inputBuf = f.inputBuf[:len(f.inputBuf)-1]
		}

	default:
		f.errMsg = ""
		if len(msg.String()) == 1 {
			f.inputBuf += msg.String()
		} else if msg.Type == tea.KeySpace {
			f.inputBuf += " "
		}
	}

	return f, nil
}

// View implements tea.Model
func (f *GroupedWizardForm) View() string {
	var sb strings.Builder

	// Tab bar - show required groups first, then optional groups
	var tabs []string
	for i, group := range f.groups {
		label := fmt.Sprintf("%d.%s", i+1, group.Title)

		// Add indicator for optional groups
		if group.Optional {
			if group.Expanded {
				label = fmt.Sprintf("%d.▼ %s", i+1, group.Title)
			} else {
				label = fmt.Sprintf("%d.▶ %s", i+1, group.Title)
			}
		}

		switch {
		case i == f.groupIndex:
			tabs = append(tabs, f.formTheme.TabActive.Render(label))
		case group.Optional && !group.Expanded:
			// Collapsed optional groups are "skipped": show them dimmed instead of as completed.
			tabs = append(tabs, f.formTheme.Help.Render(label))
		case f.isGroupComplete(i):
			tabs = append(tabs, f.formTheme.TabCompleted.Render("✓ "+label))
		default:
			tabs = append(tabs, f.formTheme.TabInactive.Render(label))
		}
	}
	sb.WriteString(strings.Join(tabs, " "))
	sb.WriteString("\n")

	// Separator
	sb.WriteString(f.formTheme.Separator.Render(strings.Repeat("─", minInt(f.width, 70))))
	sb.WriteString("\n\n")

	// Render current group
	group := f.currentGroup()
	if group == nil || len(group.Fields) == 0 {
		sb.WriteString("(No fields in this group)\n")
	} else if group.Optional && !group.Expanded {
		// Collapsed optional group - show toggle prompt
		sb.WriteString(f.formTheme.Description.Render(fmt.Sprintf("  %s (Optional)", group.Title)))
		sb.WriteString("\n\n")
		sb.WriteString(f.formTheme.Help.Render("  Press Enter or Space to expand, or Tab to skip"))
		sb.WriteString("\n")
	} else {
		// Show all fields in current group
		for i, field := range group.Fields {
			sb.WriteString(f.renderField(field, i == f.fieldIndex))
			sb.WriteString("\n")
		}
	}

	// Error message
	if strings.TrimSpace(f.errMsg) != "" {
		sb.WriteString("\n")
		sb.WriteString(f.formTheme.Error.Render("Error: " + f.errMsg))
	}

	// Help text
	sb.WriteString("\n")
	sb.WriteString(f.renderHelp())

	return sb.String()
}

// renderField renders a single field with all its options visible
func (f *GroupedWizardForm) renderField(field *FormField, isFocused bool) string {
	var sb strings.Builder

	// Title with focus indicator
	if isFocused {
		sb.WriteString(f.formTheme.FocusedTitle.Render("> " + field.Title))
	} else {
		sb.WriteString(f.formTheme.NormalTitle.Render("  " + field.Title))
	}

	// Description on same line if short
	if field.Description != "" && len(field.Description) < 40 {
		sb.WriteString(f.formTheme.Description.Render("  " + field.Description))
	}
	sb.WriteString("\n")

	// Render options based on field kind
	sb.WriteString("  ")
	switch field.Kind {
	case KindSelect:
		sb.WriteString(f.renderSelectOptions(field, isFocused))
	case KindMultiSelect:
		sb.WriteString(f.renderMultiSelectOptions(field, isFocused))
	case KindConfirm:
		sb.WriteString(f.renderConfirmOptions(field, isFocused))
	case KindInput, KindNumber:
		sb.WriteString(f.renderInputField(field, isFocused))
	}

	return sb.String()
}

// selectOptionStyle returns the appropriate style based on focus and selection state
func (f *GroupedWizardForm) selectOptionStyle(isFocused, isSelected bool) lipgloss.Style {
	if isSelected {
		return f.formTheme.SelectedOption
	}
	if isFocused {
		return f.formTheme.FocusedUnselected
	}
	return f.formTheme.UnselectedOption
}

func (f *GroupedWizardForm) renderSelectOptions(field *FormField, isFocused bool) string {
	parts := make([]string, 0, len(field.Options))
	for i, opt := range field.Options {
		display := opt
		if display == "" {
			display = "(empty)"
		}
		style := f.selectOptionStyle(isFocused, i == field.Selected)
		parts = append(parts, style.Render(display))
	}
	return strings.Join(parts, " ")
}

func (f *GroupedWizardForm) renderMultiSelectOptions(field *FormField, isFocused bool) string {
	parts := make([]string, 0, len(field.Options))
	for i, opt := range field.Options {
		marker := "○"
		if field.MultiSelect[i] {
			marker = "●"
		}
		display := fmt.Sprintf("%s %s", marker, opt)
		isCursor := isFocused && i == f.cursor

		var style lipgloss.Style
		switch {
		case isCursor:
			style = f.formTheme.SelectedOption
		case field.MultiSelect[i]:
			style = f.formTheme.MultiSelectChecked
		case isFocused:
			style = f.formTheme.FocusedUnselected
		default:
			style = f.formTheme.UnselectedOption
		}
		parts = append(parts, style.Render(display))
	}
	return strings.Join(parts, " ")
}

func (f *GroupedWizardForm) renderConfirmOptions(field *FormField, isFocused bool) string {
	yesStyle := f.selectOptionStyle(isFocused, field.ConfirmVal)
	noStyle := f.selectOptionStyle(isFocused, !field.ConfirmVal)
	return yesStyle.Render("Yes") + " " + noStyle.Render("No")
}

func (f *GroupedWizardForm) renderInputField(field *FormField, isFocused bool) string {
	if isFocused && f.inputMode {
		return f.formTheme.InputFocused.Render("[" + f.inputBuf + "█]")
	}
	display := field.InputValue
	if display == "" {
		display = "(empty)"
	}
	if isFocused {
		return f.formTheme.InputFocused.Render("["+display+"]") + f.formTheme.Description.Render("  Enter to edit")
	}
	return f.formTheme.InputBlurred.Render("[" + display + "]")
}

func (f *GroupedWizardForm) renderHelp() string {
	group := f.currentGroup()

	// Check if current group is a collapsed optional group
	if group != nil && group.Optional && !group.Expanded {
		return f.formTheme.Help.Render("Enter/Space: expand  Tab: skip group  1-9: jump  Ctrl+D: submit")
	}

	// Check if current group is an expanded optional group
	if group != nil && group.Optional && group.Expanded {
		field := f.currentField()
		baseHelp := "↑/↓: field  c: collapse  Tab: group  "
		if field == nil {
			return f.formTheme.Help.Render(baseHelp + "Ctrl+D: submit")
		}
		switch field.Kind {
		case KindMultiSelect:
			return f.formTheme.Help.Render(baseHelp + "Space: toggle  a: all  Ctrl+D: submit")
		case KindConfirm:
			return f.formTheme.Help.Render(baseHelp + "←/→: toggle  Ctrl+D: submit")
		case KindInput, KindNumber:
			if f.inputMode {
				return f.formTheme.Help.Render("Enter: save  Esc: cancel  Ctrl+D: save & submit")
			}
			return f.formTheme.Help.Render(baseHelp + "Enter: edit  Ctrl+D: submit")
		default:
			return f.formTheme.Help.Render(baseHelp + "←/→: select  Ctrl+D: submit")
		}
	}

	field := f.currentField()
	if field == nil {
		return f.formTheme.Help.Render("Tab: next group  Shift+Tab: prev group  1-9: jump to group  Ctrl+D: submit")
	}

	baseHelp := "↑/↓: field  Tab/Shift+Tab: group  1-9: jump  "

	switch field.Kind {
	case KindMultiSelect:
		return f.formTheme.Help.Render(baseHelp + "←/→: move  Space: toggle  a: all  n: none  Ctrl+D: submit")
	case KindConfirm:
		return f.formTheme.Help.Render(baseHelp + "←/→: toggle  y: Yes  n: No  Ctrl+D: submit")
	case KindInput, KindNumber:
		if f.inputMode {
			return f.formTheme.Help.Render("Enter: save  Esc: cancel  Ctrl+D: save & submit")
		}
		return f.formTheme.Help.Render(baseHelp + "Enter: edit  Ctrl+D: submit")
	default:
		return f.formTheme.Help.Render(baseHelp + "←/→: select  Ctrl+D: submit")
	}
}

// Helper methods

func (f *GroupedWizardForm) currentGroup() *FormGroup {
	if f.groupIndex >= 0 && f.groupIndex < len(f.groups) {
		return f.groups[f.groupIndex]
	}
	return nil
}

func (f *GroupedWizardForm) currentField() *FormField {
	group := f.currentGroup()
	if group == nil {
		return nil
	}
	if f.fieldIndex >= 0 && f.fieldIndex < len(group.Fields) {
		return group.Fields[f.fieldIndex]
	}
	return nil
}

func (f *GroupedWizardForm) nextGroup() {
	f.groupIndex++
	if f.groupIndex >= len(f.groups) {
		f.groupIndex = 0
	}
	f.fieldIndex = 0
	f.initCursorForField()
}

func (f *GroupedWizardForm) prevGroup() {
	f.groupIndex--
	if f.groupIndex < 0 {
		f.groupIndex = len(f.groups) - 1
	}
	f.fieldIndex = 0
	f.initCursorForField()
}

func (f *GroupedWizardForm) nextField() {
	group := f.currentGroup()
	if group == nil {
		return
	}
	f.fieldIndex++
	if f.fieldIndex >= len(group.Fields) {
		f.fieldIndex = 0
	}
	f.initCursorForField()
}

func (f *GroupedWizardForm) prevField() {
	group := f.currentGroup()
	if group == nil {
		return
	}
	f.fieldIndex--
	if f.fieldIndex < 0 {
		f.fieldIndex = len(group.Fields) - 1
	}
	f.initCursorForField()
}

func (f *GroupedWizardForm) initCursorForField() {
	field := f.currentField()
	if field == nil {
		f.cursor = 0
		return
	}
	switch field.Kind {
	case KindSelect:
		f.cursor = field.Selected
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

// wrapIndex wraps index in range [0, max) with cycling
func wrapIndex(index, delta, max int) int {
	if max <= 0 {
		return 0
	}
	return (index + delta + max) % max
}

func (f *GroupedWizardForm) nextOption() {
	field := f.currentField()
	if field == nil {
		return
	}
	switch field.Kind {
	case KindSelect:
		f.cursor = wrapIndex(f.cursor, 1, len(field.Options))
		field.Selected = f.cursor
		f.saveCurrentField()
	case KindMultiSelect:
		f.cursor = wrapIndex(f.cursor, 1, len(field.Options))
	case KindConfirm:
		f.cursor = 1 - f.cursor
		f.saveCurrentField()
	case KindInput, KindNumber:
		if !f.inputMode {
			f.saveCurrentField()
			f.nextField()
		}
	}
}

func (f *GroupedWizardForm) prevOption() {
	field := f.currentField()
	if field == nil {
		return
	}
	switch field.Kind {
	case KindSelect:
		f.cursor = wrapIndex(f.cursor, -1, len(field.Options))
		field.Selected = f.cursor
		f.saveCurrentField()
	case KindMultiSelect:
		f.cursor = wrapIndex(f.cursor, -1, len(field.Options))
	case KindConfirm:
		f.cursor = 1 - f.cursor
		f.saveCurrentField()
	case KindInput, KindNumber:
		if !f.inputMode {
			f.saveCurrentField()
			f.prevField()
		}
	}
}

func (f *GroupedWizardForm) ensureMultiSelect(field *FormField) {
	if field.MultiSelect == nil {
		field.MultiSelect = make(map[int]bool)
	}
}

func (f *GroupedWizardForm) toggleSelection() {
	field := f.currentField()
	if field == nil {
		return
	}
	f.ensureMultiSelect(field)
	field.MultiSelect[f.cursor] = !field.MultiSelect[f.cursor]
	f.saveCurrentField()
}

func (f *GroupedWizardForm) selectAll() {
	field := f.currentField()
	if field == nil {
		return
	}
	f.ensureMultiSelect(field)
	for i := range field.Options {
		field.MultiSelect[i] = true
	}
	f.saveCurrentField()
}

func (f *GroupedWizardForm) deselectAll() {
	field := f.currentField()
	if field == nil {
		return
	}
	field.MultiSelect = make(map[int]bool)
	f.saveCurrentField()
}

func (f *GroupedWizardForm) saveCurrentField() {
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
		field.ConfirmVal = (f.cursor == 0)
		if ptr, ok := field.Value.(*bool); ok && ptr != nil {
			*ptr = field.ConfirmVal
		}
	case KindInput, KindNumber:
		if ptr, ok := field.Value.(*string); ok && ptr != nil {
			*ptr = field.InputValue
		}
	}
}

func (f *GroupedWizardForm) isGroupComplete(groupIdx int) bool {
	if groupIdx < 0 || groupIdx >= len(f.groups) {
		return false
	}
	group := f.groups[groupIdx]

	// Collapsed optional groups are considered "complete" (skipped)
	if group.Optional && !group.Expanded {
		return true
	}

	for _, field := range group.Fields {
		if err := f.validateField(field); err != nil {
			return false
		}
	}
	return true
}

func (f *GroupedWizardForm) trySubmit() (tea.Model, tea.Cmd) {
	f.saveCurrentField()
	if err := f.validateAllFields(); err != nil {
		return f, nil
	}
	f.quitting = true
	return f, tea.Quit
}

func (f *GroupedWizardForm) validateAllFields() error {
	for gi, group := range f.groups {
		// Skip collapsed optional groups (user chose to skip)
		if group.Optional && !group.Expanded {
			continue
		}

		for fi, field := range group.Fields {
			if err := f.validateField(field); err != nil {
				f.errMsg = err.Error()
				f.inputMode = false
				f.inputBuf = ""
				f.groupIndex = gi
				f.fieldIndex = fi
				f.initCursorForField()
				return err
			}
		}
	}
	f.errMsg = ""
	return nil
}

// validateStringField validates string-like fields (Select, Input)
func (f *GroupedWizardForm) validateStringField(value string, field *FormField, label string) error {
	if !field.Required && field.Validate == nil {
		return nil
	}
	var required func(string) error
	if field.Required {
		required = requiredStringValidator(label)
	}
	return chainStringValidators(required, field.Validate)(value)
}


func (f *GroupedWizardForm) validateField(field *FormField) error {
	if field == nil {
		return nil
	}

	label := field.Title
	if strings.TrimSpace(label) == "" {
		label = field.Name
	}

	switch field.Kind {
	case KindSelect:
		val := ""
		if field.Selected >= 0 && field.Selected < len(field.Options) {
			val = field.Options[field.Selected]
		}
		return f.validateStringField(val, field, label)

	case KindMultiSelect:
		if !field.Required {
			return nil
		}
		for _, selected := range field.MultiSelect {
			if selected {
				return nil
			}
		}
		return requiredStringValidator(label)("")

	case KindInput:
		return f.validateStringField(field.InputValue, field, label)

	case KindNumber:
		s := strings.TrimSpace(field.InputValue)
		if s == "" {
			if field.Required {
				return requiredStringValidator(label)("")
			}
			return nil
		}
		if field.Validate != nil {
			if err := field.Validate(s); err != nil {
				return err
			}
			return nil
		}
		if _, err := strconv.Atoi(s); err != nil {
			return fmt.Errorf("please enter a valid number")
		}
		return nil

	case KindConfirm:
		return nil
	default:
		return nil
	}
}

// Run executes the grouped form
func (f *GroupedWizardForm) Run() error {
	// Prevent lipgloss from sending OSC terminal queries (like \x1b]11;?)
	// which can conflict with readline's input handling and cause garbled output.
	// We set HasDarkBackground once at startup to avoid runtime OSC queries.
	lipglossInitOnce.Do(func() {
		lipgloss.SetHasDarkBackground(true)
	})

	p := tea.NewProgram(f)
	_, err := p.Run()
	if err != nil {
		return err
	}
	if f.aborted {
		return fmt.Errorf("wizard aborted")
	}
	// Final save of all fields
	for gi := range f.groups {
		for fi := range f.groups[gi].Fields {
			f.groupIndex = gi
			f.fieldIndex = fi
			f.initCursorForField()
			f.saveCurrentField()
		}
	}
	return nil
}

// Aborted returns true if the user cancelled
func (f *GroupedWizardForm) Aborted() bool {
	return f.aborted
}

// requiredStringValidator creates a validator that checks for non-empty strings
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

// chainStringValidators chains multiple string validators together
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
