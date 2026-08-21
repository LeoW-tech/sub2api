<template>
  <BaseDialog
    :show="show"
    :title="t('keys.useKeyModal.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <!-- No Group Assigned Warning -->
      <div v-if="!platform" class="flex items-start gap-3 p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
        <svg class="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
        </svg>
        <div>
          <p class="text-sm font-medium text-yellow-800 dark:text-yellow-200">
            {{ t('keys.useKeyModal.noGroupTitle') }}
          </p>
          <p class="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
            {{ t('keys.useKeyModal.noGroupDescription') }}
          </p>
        </div>
      </div>

      <!-- Platform-specific content -->
      <template v-else>
        <!-- Description -->
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ platformDescription }}
        </p>

        <!-- Client Tabs -->
        <div v-if="clientTabs.length" class="overflow-x-auto border-b border-gray-200 dark:border-dark-700">
          <nav class="-mb-px flex min-w-max gap-4 sm:gap-6" aria-label="Client">
            <button
              v-for="tab in clientTabs"
              :key="tab.id"
              type="button"
              @click="activeClientTab = tab.id"
              :class="[
                'whitespace-nowrap py-2.5 px-1 border-b-2 font-medium text-sm transition-colors',
                activeClientTab === tab.id
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              ]"
            >
              <span class="flex items-center gap-2">
                <component :is="tab.icon" class="w-4 h-4" />
                {{ tab.label }}
              </span>
            </button>
          </nav>
        </div>

        <!-- Codex Authentication Mode -->
        <div
          v-if="showCodexAuthMode"
          class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
        >
          <div class="mb-2">
            <p class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('keys.useKeyModal.openai.authModeTitle') }}
            </p>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t('keys.useKeyModal.openai.authModeDescription') }}
            </p>
          </div>
          <div
            class="grid grid-cols-2 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-700"
            role="radiogroup"
            :aria-label="t('keys.useKeyModal.openai.authModeTitle')"
          >
            <button
              type="button"
              role="radio"
              data-testid="codex-auth-mode-legacy"
              :aria-checked="codexAuthMode === 'legacy'"
              :class="[
                'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                codexAuthMode === 'legacy'
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                  : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
              ]"
              @click="codexAuthMode = 'legacy'"
            >
              {{ t('keys.useKeyModal.openai.authModeLegacy') }}
            </button>
            <button
              type="button"
              role="radio"
              data-testid="codex-auth-mode-api-key"
              :aria-checked="codexAuthMode === 'api-key'"
              :class="[
                'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                codexAuthMode === 'api-key'
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                  : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
              ]"
              @click="codexAuthMode = 'api-key'"
            >
              {{ t('keys.useKeyModal.openai.authModeApiKey') }}
            </button>
          </div>
          <div
            v-if="codexAuthMode === 'api-key'"
            data-testid="codex-api-key-restart-notice"
            class="mt-3 flex items-start gap-2 border-l-2 border-amber-400 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800 dark:border-amber-500 dark:bg-amber-950/30 dark:text-amber-200"
          >
            <Icon name="exclamationCircle" size="sm" class="mt-0.5 flex-shrink-0" />
            <p>{{ t('keys.useKeyModal.openai.authModeApiKeyRestartNotice') }}</p>
          </div>
        </div>

        <template v-if="isOpenAICodex">
          <!-- Beginner one-command setup -->
          <section class="rounded-xl border border-primary-200 dark:border-primary-800 bg-primary-50/60 dark:bg-primary-900/10 p-4 space-y-4">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h3 class="text-base font-semibold text-gray-900 dark:text-gray-100">
                  {{ t('keys.useKeyModal.openai.beginner.title') }}
                </h3>
              </div>
              <span class="inline-flex w-fit items-center rounded-full bg-primary-100 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
                {{ detectedSystemLabel }}
              </span>
            </div>

            <div v-if="!isSupportedOneClickSystem" class="rounded-lg border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800 dark:border-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-200">
              {{ t('keys.useKeyModal.openai.unsupportedSystem') }}
            </div>

            <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ t('keys.useKeyModal.openai.steps.install') }}
              </p>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {{ t('keys.useKeyModal.openai.installCodexDescription') }}
              </p>
              <div class="mt-2 flex flex-col items-start gap-2">
                <a
                  :href="CODEX_OFFICIAL_DOWNLOAD_URL"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
                >
                  {{ t('keys.useKeyModal.openai.downloadCodex') }}
                </a>
                <a
                  :href="CODEX_BACKUP_DOWNLOAD_URL"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-sm font-medium text-gray-700 hover:text-gray-900 dark:text-gray-200 dark:hover:text-white"
                >
                  {{ t('keys.useKeyModal.openai.backupDownload') }}
                </a>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ t('keys.useKeyModal.openai.steps.quit') }}
              </p>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {{ t('keys.useKeyModal.openai.quitCodexDescriptionPrefix') }}
              </p>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {{ t('keys.useKeyModal.openai.quitCodexDescription') }}
              </p>
            </div>

            <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
              <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                    {{ terminalStepTitle }}
                  </p>
                  <ol class="mt-2 list-decimal space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-400">
                    <li v-for="step in terminalSteps" :key="step">{{ step }}</li>
                  </ol>
                </div>
                <div class="w-full max-w-md">
                  <div class="rounded-xl bg-gray-950 p-5 font-mono text-sm text-green-300 shadow-inner" :class="detectedSystem === 'windows' ? 'bg-blue-950 text-blue-100' : ''">
                    <div class="mb-4 flex items-center gap-2">
                      <span class="h-3 w-3 rounded-full bg-red-400"></span>
                      <span class="h-3 w-3 rounded-full bg-yellow-400"></span>
                      <span class="h-3 w-3 rounded-full bg-green-400"></span>
                      <span class="ml-3 text-xs text-gray-300">{{ terminalWindowTitle }}</span>
                    </div>
                    <div class="py-3">{{ terminalPromptPreview }}</div>
                  </div>
                  <p class="mt-2 text-center text-xs text-gray-500 dark:text-gray-400">
                    {{ terminalPreviewCaption }}
                  </p>
                </div>
              </div>
              <div class="mt-3 flex flex-wrap gap-2">
                <button
                  v-for="item in terminalCopyItems"
                  :key="item"
                  type="button"
                  data-testid="copy-snippet-button"
                  class="rounded-lg bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600"
                  @click="copySnippet(item)"
                >
                  {{ t('keys.useKeyModal.openai.copyInput') }}：{{ item }}
                </button>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ t('keys.useKeyModal.openai.steps.copyCommand') }}
              </p>
              <ol class="list-decimal space-y-1 pl-5 text-sm text-gray-600 dark:text-gray-400">
                <li>
                  <button
                    type="button"
                    data-testid="copy-codex-one-click-command"
                    class="btn btn-primary btn-sm"
                    @click="copySnippet(oneClickCommand)"
                  >
                    {{ t('keys.useKeyModal.openai.copyCommand') }}
                  </button>
                </li>
                <li>{{ commandPasteStep }}</li>
                <li>{{ t('keys.useKeyModal.openai.commandStep3') }}</li>
              </ol>
            </div>
          </section>

          <!-- Professional manual setup -->
          <section class="rounded-xl border border-gray-200 dark:border-dark-700 p-4 space-y-4">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-gray-100">
                {{ t('keys.useKeyModal.openai.professional.title') }}
              </h3>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                {{ t('keys.useKeyModal.openai.professional.description') }}
              </p>
            </div>

            <section class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
              <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ t('keys.useKeyModal.openai.professional.steps.openDir') }}
              </h4>
              <div class="mt-3 grid gap-3 md:grid-cols-2">
                <div>
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100">macOS</p>
                  <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                    {{ t('keys.useKeyModal.openai.professional.macOpenDir') }}
                  </p>
                  <div class="mt-2 flex flex-wrap gap-2">
                    <button type="button" data-testid="copy-snippet-button" class="rounded-lg bg-white px-2.5 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-200" @click="copySnippet('Command + Shift + G')">Command + Shift + G</button>
                    <button type="button" data-testid="copy-snippet-button" class="rounded-lg bg-white px-2.5 py-1 text-xs font-mono font-medium text-gray-700 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-200" @click="copySnippet('~/.codex')">~/.codex</button>
                  </div>
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100">Windows</p>
                  <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                    {{ t('keys.useKeyModal.openai.professional.windowsOpenDir') }}
                  </p>
                  <div class="mt-2 flex flex-wrap gap-2">
                    <button type="button" data-testid="copy-snippet-button" class="rounded-lg bg-white px-2.5 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-200" @click="copySnippet('Win + R')">Win + R</button>
                    <button type="button" data-testid="copy-snippet-button" class="rounded-lg bg-white px-2.5 py-1 text-xs font-mono font-medium text-gray-700 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-200" @click="copySnippet('%USERPROFILE%\\.codex')">%USERPROFILE%\.codex</button>
                  </div>
                </div>
              </div>
            </section>

            <section
              v-if="configTomlFile"
              data-testid="professional-config-toml-step"
              class="space-y-3"
            >
              <div>
                <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  {{ t('keys.useKeyModal.openai.professional.steps.configToml') }}
                </h4>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                  {{ t('keys.useKeyModal.openai.professional.configTomlDownloadHint') }}
                </p>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                  {{ t('keys.useKeyModal.openai.professional.configTomlCopyHint') }}
                </p>
              </div>
              <div class="relative">
                <p v-if="configTomlFile.hint" class="text-xs text-amber-600 dark:text-amber-400 mb-1.5 flex items-center gap-1">
                  <Icon name="exclamationCircle" size="sm" class="flex-shrink-0" />
                  {{ configTomlFile.hint }}
                </p>
                <div class="bg-gray-900 dark:bg-dark-900 rounded-xl overflow-hidden">
                  <div class="flex items-center justify-between px-4 py-2 bg-gray-800 dark:bg-dark-800 border-b border-gray-700 dark:border-dark-700">
                    <span class="text-xs text-gray-400 font-mono">{{ configTomlFile.path }}</span>
                    <div class="flex gap-2">
                      <button type="button" data-testid="download-config-button" @click="downloadConfigFile(configTomlFile)" class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white transition-colors">
                        {{ t('keys.useKeyModal.download') }}
                      </button>
                      <button type="button" @click="copyContent(configTomlFile.content, 0)" class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg transition-colors" :class="copiedIndex === 0 ? 'bg-green-500/20 text-green-400' : 'bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white'">
                        {{ copiedIndex === 0 ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy') }}
                      </button>
                    </div>
                  </div>
                  <pre class="p-4 text-sm font-mono text-gray-100 overflow-x-auto"><code v-text="configTomlFile.content"></code></pre>
                </div>
              </div>
            </section>

            <section
              v-if="authJsonFile"
              data-testid="professional-auth-json-step"
              class="space-y-3"
            >
              <div>
                <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  {{ t('keys.useKeyModal.openai.professional.steps.authJson') }}
                </h4>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
                  {{ t('keys.useKeyModal.openai.professional.authJsonDownloadHint') }}
                </p>
              </div>
              <div class="relative">
                <div class="bg-gray-900 dark:bg-dark-900 rounded-xl overflow-hidden">
                  <div class="flex items-center justify-between px-4 py-2 bg-gray-800 dark:bg-dark-800 border-b border-gray-700 dark:border-dark-700">
                    <span class="text-xs text-gray-400 font-mono">{{ authJsonFile.path }}</span>
                    <div class="flex gap-2">
                      <button type="button" data-testid="download-config-button" @click="downloadConfigFile(authJsonFile)" class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white transition-colors">
                        {{ t('keys.useKeyModal.download') }}
                      </button>
                      <button type="button" @click="copyContent(authJsonFile.content, 1)" class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg transition-colors" :class="copiedIndex === 1 ? 'bg-green-500/20 text-green-400' : 'bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white'">
                        {{ copiedIndex === 1 ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy') }}
                      </button>
                    </div>
                  </div>
                  <pre class="p-4 text-sm font-mono text-gray-100 overflow-x-auto"><code v-text="authJsonFile.content"></code></pre>
                </div>
              </div>
            </section>
          </section>
        </template>

        <template v-else>
          <!-- OS/Shell Tabs -->
          <div v-if="showShellTabs" class="border-b border-gray-200 dark:border-dark-700">
            <nav class="-mb-px flex space-x-4" aria-label="Tabs">
              <button
                v-for="tab in currentTabs"
                :key="tab.id"
                @click="activeTab = tab.id"
                :class="[
                  'whitespace-nowrap py-2.5 px-1 border-b-2 font-medium text-sm transition-colors',
                  activeTab === tab.id
                    ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                ]"
              >
                <span class="flex items-center gap-2">
                  <component :is="tab.icon" class="w-4 h-4" />
                  {{ tab.label }}
                </span>
              </button>
            </nav>
          </div>

          <!-- Code Blocks (Stacked for multi-file platforms) -->
          <div class="space-y-4">
            <div
              v-for="(file, index) in currentFiles"
              :key="index"
              class="relative"
            >
              <!-- File Hint (if exists) -->
              <p v-if="file.hint" class="text-xs text-amber-600 dark:text-amber-400 mb-1.5 flex items-center gap-1">
                <Icon name="exclamationCircle" size="sm" class="flex-shrink-0" />
                {{ file.hint }}
              </p>
              <div class="bg-gray-900 dark:bg-dark-900 rounded-xl overflow-hidden">
                <!-- Code Header -->
                <div class="flex items-center justify-between px-4 py-2 bg-gray-800 dark:bg-dark-800 border-b border-gray-700 dark:border-dark-700">
                  <span class="text-xs text-gray-400 font-mono">{{ file.path }}</span>
                  <button
                    @click="copyContent(file.content, index)"
                    class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg transition-colors"
                    :class="copiedIndex === index
                      ? 'bg-green-500/20 text-green-400'
                      : 'bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white'"
                  >
                    <svg v-if="copiedIndex === index" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                    <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                    </svg>
                    {{ copiedIndex === index ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy') }}
                  </button>
                </div>
                <!-- Code Content -->
                <pre class="p-4 text-sm font-mono text-gray-100 overflow-x-auto"><code v-if="file.highlighted" v-html="file.highlighted"></code><code v-else v-text="file.content"></code></pre>
              </div>
            </div>
          </div>
        </template>
        <!-- Usage Note -->
        <div v-if="showPlatformNote" class="flex items-start gap-3 p-3 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-100 dark:border-blue-800">
          <Icon name="infoCircle" size="md" class="text-blue-500 flex-shrink-0 mt-0.5" />
          <p class="text-sm text-blue-700 dark:text-blue-300">
            {{ platformNote }}
          </p>
        </div>
      </template>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button
          @click="emit('close')"
          class="btn btn-secondary"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, h, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { CODEX_BACKUP_DOWNLOAD_URL, CODEX_OFFICIAL_DOWNLOAD_URL } from '@/constants/codexDownload'
import type { GroupPlatform } from '@/types'

interface Props {
  show: boolean
  apiKey: string
  baseUrl: string
  platform: GroupPlatform | null
  allowMessagesDispatch?: boolean
}

interface Emits {
  (e: 'close'): void
}

interface TabConfig {
  id: string
  label: string
  icon: Component
}

interface FileConfig {
  path: string
  content: string
  hint?: string  // Optional hint message for this file
  highlighted?: string
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const { copyToClipboard: clipboardCopy } = useClipboard()

const copiedIndex = ref<number | null>(null)
const activeTab = ref<string>('unix')
const activeClientTab = ref<string>('claude')
type CodexAuthMode = 'legacy' | 'api-key'
const codexAuthMode = ref<CodexAuthMode>('legacy')

// Reset tabs when platform changes
const defaultClientTab = computed(() => {
  switch (props.platform) {
    case 'openai':
      return 'codex'
    case 'grok':
      return 'grok'
    case 'gemini':
      return 'gemini'
    case 'antigravity':
      return 'claude'
    default:
      return 'claude'
  }
})

watch(() => props.platform, () => {
  activeTab.value = 'unix'
  activeClientTab.value = defaultClientTab.value
  codexAuthMode.value = 'legacy'
}, { immediate: true })

watch(() => props.show, (show) => {
  if (show) codexAuthMode.value = 'legacy'
})

// Reset shell tab when client changes
watch(activeClientTab, () => {
  activeTab.value = 'unix'
})

// Icon components
const AppleIcon = {
  render() {
    return h('svg', {
      fill: 'currentColor',
      viewBox: '0 0 24 24',
      class: 'w-4 h-4'
    }, [
      h('path', { d: 'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z' })
    ])
  }
}

const WindowsIcon = {
  render() {
    return h('svg', {
      fill: 'currentColor',
      viewBox: '0 0 24 24',
      class: 'w-4 h-4'
    }, [
      h('path', { d: 'M3 12V6.75l6-1.32v6.48L3 12zm17-9v8.75l-10 .15V5.21L20 3zM3 13l6 .09v6.81l-6-1.15V13zm7 .25l10 .15V21l-10-1.91v-5.84z' })
    ])
  }
}

// Terminal icon for Claude Code
const TerminalIcon = {
  render() {
    return h('svg', {
      fill: 'none',
      stroke: 'currentColor',
      viewBox: '0 0 24 24',
      'stroke-width': '1.5',
      class: 'w-4 h-4'
    }, [
      h('path', {
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'm6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 17.25V6.75A2.25 2.25 0 0 0 18.75 4.5H5.25A2.25 2.25 0 0 0 3 6.75v10.5A2.25 2.25 0 0 0 5.25 20.25Z'
      })
    ])
  }
}

// Sparkle icon for Gemini
const SparkleIcon = {
  render() {
    return h('svg', {
      fill: 'none',
      stroke: 'currentColor',
      viewBox: '0 0 24 24',
      'stroke-width': '1.5',
      class: 'w-4 h-4'
    }, [
      h('path', {
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z'
      })
    ])
  }
}

const clientTabs = computed((): TabConfig[] => {
  if (!props.platform) return []
  switch (props.platform) {
    case 'openai': {
      const tabs: TabConfig[] = [
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'codex-ws', label: t('keys.useKeyModal.cliTabs.codexCliWs'), icon: TerminalIcon }
      ]
      if (props.allowMessagesDispatch) {
        tabs.push({ id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon })
      }
      tabs.push({ id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon })
      return tabs
    }
    case 'gemini':
      return [
        { id: 'gemini', label: t('keys.useKeyModal.cliTabs.geminiCli'), icon: SparkleIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    case 'antigravity':
      return [
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'gemini', label: t('keys.useKeyModal.cliTabs.geminiCli'), icon: SparkleIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    case 'grok':
      return [
        { id: 'grok', label: t('keys.useKeyModal.cliTabs.grokCli'), icon: TerminalIcon },
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    default:
      return [
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
  }
})

// Shell tabs (3 types for environment variable based configs)
const shellTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: AppleIcon },
  { id: 'cmd', label: 'Windows CMD', icon: WindowsIcon },
  { id: 'powershell', label: 'PowerShell', icon: WindowsIcon }
]

// OpenAI tabs (2 OS types)
const openaiTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: AppleIcon },
  { id: 'windows', label: 'Windows', icon: WindowsIcon }
]

const showCodexAuthMode = computed(() =>
  props.platform === 'openai' &&
  (activeClientTab.value === 'codex' || activeClientTab.value === 'codex-ws')
)

const isOpenAICodex = computed(() => props.platform === 'openai' && activeClientTab.value === 'codex')

const showShellTabs = computed(() => activeClientTab.value !== 'opencode')

const currentTabs = computed(() => {
  if (!showShellTabs.value) return []
  if (activeClientTab.value === 'codex' || activeClientTab.value === 'codex-ws' || activeClientTab.value === 'grok') {
    return openaiTabs
  }
  return shellTabs
})

const platformDescription = computed(() => {
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.description')
      }
      return t('keys.useKeyModal.openai.description')
    case 'gemini':
      return t('keys.useKeyModal.gemini.description')
    case 'antigravity':
      return t('keys.useKeyModal.antigravity.description')
    case 'grok':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.grok.claudeDescription')
      }
      if (activeClientTab.value === 'codex') {
        return t('keys.useKeyModal.grok.codexDescription')
      }
      return t('keys.useKeyModal.grok.description')
    default:
      return t('keys.useKeyModal.description')
  }
})

const platformNote = computed(() => {
  switch (props.platform) {
    case 'openai':
      return activeTab.value === 'windows'
        ? t('keys.useKeyModal.openai.noteWindows')
        : t('keys.useKeyModal.openai.note')
    case 'gemini':
      return t('keys.useKeyModal.gemini.note')
    case 'antigravity':
      return activeClientTab.value === 'claude'
        ? t('keys.useKeyModal.antigravity.claudeNote')
        : t('keys.useKeyModal.antigravity.geminiNote')
    case 'grok':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.grok.claudeNote')
      }
      if (activeClientTab.value === 'codex') {
        return activeTab.value === 'windows'
          ? t('keys.useKeyModal.grok.codexNoteWindows')
          : t('keys.useKeyModal.grok.codexNote')
      }
      // Grok CLI: shell-specific path guidance (env + ~/.grok/config.toml).
      if (activeClientTab.value === 'grok' && (activeTab.value === 'cmd' || activeTab.value === 'powershell')) {
        return t('keys.useKeyModal.grok.noteWindows')
      }
      if (activeClientTab.value === 'grok' && activeTab.value === 'windows') {
        return t('keys.useKeyModal.grok.noteWindows')
      }
      return t('keys.useKeyModal.grok.note')
    default:
      return t('keys.useKeyModal.note')
  }
})

const showPlatformNote = computed(() => !isOpenAICodex.value && activeClientTab.value !== 'opencode')

type DetectedSystem = 'mac' | 'windows' | 'other'

const detectedSystem = computed<DetectedSystem>(() => {
  const userAgent = window.navigator.userAgent.toLowerCase()
  if (userAgent.includes('mac')) return 'mac'
  if (userAgent.includes('win')) return 'windows'
  return 'other'
})

const isSupportedOneClickSystem = computed(() => detectedSystem.value === 'mac' || detectedSystem.value === 'windows')

const detectedSystemLabel = computed(() => {
  if (detectedSystem.value === 'mac') return t('keys.useKeyModal.openai.detectedMac')
  if (detectedSystem.value === 'windows') return t('keys.useKeyModal.openai.detectedWindows')
  return t('keys.useKeyModal.openai.detectedOther')
})

const terminalStepTitle = computed(() =>
  detectedSystem.value === 'windows'
    ? t('keys.useKeyModal.openai.openPowerShellTitle')
    : t('keys.useKeyModal.openai.steps.openTerminal')
)

const terminalSteps = computed(() => {
  if (detectedSystem.value === 'windows') {
    return [
      t('keys.useKeyModal.openai.windowsTerminalStep1'),
      t('keys.useKeyModal.openai.windowsTerminalStep2')
    ]
  }
  return [
    t('keys.useKeyModal.openai.macTerminalStep1'),
    t('keys.useKeyModal.openai.macTerminalStep2')
  ]
})

const terminalCopyItems = computed(() =>
  detectedSystem.value === 'windows'
    ? ['powershell']
    : ['终端', 'Terminal']
)

const terminalWindowTitle = computed(() =>
  detectedSystem.value === 'windows' ? 'Windows PowerShell' : 'Terminal'
)

const terminalPromptPreview = computed(() =>
  detectedSystem.value === 'windows' ? 'PS C:\\Users\\you>' : '$'
)

const terminalPreviewCaption = computed(() =>
  detectedSystem.value === 'windows'
    ? t('keys.useKeyModal.openai.powerShellPreviewCaption')
    : t('keys.useKeyModal.openai.terminalPreviewCaption')
)

const commandPasteStep = computed(() =>
  detectedSystem.value === 'windows'
    ? t('keys.useKeyModal.openai.commandStep2Windows')
    : t('keys.useKeyModal.openai.commandStep2Mac')
)

const escapeHtml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const wrapToken = (className: string, value: string) =>
  `<span class="${className}">${escapeHtml(value)}</span>`

const keyword = (value: string) => wrapToken('text-emerald-300', value)
const variable = (value: string) => wrapToken('text-sky-200', value)
const operator = (value: string) => wrapToken('text-slate-400', value)
const string = (value: string) => wrapToken('text-amber-200', value)
const comment = (value: string) => wrapToken('text-slate-500', value)

// Syntax highlighting helpers
// Generate file configs based on platform and active tab
const currentFiles = computed((): FileConfig[] => {
  const baseUrl = props.baseUrl || window.location.origin
  const apiKey = props.apiKey
  const baseRoot = baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  const ensureV1 = (value: string) => {
    const trimmed = value.replace(/\/+$/, '')
    return trimmed.endsWith('/v1') ? trimmed : `${trimmed}/v1`
  }
  const apiBase = ensureV1(baseRoot)
  const antigravityBase = ensureV1(`${baseRoot}/antigravity`)
  const antigravityGeminiBase = (() => {
    const trimmed = `${baseRoot}/antigravity`.replace(/\/+$/, '')
    return trimmed.endsWith('/v1beta') ? trimmed : `${trimmed}/v1beta`
  })()
  const geminiBase = (() => {
    const trimmed = baseRoot.replace(/\/+$/, '')
    return trimmed.endsWith('/v1beta') ? trimmed : `${trimmed}/v1beta`
  })()

  if (activeClientTab.value === 'opencode') {
    switch (props.platform) {
      case 'anthropic':
        return [generateOpenCodeConfig('anthropic', apiBase, apiKey)]
      case 'openai':
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
      case 'gemini':
        return [generateOpenCodeConfig('gemini', geminiBase, apiKey)]
      case 'antigravity':
        return [
          generateOpenCodeConfig('antigravity-claude', antigravityBase, apiKey, 'opencode.json (Claude)'),
          generateOpenCodeConfig('antigravity-gemini', antigravityGeminiBase, apiKey, 'opencode.json (Gemini)')
        ]
      case 'grok':
        return [generateOpenCodeConfig('grok', apiBase, apiKey)]
      default:
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
    }
  }

  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return generateAnthropicFiles(baseUrl, apiKey)
      }
      if (activeClientTab.value === 'codex-ws') {
        return generateOpenAIWsFiles(baseUrl, apiKey)
      }
      return generateOpenAIFiles(baseUrl, apiKey)
    case 'gemini':
      return [generateGeminiCliContent(baseUrl, apiKey)]
    case 'antigravity':
      if (activeClientTab.value === 'gemini') {
        return [generateGeminiCliContent(`${baseUrl}/antigravity`, apiKey)]
      }
      return generateAnthropicFiles(`${baseUrl}/antigravity`, apiKey)
    case 'grok':
      if (activeClientTab.value === 'claude') {
        return generateGrokClaudeFiles(baseRoot, apiKey)
      }
      if (activeClientTab.value === 'codex') {
        return generateGrokCodexFiles(apiBase, apiKey)
      }
      return generateGrokFiles(apiBase, apiKey)
    default:
      return generateAnthropicFiles(baseUrl, apiKey)
  }
})

