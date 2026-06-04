import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ProxiesView from '../ProxiesView.vue'

const {
  listProxies,
  getNetworkMonitorStatus,
  updateNetworkMonitor,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  listProxies: vi.fn(),
  getNetworkMonitorStatus: vi.fn(),
  updateNetworkMonitor: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      getNetworkMonitorStatus,
      updateNetworkMonitor
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showInfo: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/composables/useSwipeSelect', () => ({
  useSwipeSelect: vi.fn()
}))

const mountView = () =>
  mount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: true,
        Pagination: true,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: true,
        ImportDataModal: true,
        Select: true,
        Icon: true,
        PlatformTypeBadge: true,
        Teleport: true
      }
    }
  })

const networkMonitorStatus = (overrides: Partial<{
  enabled: boolean
  running: boolean
  scan_running: boolean
  interval_seconds: number
  last_summary: null
}> = {}) => ({
  enabled: false,
  running: false,
  scan_running: false,
  interval_seconds: 300,
  last_summary: null,
  ...overrides
})

describe('admin ProxiesView network monitor', () => {
  beforeEach(() => {
    localStorage.clear()
    listProxies.mockReset()
    getNetworkMonitorStatus.mockReset()
    updateNetworkMonitor.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    listProxies.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    getNetworkMonitorStatus.mockResolvedValue(networkMonitorStatus())
  })

  it('loads proxies and network monitor status on mount', async () => {
    mountView()

    await flushPromises()

    expect(listProxies).toHaveBeenCalledTimes(1)
    expect(getNetworkMonitorStatus).toHaveBeenCalledTimes(1)
  })

  it('updates network monitor and shows success when toggled on', async () => {
    getNetworkMonitorStatus.mockResolvedValue(networkMonitorStatus())
    updateNetworkMonitor.mockResolvedValue(networkMonitorStatus({ enabled: true, running: true }))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[role="switch"]').trigger('click')
    await flushPromises()

    expect(updateNetworkMonitor).toHaveBeenCalledWith(true)
    expect(wrapper.text()).toContain('admin.proxies.networkMonitorRunning')
    expect(showSuccess).toHaveBeenCalledWith('admin.proxies.networkMonitorEnabled')
  })

  it('disables network monitor and shows restore success when toggled off', async () => {
    getNetworkMonitorStatus.mockResolvedValue(networkMonitorStatus({ enabled: true, running: true }))
    updateNetworkMonitor.mockResolvedValue(networkMonitorStatus({ enabled: false, running: false }))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[role="switch"]').trigger('click')
    await flushPromises()

    expect(updateNetworkMonitor).toHaveBeenCalledWith(false)
    expect(wrapper.text()).toContain('admin.proxies.networkMonitorDisabled')
    expect(showSuccess).toHaveBeenCalledWith('admin.proxies.networkMonitorDisabledSuccess')
  })

  it('rolls back network monitor state and shows error when update fails', async () => {
    getNetworkMonitorStatus.mockResolvedValue(networkMonitorStatus({ enabled: true, running: true }))
    updateNetworkMonitor.mockRejectedValue(new Error('failed'))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[role="switch"]').trigger('click')
    await flushPromises()

    expect(updateNetworkMonitor).toHaveBeenCalledWith(false)
    expect(wrapper.text()).toContain('admin.proxies.networkMonitorRunning')
    expect(showError).toHaveBeenCalledWith('failed')
  })
})
