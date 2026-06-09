import { describe, expect, it, vi } from 'vitest'
import { isBackendModePublicRouteAllowed } from '@/router/backendMode'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('router help route', () => {
  it('registers the user help route as an authenticated route', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'UserHelp')

    expect(route?.path).toBe('/help')
    expect(route?.meta.requiresAuth).toBe(true)
    expect(route?.meta.requiresAdmin).toBe(false)
    expect(route?.meta.titleKey).toBe('help.title')
    expect(route?.meta.descriptionKey).toBe('help.description')
  })

  it('requires the affiliate feature flag for the affiliate route', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'Affiliate')

    expect(route?.path).toBe('/affiliate')
    expect(route?.meta.requiresAffiliate).toBe(true)
  })

  it('requires the payment feature flag for purchase routes', async () => {
    const { default: router } = await import('@/router')

    expect(router.getRoutes().find((record) => record.name === 'PurchaseSubscription')?.meta.requiresPayment).toBe(true)
    expect(router.getRoutes().find((record) => record.name === 'OrderList')?.meta.requiresPayment).toBe(true)
    expect(router.getRoutes().find((record) => record.name === 'PaymentQRCode')?.meta.requiresPayment).toBe(true)
  })

  it('requires the payment feature flag for admin payment routes', async () => {
    const { default: router } = await import('@/router')
    const routeNames = ['AdminPaymentDashboard', 'AdminOrders', 'AdminPaymentPlans']

    for (const name of routeNames) {
      expect(router.getRoutes().find((record) => record.name === name)?.meta.requiresPayment).toBe(true)
    }
  })

  it('requires the affiliate feature flag for admin affiliate routes', async () => {
    const { default: router } = await import('@/router')
    const routeNames = ['AdminAffiliateInvites', 'AdminAffiliateRebates', 'AdminAffiliateTransfers']

    for (const name of routeNames) {
      expect(router.getRoutes().find((record) => record.name === name)?.meta.requiresAffiliate).toBe(true)
    }
  })

  it('allows the OAuth callback alias in backend mode', async () => {
    const { default: router } = await import('@/router')
    const resolved = router.resolve('/auth/oauth/callback')

    expect(resolved.matched.some((record) => record.name === 'OAuthCallback')).toBe(true)
    expect(resolved.meta.requiresAuth).toBe(false)
    expect(isBackendModePublicRouteAllowed(resolved.path, false)).toBe(true)
  })
})
