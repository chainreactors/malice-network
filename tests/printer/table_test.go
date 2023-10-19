package printer

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"testing"
)

var (
	colTitleIndex     = "#"
	colTitleFirstName = "First Name"
	colTitleLastName  = "Last Name"
	colTitleSalary    = "Salary"
	rowHeader         = table.Row{colTitleIndex, colTitleFirstName, colTitleLastName, colTitleSalary}
	row1              = table.Row{1, "Arya", "Stark", 3000}
	row2              = table.Row{20, "Jon", "Snow", 2000, "You know nothing, Jon Snow!"}
	row3              = table.Row{300, "Tyrion", "Lannister", 5000}
	rowFooter         = table.Row{"", "", "Total", 10000}
)

func TestPrettyTable(t *testing.T) {
	tw := table.NewWriter()
	tw.AppendHeader(rowHeader)
	tw.AppendRows([]table.Row{row1, row2, row3})
	tw.AppendFooter(rowFooter)
	tw.SetIndexColumn(1)
	tw.SetTitle("Game Of Thrones")

	stylePairs := [][]table.Style{
		{table.StyleColoredBright, table.StyleColoredDark},
		{table.StyleColoredBlackOnBlueWhite, table.StyleColoredBlueWhiteOnBlack},
		{table.StyleColoredBlackOnCyanWhite, table.StyleColoredCyanWhiteOnBlack},
		{table.StyleColoredBlackOnGreenWhite, table.StyleColoredGreenWhiteOnBlack},
		{table.StyleColoredBlackOnMagentaWhite, table.StyleColoredMagentaWhiteOnBlack},
		{table.StyleColoredBlackOnRedWhite, table.StyleColoredRedWhiteOnBlack},
		{table.StyleColoredBlackOnYellowWhite, table.StyleColoredYellowWhiteOnBlack},
	}

	twOuter := table.NewWriter()
	twOuter.AppendHeader(table.Row{"Bright", "Dark"})
	for _, stylePair := range stylePairs {
		row := make(table.Row, 2)
		for idx, style := range stylePair {
			tw.SetCaption(style.Name)
			tw.SetStyle(style)
			tw.Style().Title.Align = text.AlignCenter
			row[idx] = tw.Render()
		}
		twOuter.AppendRow(row)
	}
	twOuter.SetColumnConfigs([]table.ColumnConfig{
		{Name: "Bright", Align: text.AlignCenter, AlignHeader: text.AlignCenter},
		{Name: "Dark", Align: text.AlignCenter, AlignHeader: text.AlignCenter},
	})
	twOuter.SetStyle(table.StyleLight)
	twOuter.Style().Title.Align = text.AlignCenter
	twOuter.SetTitle("C O L O R S")
	twOuter.Style().Options.SeparateRows = true
	fmt.Println(twOuter.Render())

	styles := []table.Style{
		table.StyleDefault,
		table.StyleLight,
		table.StyleColoredBright,
	}
	for _, style := range styles {
		tw := table.NewWriter()
		tw.AppendHeader(table.Row{"Key", "Value"})
		tw.AppendRows([]table.Row{
			{"Emoji 1 🥰", 1000},
			{"Emoji 2 ⚔️", 2000},
			{"Emoji 3 🎁", 3000},
			{"Emoji 4 ツ", 4000},
		})
		tw.AppendFooter(table.Row{"Total", 10000})
		tw.SetAutoIndex(true)
		tw.SetStyle(style)

		fmt.Println(tw.Render())
		fmt.Println()
	}
}
