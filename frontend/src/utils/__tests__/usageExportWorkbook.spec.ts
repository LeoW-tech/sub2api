import { describe, expect, it } from 'vitest'

import { buildUsageWorkbookBuffer } from '@/utils/usageExportWorkbook'

describe('buildUsageWorkbookBuffer', () => {
  it('生成可读取的 xlsx 工作簿并保留表头与数据行', async () => {
    const headers = ['Time', 'User', 'Cost']
    const rows = [
      ['2026-04-21 12:00:00', 'alice@example.com', '1.250000'],
      ['2026-04-21 12:05:00', 'bob@example.com', '2.500000'],
    ]

    const buffer = await buildUsageWorkbookBuffer(headers, rows)
    const ExcelJS = await import('exceljs')
    const workbook = new ExcelJS.Workbook()

    await workbook.xlsx.load(buffer)

    const sheet = workbook.getWorksheet('Usage')
    expect(sheet).toBeDefined()
    expect(sheet?.getRow(1).values).toEqual([, ...headers])
    expect(sheet?.getRow(2).values).toEqual([, ...rows[0]])
    expect(sheet?.getRow(3).values).toEqual([, ...rows[1]])
  })
})
