import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageFilters from '../UsageFilters.vue'

const { listGroups, getModelStats, searchUsers, searchApiKeys, listAccounts } = vi.hoisted(() => ({
  listGroups: vi.fn(),
  getModelStats: vi.fn(),
  searchUsers: vi.fn(),
  searchApiKeys: vi.fn(),
  listAccounts: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: listGroups,
    },
    dashboard: {
      getModelStats,
    },
    usage: {
      searchUsers,
      searchApiKeys,
    },
    accounts: {
      list: listAccounts,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const SelectStub = {
  props: ['options'],
  template: '<div class="select-stub">{{ options.length }}</div>',
}

describe('UsageFilters', () => {
  beforeEach(() => {
    listGroups.mockReset()
    getModelStats.mockReset()
    searchUsers.mockReset()
    searchApiKeys.mockReset()
    listAccounts.mockReset()

    listGroups.mockResolvedValue({
      items: [{ id: 1, name: 'group-a' }],
      total: 1,
      page: 1,
      page_size: 1000,
      pages: 1,
    })
  })

  it('reuses parent-provided model options and does not issue an extra model stats request on mount', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: {},
        exporting: false,
        startDate: '2026-03-01',
        endDate: '2026-03-02',
        modelOptions: [{ value: 'claude-sonnet-4', label: 'claude-sonnet-4' }],
      },
      global: {
        stubs: {
          Select: SelectStub,
        },
      },
    })

    await flushPromises()

    expect(listGroups).toHaveBeenCalledTimes(1)
    expect(getModelStats).not.toHaveBeenCalled()
    expect(wrapper.find('.select-stub').text()).toBe('2')
  })
})
