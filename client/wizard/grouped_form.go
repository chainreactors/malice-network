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
	NormalTitle          lipgloss.Style
	Description         lipgloss.Style
	SelectedOption      lipgloss.Style
	UnselectedOption    lipgloss.Style
	FocusedUnselected   lipgloss.Style
	MultiSelectChecked  lipgloss.Style
	InputFocused        lipgloss.Style
	InputBlurred        lipgloss.Style
	GroupHeader         lipgloss.Style
	GroupHeaderDim      lipgloss.Style
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
		GroupHeader:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		GroupHeaderDim:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	}
}

// Package-level default theme instance
var defaultFormTheme = DefaultFormTheme()

// defaultTerminalWidth is the fallback width when terminal size cannot be determined
const defaultTerminalWidth = 80
const defaultTerminalHeight = 40

// getTerminalWidth returns the current terminal width or a default value
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return defaultTerminalWidth
	}
	return width
}

// getTerminalHeight returns the current terminal height or a default value
func getTerminalHeight() int {
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || height <= 0 {
		return defaultTerminalHeight
	}
	return height
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

// GroupedWizardForm is a single-page wizard form with all groups visible
type GroupedWizardForm struct {
	groups     []*FormGroup
	groupIndex int // Current group being edited

	// Current field within group
	fieldIndex  int
	cursor      int  // Cursor within field options
	onSubmitBtn bool // True when focus is on the Submit button at the bottom

	inputMode   bool
	inputBuf    string
	inputCurPos int

	scrollOffset int // Viewport scroll offset (in lines)
	width        int
	height       int
	theme        *huh.Theme
	formTheme    *FormTheme
	quitting     bool
	aborted      bool

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
		height:     getTerminalHeight(),
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

		// Handle submit button focus
		if f.onSubmitBtn {
			return f.handleSubmitBtn(msg)
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
				f.onSubmitBtn = false
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
			// Next group (quick jump)
			f.errMsg = ""
			f.saveCurrentField()
			f.nextGroup()

		case "shift+tab":
			// Previous group (quick jump)
			f.errMsg = ""
			f.saveCurrentField()
			f.prevGroup()

		case "up", "k":
			f.errMsg = ""
			f.saveCurrentField()
			f.prevField()

		case "down", "j":
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
			// Select/Confirm/MultiSelect: advance to next field
			f.errMsg = ""
			f.saveCurrentField()
			f.nextField()

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

	case "enter", "down", "j":
		if !f.commitInput() {
			return f, nil
		}
		f.nextField()

	case "up", "k":
		if !f.commitInput() {
			return f, nil
		}
		f.prevField()

	case "tab":
		if !f.commitInput() {
			return f, nil
		}
		f.nextGroup()

	case "shift+tab":
		if !f.commitInput() {
			return f, nil
		}
		f.prevGroup()

	case "ctrl+d":
		if !f.commitInput() {
			return f, nil
		}
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

// commitInput validates and saves the current input buffer, exits input mode.
// Returns true on success, false on validation error (stays in input mode).
func (f *GroupedWizardForm) commitInput() bool {
	field := f.currentField()
	if field == nil {
		f.inputMode = false
		return true
	}
	candidate := f.inputBuf
	old := field.InputValue
	field.InputValue = candidate
	if err := f.validateField(field); err != nil {
		field.InputValue = old
		f.errMsg = err.Error()
		return false
	}
	f.saveCurrentField()
	f.inputMode = false
	f.inputBuf = ""
	f.errMsg = ""
	return true
}

// handleSubmitBtn handles key events when focus is on the Submit button
func (f *GroupedWizardForm) handleSubmitBtn(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		f.aborted = true
		f.quitting = true
		return f, tea.Quit
	case "enter", " ", "ctrl+d":
		return f.trySubmit()
	case "up", "k":
		f.errMsg = ""
		f.onSubmitBtn = false
		// Go back to last visible field
		f.goToLastField()
	case "tab":
		f.errMsg = ""
		f.onSubmitBtn = false
		f.groupIndex = 0
		f.fieldIndex = 0
		f.initCursorForField()
	case "shift+tab":
		f.errMsg = ""
		f.onSubmitBtn = false
		f.goToLastField()
	default:
		// Number keys for group jump
		key := msg.String()
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			groupNum := int(key[0] - '1')
			if groupNum < len(f.groups) {
				f.errMsg = ""
				f.onSubmitBtn = false
				f.groupIndex = groupNum
				f.fieldIndex = 0
				f.initCursorForField()
			}
		}
	}
	return f, nil
}

// goToLastField moves focus to the last visible field or collapsed group
func (f *GroupedWizardForm) goToLastField() {
	for gi := len(f.groups) - 1; gi >= 0; gi-- {
		group := f.groups[gi]
		if group.Optional && !group.Expanded {
			f.groupIndex = gi
			f.fieldIndex = 0
			f.initCursorForField()
			return
		}
		if len(group.Fields) > 0 {
			f.groupIndex = gi
			f.fieldIndex = len(group.Fields) - 1
			f.initCursorForField()
			return
		}
	}
}

// View implements tea.Model - renders all groups on a single page
func (f *GroupedWizardForm) View() string {
	var sb strings.Builder

	// Status bar - compact group indicators
	var tabs []string
	for i, group := range f.groups {
		label := fmt.Sprintf("%d.%s", i+1, group.Title)
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
			tabs = append(tabs, f.formTheme.Help.Render(label))
		case f.isGroupComplete(i):
			tabs = append(tabs, f.formTheme.TabCompleted.Render("✓ "+label))
		default:
			tabs = append(tabs, f.formTheme.TabInactive.Render(label))
		}
	}
	sb.WriteString(strings.Join(tabs, " "))
	sb.WriteString("\n")
	sb.WriteString(f.formTheme.Separator.Render(strings.Repeat("─", minInt(f.width, 70))))
	sb.WriteString("\n")

	// Render ALL groups
	focusLineStart := 0
	lineCount := 0

	for gi, group := range f.groups {
		isCurrent := gi == f.groupIndex

		if group.Optional && !group.Expanded {
			// Collapsed optional group - single line
			if isCurrent {
				focusLineStart = lineCount
				sb.WriteString(f.formTheme.GroupHeader.Render(fmt.Sprintf("> ▶ %s (Optional)", group.Title)))
				sb.WriteString(f.formTheme.Description.Render("  Enter to expand"))
			} else {
				sb.WriteString(f.formTheme.GroupHeaderDim.Render(fmt.Sprintf("  ▶ %s (Optional)", group.Title)))
			}
			sb.WriteString("\n")
			lineCount++
		} else {
			// Group section header
			headerStyle := f.formTheme.GroupHeaderDim
			if isCurrent {
				headerStyle = f.formTheme.GroupHeader
			}
			sb.WriteString(headerStyle.Render(fmt.Sprintf("━━ %s ━━", group.Title)))
			sb.WriteString("\n")
			lineCount++

			// Render all fields in this group
			for fi, field := range group.Fields {
				isFocused := isCurrent && fi == f.fieldIndex
				if isFocused {
					focusLineStart = lineCount
				}
				rendered := f.renderField(field, isFocused)
				sb.WriteString(rendered)
				sb.WriteString("\n")
				lineCount += strings.Count(rendered, "\n") + 1
			}
		}
	}

	// Submit button
	sb.WriteString("\n")
	if f.onSubmitBtn {
		focusLineStart = lineCount
		sb.WriteString(f.formTheme.SelectedOption.Render("  [ Submit ]"))
	} else {
		sb.WriteString(f.formTheme.UnselectedOption.Render("  [ Submit ]"))
	}
	sb.WriteString("\n")
	lineCount++

	// Error message
	if strings.TrimSpace(f.errMsg) != "" {
		sb.WriteString("\n")
		sb.WriteString(f.formTheme.Error.Render("Error: " + f.errMsg))
	}

	// Help text
	sb.WriteString("\n")
	sb.WriteString(f.renderHelp())

	// Apply viewport scrolling
	return f.applyScroll(sb.String(), focusLineStart)
}

