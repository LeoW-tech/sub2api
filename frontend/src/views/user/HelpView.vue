<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <section class="overflow-hidden rounded-3xl border border-primary-100 bg-gradient-to-br from-primary-50 via-white to-sky-50 p-6 shadow-sm dark:border-primary-900/40 dark:from-primary-950/40 dark:via-dark-900 dark:to-sky-950/30 md:p-8">
        <div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
          <div class="max-w-2xl">
            <div class="mb-4 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-600 text-white shadow-lg shadow-primary-500/20">
              <Icon name="questionCircle" size="lg" />
            </div>
            <h2 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white md:text-3xl">
              {{ t('help.intro.title') }}
            </h2>
            <p class="mt-3 text-base leading-7 text-gray-600 dark:text-dark-300">
              {{ t('help.intro.description') }}
            </p>
          </div>
          <div class="grid gap-3 sm:grid-cols-3 lg:w-[420px]">
            <div
              v-for="item in visibleSummaryItems"
              :key="item.labelKey"
              class="rounded-2xl border border-white/70 bg-white/70 p-4 text-center shadow-sm backdrop-blur dark:border-dark-700/70 dark:bg-dark-800/70"
            >
              <div class="mx-auto flex h-10 w-10 items-center justify-center rounded-xl bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300">
                <Icon :name="item.icon" size="md" />
              </div>
              <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
                {{ t(item.labelKey) }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="space-y-5">
        <article
          v-for="(step, index) in visibleSteps"
          :key="step.titleKey"
          class="grid gap-5 rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 md:grid-cols-[minmax(0,1fr)_360px] md:p-6"
        >
          <div class="flex gap-4">
            <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
              <span class="text-sm font-semibold">{{ String(index + 1).padStart(2, '0') }}</span>
            </div>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <Icon :name="step.icon" size="md" class="text-primary-600 dark:text-primary-400" />
                <h3 class="text-xl font-semibold text-gray-900 dark:text-white">
                  {{ t(step.titleKey) }}
                </h3>
              </div>
              <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ t(step.descriptionKey) }}
              </p>
              <p
                v-if="step.noteKey"
                class="mt-3 rounded-2xl bg-gray-50 px-4 py-3 text-sm leading-6 text-gray-600 dark:bg-dark-900/60 dark:text-dark-300"
              >
                {{ t(step.noteKey) }}
              </p>

              <div class="mt-5 flex flex-wrap gap-3">
                <router-link
                  :to="step.primaryTo"
                  class="btn btn-primary"
                >
                  <span>{{ t(step.primaryLabelKey) }}</span>
                  <Icon name="arrowRight" size="sm" />
                </router-link>
                <router-link
                  v-if="shouldShowSecondaryLink(step)"
                  :to="step.secondaryTo"
                  class="btn btn-secondary"
                >
                  <span>{{ t(step.secondaryLabelKey) }}</span>
                  <Icon name="arrowRight" size="sm" />
                </router-link>
              </div>
            </div>
          </div>

          <div
            data-testid="help-screenshot-card"
            class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="mb-3 flex items-center justify-between">
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t(step.screenshotTitleKey) }}
              </p>
              <div class="flex gap-1">
                <span class="h-2.5 w-2.5 rounded-full bg-red-300"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-emerald-300"></span>
              </div>
            </div>

            <div v-if="step.screenshot === 'purchase'" class="space-y-3">
              <div class="rounded-xl bg-white p-4 shadow-sm dark:bg-dark-800">
                <div class="flex items-center justify-between">
                  <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('help.screenshots.purchase.wallet') }}</span>
                  <span class="rounded-full bg-primary-50 px-3 py-1 text-sm font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">$20.00</span>
                </div>
                <div class="mt-4 h-2 rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-2 w-2/3 rounded-full bg-primary-500"></div>
                </div>
              </div>
              <div class="rounded-xl border border-dashed border-primary-200 bg-primary-50/70 p-3 text-sm text-primary-700 dark:border-primary-900/50 dark:bg-primary-900/20 dark:text-primary-300">
                {{ t('help.screenshots.purchase.subscription') }}
              </div>
            </div>

            <div v-else-if="step.screenshot === 'key'" class="space-y-3">
              <div class="rounded-xl bg-white p-3 shadow-sm dark:bg-dark-800">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('help.screenshots.key.name') }}</p>
                    <p class="text-xs text-gray-500 dark:text-dark-400">sk-••••••••••</p>
                  </div>
                  <span class="rounded-lg bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                    {{ t('help.screenshots.key.useKey') }}
                  </span>
                </div>
              </div>
              <div class="rounded-xl bg-dark-950 p-3 font-mono text-xs leading-5 text-emerald-300">
                <p>OPENAI_API_KEY=sk-••••••</p>
                <p>OPENAI_BASE_URL=https://...</p>
              </div>
            </div>

            <div v-else class="space-y-3">
              <div class="rounded-xl bg-white p-3 shadow-sm dark:bg-dark-800">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('help.screenshots.affiliate.linkLabel') }}</p>
                <div class="mt-2 flex items-center gap-2 rounded-lg bg-gray-50 p-2 dark:bg-dark-900">
                  <code class="flex-1 truncate text-xs text-gray-700 dark:text-dark-200">/register?aff=INVITE</code>
                  <span class="rounded-md bg-primary-600 px-2 py-1 text-xs font-medium text-white">{{ t('help.screenshots.affiliate.copy') }}</span>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-2">
                <div class="rounded-xl bg-emerald-50 p-3 text-center dark:bg-emerald-900/20">
                  <p class="text-lg font-semibold text-emerald-600 dark:text-emerald-300">20%</p>
                  <p class="text-xs text-emerald-700 dark:text-emerald-300">{{ t('help.screenshots.affiliate.rebate') }}</p>
                </div>
                <div class="rounded-xl bg-white p-3 text-center shadow-sm dark:bg-dark-800">
                  <p class="text-lg font-semibold text-gray-900 dark:text-white">$8.00</p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('help.screenshots.affiliate.reward') }}</p>
                </div>
              </div>
            </div>
          </div>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()
