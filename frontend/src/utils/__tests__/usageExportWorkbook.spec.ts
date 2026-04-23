// @vitest-environment node

import { strFromU8, unzipSync } from 'fflate'
import { describe, expect, it } from 'vitest'

import { buildUsageWorkbookBlob } from '@/utils/usageExportWorkbook'

describe('buildUsageWorkbookBlob', () => {
  it('生成包含 Usage sheet、冻结首行且保留表头与数据行的 xlsx 工作簿', async () => {
    const headers = ['Time', 'User', 'Cost']
    const rows = [
      ['2026-04-21 12:00:00', 'alice@example.com', '1.250000'],
      ['2026-04-21 12:05:00', 'bob@example.com', '2.500000'],
    ]

    const blob = await buildUsageWorkbookBlob(headers, rows)
    expect(blob).toBeInstanceOf(Blob)
    expect(blob.size).toBeGreaterThan(0)

    const archiveBuffer =
      typeof (blob as Blob).arrayBuffer === 'function'
        ? await (blob as Blob).arrayBuffer()
        : await new Response(blob as BodyInit).arrayBuffer()
    const files = unzipSync(new Uint8Array(archiveBuffer))
    const xmlEntries = Object.entries(files)
      .filter(([name]) => name.endsWith('.xml'))
      .map(([, content]) => strFromU8(content))
    const workbookXML = strFromU8(files['xl/workbook.xml'])
    const sheetXML = strFromU8(files['xl/worksheets/sheet1.xml'])
    const allXML = xmlEntries.join('\n')

    expect(workbookXML).toContain('name="Usage"')
    expect(sheetXML).toContain('state="frozen"')
    expect(sheetXML).toContain('ySplit="1"')
    expect(sheetXML).toContain('topLeftCell="A2"')
    expect(sheetXML).toContain('<row r="1">')
    expect(sheetXML).toContain('<row r="2">')
    expect(sheetXML).toContain('<row r="3">')
    for (const value of [...headers, ...rows.flat()]) {
      expect(allXML).toContain(value)
    }
  })
})