function generateAnthropicFiles(baseUrl: string, apiKey: string): FileConfig[] {
  let path: string
  let content: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export ANTHROPIC_BASE_URL="${baseUrl}"
export ANTHROPIC_AUTH_TOKEN="${apiKey}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
export CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set ANTHROPIC_BASE_URL=${baseUrl}
set ANTHROPIC_AUTH_TOKEN=${apiKey}
set CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
set CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:ANTHROPIC_BASE_URL="${baseUrl}"
$env:ANTHROPIC_AUTH_TOKEN="${apiKey}"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
$env:CLAUDE_CODE_ATTRIBUTION_HEADER=0`
      break
    default:
      path = 'Terminal'
      content = ''
  }

  const vscodeSettingsPath = activeTab.value === 'unix'
    ? '~/.claude/settings.json'
    : '%USERPROFILE%\\.claude\\settings.json'

  const vscodeContent = `{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "env": {
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_AUTH_TOKEN": "${apiKey}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}`

  return [
    { path, content },
    {
      path: vscodeSettingsPath,
      content: vscodeContent,
      hint: t('keys.useKeyModal.claudeSettingsHint')
    }
  ]
}

function generateGrokClaudeFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const environment = {
    ANTHROPIC_BASE_URL: baseUrl,
    ANTHROPIC_AUTH_TOKEN: apiKey,
    ANTHROPIC_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_OPUS_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_SONNET_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_HAIKU_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_FABLE_MODEL: 'grok-4.5',
    CLAUDE_CODE_SUBAGENT_MODEL: 'grok-4.5',
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: '1',
    CLAUDE_CODE_ATTRIBUTION_HEADER: '0'
  }
  let path: string
  let content: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = Object.entries(environment)
        .map(([name, value]) => `export ${name}="${value}"`)
        .join('\n')
      break
    case 'cmd':
      path = 'Command Prompt'
      content = Object.entries(environment)
        .map(([name, value]) => `set ${name}=${value}`)
        .join('\n')
      break
    case 'powershell':
      path = 'PowerShell'
      content = Object.entries(environment)
        .map(([name, value]) => `$env:${name}="${value}"`)
        .join('\n')
      break
    default:
      path = 'Terminal'
      content = ''
  }

  const settingsPath = activeTab.value === 'unix'
    ? '~/.claude/settings.json'
    : '%USERPROFILE%\\.claude\\settings.json'

  return [
    { path, content },
    {
      path: settingsPath,
      content: JSON.stringify({
        $schema: 'https://json.schemastore.org/claude-code-settings.json',
        env: environment
      }, null, 2),
      hint: t('keys.useKeyModal.claudeSettingsHint')
    }
  ]
}

function generateGeminiCliContent(baseUrl: string, apiKey: string): FileConfig {
  const model = 'gemini-2.0-flash'
  const modelComment = t('keys.useKeyModal.gemini.modelComment')
  let path: string
  let content: string
  let highlighted: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export GOOGLE_GEMINI_BASE_URL="${baseUrl}"
export GEMINI_API_KEY="${apiKey}"
export GEMINI_MODEL="${model}"  # ${modelComment}`
      highlighted = `${keyword('export')} ${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(`"${baseUrl}"`)}
${keyword('export')} ${variable('GEMINI_API_KEY')}${operator('=')}${string(`"${apiKey}"`)}
${keyword('export')} ${variable('GEMINI_MODEL')}${operator('=')}${string(`"${model}"`)}  ${comment(`# ${modelComment}`)}`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set GOOGLE_GEMINI_BASE_URL=${baseUrl}
set GEMINI_API_KEY=${apiKey}
set GEMINI_MODEL=${model}`
      highlighted = `${keyword('set')} ${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(baseUrl)}
${keyword('set')} ${variable('GEMINI_API_KEY')}${operator('=')}${string(apiKey)}
${keyword('set')} ${variable('GEMINI_MODEL')}${operator('=')}${string(model)}
${comment(`REM ${modelComment}`)}`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:GOOGLE_GEMINI_BASE_URL="${baseUrl}"
$env:GEMINI_API_KEY="${apiKey}"
$env:GEMINI_MODEL="${model}"  # ${modelComment}`
      highlighted = `${keyword('$env:')}${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(`"${baseUrl}"`)}
${keyword('$env:')}${variable('GEMINI_API_KEY')}${operator('=')}${string(`"${apiKey}"`)}
${keyword('$env:')}${variable('GEMINI_MODEL')}${operator('=')}${string(`"${model}"`)}  ${comment(`# ${modelComment}`)}`
      break
    default:
      path = 'Terminal'
      content = ''
      highlighted = ''
  }

  return { path, content, highlighted }
}

