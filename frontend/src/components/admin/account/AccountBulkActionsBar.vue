<template>
  <div
    class="mb-4 flex items-center justify-between rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20"
  >
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkEdit.title') }}
      </span>
      <template v-if="selectedIds.length > 0">
      <button
        @click="$emit('select-page')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
      </button>
      <span class="text-gray-300 dark:text-primary-800">•</span>
      <button
        @click="$emit('clear')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.clear') }}
      </button>
      </template>
    </div>
    <div class="flex gap-2">
      <template v-if="selectedIds.length > 0">
        <button @click="$emit('delete')" class="btn btn-danger btn-sm">{{ t('admin.accounts.bulkActions.delete') }}</button>
        <button @click="$emit('reset-status')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
        <button @click="$emit('refresh-token')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
        <button @click="$emit('true-refresh-token')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.trueRefreshToken') }}</button>
        <button
          @click="$emit('test-activate')"
          :disabled="testingActivate"
          class="btn btn-secondary btn-sm"
        >
          {{
            testingActivate
              ? t('admin.accounts.bulkActions.testActivating')
              : t('admin.accounts.bulkActions.testActivate')
          }}
        </button>
        <button @click="$emit('probe-upstream-billing')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}</button>
        <button @click="$emit('toggle-schedulable', true)" class="btn btn-success btn-sm">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
        <button @click="$emit('toggle-schedulable', false)" class="btn btn-warning btn-sm">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
        <button @click="$emit('edit-selected')" class="btn btn-primary btn-sm">{{ t('admin.accounts.bulkActions.edit') }}</button>
      </template>
      <button @click="$emit('refresh-subscriptions')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.refreshSubscriptions') }}</button>
      <button @click="$emit('edit-filtered')" class="btn btn-primary btn-sm">
        {{ t('admin.accounts.bulkEdit.submit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  selectedIds: number[]
  testingActivate?: boolean
}>()

defineEmits<{
  (e: 'delete'): void
  (e: 'edit-selected'): void
  (e: 'edit-filtered'): void
  (e: 'clear'): void
  (e: 'select-page'): void
  (e: 'toggle-schedulable', value: boolean): void
  (e: 'reset-status'): void
  (e: 'refresh-token'): void
  (e: 'true-refresh-token'): void
  (e: 'refresh-subscriptions'): void
  (e: 'test-activate'): void
  (e: 'probe-upstream-billing'): void
}>()

const { t } = useI18n()
</script>
