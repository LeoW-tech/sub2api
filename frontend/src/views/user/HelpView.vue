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
          <div class="grid gap-3 sm:grid-cols-2 lg:w-[520px] lg:grid-cols-5">
            <div
              v-for="item in visibleSummaryItems"
              :key="item.labelKey"
              class="rounded-2xl border border-white/70 bg-white/70 p-3 text-center shadow-sm backdrop-blur dark:border-dark-700/70 dark:bg-dark-800/70"
            >
              <div class="mx-auto flex h-9 w-9 items-center justify-center rounded-xl bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300">
                <Icon :name="item.icon" size="sm" />
              </div>
              <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
                {{ t(item.labelKey) }}
              </p>
            </div>
          </div>
        </div>
      </section>

      <section class="relative space-y-5 before:absolute before:bottom-6 before:left-6 before:top-6 before:hidden before:w-px before:bg-gradient-to-b before:from-primary-200 before:via-primary-100 before:to-transparent dark:before:from-primary-800/70 dark:before:via-primary-900/60 md:before:block">
        <article
          v-for="(step, index) in visibleSteps"
          :key="step.titleKey"
          class="relative grid gap-5 rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 md:grid-cols-[minmax(0,1fr)_360px] md:p-6"
        >
          <div class="flex gap-4">
            <div class="z-10 flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl bg-primary-600 text-white shadow-lg shadow-primary-500/20 dark:bg-primary-500">
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
                  v-for="action in visibleInternalActions(step)"
                  :key="action.labelKey"
                  :to="action.to"
                  class="btn"
                  :class="action.variant === 'secondary' ? 'btn-secondary' : 'btn-primary'"
                >
                  <span>{{ t(action.labelKey) }}</span>
                  <Icon name="arrowRight" size="sm" />
                </router-link>
                <a
                  v-for="action in externalActions(step)"
                  :key="action.labelKey"
                  :href="action.href"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="btn"
                  :class="action.variant === 'secondary' ? 'btn-secondary' : 'btn-primary'"
                >
                  <span>{{ t(action.labelKey) }}</span>
                  <Icon name="externalLink" size="sm" />
                </a>
              </div>
            </div>
          </div>

          <div
            data-testid="help-screenshot-card"
            class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="mb-3 flex items-center justify-between">
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t(step.cardTitleKey) }}
              </p>
              <div class="flex gap-1">
                <span class="h-2.5 w-2.5 rounded-full bg-red-300"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-emerald-300"></span>
              </div>
            </div>

            <div v-if="step.card === 'install'" class="space-y-3">
              <div class="rounded-xl bg-white p-4 shadow-sm dark:bg-dark-800">
                <div class="flex items-center gap-3">
                  <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                    <Icon name="download" size="md" />
                  </div>
                  <div>
                    <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('help.screenshots.install.official') }}</p>
                    <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('help.screenshots.install.backup') }}</p>
                  </div>
                </div>
              </div>
              <div class="rounded-xl border border-dashed border-primary-200 bg-primary-50/70 p-3 text-sm text-primary-700 dark:border-primary-900/50 dark:bg-primary-900/20 dark:text-primary-300">
                {{ t('help.steps.install.note') }}
              </div>
            </div>

            <div v-else-if="step.card === 'purchase'" class="space-y-3">
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

            <div v-else-if="step.card === 'key'" class="space-y-3">
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

            <div v-else-if="step.card === 'learn'" class="space-y-3">
              <div class="rounded-xl bg-white p-4 shadow-sm dark:bg-dark-800">
                <div class="flex items-center gap-3">
                  <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-300">
                    <Icon name="chatBubble" size="md" />
                  </div>
                  <div>
                    <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('help.screenshots.learn.prompt') }}</p>
                    <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('help.screenshots.learn.video') }}</p>
                  </div>
                </div>
              </div>
              <div class="grid gap-2">
                <div
                  v-for="action in externalActions(step)"
                  :key="action.labelKey"
                  class="flex items-center gap-2 rounded-xl bg-primary-50/70 p-3 text-sm font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                >
                  <Icon name="play" size="sm" />
                  <span>{{ t(action.labelKey) }}</span>
                </div>
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
import { CODEX_BACKUP_DOWNLOAD_URL, CODEX_OFFICIAL_DOWNLOAD_URL } from '@/constants/codexDownload'

const { t } = useI18n()
const authStore = useAuthStore()

type HelpIcon = 'chatBubble' | 'creditCard' | 'download' | 'gift' | 'key'
type HelpCardKind = 'affiliate' | 'install' | 'key' | 'learn' | 'purchase'
type HelpFeature = 'affiliate' | 'payment'

type HelpAction =
  | {
      type: 'internal'
      to: string
      labelKey: string
      variant?: 'primary' | 'secondary'
    }
  | {
      type: 'external'
      href: string
      labelKey: string
      variant?: 'primary' | 'secondary'
    }

