package common

import (
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
)

// NewKVTable creates a two-column table using the standard border style,
// consistent with all other tables in the client. The header row displays
// the section title in the "Key" column.
func NewKVTable(title string, keys []string, values map[string]string) *tui.TableModel {
	var rows []table.Row
	for _, k := range keys {
		rows = append(rows, table.NewRow(table.RowData{
			"Key":   tui.BlueFg.Bold(true).Render(k),
			"Value": values[k],
		}))
	}

	t := tui.NewTable([]table.Column{
		table.NewColumn("Key", title, 16),
		table.NewFlexColumn("Value", "", 1),
	}, true)
	t.SetRows(rows)
	return t
}