function generateOpenAIFiles(baseUrl: string, apiKey: string, isWindows = activeTab.value === 'windows'): FileConfig[] {
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'

  // config.toml content
  const configContent = `model_provider = "OpenAI"
model = "gpt-5.6-sol"
review_model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
${generateCodexProviderAuthConfig()}
requires_openai_auth = true

[features]
goals = true`

  // auth.json content
  const authContent = `{
  "OPENAI_API_KEY": "${apiKey}"
}`

  return [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    },
    {
      path: `${configDir}/auth.json`,
      content: authContent
    }
  ]
}

function generateCodexProviderAuthConfig(): string {
  if (codexAuthMode.value === 'api-key') {
    return `requires_openai_auth = false
http_headers = { "x-openai-actor-authorization" = "local-image-extension" }`
  }

  return 'requires_openai_auth = true'
}

function generateOpenAIWsFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'

  const configContent = `model_provider = "OpenAI"
model = "gpt-5.6-sol"
review_model = "gpt-5.6-sol"
model_reasoning_effort = "xhigh"
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "${baseUrl}"
wire_api = "responses"
supports_websockets = true
${generateCodexProviderAuthConfig()}

[features]
responses_websockets_v2 = true
goals = true`

  const authContent = `{
  "OPENAI_API_KEY": "${apiKey}"
}`

  return [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    },
    {
      path: `${configDir}/auth.json`,
      content: authContent
    }
  ]
}

