package wizard

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Command functions to emit huh field navigation messages
func nextFieldCmd() tea.Msg {
	return huh.NextField()
}

func prevFieldCmd() tea.Msg {
	return huh.PrevField()
}

// HorizontalMultiSelect is a custom multi-select field that displays options horizontally
type HorizontalMultiSelect struct {
	title       string
	description string
	options     []string
	selected    map[int]bool
	cursor      int
	focused     bool
	width       int
	height      int
	theme       *huh.Theme
	keyMap      *huh.KeyMap
	accessible  bool
	position    huh.FieldPosition
	validate    func([]string) error
	err         error
	fieldKey    string

	// Reference to store the result
	value *[]string
}

// NewHorizontalMultiSelect creates a new horizontal multi-select field
func NewHorizontalMultiSelect(options []string) *HorizontalMultiSelect {
	return &HorizontalMultiSelect{
		options:  options,
		selected: make(map[int]bool),
		cursor:   0,
		focused:  false,
		width:    80,
	}
}

// Title sets the title of the field
func (m *HorizontalMultiSelect) Title(title string) *HorizontalMultiSelect {
	m.title = title
	return m
}

// Description sets the description of the field
func (m *HorizontalMultiSelect) Description(desc string) *HorizontalMultiSelect {
	m.description = desc
	return m
}

// Value sets the pointer to store the selected values
func (m *HorizontalMultiSelect) Value(value *[]string) *HorizontalMultiSelect {
	m.value = value
	// Initialize selected based on existing values
	if value != nil && *value != nil {
		for _, v := range *value {
			for i, opt := range m.options {
				if opt == v {
					m.selected[i] = true
					break
				}
			}
		}
	}
	return m
}

// Key sets the field key
func (m *HorizontalMultiSelect) Key(key string) *HorizontalMultiSelect {
	m.fieldKey = key
	return m
}

// Validate sets the validation function
func (m *HorizontalMultiSelect) Validate(fn func([]string) error) *HorizontalMultiSelect {
	m.validate = fn
	return m
}

// Init implements tea.Model
func (m *HorizontalMultiSelect) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *HorizontalMultiSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.err = nil
		switch msg.String() {
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case " ":
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.updateValue()
		case "a":
			// Select all
			for i := range m.options {
				m.selected[i] = true
			}
			m.updateValue()
		case "n":
			// Deselect all
			m.selected = make(map[int]bool)
			m.updateValue()
		case "enter", "tab", "shift+tab":
			m.updateValue()
			if msg.String() == "tab" {
				if m.validate != nil {
					m.err = m.validate(m.getSelectedValues())
					if m.err != nil {
						return m, nil
					}
				}
				return m, nextFieldCmd
			}
			if msg.String() == "shift+tab" {
				if m.validate != nil {
					m.err = m.validate(m.getSelectedValues())
					if m.err != nil {
						return m, nil
					}
				}
				return m, prevFieldCmd
			}
			if m.validate != nil {
				m.err = m.validate(m.getSelectedValues())
				if m.err != nil {
					return m, nil
				}
			}
			return m, nextFieldCmd
		}
	}
	return m, nil
}

// updateValue syncs the selected values back to the value pointer
func (m *HorizontalMultiSelect) updateValue() {
	if m.value != nil {
		*m.value = m.getSelectedValues()
	}
}

// getSelectedValues returns the list of selected option values
func (m *HorizontalMultiSelect) getSelectedValues() []string {
	var result []string
	for i, opt := range m.options {
		if m.selected[i] {
			result = append(result, opt)
		}
	}
	return result
}

// View implements tea.Model
func (m *HorizontalMultiSelect) View() string {
	var sb strings.Builder

	// Get styles from theme or use defaults
	var titleStyle, descStyle lipgloss.Style

	if m.theme != nil {
		if m.focused {
			titleStyle = m.theme.Focused.Title
			descStyle = m.theme.Focused.Description
		} else {
			titleStyle = m.theme.Blurred.Title
			descStyle = m.theme.Blurred.Description
		}
	} else {
		titleStyle = lipgloss.NewStyle().Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	}

	// Claude Code plan mode style: background highlight for current item
	cursorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).   // black text
		Background(lipgloss.Color("212")). // pink/magenta background
		Padding(0, 1)

	// Arrow, index, and selected count styles
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	indexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedCountStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// Title
	if m.title != "" {
		sb.WriteString(titleStyle.Render(m.title))
		sb.WriteString("\n")
	}

	// Description
	if m.description != "" {
		sb.WriteString(descStyle.Render(m.description))
		sb.WriteString("\n")
	}

	// Single-line carousel style display
	// Left arrow (if not first option)
	if m.cursor > 0 {
		sb.WriteString(arrowStyle.Render(" ◀ "))
	} else {
		sb.WriteString("   ")
	}

	// Current option with selection marker
	marker := "○"
	if m.selected[m.cursor] {
		marker = "●"
	}
	label := fmt.Sprintf("%s %s", marker, m.options[m.cursor])
	sb.WriteString(cursorStyle.Render(label))

	// Right arrow (if not last option)
	if m.cursor < len(m.options)-1 {
		sb.WriteString(arrowStyle.Render(" ▶ "))
	} else {
		sb.WriteString("   ")
	}

	// Position indicator (n/total)
	sb.WriteString(indexStyle.Render(fmt.Sprintf(" (%d/%d)", m.cursor+1, len(m.options))))

	// Selected count
	selectedCount := len(m.getSelectedValues())
	if selectedCount > 0 {
		sb.WriteString(selectedCountStyle.Render(fmt.Sprintf("  已选: %d 项", selectedCount)))
	}

	// Error message
	if m.err != nil {
		sb.WriteString("\n")
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		sb.WriteString(errStyle.Render(m.err.Error()))
	}

	return sb.String()
}

