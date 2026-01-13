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

// HorizontalSelect is a custom select field that displays options horizontally
// Similar to Claude Code's plan mode style
type HorizontalSelect struct {
	title       string
	description string
	options     []string
	cursor      int // cursor position is also the selected item
	focused     bool
	width       int
	height      int
	theme       *huh.Theme
	keyMap      *huh.KeyMap
	accessible  bool
	position    huh.FieldPosition
	validate    func(string) error
	err         error
	fieldKey    string

	// Reference to store the result
	value *string

	// Edit mode state
	editing    bool
	editBuffer string
}

// NewHorizontalSelect creates a new horizontal select field
func NewHorizontalSelect(options []string) *HorizontalSelect {
	return &HorizontalSelect{
		options: options,
		cursor:  0,
		focused: false,
		width:   80,
	}
}

// Title sets the title of the field
func (m *HorizontalSelect) Title(title string) *HorizontalSelect {
	m.title = title
	return m
}

// Description sets the description of the field
func (m *HorizontalSelect) Description(desc string) *HorizontalSelect {
	m.description = desc
	return m
}

// Value sets the pointer to store the selected value
func (m *HorizontalSelect) Value(value *string) *HorizontalSelect {
	m.value = value
	// Initialize cursor based on existing value
	if value != nil && *value != "" {
		for i, opt := range m.options {
			if opt == *value {
				m.cursor = i
				break
			}
		}
	}
	return m
}

// Key sets the field key
func (m *HorizontalSelect) Key(key string) *HorizontalSelect {
	m.fieldKey = key
	return m
}

// Validate sets the validation function
func (m *HorizontalSelect) Validate(fn func(string) error) *HorizontalSelect {
	m.validate = fn
	return m
}

// Init implements tea.Model
func (m *HorizontalSelect) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *HorizontalSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.err = nil

		// Handle edit mode
		if m.editing {
			switch msg.String() {
			case "enter":
				// Finish editing
				m.editing = false
				if m.value != nil {
					*m.value = m.editBuffer
				}
				if m.validate != nil {
					m.err = m.validate(m.editBuffer)
					if m.err != nil {
						return m, nil
					}
				}
				return m, nextFieldCmd
			case "esc":
				// Cancel editing
				m.editing = false
				m.editBuffer = ""
				return m, nil
			case "backspace", "ctrl+h":
				if len(m.editBuffer) > 0 {
					m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
				}
			default:
				// Handle regular character input
				if len(msg.String()) == 1 {
					m.editBuffer += msg.String()
				}
			}
			return m, nil
		}

		// Normal mode
		switch msg.String() {
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
				m.updateValue()
			}
		case "right", "l":
			if m.cursor < len(m.options)-1 {
				m.cursor++
				m.updateValue()
			}
		case "e":
			// Enter edit mode
			m.editing = true
			m.editBuffer = m.getSelectedValue()
			return m, nil
		case "enter", "tab", "shift+tab":
			m.updateValue()
			if m.validate != nil {
				m.err = m.validate(m.getSelectedValue())
				if m.err != nil {
					return m, nil
				}
			}
			if msg.String() == "shift+tab" {
				return m, prevFieldCmd
			}
			return m, nextFieldCmd
		}
	}
	return m, nil
}

// updateValue syncs the selected value back to the value pointer
func (m *HorizontalSelect) updateValue() {
	if m.value != nil {
		*m.value = m.getSelectedValue()
	}
}

// getSelectedValue returns the currently selected option value
func (m *HorizontalSelect) getSelectedValue() string {
	if m.cursor >= 0 && m.cursor < len(m.options) {
		return m.options[m.cursor]
	}
	return ""
}