// applyScroll applies viewport scrolling to keep the focused line visible
func (f *GroupedWizardForm) applyScroll(content string, focusLine int) string {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Reserve lines for status bar (2) + help (2) + error (2)
	visibleHeight := f.height - 2
	if visibleHeight <= 0 || totalLines <= visibleHeight {
		return content
	}

	// Ensure focused line is visible with some padding
	padding := 3
	if focusLine < f.scrollOffset+padding {
		f.scrollOffset = maxInt(0, focusLine-padding)
	}
	if focusLine >= f.scrollOffset+visibleHeight-padding {
		f.scrollOffset = minInt(totalLines-visibleHeight, focusLine-visibleHeight+padding+1)
	}

	// Clamp
	if f.scrollOffset < 0 {
		f.scrollOffset = 0
	}
	if f.scrollOffset+visibleHeight > totalLines {
		f.scrollOffset = maxInt(0, totalLines-visibleHeight)
	}

	end := minInt(f.scrollOffset+visibleHeight, totalLines)
	visible := lines[f.scrollOffset:end]

	// Add scroll indicator if not showing everything
	if f.scrollOffset > 0 {
		visible[0] = f.formTheme.Help.Render("▲ scroll up")
	}
	if end < totalLines {
		visible[len(visible)-1] = f.formTheme.Help.Render("▼ scroll down")
	}

	return strings.Join(visible, "\n")
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
		return f.formTheme.InputFocused.Render("[" + display + "]")
	}
	return f.formTheme.InputBlurred.Render("[" + display + "]")
}

