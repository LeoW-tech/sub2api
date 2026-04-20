import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue', 'change'],
  template: '<div class="select-stub"></div>'
})

const SearchInputStub = defineComponent({
  name: 'SearchInputStub',
  props: {
    modelValue: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue', 'search'],
  template: '<div class="search-input-stub"></div>'
})

describe('AccountTableFilters', () => {
  it('状态筛选项包含 disabled', () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          type: '',
          status: '',
          privacy_mode: '',
          group: ''
        },
        groups: []
      },
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub
        }
      }
    })

    const selects = wrapper.findAllComponents(SelectStub)
    const statusOptions = selects[2]?.props('options') as Array<{ value: string; label: string }>

    expect(statusOptions).toEqual(
      expect.arrayContaining([
        { value: 'disabled', label: 'admin.accounts.status.disabled' }
      ])
    )
  })

  it('选择 disabled 时会发出对应状态值', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          type: '',
          status: '',
          privacy_mode: '',
          group: ''
        },
        groups: []
      },
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub
        }
      }
    })

    const selects = wrapper.findAllComponents(SelectStub)
    const statusSelect = selects[2]

    await statusSelect.vm.$emit('update:modelValue', 'disabled')
    await statusSelect.vm.$emit('change', 'disabled')

    expect(wrapper.emitted('update:filters')).toEqual([
      [
        {
          platform: '',
          type: '',
          status: 'disabled',
          privacy_mode: '',
          group: ''
        }
      ]
    ])
    expect(wrapper.emitted('change')).toHaveLength(1)
  })
})