function joinConfigPath(dir: string, file: string, windows: boolean): string {
  if (!windows) return `${dir}/${file}`
  return `${dir}\\${file}`
}

function generateGrokFiles(baseUrl: string, apiKey: string): FileConfig[] {
  // Prefer unix/cmd/powershell when shell tabs are shown; fall back to windows tab.
  const shell = activeTab.value
  const isWindowsPath = shell === 'windows' || shell === 'cmd' || shell === 'powershell'
  const configDir = isWindowsPath ? '%userprofile%\\.grok' : '~/.grok'

  let envPath: string
  let envContent: string
  switch (shell) {
    case 'cmd':
      envPath = 'Command Prompt'
      envContent = `set GROK_MODELS_BASE_URL=${baseUrl}
set XAI_API_KEY=${apiKey}`
      break
    case 'powershell':
    case 'windows':
      envPath = 'PowerShell'
      envContent = `$env:GROK_MODELS_BASE_URL="${baseUrl}"
$env:XAI_API_KEY="${apiKey}"`
      break
    default:
      envPath = 'Terminal'
      envContent = `export GROK_MODELS_BASE_URL="${baseUrl}"
export XAI_API_KEY="${apiKey}"`
  }

  // Shape follows Grok Build user guide (~/.grok/docs + custom-models) and production-ready Sub2API setups.
  // Text models only (Responses). Image/video: Imagine model IDs on media endpoints / feature overrides.
  // Credential order: api_key field → env_key → signed-in session → XAI_API_KEY global fallback.
  const modelsListUrl = `${baseUrl.replace(/\/+$/, '')}/models`
  const configContent = `# Grok Build CLI → Sub2API Grok group (API key auth).
# Docs: ~/.grok/docs/user-guide/05-configuration.md + 11-custom-models.md
# Verify after save: grok inspect
#
# IMPORTANT: api_backend must be "responses" for Sub2API Grok (POST /v1/responses).
# If omitted, Grok Build defaults to chat_completions (/v1/chat/completions).
# Keep api_backend = "responses" on every model entry.
#
# Prefer env_key over hardcoding api_key (never commit secrets).
# Also export GROK_MODELS_BASE_URL + XAI_API_KEY in the shell block above.

# Global inference / catalog endpoints (same role as env GROK_MODELS_BASE_URL).
# When models_base_url is set, Grok uses API-key Bearer auth (no grok login required).
[endpoints]
models_base_url = "${baseUrl}"              # inference base; model list defaults to {base}/models
models_list_url = "${modelsListUrl}"        # optional override (env: GROK_MODELS_LIST_URL)
xai_api_base_url = "${baseUrl}"             # public xAI API base override for gateway routing
cli_chat_proxy_base_url = "${baseUrl}"      # CLI chat-proxy base (env: GROK_CLI_CHAT_PROXY_BASE_URL)

# Prefer API key when using a custom gateway (matches Sub2API).
# Requires XAI_API_KEY env or per-model env_key / api_key.
[auth]
preferred_method = "api_key"

[model."grok-4.5"]
model = "grok-4.5"                          # id sent to the API
name = "Grok 4.5"                           # shown in /model picker
description = "Grok 4.5 via Sub2API (Responses)"
# base_url inherits from [endpoints].models_base_url; override only if needed:
# base_url = "${baseUrl}"
env_key = "XAI_API_KEY"                     # or: api_key = "${apiKey}"  (not recommended)
api_backend = "responses"                   # chat_completions | responses | messages
context_window = 500000                     # drives auto-compaction timing
# Optional sampling (global defaults can live under [models] instead):
# temperature = 0.7
# top_p = 0.95
# max_completion_tokens = 8192
# Server-side (backend) web_search tools — only if your gateway exposes them:
supports_backend_search = true

[model."grok-build-0.1"]
model = "grok-build-0.1"
name = "Grok Build"
description = "Coding / agent sessions (xAI recommends grok-build* for coding)"
env_key = "XAI_API_KEY"
api_backend = "responses"
context_window = 256000
supports_backend_search = true

# Text multi-agent / client web_search sub-agent (NOT Imagine image/video).
[model."grok-4.20-multi-agent-0309"]
model = "grok-4.20-multi-agent-0309"
name = "Grok 4.20 Multi Agent (text / web_search)"
description = "Text multi-agent; use for web_search sub-agent, not image/video"
env_key = "XAI_API_KEY"
api_backend = "responses"
context_window = 1000000
supports_backend_search = true

[model."grok-4.3"]
model = "grok-4.3"
name = "Grok 4.3"
env_key = "XAI_API_KEY"
api_backend = "responses"
context_window = 1000000
supports_backend_search = true

# Optional short alias for /model grok:
# [model."grok"]
# model = "grok-4.5"
# name = "Grok"
# env_key = "XAI_API_KEY"
# api_backend = "responses"
# context_window = 1000000
# supports_backend_search = true

[models]
# xAI recommends grok-build* for coding/agent sessions; use grok-4.5 for general chat.
default = "grok-4.5"
web_search = "grok-4.5"                     # client-side web_search tool model (must exist as [model.*])
image_description = "grok-4.5"              # vision/describe-image helper model
# Optional environment-wide sampling defaults (per-model values win):
# temperature = 0.7
# top_p = 0.95
# max_completion_tokens = 8192
# max_retries = 8

[session]
auto_compact_threshold_percent = 80         # auto-compact at this % of context_window (default 85)

# Imagine tools: model IDs go to Sub2API media endpoints (not the text [model.*] catalog).
# Enable only if the Grok group allows image/video generation.
[features]
image_gen = true
video_gen = true
image_gen_model_override = "grok-imagine-image-quality"   # or grok-imagine-image
image_edit_model_override = "grok-imagine-edit"
# Optional feature flags (defaults shown in docs):
# telemetry = false
# remote_fetch = true                         # set false for air-gapped / pure-gateway catalogs
# lsp_tools = false`

  return [
    { path: envPath, content: envContent },
    {
      path: joinConfigPath(configDir, 'config.toml', isWindowsPath),
      content: configContent,
      hint: t('keys.useKeyModal.grok.configTomlHint')
    }
  ]
}

