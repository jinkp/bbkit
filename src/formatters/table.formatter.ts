import Table from 'cli-table3'

export interface TableColumn {
  header: string
  key: string
  width?: number
}

export function formatTable(columns: TableColumn[], rows: Record<string, unknown>[]): string {
  const table = new Table({
    head: columns.map((c) => c.header),
    colWidths: columns.map((c) => c.width ?? null),
    style: { head: ['cyan'] },
  })

  for (const row of rows) {
    table.push(columns.map((c) => String(row[c.key] ?? '')))
  }

  return table.toString()
}
