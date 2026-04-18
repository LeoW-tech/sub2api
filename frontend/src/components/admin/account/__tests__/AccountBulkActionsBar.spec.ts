import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, number>) => {
      const messages: Record<string, string> = {
        'admin.accounts.bulkActions.selected': `已选择 ${params?.count ?? 0} 个账号`,
        'admin.accounts.bulkActions.selectCurrentPage': '本页全选',
        'admin.accounts.bulkActions.clear': '清除选择',
        'admin.accounts.bulkActions.delete': '批量删除',
        'admin.accounts.bulkActions.resetStatus': '批量重置状态',
        'admin.accounts.bulkActions.refreshToken': '批量刷新令牌',
        'admin.accounts.bulkActions.testActivate': '批量测试激活',
        'admin.accounts.bulkActions.testActivating': '批量测试激活中...',
        'admin.accounts.bulkActions.enableScheduling': '批量启用调度',
        'admin.accounts.bulkActions.disableScheduling': '批量停止调度',
        'admin.accounts.bulkActions.edit': '批量编辑账号'
      }
      return messages[key] || key
    }
  })
}))

describe('AccountBulkActionsBar', () => {
  it('在刷新令牌和启用调度之间渲染批量测试激活按钮', () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: {
        selectedIds: [1, 2],
        testingActivate: false
      }
    })

    const labels = wrapper.findAll('button').map((button) => button.text())
    const refreshIndex = labels.indexOf('批量刷新令牌')
    const bulkTestIndex = labels.indexOf('批量测试激活')
    const enableSchedulingIndex = labels.indexOf('批量启用调度')

    expect(refreshIndex).toBeGreaterThanOrEqual(0)
    expect(bulkTestIndex).toBe(refreshIndex + 1)
    expect(enableSchedulingIndex).toBe(bulkTestIndex + 1)
  })

  it('点击批量测试激活会派发事件，执行中按钮禁用', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: {
        selectedIds: [1],
        testingActivate: false
      }
    })

    const actionButtons = wrapper.findAll('button')
    const activateButton = actionButtons.find((button) => button.text() === '批量测试激活')
    expect(activateButton).toBeTruthy()

    await activateButton!.trigger('click')
    expect(wrapper.emitted('test-activate')).toHaveLength(1)

    await wrapper.setProps({ testingActivate: true })
    const disabledButton = wrapper.findAll('button').find((button) => button.text() === '批量测试激活中...')
    expect(disabledButton).toBeTruthy()
    expect(disabledButton!.attributes('disabled')).toBeDefined()
  })
})