function generateGrokCodexFiles(baseUrl: string, apiKey: string): FileConfig[] {
  // Codex config reference: wire_api = "responses" only; prefer env_key over experimental_bearer_token.
  // Non-OpenAI gateways should set supports_websockets = false (HTTP/SSE).
  const shell = activeTab.value
  const isWindowsPath = shell === 'windows' || shell === 'cmd' || shell === 'powershell'
  const configDir = isWindowsPath ? '%userprofile%\\.codex' : '~/.codex'

  let envPath: string
  let envContent: string
  switch (shell) {
    case 'cmd':
      envPath = 'Command Prompt'
      envContent = `set SUB2API_API_KEY=${apiKey}`
      break
    case 'powershell':
    case 'windows':
      envPath = 'PowerShell'
      envContent = `$env:SUB2API_API_KEY="${apiKey}"`
      break
    default:
      envPath = 'Terminal'
      envContent = `export SUB2API_API_KEY="${apiKey}"`
  }

  const configContent = `# Codex CLI → Sub2API Grok group
# Docs: Codex config reference (model_providers.*, wire_api = "responses")
#
# Text models only. Image/video: grok-imagine-image / grok-imagine-video on media endpoints.
# Switch model: grok-4.5 | grok-4.3 | grok-build-0.1 | grok-4.20-multi-agent-0309 (text / web_search)

model_provider = "sub2api"
model = "grok-4.5"
# Optional:
# review_model = "grok-4.5"
# model_reasoning_effort = "medium"
# model_context_window = 500000
# disable_response_storage = true
# network_access = "enabled"
# windows_wsl_setup_acknowledged = true

[model_providers.sub2api]
name = "Sub2API Grok"
base_url = "${baseUrl}"
# Prefer env_key (variable NAME). Do not combine with experimental_bearer_token.
env_key = "SUB2API_API_KEY"
# Fallback only if you cannot set env (discouraged — keeps secret on disk):
# experimental_bearer_token = "${apiKey}"
wire_api = "responses"
# API-key providers: do not require ChatGPT OAuth login
requires_openai_auth = false
# Grok/Sub2API path is HTTP/SSE; disable WS (Codex may otherwise try WebSocket first)
supports_websockets = false

# Optional:
# [features]
# goals = true`

  return [
    { path: envPath, content: envContent },
    {
      path: joinConfigPath(configDir, 'config.toml', isWindowsPath),
      content: configContent,
      hint: t('keys.useKeyModal.grok.codexConfigTomlHint')
    }
  ]
}

