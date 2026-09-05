import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('usage service tier locale keys', () => {
  it('contains zh labels for service tier tooltip', () => {
    expect(zh.usage.serviceTier).toBe('服务档位')
    expect(zh.usage.serviceTierPriority).toBe('Fast')
    expect(zh.usage.serviceTierUltrafast).toBe('Ultrafast')
    expect(zh.usage.serviceTierFlex).toBe('Flex')
    expect(zh.usage.serviceTierStandard).toBe('Standard')
  })

  it('contains en labels for service tier tooltip', () => {
    expect(en.usage.serviceTier).toBe('Service tier')
    expect(en.usage.serviceTierPriority).toBe('Fast')
    expect(en.usage.serviceTierUltrafast).toBe('Ultrafast')
    expect(en.usage.serviceTierFlex).toBe('Flex')
    expect(en.usage.serviceTierStandard).toBe('Standard')
  })

  it('contains zh label for disabled account status', () => {
    expect(zh.admin.accounts.status.disabled).toBe('已禁用')
  })

  it('contains en label for disabled account status', () => {
    expect(en.admin.accounts.status.disabled).toBe('Disabled')
  })

  it('contains zh labels for account filter and network status keys', () => {
    expect(zh.admin.accounts.allTokenStatus).toBe('全部 RT 状态')
    expect(zh.admin.accounts.hasRT).toBe('有 RT')
    expect(zh.admin.accounts.noRT).toBe('无 RT')
    expect(zh.admin.accounts.allCapacity).toBe('全部容量状态')
    expect(zh.admin.accounts.capacityConcurrent).toBe('有并发占用')
    expect(zh.admin.accounts.allNetworkStatus).toBe('全部网络状态')
    expect(zh.admin.accounts.allIPs).toBe('全部出口 IP')
    expect(zh.admin.accounts.columns.networkStatus).toBe('网络状态')
    expect(zh.admin.accounts.networkStatus.online).toBe('在线')
    expect(zh.admin.accounts.networkStatus.offline).toBe('离线')
  })

  it('contains en labels for account filter and network status keys', () => {
    expect(en.admin.accounts.allTokenStatus).toBe('All RT Status')
    expect(en.admin.accounts.hasRT).toBe('Has RT')
    expect(en.admin.accounts.noRT).toBe('No RT')
    expect(en.admin.accounts.allCapacity).toBe('All Capacity Status')
    expect(en.admin.accounts.capacityConcurrent).toBe('Has Active Concurrency')
    expect(en.admin.accounts.allNetworkStatus).toBe('All Network Status')
    expect(en.admin.accounts.allIPs).toBe('All Exit IPs')
    expect(en.admin.accounts.columns.networkStatus).toBe('Network Status')
    expect(en.admin.accounts.networkStatus.online).toBe('Online')
    expect(en.admin.accounts.networkStatus.offline).toBe('Offline')
  })
})