const authStore = useAuthStore()

type HelpIcon = 'creditCard' | 'gift' | 'key'
type ScreenshotKind = 'affiliate' | 'key' | 'purchase'

interface SummaryItem {
  icon: HelpIcon
  labelKey: string
  feature?: 'affiliate' | 'payment'
}

interface HelpStep {
  icon: HelpIcon
  titleKey: string
  descriptionKey: string
  noteKey?: string
  primaryTo: string
  primaryLabelKey: string
  secondaryTo?: string
  secondaryLabelKey?: string
  screenshotTitleKey: string
  screenshot: ScreenshotKind
  feature?: 'affiliate' | 'payment'
}

const summaryItems: SummaryItem[] = [
  { icon: 'creditCard', labelKey: 'help.steps.purchase.title', feature: 'payment' },
  { icon: 'key', labelKey: 'help.steps.key.title' },
  { icon: 'gift', labelKey: 'help.steps.affiliate.title', feature: 'affiliate' },
] 

const steps: HelpStep[] = [
  {
    icon: 'creditCard',
    titleKey: 'help.steps.purchase.title',
    descriptionKey: 'help.steps.purchase.description',
    primaryTo: '/purchase',
    primaryLabelKey: 'help.links.purchase',
    secondaryTo: '/subscriptions',
    secondaryLabelKey: 'help.links.subscriptions',
    screenshotTitleKey: 'help.screenshots.purchase.title',
    screenshot: 'purchase',
    feature: 'payment',
  },
  {
    icon: 'key',
    titleKey: 'help.steps.key.title',
    descriptionKey: 'help.steps.key.description',
    noteKey: 'help.steps.key.note',
    primaryTo: '/keys',
    primaryLabelKey: 'help.links.keys',
    screenshotTitleKey: 'help.screenshots.key.title',
    screenshot: 'key',
  },
  {
    icon: 'gift',
    titleKey: 'help.steps.affiliate.title',
    descriptionKey: 'help.steps.affiliate.description',
    primaryTo: '/affiliate',
    primaryLabelKey: 'help.links.affiliate',
    screenshotTitleKey: 'help.screenshots.affiliate.title',
    screenshot: 'affiliate',
    feature: 'affiliate',
  },
]

function isHelpFeatureVisible(feature?: HelpStep['feature']): boolean {
  if (!feature) return true
  if (feature === 'payment') {
    // Keep opt-out payment semantics aligned with the sidebar and route guard.
    return isFeatureFlagEnabled(FeatureFlags.payment)
  }
  return isFeatureFlagEnabled(FeatureFlags.affiliate)
}

function shouldShowSecondaryLink(step: HelpStep): step is HelpStep & { secondaryTo: string; secondaryLabelKey: string } {
  if (!step.secondaryTo || !step.secondaryLabelKey) return false
  return !(authStore.isSimpleMode && step.secondaryTo === '/subscriptions')
}

const visibleSteps = computed(() => steps.filter((step) => isHelpFeatureVisible(step.feature)))
const visibleSummaryItems = computed(() => summaryItems.filter((item) => isHelpFeatureVisible(item.feature)))
</script>
