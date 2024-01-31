package styles

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TableModel struct {
	table       table.Model
	Style       *table.Styles
	Columns     []table.Column
	Rows        []table.Row
	currentPage int
	totalPages  int
	rowsPerPage int
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240"))

func (t *TableModel) SetDefaultStyle() {
	defaultStyle := table.Styles{
		Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
	}
	defaultStyle.Header = defaultStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	defaultStyle.Selected = defaultStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyle(&defaultStyle)
}

func (t *TableModel) UpdatePagination() {
	t.totalPages = (len(t.Rows) + t.rowsPerPage - 1) / t.rowsPerPage
	if t.currentPage > t.totalPages {
		t.currentPage = t.totalPages
	}
	if t.currentPage < 1 {
		t.currentPage = 1
	}
}

func (t *TableModel) Init() tea.Cmd { return nil }

func (t *TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if t.table.Focused() {
				t.table.Blur()
			} else {
				t.table.Focus()
			}
		case "q", "ctrl+c":
			return t, tea.Quit
		case "enter":
			return t, tea.Quit
		case "n": // Next page
			if t.currentPage < t.totalPages {
				t.currentPage++
			}
		case "p": // Previous page
			if t.currentPage > 1 {
				t.currentPage--
			}
		}
		t.UpdatePagination()
	}
	t.table, cmd = t.table.Update(msg)
	return t, cmd
}

func (t *TableModel) View() string {
	startIndex := (t.currentPage - 1) * t.rowsPerPage
	endIndex := startIndex + t.rowsPerPage
	if endIndex > len(t.Rows) {
		endIndex = len(t.Rows)
	}

	t.table.SetRows(t.Rows[startIndex:endIndex])

	return baseStyle.Render(t.table.View()) +
		fmt.Sprintf("\nPage %d of %d\n", t.currentPage, t.totalPages)
}

func (t *TableModel) SetStyle(s *table.Styles) {
	t.table.SetStyles(*s)
}

func (t *TableModel) Run() {
	t.table = table.New(
		table.WithColumns(t.Columns),
		table.WithRows(t.Rows),
		table.WithFocused(true))
	if t.Style != nil {
		t.SetStyle(t.Style)
	} else {
		t.SetDefaultStyle()
	}
	if _, err := tea.NewProgram(t).Run(); err != nil {
		console.Log.Errorf("Can't print aliases: %s", err)
	}
}
