import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SubscriptionPlanCard from '../SubscriptionPlanCard.vue'
import type { SubscriptionPlan } from '@/types/payment'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const planFactory = (overrides: Partial<SubscriptionPlan>): SubscriptionPlan => ({
  id: 1,
  group_id: 10,
  group_platform: 'openai',
  group_name: 'OpenAI',
  rate_multiplier: 1,
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'],
  name: 'Codex 150K',
  description: '',
  price: 68,
  original_price: 150,
  validity_days: 30,
  validity_unit: 'day',
  features: [],
  for_sale: true,
  sort_order: 1,
  ...overrides,
})

describe('SubscriptionPlanCard', () => {
  it('hides Antigravity model scopes for OpenAI plans', () => {
    const wrapper = mount(SubscriptionPlanCard, {
      props: {
        plan: planFactory({
          group_platform: 'openai',
          supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'],
        }),
      },
    })

    expect(wrapper.text()).not.toContain('Claude')
    expect(wrapper.text()).not.toContain('Gemini')
    expect(wrapper.text()).not.toContain('Imagen')
    expect(wrapper.text()).not.toContain('payment.planCard.models')
  })

  it('shows Antigravity model scopes for Antigravity plans', () => {
    const wrapper = mount(SubscriptionPlanCard, {
      props: {
        plan: planFactory({
          group_platform: 'antigravity',
          supported_model_scopes: ['claude', 'gemini_text', 'gemini_image'],
        }),
      },
    })

    expect(wrapper.text()).toContain('Claude')
    expect(wrapper.text()).toContain('Gemini')
    expect(wrapper.text()).toContain('Imagen')
    expect(wrapper.text()).toContain('payment.planCard.models')
  })
})
