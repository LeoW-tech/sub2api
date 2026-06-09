import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import HelpView from '../HelpView.vue'

const flagState = vi.hoisted(() => ({
  payment: true as boolean | undefined,
  affiliate: true as boolean | undefined,
  isSimpleMode: false,
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' }
}))

vi.mock('@/utils/featureFlags', () => ({
  FeatureFlags: {
    payment: { key: 'payment_enabled' },
    affiliate: { key: 'affiliate_enabled' },
  },
  isFeatureFlagEnabled: (flag: { key: string }) => {
    if (flag.key === 'payment_enabled') return flagState.payment ?? true
    if (flag.key === 'affiliate_enabled') return flagState.affiliate === true
    return true
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return flagState.isSimpleMode
    }
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const messages: Record<string, string> = {
          'help.title': '使用帮助',
          'help.description': '按页面提示完成可用步骤，即可开始使用 AI 服务。',
          'help.intro.title': '快速开始使用 AI 服务',
          'help.intro.description': '进入系统后，先确认余额或订阅，再创建 API 密钥并配置到 Codex 等工具中。',
          'help.steps.purchase.title': '购买服务',
          'help.steps.purchase.description': '先确保界面右上角钱包有余额，或者有可用的订阅包。至少满足一个条件才能正常使用。',
          'help.steps.key.title': '配置 API 密钥',
          'help.steps.key.description': '打开 API 密钥菜单，创建密钥并配置到 Codex 等大模型工具里面。密钥只用填写名称和选择你购买的服务。',
          'help.steps.key.note': '创建后，点击密钥右侧菜单栏“使用密钥”，里面有配置方法。参考方法完成配置后，就可以开始使用 AI 模型。',
          'help.steps.affiliate.title': '邀请返利',
          'help.steps.affiliate.description': '如果你觉得好用，可以进入邀请返利，发送你的邀请链接邀请好友使用服务，并获得邀请奖励。',
          'help.links.purchase': '进入充值/订阅',
          'help.links.subscriptions': '查看我的订阅',
          'help.links.keys': '创建 API 密钥',
          'help.links.affiliate': '进入邀请返利',
          'help.screenshots.purchase.title': '钱包与订阅',
          'help.screenshots.purchase.wallet': '右上角钱包余额',
          'help.screenshots.purchase.subscription': '也可以购买可用订阅包',
          'help.screenshots.key.title': '使用密钥',
          'help.screenshots.key.name': 'Codex 密钥',
          'help.screenshots.key.useKey': '使用密钥',
          'help.screenshots.affiliate.title': '邀请链接',
          'help.screenshots.affiliate.linkLabel': '你的邀请链接',
          'help.screenshots.affiliate.copy': '复制',
          'help.screenshots.affiliate.rebate': '返利比例',
          'help.screenshots.affiliate.reward': '可提奖励',
        }
        return messages[key] ?? key
      }
    })
  }
})

describe('HelpView', () => {
  beforeEach(() => {
    flagState.payment = true
    flagState.affiliate = true
    flagState.isSimpleMode = false
  })

  function mountView() {
    return mount(HelpView, {
      global: {
        stubs: {
          Icon: { props: ['name'], template: '<span :data-icon="name" />' },
          RouterLink: { props: ['to'], template: '<a :href="to" data-router-link="true"><slot /></a>' },
        }
      }
    })
  }

  it('renders the three quick-start steps when all features are enabled', () => {
    const wrapper = mountView()

    expect(wrapper.text()).toContain('购买服务')
    expect(wrapper.text()).toContain('配置 API 密钥')
    expect(wrapper.text()).toContain('邀请返利')
  })

  it('uses in-app router links for system destinations instead of new windows', () => {
    const wrapper = mountView()
    const links = wrapper.findAll('[data-router-link="true"]')
    const hrefs = links.map((link) => link.attributes('href'))

    expect(hrefs).toContain('/purchase')
    expect(hrefs).toContain('/subscriptions')
    expect(hrefs).toContain('/keys')
    expect(hrefs).toContain('/affiliate')
    for (const link of links) {
      expect(link.attributes('target')).toBeUndefined()
      expect(link.attributes('rel')).toBeUndefined()
    }
  })

  it('hides payment and affiliate steps when their features are disabled', () => {
    flagState.payment = false
    flagState.affiliate = false

    const wrapper = mountView()

    expect(wrapper.text()).not.toContain('购买服务')
    expect(wrapper.text()).toContain('配置 API 密钥')
    expect(wrapper.text()).not.toContain('邀请返利')
    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(1)
  })

  it('uses default feature flag semantics when settings are not loaded', () => {
    flagState.payment = undefined
    flagState.affiliate = undefined

    const wrapper = mountView()

    expect(wrapper.text()).toContain('购买服务')
    expect(wrapper.text()).toContain('配置 API 密钥')
    expect(wrapper.text()).not.toContain('邀请返利')
    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(2)
  })

  it('does not claim a fixed three-step flow when optional feature steps are hidden', () => {
    flagState.payment = undefined
    flagState.affiliate = undefined

    const wrapper = mountView()

    expect(wrapper.text()).not.toContain('三步')
    expect(wrapper.text()).toContain('快速开始使用 AI 服务')
  })

  it('hides the subscriptions shortcut in simple mode', () => {
    flagState.isSimpleMode = true

    const wrapper = mountView()
    const hrefs = wrapper.findAll('[data-router-link="true"]').map((link) => link.attributes('href'))

    expect(hrefs).toContain('/purchase')
    expect(hrefs).not.toContain('/subscriptions')
  })

  it('keeps lightweight screenshot cards for every visible step', () => {
    const wrapper = mountView()

    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(3)
    expect(wrapper.text()).toContain('钱包与订阅')
    expect(wrapper.text()).toContain('使用密钥')
    expect(wrapper.text()).toContain('邀请链接')
  })
})