// Focus implements huh.Field
func (m *HorizontalMultiSelect) Focus() tea.Cmd {
	m.focused = true
	return nil
}

// Blur implements huh.Field
func (m *HorizontalMultiSelect) Blur() tea.Cmd {
	m.focused = false
	m.updateValue()
	if m.validate != nil {
		m.err = m.validate(m.getSelectedValues())
	}
	return nil
}

// Error implements huh.Field
func (m *HorizontalMultiSelect) Error() error {
	return m.err
}

// Run implements huh.Field
func (m *HorizontalMultiSelect) Run() error {
	if m.accessible { // TODO: remove in a future release (parity with huh fields).
		return m.RunAccessible(os.Stdout, os.Stdin)
	}
	return huh.Run(m)
}

// RunAccessible implements huh.Field
func (m *HorizontalMultiSelect) RunAccessible(w io.Writer, r io.Reader) error {
	fmt.Fprintf(w, "%s\n", m.title)
	if m.description != "" {
		fmt.Fprintf(w, "%s\n", m.description)
	}

	scanner := bufio.NewScanner(r)
	for {
		fmt.Fprintln(w, "Options (comma-separated numbers, empty to continue):")
		for i, opt := range m.options {
			marker := " "
			if m.selected[i] {
				marker = "*"
			}
			fmt.Fprintf(w, "  [%s] %d. %s\n", marker, i+1, opt)
		}

		fmt.Fprint(w, "> ")
		if !scanner.Scan() {
			m.updateValue()
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			nextSelected := make(map[int]bool)
			var invalidToken string
			for _, part := range strings.Split(line, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				num, err := strconv.Atoi(part)
				if err != nil || num <= 0 || num > len(m.options) {
					invalidToken = part
					break
				}
				nextSelected[num-1] = true
			}
			if invalidToken != "" {
				fmt.Fprintf(w, "Invalid option: %q\n\n", invalidToken)
				continue
			}
			m.selected = nextSelected
		}

		m.updateValue()
		if m.validate != nil {
			m.err = m.validate(m.getSelectedValues())
			if m.err != nil {
				fmt.Fprintf(w, "%s\n\n", m.err)
				continue
			}
		}
		return nil
	}
}

// Skip implements huh.Field
func (m *HorizontalMultiSelect) Skip() bool {
	return false
}

// Zoom implements huh.Field
func (m *HorizontalMultiSelect) Zoom() bool {
	return false
}

// KeyBinds implements huh.Field
func (m *HorizontalMultiSelect) KeyBinds() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev")),
		key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next")),
		key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all")),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "none")),
		key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter/tab", "confirm")),
		key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
	}
}

// WithTheme implements huh.Field
func (m *HorizontalMultiSelect) WithTheme(theme *huh.Theme) huh.Field {
	m.theme = theme
	return m
}

// WithAccessible implements huh.Field
func (m *HorizontalMultiSelect) WithAccessible(accessible bool) huh.Field {
	m.accessible = accessible
	return m
}

// WithKeyMap implements huh.Field
func (m *HorizontalMultiSelect) WithKeyMap(keyMap *huh.KeyMap) huh.Field {
	m.keyMap = keyMap
	return m
}

// WithWidth implements huh.Field
func (m *HorizontalMultiSelect) WithWidth(width int) huh.Field {
	m.width = width
	return m
}

// WithHeight implements huh.Field
func (m *HorizontalMultiSelect) WithHeight(height int) huh.Field {
	m.height = height
	return m
}

// WithPosition implements huh.Field
func (m *HorizontalMultiSelect) WithPosition(pos huh.FieldPosition) huh.Field {
	m.position = pos
	return m
}

// GetKey implements huh.Field
func (m *HorizontalMultiSelect) GetKey() string {
	return m.fieldKey
}

// GetValue implements huh.Field
func (m *HorizontalMultiSelect) GetValue() any {
	return m.getSelectedValues()
}
