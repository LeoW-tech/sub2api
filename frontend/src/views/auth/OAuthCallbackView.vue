<template>
  <div class="min-h-screen bg-gray-50 px-4 py-10 dark:bg-dark-900">
    <div class="mx-auto max-w-2xl">
      <div class="card p-6">
        <h1 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('auth.oauth.callbackTitle') }}
        </h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {{ t('auth.oauth.callbackHint') }}
        </p>
        <div
          v-if="completingOpenAI"
          class="mt-4 rounded border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-900/30 dark:text-blue-200"
        >
          {{ t('auth.oauth.openaiCompleting') }}
        </div>

        <div class="mt-6 space-y-4">
          <div>
            <label class="input-label">{{ t('auth.oauth.code') }}</label>
            <div class="flex gap-2">
              <input class="input flex-1 font-mono text-sm" :value="code" readonly />
              <button class="btn btn-secondary" type="button" :disabled="!code" @click="copy(code)">
                {{ t('common.copy') }}
              </button>
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('auth.oauth.state') }}</label>
            <div class="flex gap-2">
              <input class="input flex-1 font-mono text-sm" :value="state" readonly />
              <button
                class="btn btn-secondary"
                type="button"
                :disabled="!state"
                @click="copy(state)"
              >
                {{ t('common.copy') }}
              </button>
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('auth.oauth.fullUrl') }}</label>
            <div class="flex gap-2">
              <input class="input flex-1 font-mono text-xs" :value="fullUrl" readonly />
              <button
                class="btn btn-secondary"
                type="button"
                :disabled="!fullUrl"
                @click="copy(fullUrl)"
              >
                {{ t('common.copy') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const appStore = useAppStore()
const authStore = useAuthStore()
const completingOpenAI = ref(false)

const code = computed(() => (route.query.code as string) || '')
const state = computed(() => (route.query.state as string) || '')
const isOpenAIAdminCallback = computed(() => route.query.admin_oauth_provider === 'openai')
const error = computed(
  () => (route.query.error as string) || (route.query.error_description as string) || ''
)

const fullUrl = computed(() => {
  if (typeof window === 'undefined') return ''
  return window.location.href
})

watch(
  error,
  (message) => {
    if (message) {
      appStore.showError(message)
    }
  },
  { immediate: true }
)

onMounted(async () => {
  if (!isOpenAIAdminCallback.value || !code.value || !state.value) {
    return
  }

  completingOpenAI.value = true
  try {
    await adminAPI.accounts.completeOpenAIPendingCreate({
      code: code.value,
      state: state.value
    })
    appStore.showSuccess(t('auth.oauth.openaiCompleteSuccess'))
    if (authStore.isAuthenticated) {
      await router.replace('/admin/accounts')
    }
  } catch (err) {
    appStore.showError(
      extractApiErrorMessage(err, t('auth.oauth.openaiCompleteFailed'))
    )
  } finally {
    completingOpenAI.value = false
  }
})

const copy = (value: string) => {
  if (!value) return
  copyToClipboard(value)
}
</script>
