import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const source = readFileSync('src/views/admin/ProxiesView.vue', 'utf8')
const filtersSlot = source.slice(
  source.indexOf('<template #filters>'),
  source.indexOf('</template>', source.indexOf('<template #filters>'))
)

describe('admin ProxiesView toolbar layout', () => {
  it('keeps filtering, monitoring, and connection actions in the primary row', () => {
    const primaryRow = filtersSlot.match(
      /data-testid="proxy-toolbar-primary"[\s\S]*?data-testid="proxy-toolbar-management"/
    )

    expect(primaryRow, 'toolbar rows are not defined').toBeTruthy()
    expect(primaryRow?.[0]).toContain('networkMonitorTitle')
    expect(primaryRow?.[0]).toContain('handleBatchTest')
    expect(primaryRow?.[0]).toContain('handleBatchQualityCheck')
    expect(primaryRow?.[0]).not.toContain('showImportData = true')
    expect(primaryRow?.[0]).not.toContain('showExportDataDialog = true')
  })

  it('keeps import, export, create, and delete actions together in the management row', () => {
    const managementRow = filtersSlot.match(
      /data-testid="proxy-toolbar-management"[\s\S]*/
    )

    expect(managementRow, 'management row is not defined').toBeTruthy()
    expect(managementRow?.[0]).toContain('showImportData = true')
    expect(managementRow?.[0]).toContain('showExportDataDialog = true')
    expect(managementRow?.[0]).toContain('showCreateModal = true')
    expect(managementRow?.[0]).toContain('openBatchDelete')
  })

  it('allows rows and controls to shrink or wrap without horizontal overflow', () => {
    expect(filtersSlot).toContain('flex-wrap')
    expect(filtersSlot).toContain('min-w-0')
    expect(filtersSlot).toContain('shrink-0')
  })
})
