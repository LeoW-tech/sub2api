import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import HelpView from '../HelpView.vue'

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const messages: Record<string, string> = {
          'help.title': '使用帮助',
          'help.description': '完成购买服务、配置 API 密钥、邀请返利三步即可开始使用。',
          'help.intro.title': '三步开始使用 AI 服务',
          'help.intro.description': '进入系统后，先确认余额或订阅，再创建 API 密钥并配置到 Codex 等工具中。',
          'help.steps.purchase.title': '购买服务',
          'help.steps.purchase.description': '先确保界面右上角钱包有余额，或者有可用的订阅包。至少满足一个条件才能正常使用。',
          'help.steps.purchase.secondary': '查看我的订阅',
          'help.steps.key.title': '配置 API 密钥',
          'help.steps.key.description': '打开 API 密钥菜单，创建密钥并配置到 Codex 等大模型工具里面。密钥只用填写名称和选择你购买的服务。',
          'help.steps.key.note': '创建后，点击密钥右侧菜单栏“使用密钥”，里面有配置方法。参考方法完成配置后，就可以开始使用 AI 模型。',
          'help.steps.affiliate.title': '邀请返利',
          'help.steps.affiliate.description': '如果你觉得好用，可以进入邀请返利，发送你的邀请链接邀请好友使用服务，并获得邀请奖励。',
          'help.links.purchase': '进入充值/订阅',
          'help.links.keys': '创建 API 密钥',
          'help.links.affiliate': '进入邀请返利',
          'help.screenshots.purchase.title': '钱包与订阅',
          'help.screenshots.key.title': '使用密钥',
          'help.screenshots.affiliate.title': '邀请链接',
        }
        return messages[key] ?? key
      }
    })
  }
})

describe('HelpView', () => {
  function mountView() {
    return mount(HelpView, {
      global: {
        stubs: {
          Icon: { props: ['name'], template: '<span :data-icon="name" />' }
        }
      }
    })
  }

  it('renders the three quick-start steps', () => {
    const wrapper = mountView()

    expect(wrapper.text()).toContain('购买服务')
    expect(wrapper.text()).toContain('配置 API 密钥')
    expect(wrapper.text()).toContain('邀请返利')
  })

  it('opens system destinations in a new window', () => {
    const wrapper = mountView()
    const links = wrapper.findAll('a[target="_blank"]')
    const hrefs = links.map((link) => link.attributes('href'))

    expect(hrefs).toContain('/purchase')
    expect(hrefs).toContain('/keys')
    expect(hrefs).toContain('/affiliate')
    for (const link of links) {
      expect(link.attributes('rel')).toContain('noopener')
      expect(link.attributes('rel')).toContain('noreferrer')
    }
  })

  it('keeps lightweight screenshot cards for every step', () => {
    const wrapper = mountView()

    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(3)
    expect(wrapper.text()).toContain('钱包与订阅')
    expect(wrapper.text()).toContain('使用密钥')
    expect(wrapper.text()).toContain('邀请链接')
  })
})
