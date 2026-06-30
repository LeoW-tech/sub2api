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
          'help.intro.title': '快速开始使用 Codex',
          'help.intro.description': '按照下面 5 个步骤完成安装、购买、密钥配置和基础学习，即可开始使用 Codex。',
          'help.steps.install.title': '下载安装 Codex',
          'help.steps.install.description': '请先安装 Codex App。优先打开 Codex 官方下载页面，进入页面后复制适合你设备的下载链接并下载安装。如果官方下载页面无法打开，可以使用备用网盘下载。',
          'help.steps.install.note': '安装完成后，请打开一次 Codex，确保应用完成初始化，再继续后面的 API 密钥配置。',
          'help.steps.purchase.title': '购买服务',
          'help.steps.purchase.description': '先确保界面右上角钱包有余额，或者有可用的订阅包。至少满足一个条件才能正常使用。',
          'help.steps.key.title': '配置 API 密钥',
          'help.steps.key.description': '打开 API 密钥菜单，创建密钥并配置到 Codex 等大模型工具里面。密钥只用填写名称和选择你购买的服务。',
          'help.steps.key.note': '创建后，点击密钥右侧菜单栏“使用密钥”，里面有配置方法。参考方法完成配置后，就可以开始使用 AI 模型。',
          'help.steps.affiliate.title': '邀请返利',
          'help.steps.affiliate.description': '如果你觉得好用，可以进入邀请返利，发送你的邀请链接邀请好友使用服务，并获得邀请奖励。',
          'help.steps.learn.title': '学习使用',
          'help.steps.learn.description': '配置完成后，直接在 Codex 对话框中描述你希望它完成的事情，或者说明你想要的结果，Codex 就可以帮你执行。',
          'help.steps.learn.note': '如果你想系统学习 Codex 的使用技巧，可以参考下面任意一个视频。三个视频都是完整的 Codex 实战教程，选择一个看完即可。',
          'help.links.officialDownload': '打开官方下载页',
          'help.links.quarkDownload': '备用网盘下载',
          'help.links.purchase': '进入充值/订阅',
          'help.links.subscriptions': '查看我的订阅',
          'help.links.keys': '创建 API 密钥',
          'help.links.affiliate': '进入邀请返利',
          'help.links.videoFullGuide': '40 分钟全面掌握 Codex',
          'help.links.videoAppGuide': 'Codex App 保姆级全攻略',
          'help.links.videoCompleteGuide': '2026 Codex 完整教程',
          'help.screenshots.install.title': '下载入口',
          'help.screenshots.install.official': '优先使用官方下载',
          'help.screenshots.install.backup': '打不开时使用备用网盘',
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
          'help.screenshots.learn.title': '学习资源',
          'help.screenshots.learn.prompt': '直接描述你的目标',
          'help.screenshots.learn.video': '选择一个教程看完即可',
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

  it('renders the five quick-start steps in the expected order when all features are enabled', () => {
    const wrapper = mountView()
    const text = wrapper.text()

    expect(text).toContain('下载安装 Codex')
    expect(text).toContain('购买服务')
    expect(text).toContain('配置 API 密钥')
    expect(text).toContain('学习使用')
    expect(text).toContain('邀请返利')
    expect(text.indexOf('下载安装 Codex')).toBeLessThan(text.indexOf('购买服务'))
    expect(text.indexOf('购买服务')).toBeLessThan(text.indexOf('配置 API 密钥'))
    expect(text.indexOf('配置 API 密钥')).toBeLessThan(text.indexOf('学习使用'))
    expect(text.indexOf('学习使用')).toBeLessThan(text.indexOf('邀请返利'))
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

    expect(wrapper.text()).toContain('下载安装 Codex')
    expect(wrapper.text()).not.toContain('购买服务')
    expect(wrapper.text()).toContain('配置 API 密钥')
    expect(wrapper.text()).toContain('学习使用')
    expect(wrapper.text()).not.toContain('邀请返利')
    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(3)
  })

  it('uses default feature flag semantics when settings are not loaded', () => {
    flagState.payment = undefined
    flagState.affiliate = undefined

    const wrapper = mountView()

    expect(wrapper.text()).toContain('下载安装 Codex')
    expect(wrapper.text()).toContain('购买服务')
    expect(wrapper.text()).toContain('配置 API 密钥')
    expect(wrapper.text()).toContain('学习使用')
    expect(wrapper.text()).not.toContain('邀请返利')
    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(4)
  })

  it('does not claim a fixed three-step flow when optional feature steps are hidden', () => {
    flagState.payment = undefined
    flagState.affiliate = undefined

    const wrapper = mountView()

    expect(wrapper.text()).not.toContain('三步')
    expect(wrapper.text()).toContain('快速开始使用 Codex')
  })

  it('hides the subscriptions shortcut in simple mode', () => {
    flagState.isSimpleMode = true

    const wrapper = mountView()
    const hrefs = wrapper.findAll('[data-router-link="true"]').map((link) => link.attributes('href'))

    expect(hrefs).toContain('/purchase')
    expect(hrefs).not.toContain('/subscriptions')
  })

  it('keeps lightweight resource or screenshot cards for every visible step', () => {
    const wrapper = mountView()

    expect(wrapper.findAll('[data-testid="help-screenshot-card"]')).toHaveLength(5)
    expect(wrapper.text()).toContain('下载入口')
    expect(wrapper.text()).toContain('钱包与订阅')
    expect(wrapper.text()).toContain('使用密钥')
    expect(wrapper.text()).toContain('学习资源')
    expect(wrapper.text()).toContain('邀请链接')
  })

  it('renders external download and learning resources as labeled links without exposing raw URLs', () => {
    const wrapper = mountView()
    const externalLinks = wrapper.findAll('a[target="_blank"]')
    const labels = externalLinks.map((link) => link.text())

    expect(labels.some((label) => label.includes('打开官方下载页'))).toBe(true)
    expect(labels.some((label) => label.includes('备用网盘下载'))).toBe(true)
    expect(labels.some((label) => label.includes('40 分钟全面掌握 Codex'))).toBe(true)
    expect(labels.some((label) => label.includes('Codex App 保姆级全攻略'))).toBe(true)
    expect(labels.some((label) => label.includes('2026 Codex 完整教程'))).toBe(true)
    for (const link of externalLinks) {
      expect(link.attributes('rel')).toBe('noopener noreferrer')
    }
    expect(wrapper.text()).not.toContain('https://chatgpt.com/codex/for-work/')
    expect(wrapper.text()).not.toContain('https://pan.quark.cn/s/4a0a101a1eba')
    expect(wrapper.text()).not.toContain('https://www.bilibili.com/video/')
  })
})