const codexManualFiles = computed((): FileConfig[] => {
  const files = generateOpenAIFiles(props.baseUrl || window.location.origin, props.apiKey, false)
  return files.map((file) => ({
    ...file,
    path: file.path.endsWith('config.toml') ? 'config.toml' : 'auth.json'
  }))
})

const configTomlFile = computed(() => codexManualFiles.value.find((file) => file.path === 'config.toml'))
const authJsonFile = computed(() => codexManualFiles.value.find((file) => file.path === 'auth.json'))

const oneClickCommand = computed(() => {
  const configFile = configTomlFile.value
  const authFile = authJsonFile.value
  if (!configFile || !authFile) return ''
  return detectedSystem.value === 'windows'
    ? generateWindowsCodexCommand(configFile.content, authFile.content)
    : generateMacCodexCommand(configFile.content, authFile.content)
})

function generateMacCodexCommand(configContent: string, authContent: string): string {
  return `CODEX_DIR="$HOME/.codex"
CONFIG_FILE="$CODEX_DIR/config.toml"
AUTH_FILE="$CODEX_DIR/auth.json"
STAMP="$(date +%Y%m%d-%H%M%S)"

big_error() {
  echo ""
  echo "============================================================"
  echo "❌❌❌  配置失败  ❌❌❌"
  echo "============================================================"
  printf "%b\n" "$1"
  echo "============================================================"
  echo ""
}

if [ ! -d "$CODEX_DIR" ]; then
  big_error "未找到 Codex 配置目录：$CODEX_DIR\n请先安装 Codex App，并打开一次完成初始化。\n官方下载：${CODEX_OFFICIAL_DOWNLOAD_URL}\n备用网盘：${CODEX_BACKUP_DOWNLOAD_URL}\n安装完成后，请回到网页从第一步重新执行配置。"
  exit 1
fi

if pgrep -if '(^|/)Codex( |$)' >/dev/null 2>&1; then
  big_error "检测到 Codex App 仍在运行。\n请先右键 Dock / 菜单栏里的 Codex，选择退出，确保它已完全退出后再重新执行本命令。"
  exit 1
fi

[ -f "$CONFIG_FILE" ] && cp "$CONFIG_FILE" "$CONFIG_FILE.sub2api.bak-$STAMP"
[ -f "$AUTH_FILE" ] && cp "$AUTH_FILE" "$AUTH_FILE.sub2api.bak-$STAMP"

cat > "$CONFIG_FILE" <<'SUB2API_CONFIG_TOML'
${configContent}
SUB2API_CONFIG_TOML

cat > "$AUTH_FILE" <<'SUB2API_AUTH_JSON'
${authContent}
SUB2API_AUTH_JSON

if ! grep -Fq 'model = "gpt-5.6-sol"' "$CONFIG_FILE" || ! grep -Fq 'wire_api = "responses"' "$CONFIG_FILE" || ! grep -Fq 'base_url = "${props.baseUrl || window.location.origin}"' "$CONFIG_FILE"; then
  big_error "config.toml 写入后校验失败。旧配置已备份在 $CODEX_DIR，请联系网页右上角客服咨询。"
  exit 1
fi

if ! grep -Fq '"OPENAI_API_KEY": "${props.apiKey}"' "$AUTH_FILE"; then
  big_error "auth.json 写入后校验失败。旧配置已备份在 $CODEX_DIR，请联系网页右上角客服咨询。"
  exit 1
fi

open "$CODEX_DIR"
echo ""
echo "============================================================"
echo "✅✅✅  配置成功！Codex CLI 已完成接入  ✅✅✅"
echo "============================================================"
echo "已写入："
echo "  $CONFIG_FILE"
echo "  $AUTH_FILE"
echo ""
echo "旧配置已自动备份在同一目录中。"
echo "现在可以打开 Codex 开始使用。"
echo "如有疑问，请点击网页右上角联系客服咨询。"
echo "============================================================"`
}

