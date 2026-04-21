type UsageExportCell = string | number
type ExcelJSImport = typeof import('exceljs')

function normalizeCellValue(value: UsageExportCell): string | number {
  return typeof value === 'number' ? value : String(value)
}

export async function buildUsageWorkbookBuffer(
  headers: UsageExportCell[],
  rows: UsageExportCell[][]
): Promise<Uint8Array> {
  const ExcelJS = (await import('exceljs')) as ExcelJSImport
  const workbook = new ExcelJS.Workbook()
  const sheet = workbook.addWorksheet('Usage')

  sheet.addRow(headers.map(normalizeCellValue))
  for (const row of rows) {
    sheet.addRow(row.map(normalizeCellValue))
  }

  sheet.views = [{ state: 'frozen', ySplit: 1 }]

  const buffer = await workbook.xlsx.writeBuffer()
  if (buffer instanceof Uint8Array) {
    return buffer
  }

  return new Uint8Array(buffer)
}
