import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import DataTable from '../DataTable.vue'

const messages: Record<string, string> = {
  'empty.noData': 'No data',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const columns = [
  { key: 'name', label: 'Name', sortable: true },
]

const rows = Array.from({ length: 5 }, (_, index) => ({
  id: index + 1,
  name: `row-${index + 1}`,
}))

const getVisibleRows = (wrapper: ReturnType<typeof mount>) => {
  const exposed = wrapper.vm as unknown as {
    visibleRows: { value: unknown[] } | unknown[]
  }

  return Array.isArray(exposed.visibleRows)
    ? exposed.visibleRows
    : exposed.visibleRows.value
}

const createMatchMedia = (matches: boolean) =>
  vi.fn().mockImplementation(() => ({
    matches,
    media: '(min-width: 768px)',
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))

describe('DataTable progressive mount', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal('matchMedia', createMatchMedia(false))
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      return setTimeout(() => callback(0), 0) as unknown as number
    })
    vi.stubGlobal('cancelAnimationFrame', (handle: number) => {
      clearTimeout(handle)
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('renders all rows immediately when progressive mounting is disabled', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
      },
      slots: {
        'cell-name': ({ row }: { row: { name: string } }) => `<span class="row-marker">${row.name}</span>`,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await nextTick()

    expect(getVisibleRows(wrapper)).toHaveLength(5)
  })

  it('mounts rows progressively without changing the final row set', async () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        progressiveMount: true,
        initialRenderCount: 2,
        renderBatchSize: 2,
      },
      slots: {
        'cell-name': ({ row }: { row: { name: string } }) => `<span class="row-marker">${row.name}</span>`,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await nextTick()
    expect(getVisibleRows(wrapper)).toHaveLength(2)

    await vi.runOnlyPendingTimersAsync()
    await nextTick()
    expect(getVisibleRows(wrapper)).toHaveLength(4)

    await vi.runOnlyPendingTimersAsync()
    await nextTick()
    expect(getVisibleRows(wrapper)).toHaveLength(5)
    expect(wrapper.text()).toContain('row-1')
    expect(wrapper.text()).toContain('row-5')
  })

  it('keeps server-side sort behavior when progressive mounting is enabled', async () => {
    vi.stubGlobal('matchMedia', createMatchMedia(true))

    const wrapper = mount(DataTable, {
      props: {
        columns,
        data: rows,
        serverSideSort: true,
        progressiveMount: true,
        initialRenderCount: 2,
        renderBatchSize: 2,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await wrapper.find('th').trigger('click')

    expect(wrapper.emitted('sort')).toEqual([['name', 'asc']])
  })
})