func (f *GroupedWizardForm) renderHelp() string {
	if f.onSubmitBtn {
		return f.formTheme.Help.Render("Enter: submit  ↑: go back  1-9: jump group  Esc: cancel")
	}

	group := f.currentGroup()

	// Check if current group is a collapsed optional group
	if group != nil && group.Optional && !group.Expanded {
		return f.formTheme.Help.Render("Enter/Space: expand  ↑/↓: navigate  Tab: jump group  Ctrl+D: submit")
	}

	field := f.currentField()
	if field == nil {
		return f.formTheme.Help.Render("↑/↓: navigate  Tab: jump group  1-9: jump  Ctrl+D: submit")
	}

	baseHelp := "↑/↓: field  Tab: jump group  "

	// Check if current group is an expanded optional group
	if group != nil && group.Optional && group.Expanded {
		baseHelp = "↑/↓: field  c: collapse  Tab: jump group  "
	}

	switch field.Kind {
	case KindMultiSelect:
		return f.formTheme.Help.Render(baseHelp + "Space: toggle  a: all  n: none  Ctrl+D: submit")
	case KindConfirm:
		return f.formTheme.Help.Render(baseHelp + "←/→: toggle  y/n  Enter: next  Ctrl+D: submit")
	case KindInput, KindNumber:
		return f.formTheme.Help.Render("Type to edit  Enter/↑/↓: save & move  Esc: discard  Ctrl+D: submit")
	default:
		return f.formTheme.Help.Render(baseHelp + "←/→: select  Enter: next  Ctrl+D: submit")
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
	f.onSubmitBtn = false
	f.groupIndex++
	if f.groupIndex >= len(f.groups) {
		f.groupIndex = 0
	}
	f.fieldIndex = 0
	f.initCursorForField()
}

func (f *GroupedWizardForm) prevGroup() {
	f.onSubmitBtn = false
	f.groupIndex--
	if f.groupIndex < 0 {
		f.groupIndex = len(f.groups) - 1
	}
	f.fieldIndex = 0
	f.initCursorForField()
}

// nextField moves to the next field, crossing group boundaries
func (f *GroupedWizardForm) nextField() {
	group := f.currentGroup()
	if group == nil {
		return
	}

	// Collapsed optional group - move to next group or submit button
	if group.Optional && !group.Expanded {
		if f.groupIndex < len(f.groups)-1 {
			f.groupIndex++
			f.fieldIndex = 0
			f.initCursorForField()
		} else {
			f.onSubmitBtn = true
		}
		return
	}

	f.fieldIndex++
	if f.fieldIndex < len(group.Fields) {
		f.initCursorForField()
		return
	}

	// Cross to next group or submit button
	if f.groupIndex < len(f.groups)-1 {
		f.groupIndex++
		f.fieldIndex = 0
		f.initCursorForField()
	} else {
		// Past the last field → focus submit button
		f.fieldIndex = len(group.Fields) - 1
		f.onSubmitBtn = true
	}
}

// prevField moves to the previous field, crossing group boundaries
func (f *GroupedWizardForm) prevField() {
	// If on submit button, go back to last field
	if f.onSubmitBtn {
		f.onSubmitBtn = false
		f.goToLastField()
		return
	}

	group := f.currentGroup()
	if group == nil {
		return
	}

	// Collapsed optional group - move to previous group
	if group.Optional && !group.Expanded {
		if f.groupIndex > 0 {
			f.groupIndex--
			prevGroup := f.groups[f.groupIndex]
			if prevGroup.Optional && !prevGroup.Expanded {
				f.fieldIndex = 0
			} else if len(prevGroup.Fields) > 0 {
				f.fieldIndex = len(prevGroup.Fields) - 1
			} else {
				f.fieldIndex = 0
			}
			f.initCursorForField()
		}
		return
	}

	f.fieldIndex--
	if f.fieldIndex >= 0 {
		f.initCursorForField()
		return
	}

	// Cross to previous group
	if f.groupIndex > 0 {
		f.groupIndex--
		prevGroup := f.groups[f.groupIndex]
		if prevGroup.Optional && !prevGroup.Expanded {
			f.fieldIndex = 0
		} else if len(prevGroup.Fields) > 0 {
			f.fieldIndex = len(prevGroup.Fields) - 1
		} else {
			f.fieldIndex = 0
		}
		f.initCursorForField()
	} else {
		// Stay at first field of first group
		f.fieldIndex = 0
	}
}

func (f *GroupedWizardForm) initCursorForField() {
	field := f.currentField()
	if field == nil {
		f.cursor = 0
		f.inputMode = false
		return
	}
	switch field.Kind {
	case KindSelect:
		f.cursor = field.Selected
		f.inputMode = false
	case KindConfirm:
		if field.ConfirmVal {
			f.cursor = 0
		} else {
			f.cursor = 1
		}
		f.inputMode = false
	case KindInput, KindNumber:
		f.cursor = 0
		f.inputMode = true
		f.inputBuf = field.InputValue
		f.inputCurPos = len(f.inputBuf)
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