interface SummaryItem {
  icon: HelpIcon
  labelKey: string
  feature?: HelpFeature
}

interface HelpStep {
  icon: HelpIcon
  titleKey: string
  descriptionKey: string
  noteKey?: string
  actions: HelpAction[]
  cardTitleKey: string
  card: HelpCardKind
  feature?: HelpFeature
}

const tutorialVideoUrls = [
  'https://www.bilibili.com/video/BV1Nd596vEyU/?share_source=copy_web&vd_source=8c193272be30b72d84fe725197bcdf59',
  'https://www.bilibili.com/video/BV1Kk9kBAEJv/?share_source=copy_web&vd_source=8c193272be30b72d84fe725197bcdf59',
  'https://www.bilibili.com/video/BV1w45v6kEDL/?share_source=copy_web&vd_source=8c193272be30b72d84fe725197bcdf59',
]

const summaryItems: SummaryItem[] = [
  { icon: 'download', labelKey: 'help.steps.install.title' },
  { icon: 'creditCard', labelKey: 'help.steps.purchase.title', feature: 'payment' },
  { icon: 'key', labelKey: 'help.steps.key.title' },
  { icon: 'chatBubble', labelKey: 'help.steps.learn.title' },
  { icon: 'gift', labelKey: 'help.steps.affiliate.title', feature: 'affiliate' },
] 

const steps: HelpStep[] = [
  {
    icon: 'download',
    titleKey: 'help.steps.install.title',
    descriptionKey: 'help.steps.install.description',
    noteKey: 'help.steps.install.note',
    actions: [
      { type: 'external', href: CODEX_OFFICIAL_DOWNLOAD_URL, labelKey: 'help.links.officialDownload' },
      { type: 'external', href: CODEX_BACKUP_DOWNLOAD_URL, labelKey: 'help.links.quarkDownload', variant: 'secondary' },
    ],
    cardTitleKey: 'help.screenshots.install.title',
    card: 'install',
  },
  {
    icon: 'creditCard',
    titleKey: 'help.steps.purchase.title',
    descriptionKey: 'help.steps.purchase.description',
    actions: [
      { type: 'internal', to: '/purchase', labelKey: 'help.links.purchase' },
      { type: 'internal', to: '/subscriptions', labelKey: 'help.links.subscriptions', variant: 'secondary' },
    ],
    cardTitleKey: 'help.screenshots.purchase.title',
    card: 'purchase',
    feature: 'payment',
  },
  {
    icon: 'key',
    titleKey: 'help.steps.key.title',
    descriptionKey: 'help.steps.key.description',
    noteKey: 'help.steps.key.note',
    actions: [
      { type: 'internal', to: '/keys', labelKey: 'help.links.keys' },
    ],
    cardTitleKey: 'help.screenshots.key.title',
    card: 'key',
  },
  {
    icon: 'chatBubble',
    titleKey: 'help.steps.learn.title',
    descriptionKey: 'help.steps.learn.description',
    noteKey: 'help.steps.learn.note',
    actions: [
      { type: 'external', href: tutorialVideoUrls[0], labelKey: 'help.links.videoFullGuide' },
      { type: 'external', href: tutorialVideoUrls[1], labelKey: 'help.links.videoAppGuide', variant: 'secondary' },
      { type: 'external', href: tutorialVideoUrls[2], labelKey: 'help.links.videoCompleteGuide', variant: 'secondary' },
    ],
    cardTitleKey: 'help.screenshots.learn.title',
    card: 'learn',
  },
  {
    icon: 'gift',
    titleKey: 'help.steps.affiliate.title',
    descriptionKey: 'help.steps.affiliate.description',
    actions: [
      { type: 'internal', to: '/affiliate', labelKey: 'help.links.affiliate' },
    ],
    cardTitleKey: 'help.screenshots.affiliate.title',
    card: 'affiliate',
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

function shouldShowAction(action: HelpAction): boolean {
  return !(action.type === 'internal' && authStore.isSimpleMode && action.to === '/subscriptions')
}

function visibleInternalActions(step: HelpStep): Array<Extract<HelpAction, { type: 'internal' }>> {
  return step.actions.filter((action): action is Extract<HelpAction, { type: 'internal' }> => (
    action.type === 'internal' && shouldShowAction(action)
  ))
}

function externalActions(step: HelpStep): Array<Extract<HelpAction, { type: 'external' }>> {
  return step.actions.filter((action): action is Extract<HelpAction, { type: 'external' }> => action.type === 'external')
}

const visibleSteps = computed(() => steps.filter((step) => isHelpFeatureVisible(step.feature)))
const visibleSummaryItems = computed(() => summaryItems.filter((item) => isHelpFeatureVisible(item.feature)))
</script>
