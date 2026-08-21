import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AppHeader from '../AppHeader.vue'

const storeState = vi.hoisted(() => ({
  app: {
    contactInfo: 'QQ：2801799698',
    docUrl: '',
    cachedPublicSettings: null,
    toggleMobileSidebar: vi.fn()
  },
  auth: {
    user: {
      username: 'tester',
      email: 'tester@example.com',
      role: 'user',
      balance: 0,
      avatar_url: ''
    },
    isAdmin: false,
    isSimpleMode: false,
    logout: vi.fn()
  },
  onboarding: {
    replay: vi.fn()
  },
  adminSettings: {
    customMenuItems: []
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => storeState.app,
  useAuthStore: () => storeState.auth,
  useOnboardingStore: () => storeState.onboarding
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => storeState.app
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => storeState.adminSettings
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ name: 'Dashboard', meta: {}, params: {} })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        if (key === 'common.contactSupport') return '联系客服'
        if (key === 'common.balance') return '余额'
        if (key === 'nav.help') return '使用帮助'
        return key
      }
    })
  }
})

describe('AppHeader contact support placement', () => {
  beforeEach(() => {
    storeState.app.contactInfo = 'QQ：2801799698'
    storeState.auth.user = {
      username: 'tester',
      email: 'tester@example.com',
      role: 'user',
      balance: 0,
      avatar_url: ''
    }
  })

  function mountHeader() {
    return mount(AppHeader, {
      global: {
        stubs: {
          AnnouncementBell: {
            template: '<div data-testid="announcement-bell" />'
          },
          LocaleSwitcher: {
            template: '<div data-testid="locale-switcher" />'
          },
          SubscriptionProgressMini: {
            template: '<div data-testid="subscription-progress" />'
          },
          Icon: {
            props: ['name'],
            template: '<span :data-icon="name" />'
          },
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>'
          }
        },
        mocks: {
          $t: (key: string) => key
        }
      }
    })
  }

  it('renders help link before configured contact support', () => {
    const wrapper = mountHeader()
    const help = wrapper.get('[data-testid="header-help-link"]')
    const contact = wrapper.get('[data-testid="header-contact-support"]')

    expect(help.text()).toContain('使用帮助')
    expect(help.attributes('href')).toBe('/help')
    expect(help.classes()).toContain('lg:flex')
    expect(help.classes()).toContain('hidden')
    expect(help.classes()).toContain('border')
    expect(help.classes()).toContain('border-primary-200')
    expect(help.classes()).toContain('bg-primary-50')
    expect(help.classes()).toContain('font-semibold')
    expect(
      help.element.compareDocumentPosition(contact.element)
        & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
  })

  it('renders help in the user dropdown for compact screens', async () => {
    const wrapper = mountHeader()

    await wrapper.get('button[aria-label="common.userMenu"]').trigger('click')

    const helpLinks = wrapper.findAll('a').filter((link) => link.attributes('href') === '/help')
    expect(helpLinks).toHaveLength(2)
    expect(helpLinks[1].classes()).toContain('dropdown-item')
  })

  it('hides the help link when user is not logged in', () => {
    storeState.auth.user = null as unknown as typeof storeState.auth.user

    const wrapper = mountHeader()

    expect(wrapper.find('[data-testid="header-help-link"]').exists()).toBe(false)
  })

  it('renders configured contact support before the announcement bell', () => {
    const wrapper = mountHeader()
    const contact = wrapper.get('[data-testid="header-contact-support"]')
    const announcementBell = wrapper.get('[data-testid="announcement-bell"]')

    expect(contact.text()).toContain('联系客服:')
    expect(contact.text()).toContain('QQ：2801799698')
    expect(contact.classes()).toContain('lg:flex')
    expect(contact.classes()).not.toContain('xl:flex')
    expect(contact.classes()).toContain('max-w-[240px]')
    expect(
      contact.element.compareDocumentPosition(announcementBell.element)
        & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
  })

  it('hides the top contact support when contact info is not configured', () => {
    storeState.app.contactInfo = ''

    const wrapper = mountHeader()

    expect(wrapper.find('[data-testid="header-contact-support"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="announcement-bell"]').exists()).toBe(true)
  })
})
