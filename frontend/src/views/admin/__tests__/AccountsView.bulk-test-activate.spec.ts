import { computed, reactive, ref } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  loadMock,
  reloadMock,
  debouncedReloadMock,
  handlePageChangeMock,
  handlePageSizeChangeMock,
  runAccountTestStreamMock,
  getByIdMock,
  bulkUpdateMock,
  getBatchTodayStatsMock,
  getAllProxiesMock,
  getAllGroupsMock,
  showSuccessMock,
  showErrorMock,
  setSelectedIdsMock,
  clearSelectionMock,
  removeManyMock
} = vi.hoisted(() => ({
  loadMock: vi.fn(),
  reloadMock: vi.fn(),
  debouncedReloadMock: vi.fn(),
  handlePageChangeMock: vi.fn(),
  handlePageSizeChangeMock: vi.fn(),
  runAccountTestStreamMock: vi.fn(),
  getByIdMock: vi.fn(),
  bulkUpdateMock: vi.fn(),
  getBatchTodayStatsMock: vi.fn(),
  getAllProxiesMock: vi.fn(),
  getAllGroupsMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  setSelectedIdsMock: vi.fn(),
  clearSelectionMock: vi.fn(),
  removeManyMock: vi.fn()
}))

const accountsRef = ref<any[]>([])
const selectedIdsRef = ref<number[]>([])
const loadingRef = ref(false)
const tableParams = reactive({
  platform: '',
  type: '',
  status: '',
  privacy_mode: '',
  group: '',
  search: '',
  sort_by: 'name',
  sort_order: 'asc'
})
const paginationState = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 1
})

vi.mock('@vueuse/core', () => ({
  useIntervalFn: () => ({
    pause: vi.fn(),
    resume: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, number>) => {
        if (key === 'admin.accounts.bulkTestActivateSummary') {
          return `测试成功 ${params?.success ?? 0} 个，测试失败 ${params?.failed ?? 0} 个，已启用 ${params?.activated ?? 0} 个`
        }
        const messages: Record<string, string> = {
          'common.confirm': '确认',
          'common.error': '错误',
          'admin.accounts.bulkActions.testActivate': '批量测试激活',
          'admin.accounts.bulkActions.testActivating': '批量测试激活中...'
        }
        return messages[key] || key
      }
    })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: false
  })
}))

vi.mock('@/api/admin/accountTestStream', () => ({
  runAccountTestStream: runAccountTestStreamMock
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getById: getByIdMock,
      bulkUpdate: bulkUpdateMock,
      getBatchTodayStats: getBatchTodayStatsMock
    },
    proxies: {
      getAll: getAllProxiesMock
    },
    groups: {
      getAll: getAllGroupsMock
    }
  }
}))

vi.mock('@/composables/useSwipeSelect', () => ({
  useSwipeSelect: vi.fn()
}))

vi.mock('@/composables/useTableLoader', () => ({
  useTableLoader: () => ({
    items: accountsRef,
    loading: loadingRef,
    params: tableParams,
    pagination: paginationState,
    load: loadMock,
    reload: reloadMock,
    debouncedReload: debouncedReloadMock,
    handlePageChange: handlePageChangeMock,
    handlePageSizeChange: handlePageSizeChangeMock
  })
}))

vi.mock('@/composables/useTableSelection', () => ({
  useTableSelection: () => ({
    selectedIds: computed(() => selectedIdsRef.value),
    allVisibleSelected: computed(() => false),
    isSelected: vi.fn(() => false),
    setSelectedIds: setSelectedIdsMock,
    select: vi.fn(),
    deselect: vi.fn(),
    toggle: vi.fn(),
    clear: clearSelectionMock,
    removeMany: removeManyMock,
    toggleVisible: vi.fn(),
    selectVisible: vi.fn(),
    batchUpdate: vi.fn()
  })
}))

import AccountsView from '../AccountsView.vue'

function mountView() {
  return shallowMount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        AccountBulkActionsBar: {
          props: ['selectedIds', 'testingActivate'],
          emits: ['test-activate'],
          template: `
            <button
              class="bulk-test-activate-trigger"
              :disabled="testingActivate"
              @click="$emit('test-activate')"
            >
              bulk-test-activate
            </button>
          `
        },
        AccountTableActions: true,
        AccountTableFilters: true,
        DataTable: true,
        Pagination: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        AccountActionMenu: true,
        SyncFromCrsModal: true,
        ImportDataModal: true,
        BulkEditAccountModal: true,
        TempUnschedStatusModal: true,
        ConfirmDialog: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        AccountStatusIndicator: true,
        AccountUsageCell: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountCapacityCell: true,
        PlatformTypeBadge: true,
        Icon: true
      }
    }
  })
}

