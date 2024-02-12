package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

func NewTable(columns []table.Column) *TableModel {
	t := &TableModel{
		table: table.New(
			table.WithColumns(columns),
			table.WithFocused(true)),
		Style: DefaultTableStyle,
	}
	return t
}

// TODO tui: table 实现自适应width 并通过左右键查看无法一次性展示的属性
type TableModel struct {
	table       table.Model
	Style       *table.Styles
	Columns     []table.Column
	Rows        []table.Row
	currentPage int
	totalPages  int
	rowsPerPage int
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

	return FootStyle.Render(t.table.View()) +
		fmt.Sprintf("\nPage %d of %d\n", t.currentPage, t.totalPages)
}

func (t *TableModel) SetStyle(s *table.Styles) {
	t.table.SetStyles(*s)
}