function generateWindowsCodexCommand(configContent: string, authContent: string): string {
  return `$codexDir = Join-Path $env:USERPROFILE '.codex'
$configFile = Join-Path $codexDir 'config.toml'
$authFile = Join-Path $codexDir 'auth.json'
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'

function Show-Sub2ApiError([string]$message) {
  Write-Host ''
  Write-Host '============================================================' -ForegroundColor Red
  Write-Host '❌❌❌  配置失败  ❌❌❌' -ForegroundColor Red
  Write-Host '============================================================' -ForegroundColor Red
  Write-Host $message -ForegroundColor Yellow
  Write-Host '============================================================' -ForegroundColor Red
  Write-Host ''
}

if (!(Test-Path $codexDir)) {
  Show-Sub2ApiError "未找到 Codex 配置目录：$codexDir\`n请先安装 Codex App，并打开一次完成初始化。\`n官方下载：${CODEX_OFFICIAL_DOWNLOAD_URL}\`n备用网盘：${CODEX_BACKUP_DOWNLOAD_URL}\`n安装完成后，请回到网页从第一步重新执行配置。"
  exit 1
}

$codexProcess = Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.ProcessName -match '^Codex$|^codex$' }
if ($codexProcess) {
  Show-Sub2ApiError "检测到 Codex App 仍在运行。\`n请先右键任务栏里的 Codex，选择退出，确保它已完全退出后再重新执行本命令。"
  exit 1
}

if (Test-Path $configFile) { Copy-Item $configFile "$configFile.sub2api.bak-$stamp" -Force }
if (Test-Path $authFile) { Copy-Item $authFile "$authFile.sub2api.bak-$stamp" -Force }

$configContent = @'
${configContent}
'@

$authContent = @'
${authContent}
'@

Set-Content -Path $configFile -Value $configContent -Encoding UTF8
Set-Content -Path $authFile -Value $authContent -Encoding UTF8

$writtenConfig = Get-Content -Path $configFile -Raw
$writtenAuth = Get-Content -Path $authFile -Raw

if (!$writtenConfig.Contains('model = "gpt-5.6-sol"') -or !$writtenConfig.Contains('wire_api = "responses"') -or !$writtenConfig.Contains('base_url = "${props.baseUrl || window.location.origin}"')) {
  Show-Sub2ApiError "config.toml 写入后校验失败。旧配置已备份在 $codexDir，请联系网页右上角客服咨询。"
  exit 1
}

if (!$writtenAuth.Contains('"OPENAI_API_KEY": "${props.apiKey}"')) {
  Show-Sub2ApiError "auth.json 写入后校验失败。旧配置已备份在 $codexDir，请联系网页右上角客服咨询。"
  exit 1
}

Invoke-Item $codexDir
Write-Host ''
Write-Host '============================================================' -ForegroundColor Green
Write-Host '✅✅✅  配置成功！Codex CLI 已完成接入  ✅✅✅' -ForegroundColor Green
Write-Host '============================================================' -ForegroundColor Green
Write-Host '已写入：'
Write-Host "  $configFile"
Write-Host "  $authFile"
Write-Host ''
Write-Host '旧配置已自动备份在同一目录中。'
Write-Host '现在可以打开 Codex 开始使用。'
Write-Host '如有疑问，请点击网页右上角联系客服咨询。'
Write-Host '============================================================' -ForegroundColor Green`
}