describe('AccountsView bulk test activate', () => {
  beforeEach(() => {
    accountsRef.value = [
      { id: 1, name: 'Account 1', status: 'inactive', schedulable: true },
      { id: 2, name: 'Account 2', status: 'active', schedulable: true },
      { id: 3, name: 'Account 3', status: 'inactive', schedulable: true },
      { id: 4, name: 'Account 4', status: 'inactive', schedulable: true }
    ]
    selectedIdsRef.value = [1, 2, 3, 4]
    paginationState.total = accountsRef.value.length
    paginationState.pages = 1

    loadMock.mockReset()
    reloadMock.mockReset()
    debouncedReloadMock.mockReset()
    handlePageChangeMock.mockReset()
    handlePageSizeChangeMock.mockReset()
    runAccountTestStreamMock.mockReset()
    getByIdMock.mockReset()
    bulkUpdateMock.mockReset()
    getBatchTodayStatsMock.mockReset()
    getAllProxiesMock.mockReset()
    getAllGroupsMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    setSelectedIdsMock.mockReset()
    clearSelectionMock.mockReset()
    removeManyMock.mockReset()

    setSelectedIdsMock.mockImplementation((ids: number[]) => {
      selectedIdsRef.value = ids
    })
    clearSelectionMock.mockImplementation(() => {
      selectedIdsRef.value = []
    })

    getBatchTodayStatsMock.mockResolvedValue({ stats: {} })
    getAllProxiesMock.mockResolvedValue([])
    getAllGroupsMock.mockResolvedValue([])
    bulkUpdateMock.mockResolvedValue({
      success: 0,
      failed: 0,
      success_ids: [],
      failed_ids: [],
      results: []
    })

    vi.stubGlobal('confirm', vi.fn(() => true))
  })

  it('批量测试激活按 3 并发启动测试任务', async () => {
    const started: number[] = []
    const resolvers = new Map<number, (value: { success: boolean }) => void>()

    runAccountTestStreamMock.mockImplementation((accountId: number) => new Promise((resolve) => {
      started.push(accountId)
      resolvers.set(accountId, resolve)
    }))
    getByIdMock.mockImplementation(async (accountId: number) => ({
      id: accountId,
      status: 'inactive'
    }))
    bulkUpdateMock.mockResolvedValue({
      success: 4,
      failed: 0,
      success_ids: [1, 2, 3, 4],
      failed_ids: [],
      results: []
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button.bulk-test-activate-trigger').trigger('click')
    await flushPromises()

    expect(started).toEqual([1, 2, 3])

    resolvers.get(1)?.({ success: true })
    await flushPromises()

    expect(started).toEqual([1, 2, 3, 4])

    resolvers.get(2)?.({ success: true })
    resolvers.get(3)?.({ success: true })
    resolvers.get(4)?.({ success: true })
    await flushPromises()
  })

  it('固定使用 GPT-5.4，失败不阻塞，并且只激活测试成功且当前非 active 的账号', async () => {
    selectedIdsRef.value = [1, 2, 3]

    runAccountTestStreamMock
      .mockResolvedValueOnce({ success: true })
      .mockResolvedValueOnce({ success: true })
      .mockRejectedValueOnce(new Error('boom'))

    getByIdMock
      .mockResolvedValueOnce({ id: 1, status: 'inactive' })
      .mockResolvedValueOnce({ id: 2, status: 'active' })

    bulkUpdateMock.mockResolvedValue({
      success: 1,
      failed: 0,
      success_ids: [1],
      failed_ids: [],
      results: [{ account_id: 1, success: true }]
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button.bulk-test-activate-trigger').trigger('click')
    await flushPromises()
    await flushPromises()

    expect(runAccountTestStreamMock).toHaveBeenCalledTimes(3)
    expect(runAccountTestStreamMock).toHaveBeenNthCalledWith(
      1,
      1,
      expect.objectContaining({ modelId: 'gpt-5.4' })
    )
    expect(runAccountTestStreamMock).toHaveBeenNthCalledWith(
      2,
      2,
      expect.objectContaining({ modelId: 'gpt-5.4' })
    )
    expect(runAccountTestStreamMock).toHaveBeenNthCalledWith(
      3,
      3,
      expect.objectContaining({ modelId: 'gpt-5.4' })
    )
    expect(getByIdMock).toHaveBeenCalledTimes(2)
    expect(getByIdMock).toHaveBeenNthCalledWith(1, 1)
    expect(getByIdMock).toHaveBeenNthCalledWith(2, 2)
    expect(bulkUpdateMock).toHaveBeenCalledWith([1], { status: 'active' })
    expect(setSelectedIdsMock).toHaveBeenCalledWith([3])
    expect(showErrorMock).toHaveBeenCalledWith('测试成功 2 个，测试失败 1 个，已启用 1 个')
  })
})
