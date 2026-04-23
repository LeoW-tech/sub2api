import writeExcelFile from 'write-excel-file/browser'

type UsageExportCell = string | number

function normalizeCellValue(value: UsageExportCell): string | number {
  return typeof value === 'number' ? value : String(value)
}

export async function buildUsageWorkbookBlob(
  headers: UsageExportCell[],
  rows: UsageExportCell[][]
): Promise<Blob> {
  const sheetData = [
    headers.map(normalizeCellValue),
    ...rows.map((row) => row.map(normalizeCellValue)),
  ]

  return writeExcelFile(sheetData, {
    sheet: 'Usage',
    stickyRowsCount: 1,
  }).toBlob()
}
