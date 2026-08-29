<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-wrap items-center gap-3">
      <SearchInput
        :model-value="searchQuery"
        :placeholder="t('admin.accounts.searchAccounts')"
        class="w-full sm:w-64"
        @update:model-value="$emit('update:searchQuery', $event)"
        @search="$emit('change')"
      />
      <Select
        data-filter="status"
        :model-value="filters.status"
        class="w-40"
        :options="statusOptions"
        @update:model-value="updateStatus"
        @change="$emit('change')"
      />
      <Select
        data-filter="capacity_status"
        :model-value="filters.capacity_status"
        class="w-40"
        :options="capacityOptions"
        @update:model-value="updateCapacityStatus"
        @change="$emit('change')"
      />
      <Select
        data-filter="ip"
        :model-value="filters.ip"
        class="w-44"
        :options="ipFilterOptions"
        searchable
        :search-placeholder="t('admin.accounts.searchIPs')"
        @update:model-value="updateIP"
        @change="$emit('change')"
      />
      <Select
        data-filter="group"
        :model-value="filters.group"
        class="w-40"
        :options="groupOptions"
        @update:model-value="updateGroup"
        @change="$emit('change')"
      />
      <button
        type="button"
        data-testid="more-filters"
        class="btn btn-secondary h-11 whitespace-nowrap px-3"
        :aria-expanded="showAdvancedFilters"
        aria-controls="advanced-account-filters"
        @click="showAdvancedFilters = !showAdvancedFilters"
      >
        <Icon name="filter" size="sm" />
        <span>{{ t("admin.accounts.moreFilters") }}</span>
        <span
          v-if="activeAdvancedFilterCount > 0"
          data-testid="advanced-filter-count"
          class="rounded-md bg-primary-500/15 px-1.5 py-0.5 text-xs font-semibold text-primary-500"
        >
          {{ activeAdvancedFilterCount }}
        </span>
        <Icon
          name="chevronDown"
          size="sm"
          :class="['transition-transform duration-200', showAdvancedFilters && 'rotate-180']"
        />
      </button>
    </div>

    <div
      v-if="showAdvancedFilters"
      id="advanced-account-filters"
      data-testid="advanced-filters"
      class="flex flex-wrap items-center gap-3"
    >
      <Select
        data-filter="platform"
        :model-value="filters.platform"
        class="w-40"
        :options="platformOptions"
        @update:model-value="updatePlatform"
        @change="$emit('change')"
      />
      <Select
        data-filter="type"
        :model-value="filters.type"
        class="w-40"
        :options="typeOptions"
        @update:model-value="updateType"
        @change="$emit('change')"
      />
      <Select
        data-filter="privacy_mode"
        :model-value="filters.privacy_mode"
        class="w-40"
        :options="privacyOptions"
        @update:model-value="updatePrivacyMode"
        @change="$emit('change')"
      />
      <Select
        data-filter="rt_status"
        :model-value="filters.rt_status"
        class="w-40"
        :options="rtStatusOptions"
        @update:model-value="updateRTStatus"
        @change="$emit('change')"
      />
      <Select
        data-filter="network_status"
        :model-value="filters.network_status"
        class="w-40"
        :options="networkOptions"
        @update:model-value="updateNetworkStatus"
        @change="$emit('change')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminGroup, ProxyIPOption } from '@/types'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'

const props = defineProps<{
  searchQuery: string;
  filters: Record<string, any>;
  groups?: AdminGroup[];
  ipOptions?: ProxyIPOption[];
}>();

const emit = defineEmits(["update:searchQuery", "update:filters", "change"]);

const { t } = useI18n();
const advancedFilterKeys = [
  "platform",
  "type",
  "privacy_mode",
  "rt_status",
  "network_status",
] as const;

const isFilterApplied = (value: unknown) => value !== "" && value !== null && value !== undefined;

const activeAdvancedFilterCount = computed(() =>
  advancedFilterKeys.filter((key) => isFilterApplied(props.filters[key])).length,
);

const showAdvancedFilters = ref(activeAdvancedFilterCount.value > 0);

watch(activeAdvancedFilterCount, (count) => {
  if (count > 0) {
    showAdvancedFilters.value = true;
  }
});

const updatePlatform = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, platform: value });
};
const updateType = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, type: value });
};
const updateStatus = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, status: value });
};
const updateRTStatus = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, rt_status: value });
};
const updateCapacityStatus = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, capacity_status: value });
};
const updatePrivacyMode = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, privacy_mode: value });
};
const updateNetworkStatus = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, network_status: value });
};
const updateIP = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, ip: value });
};
const updateGroup = (value: string | number | boolean | null) => {
  emit("update:filters", { ...props.filters, group: value });
};

const platformOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allPlatforms") },
  ...CONCRETE_PLATFORM_OPTIONS,
]);

const typeOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allTypes") },
  { value: "oauth", label: t("admin.accounts.oauthType") },
  { value: "setup-token", label: t("admin.accounts.setupToken") },
  { value: "apikey", label: t("admin.accounts.apiKey") },
  { value: "bedrock", label: "AWS Bedrock" },
]);

const rtStatusOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allTokenStatus") },
  { value: "has_rt", label: t("admin.accounts.hasRT") },
  { value: "no_rt", label: t("admin.accounts.noRT") },
]);

const statusOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allStatus") },
  { value: "active", label: t("admin.accounts.status.active") },
  { value: "inactive", label: t("admin.accounts.status.inactive") },
  { value: "disabled", label: t("admin.accounts.status.disabled") },
  { value: "error", label: t("admin.accounts.status.error") },
  { value: "rate_limited", label: t("admin.accounts.status.rateLimited") },
  {
    value: "temp_unschedulable",
    label: t("admin.accounts.status.tempUnschedulable"),
  },
  { value: "unschedulable", label: t("admin.accounts.status.unschedulable") },
]);

const capacityOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allCapacity") },
  {
    value: "concurrent",
    label: t("admin.accounts.capacityConcurrent"),
  },
]);

const privacyOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allPrivacyModes") },
  { value: "__unset__", label: t("admin.accounts.privacyUnset") },
  { value: "training_off", label: "Privacy" },
  { value: "training_set_cf_blocked", label: "CF" },
  { value: "training_set_failed", label: "Fail" },
]);

const networkOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allNetworkStatus") },
  { value: "online", label: t("admin.accounts.networkStatus.online") },
  { value: "offline", label: t("admin.accounts.networkStatus.offline") },
]);

const ipFilterOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allIPs") },
  ...(props.ipOptions || []).map((option) => ({
    value: option.ip,
    label: option.ip,
    description: option.proxy_names.join(" "),
  })),
]);

const groupOptions = computed<SelectOption[]>(() => [
  { value: "", label: t("admin.accounts.allGroups") },
  { value: "ungrouped", label: t("admin.accounts.ungroupedGroup") },
  ...(props.groups || []).map((group) => ({
    value: String(group.id),
    label: group.name,
  })),
]);

</script>
