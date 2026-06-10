import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (!value || typeof value !== 'object') return [prefix]
  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) => {
    const nextPrefix = prefix ? `${prefix}.${key}` : key
    return flattenKeys(child, nextPrefix)
  })
}

describe('help page locales', () => {
  it('defines navigation and help page copy in Chinese and English', () => {
    expect(zh.nav.help).toBe('使用帮助')
    expect(en.nav.help).toBe('Help')

    expect(zh.help.steps.purchase.title).toBe('购买服务')
    expect(zh.help.steps.key.title).toBe('配置 API 密钥')
    expect(zh.help.steps.affiliate.title).toBe('邀请返利')
    expect(zh.help.links.subscriptions).toBe('查看我的订阅')
    expect(zh.help.screenshots.key.name).toBe('Codex 密钥')

    expect(en.help.steps.purchase.title).toBe('Purchase service')
    expect(en.help.steps.key.title).toBe('Configure API key')
    expect(en.help.steps.affiliate.title).toBe('Invite for rebates')
    expect(en.help.links.subscriptions).toBe('View my subscriptions')
    expect(en.help.screenshots.key.name).toBe('Codex Key')
  })

  it('keeps Chinese and English help locale structures aligned', () => {
    expect(flattenKeys(en.help).sort()).toEqual(flattenKeys(zh.help).sort())
  })
})