// View implements tea.Model
func (m *HorizontalSelect) View() string {
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

	// Claude Code plan mode style: background highlight for selected item
	cursorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).   // black text
		Background(lipgloss.Color("212")). // pink/magenta background
		Padding(0, 1)

	// Arrow and index styles
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	indexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

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

	// Edit mode display
	if m.editing {
		editStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("226")). // yellow background for edit
			Padding(0, 1)

		sb.WriteString("   ")
		sb.WriteString(editStyle.Render(m.editBuffer + "▌"))
		sb.WriteString("\n")
		sb.WriteString(hintStyle.Render("   按 Enter 确认, Esc 取消"))
	} else {
		// Single-line carousel style display
		// Left arrow (if not first option)
		if m.cursor > 0 {
			sb.WriteString(arrowStyle.Render(" ◀ "))
		} else {
			sb.WriteString("   ")
		}

		// Current option with highlight
		displayVal := m.options[m.cursor]
		if displayVal == "" {
			displayVal = "(空)"
		}
		sb.WriteString(cursorStyle.Render(displayVal))

		// Right arrow (if not last option)
		if m.cursor < len(m.options)-1 {
			sb.WriteString(arrowStyle.Render(" ▶ "))
		} else {
			sb.WriteString("   ")
		}

		// Position indicator (n/total)
		sb.WriteString(indexStyle.Render(fmt.Sprintf(" (%d/%d)", m.cursor+1, len(m.options))))

		// Edit hint
		sb.WriteString(hintStyle.Render("  按 e 编辑"))
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
func (m *HorizontalSelect) Focus() tea.Cmd {
	m.focused = true
	return nil
}

// Blur implements huh.Field
func (m *HorizontalSelect) Blur() tea.Cmd {
	m.focused = false
	m.updateValue()
	if m.validate != nil {
		m.err = m.validate(m.getSelectedValue())
	}
	return nil
}

// Error implements huh.Field
func (m *HorizontalSelect) Error() error {
	return m.err
}

// Run implements huh.Field
func (m *HorizontalSelect) Run() error {
	if m.accessible { // TODO: remove in a future release (parity with huh fields).
		return m.RunAccessible(os.Stdout, os.Stdin)
	}
	return huh.Run(m)
}

// RunAccessible implements huh.Field
func (m *HorizontalSelect) RunAccessible(w io.Writer, r io.Reader) error {
	fmt.Fprintf(w, "%s\n", m.title)
	if m.description != "" {
		fmt.Fprintf(w, "%s\n", m.description)
	}

	scanner := bufio.NewScanner(r)
	for {
		fmt.Fprintln(w, "Options (enter number to select, empty to continue):")
		for i, opt := range m.options {
			marker := " "
			if i == m.cursor {
				marker = ">"
			}
			fmt.Fprintf(w, "  %s %d. %s\n", marker, i+1, opt)
		}

		fmt.Fprint(w, "> ")
		if !scanner.Scan() {
			m.updateValue()
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			num, err := strconv.Atoi(line)
			if err != nil || num <= 0 || num > len(m.options) {
				fmt.Fprintf(w, "Invalid option: %q\n\n", line)
				continue
			}
			m.cursor = num - 1
		}

		m.updateValue()
		if m.validate != nil {
			m.err = m.validate(m.getSelectedValue())
			if m.err != nil {
				fmt.Fprintf(w, "%s\n\n", m.err)
				continue
			}
		}
		return nil
	}
}

// Skip implements huh.Field
func (m *HorizontalSelect) Skip() bool {
	return false
}

// Zoom implements huh.Field
func (m *HorizontalSelect) Zoom() bool {
	return false
}

// KeyBinds implements huh.Field
func (m *HorizontalSelect) KeyBinds() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev")),
		key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next")),
		key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter/tab", "confirm")),
		key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back")),
	}
}

// WithTheme implements huh.Field
func (m *HorizontalSelect) WithTheme(theme *huh.Theme) huh.Field {
	m.theme = theme
	return m
}

// WithAccessible implements huh.Field
func (m *HorizontalSelect) WithAccessible(accessible bool) huh.Field {
	m.accessible = accessible
	return m
}

// WithKeyMap implements huh.Field
func (m *HorizontalSelect) WithKeyMap(keyMap *huh.KeyMap) huh.Field {
	m.keyMap = keyMap
	return m
}

// WithWidth implements huh.Field
func (m *HorizontalSelect) WithWidth(width int) huh.Field {
	m.width = width
	return m
}

// WithHeight implements huh.Field
func (m *HorizontalSelect) WithHeight(height int) huh.Field {
	m.height = height
	return m
}

// WithPosition implements huh.Field
func (m *HorizontalSelect) WithPosition(pos huh.FieldPosition) huh.Field {
	m.position = pos
	return m
}

// GetKey implements huh.Field
func (m *HorizontalSelect) GetKey() string {
	return m.fieldKey
}

// GetValue implements huh.Field
func (m *HorizontalSelect) GetValue() any {
	return m.getSelectedValue()
}
