import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OAuthCallbackView from '@/views/auth/OAuthCallbackView.vue'

const { routeState, replaceMock, showErrorMock, showSuccessMock, copyToClipboardMock, completeOpenAIPendingCreateMock, authState } = vi.hoisted(() => ({
  routeState: {
    query: {} as Record<string, unknown>,
  },
  replaceMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  copyToClipboardMock: vi.fn(),
  completeOpenAIPendingCreateMock: vi.fn(),
  authState: {
    isAuthenticated: true,
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    replace: (...args: any[]) => replaceMock(...args),
  }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: (...args: any[]) => showErrorMock(...args),
    showSuccess: (...args: any[]) => showSuccessMock(...args),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState,
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: (...args: any[]) => copyToClipboardMock(...args),
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      completeOpenAIPendingCreate: (...args: any[]) => completeOpenAIPendingCreateMock(...args),
    },
  },
}))

describe('OAuthCallbackView', () => {
  beforeEach(() => {
    routeState.query = {}
    replaceMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
    copyToClipboardMock.mockReset()
    completeOpenAIPendingCreateMock.mockReset()
    authState.isAuthenticated = true
  })

  it('renders localized callback copy actions', () => {
    routeState.query = {
      code: 'oauth-code',
      state: 'oauth-state',
    }

    const wrapper = mount(OAuthCallbackView)

    expect(wrapper.text()).toContain('auth.oauth.callbackTitle')
    expect(wrapper.text()).toContain('auth.oauth.callbackHint')
    expect(wrapper.text()).toContain('common.copy')
    expect(wrapper.find('input[value="oauth-code"]').exists()).toBe(true)
    expect(wrapper.find('input[value="oauth-state"]').exists()).toBe(true)
  })

  it('sends callback errors to toast instead of rendering inline red text', () => {
    routeState.query = {
      error: 'oauth failed',
    }

    const wrapper = mount(OAuthCallbackView)

    expect(showErrorMock).toHaveBeenCalledWith('oauth failed')
    expect(wrapper.text()).not.toContain('oauth failed')
    expect(wrapper.find('.bg-red-50').exists()).toBe(false)
  })

  it('automatically completes OpenAI pending account creation and returns to accounts page', async () => {
    routeState.query = {
      admin_oauth_provider: 'openai',
      code: 'oauth-code',
      state: 'oauth-state',
    }
    completeOpenAIPendingCreateMock.mockResolvedValueOnce({ id: 10 })

    mount(OAuthCallbackView)
    await flushPromises()

    expect(completeOpenAIPendingCreateMock).toHaveBeenCalledWith({
      code: 'oauth-code',
      state: 'oauth-state',
    })
    expect(showSuccessMock).toHaveBeenCalled()
    expect(replaceMock).toHaveBeenCalledWith('/admin/accounts')
  })

  it('keeps the success callback visible when the current browser is not logged into admin', async () => {
    authState.isAuthenticated = false
    routeState.query = {
      admin_oauth_provider: 'openai',
      code: 'oauth-code',
      state: 'oauth-state',
    }
    completeOpenAIPendingCreateMock.mockResolvedValueOnce({ id: 10 })

    const wrapper = mount(OAuthCallbackView)
    await flushPromises()

    expect(completeOpenAIPendingCreateMock).toHaveBeenCalledWith({
      code: 'oauth-code',
      state: 'oauth-state',
    })
    expect(showSuccessMock).toHaveBeenCalled()
    expect(replaceMock).not.toHaveBeenCalled()
    expect(wrapper.find('input[value="oauth-code"]').exists()).toBe(true)
  })

  it('keeps the manual copy fallback visible when OpenAI pending completion fails', async () => {
    routeState.query = {
      admin_oauth_provider: 'openai',
      code: 'oauth-code',
      state: 'oauth-state',
    }
    completeOpenAIPendingCreateMock.mockRejectedValueOnce(new Error('expired session'))

    const wrapper = mount(OAuthCallbackView)
    await flushPromises()

    expect(showErrorMock).toHaveBeenCalled()
    expect(replaceMock).not.toHaveBeenCalled()
    expect(wrapper.find('input[value="oauth-code"]').exists()).toBe(true)
    expect(wrapper.find('input[value="oauth-state"]').exists()).toBe(true)
  })

  it('does not call pending completion for non-admin callback URLs', async () => {
    routeState.query = {
      code: 'oauth-code',
      state: 'oauth-state',
    }

    mount(OAuthCallbackView)
    await flushPromises()

    expect(completeOpenAIPendingCreateMock).not.toHaveBeenCalled()
    expect(replaceMock).not.toHaveBeenCalled()
  })
})