function generateOpenCodeConfig(platform: string, baseUrl: string, apiKey: string, pathLabel?: string): FileConfig {
  const provider: Record<string, any> = {
    [platform]: {
      options: {
        baseURL: baseUrl,
        apiKey
      }
    }
  }
  const openaiModels = {
    'gpt-5.2': {
      name: 'GPT-5.2',
      limit: {
        context: 400000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.6': {
      name: 'GPT-5.6 (Sol)',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-sol': {
      name: 'GPT-5.6 Sol',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-terra': {
      name: 'GPT-5.6 Terra',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-luna': {
      name: 'GPT-5.6 Luna',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.5': {
      name: 'GPT-5.5',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.4': {
      name: 'GPT-5.4',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.4-mini': {
      name: 'GPT-5.4 Mini',
      limit: {
        context: 400000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.3-codex-spark': {
      name: 'GPT-5.3 Codex Spark',
      limit: {
        context: 128000,
        output: 32000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'codex-mini-latest': {
      name: 'Codex Mini',
      limit: {
        context: 200000,
        output: 100000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {}
      }
    }
  }
  const geminiModels = {
    'gemini-2.0-flash': {
      name: 'Gemini 2.0 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-2.5-flash': {
      name: 'Gemini 2.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-2.5-pro': {
      name: 'Gemini 2.5 Pro',
      limit: {
        context: 2097152,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.5-flash': {
      name: 'Gemini 3.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-3-flash-preview': {
      name: 'Gemini 3 Flash Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-3-pro-preview': {
      name: 'Gemini 3 Pro Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-preview': {
      name: 'Gemini 3.1 Pro Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }

  const antigravityGeminiModels = {
    'gemini-2.5-flash': {
      name: 'Gemini 2.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'disable'
        }
      }
    },
    'gemini-2.5-flash-lite': {
      name: 'Gemini 2.5 Flash Lite',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-2.5-flash-thinking': {
      name: 'Gemini 2.5 Flash (Thinking)',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3-flash': {
      name: 'Gemini 3 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-low': {
      name: 'Gemini 3.1 Pro Low',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-high': {
      name: 'Gemini 3.1 Pro High',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-2.5-flash-image': {
      name: 'Gemini 2.5 Flash Image',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image'],
        output: ['image']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-flash-image': {
      name: 'Gemini 3.1 Flash Image',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image'],
        output: ['image']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }
  const claudeModels = {
    'claude-fable-5': {
      name: 'Claude Fable 5',
      limit: {
        context: 1048576,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          type: 'adaptive'
        }
      }
    },
    'claude-opus-4-6-thinking': {
      name: 'Claude 4.6 Opus (Thinking)',
      limit: {
        context: 200000,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'claude-sonnet-4-6': {
      name: 'Claude 4.6 Sonnet',
      limit: {
        context: 200000,
        output: 64000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }
  // Align context_window with Grok Build official sample (docs.x.ai/build/settings) where known.
  // Image/video: grok-imagine-image / grok-imagine-video on media endpoints — not this list.
  const grokModels = {
    'grok-4.5': {
      name: 'Grok 4.5',
      limit: { context: 500000, output: 64000 }
    },
    'grok-build-0.1': {
      name: 'Grok Build 0.1',
      limit: { context: 256000, output: 64000 }
    },
    'grok-4.20-multi-agent-0309': {
      name: 'Grok 4.20 Multi Agent (text / web_search)',
      limit: { context: 1000000, output: 64000 }
    },
    'grok-4.3': {
      name: 'Grok 4.3',
      limit: { context: 1000000, output: 64000 }
    },
    'grok-composer-2.5-fast': {
      name: 'Grok Composer 2.5 Fast',
      limit: { context: 500000, output: 64000 }
    }
  }

  if (platform === 'gemini') {
    provider[platform].npm = '@ai-sdk/google'
    provider[platform].models = geminiModels
  } else if (platform === 'anthropic') {
    provider[platform].npm = '@ai-sdk/anthropic'
  } else if (platform === 'antigravity-claude') {
    provider[platform].npm = '@ai-sdk/anthropic'
    provider[platform].name = 'Antigravity (Claude)'
    provider[platform].models = claudeModels
  } else if (platform === 'antigravity-gemini') {
    provider[platform].npm = '@ai-sdk/google'
    provider[platform].name = 'Antigravity (Gemini)'
    provider[platform].models = antigravityGeminiModels
  } else if (platform === 'openai') {
    provider[platform].models = openaiModels
  } else if (platform === 'grok') {
    // Custom provider pointing at Sub2API OpenAI-compatible Responses/Chat endpoints.
    provider[platform].npm = '@ai-sdk/openai-compatible'
    provider[platform].name = 'Grok via Sub2API'
    provider[platform].models = grokModels
  }

  const agent =
    platform === 'openai'
      ? {
          build: {
            options: {
              store: false
            }
          },
          plan: {
            options: {
              store: false
            }
          }
        }
      : undefined

  const content = JSON.stringify(
    {
      provider,
      ...(agent ? { agent } : {}),
      $schema: 'https://opencode.ai/config.json'
    },
    null,
    2
  )

  return {
    path: pathLabel ?? 'opencode.json',
    content,
    hint: t('keys.useKeyModal.opencode.hint')
  }
}

const copyContent = async (content: string, index: number) => {
  const success = await clipboardCopy(content, t('keys.copied'))
  if (success) {
    copiedIndex.value = index
    setTimeout(() => {
      copiedIndex.value = null
    }, 2000)
  }
}

const copySnippet = async (content: string) => {
  await clipboardCopy(content, t('keys.copied'))
}

const downloadConfigFile = (file: FileConfig) => {
  const filename = file.path.endsWith('config.toml') ? 'config.toml'
    : file.path.endsWith('auth.json') ? 'auth.json'
      : file.path.split(/[\\/]/).pop() || 'config.txt'
  const blob = new Blob([file.content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
</script>
