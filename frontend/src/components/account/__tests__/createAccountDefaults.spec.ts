import { describe, expect, it } from 'vitest'
import {
  DEFAULT_OPENAI_ACCOUNT_PRIORITY,
  normalizeOpenAIPlusGroupName,
  resolveDefaultOpenAIPlusGroup,
  shouldApplyDefaultOpenAIPlusGroup
} from '@/components/account/createAccountDefaults'
import type { AdminGroup } from '@/types'

const makeGroup = (overrides: Partial<AdminGroup>): AdminGroup => ({
  id: 1,
  name: 'default',
  description: null,
  platform: 'openai',
  rate_multiplier: 1,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '',
  updated_at: '',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: false,
  ...overrides
})

describe('createAccountDefaults', () => {
  it('uses priority 50 for default OpenAI OAuth accounts', () => {
    expect(DEFAULT_OPENAI_ACCOUNT_PRIORITY).toBe(50)
  })

  it('normalizes plus group names by removing whitespace and lowercasing', () => {
    expect(normalizeOpenAIPlusGroupName(' Plus Group ')).toBe('plusgroup')
    expect(normalizeOpenAIPlusGroupName('plus 组')).toBe('plus组')
  })

  it.each([
    ['plus 组'],
    ['plus组'],
    ['Plus Group']
  ])('matches active OpenAI group named %s', (name) => {
    expect(resolveDefaultOpenAIPlusGroup([
      makeGroup({ id: 8, name })
    ])).toBe(8)
  })

  it('ignores inactive or non-OpenAI groups and does not fall back to other names', () => {
    expect(resolveDefaultOpenAIPlusGroup([
      makeGroup({ id: 2, name: 'plus 组', status: 'inactive' }),
      makeGroup({ id: 3, name: 'plus 组', platform: 'anthropic' }),
      makeGroup({ id: 4, name: '标准余额' })
    ])).toBeNull()
  })

  it('only applies the default plus group for untouched OpenAI forms with empty group_ids', () => {
    expect(shouldApplyDefaultOpenAIPlusGroup({
      platform: 'openai',
      groupIds: [],
      userTouchedGroups: false
    })).toBe(true)
    expect(shouldApplyDefaultOpenAIPlusGroup({
      platform: 'openai',
      groupIds: [7],
      userTouchedGroups: false
    })).toBe(false)
    expect(shouldApplyDefaultOpenAIPlusGroup({
      platform: 'openai',
      groupIds: [],
      userTouchedGroups: true
    })).toBe(false)
    expect(shouldApplyDefaultOpenAIPlusGroup({
      platform: 'anthropic',
      groupIds: [],
      userTouchedGroups: false
    })).toBe(false)
  })
})
