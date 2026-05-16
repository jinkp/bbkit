package output

import (
	"github.com/olekukonko/tablewriter"
)

func PrintTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(stdout)
	table.SetHeader(headers)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(true)
	table.SetHeaderLine(true)

	if IsTTY() {
		headerColors := make([]tablewriter.Colors, 0, len(headers))
		for range headers {
			headerColors = append(headerColors, tablewriter.Colors{tablewriter.FgCyanColor, tablewriter.Bold})
		}
		table.SetHeaderColor(headerColors...)
	}

	for _, row := range rows {
		table.Append(row)
	}

	table.Render()
}
