import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'

describe('help page locales', () => {
  it('defines navigation and help page copy in Chinese and English', () => {
    expect(zh.nav.help).toBe('使用帮助')
    expect(en.nav.help).toBe('Help')

    expect(zh.help.steps.purchase.title).toBe('购买服务')
    expect(zh.help.steps.key.title).toBe('配置 API 密钥')
    expect(zh.help.steps.affiliate.title).toBe('邀请返利')

    expect(en.help.steps.purchase.title).toBe('Purchase service')
    expect(en.help.steps.key.title).toBe('Configure API key')
    expect(en.help.steps.affiliate.title).toBe('Invite for rebates')
  })
})
