import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File(['invalid json'], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('invalid json')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
  })

  it('导入时保留门标识和可选出口 IP 字段', async () => {
    const importData = vi.mocked(adminAPI.accounts.importData)
    importData.mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 1,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0,
      errors: []
    })

    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const payload = {
      type: 'sub2api-data',
      version: 1,
      exported_at: '2026-04-08T00:00:00Z',
      proxies: [
        {
          proxy_key: 'http|host.docker.internal|58052||',
          proxy_external_key: 'door-hk-w10',
          name: '🇭🇰 香港 W10 | IEPL',
          protocol: 'http',
          host: 'host.docker.internal',
          port: 58052,
          status: 'active',
          exit_ip: '203.0.113.10',
        }
      ],
      accounts: [
        {
          name: 'acc-1',
          platform: 'openai',
          type: 'oauth',
          credentials: { token: 'x' },
          proxy_external_key: 'door-hk-w10',
          proxy_name: '🇭🇰 香港 W10 | IEPL',
          exit_ip: '203.0.113.10',
          concurrency: 3,
          priority: 50
        }
      ]
    }

    const input = wrapper.find('input[type="file"]')
    const file = new File([JSON.stringify(payload)], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve(JSON.stringify(payload))
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(importData).toHaveBeenCalledWith({
      data: payload,
      skip_default_group_bind: true
    })
  })
})
