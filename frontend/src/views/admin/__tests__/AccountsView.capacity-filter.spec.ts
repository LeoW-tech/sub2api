import { computed, defineComponent, reactive, ref } from "vue";
import { flushPromises, shallowMount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

const {
  loadMock,
  reloadMock,
  debouncedReloadMock,
  handlePageChangeMock,
  handlePageSizeChangeMock,
  getBatchTodayStatsMock,
  getAllProxiesMock,
  getIPOptionsMock,
  getAllGroupsMock,
  removeManyMock,
} = vi.hoisted(() => ({
  loadMock: vi.fn(),
  reloadMock: vi.fn(),
  debouncedReloadMock: vi.fn(),
  handlePageChangeMock: vi.fn(),
  handlePageSizeChangeMock: vi.fn(),
  getBatchTodayStatsMock: vi.fn(),
  getAllProxiesMock: vi.fn(),
  getIPOptionsMock: vi.fn(),
  getAllGroupsMock: vi.fn(),
  removeManyMock: vi.fn(),
}));

const accountsRef = ref<any[]>([]);
const selectedIdsRef = ref<number[]>([]);
const loadingRef = ref(false);
const tableParams = reactive({
  platform: "",
  type: "",
  status: "",
  capacity_status: "",
  privacy_mode: "",
  network_status: "",
  ip: "",
  group: "",
  search: "",
  sort_by: "name",
  sort_order: "asc",
});
const paginationState = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 1,
});

vi.mock("@vueuse/core", () => ({
  useIntervalFn: () => ({
    pause: vi.fn(),
    resume: vi.fn(),
  }),
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
  }),
}));

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    isSimpleMode: false,
  }),
}));

vi.mock("@/api/admin", () => ({
  adminAPI: {
    accounts: {
      getBatchTodayStats: getBatchTodayStatsMock,
    },
    proxies: {
      getAll: getAllProxiesMock,
      getIPOptions: getIPOptionsMock,
    },
    groups: {
      getAll: getAllGroupsMock,
    },
  },
}));

vi.mock("@/composables/useSwipeSelect", () => ({
  useSwipeSelect: vi.fn(),
}));

vi.mock("@/composables/useTableLoader", () => ({
  useTableLoader: () => ({
    items: accountsRef,
    loading: loadingRef,
    params: tableParams,
    pagination: paginationState,
    load: loadMock,
    reload: reloadMock,
    debouncedReload: debouncedReloadMock,
    handlePageChange: handlePageChangeMock,
    handlePageSizeChange: handlePageSizeChangeMock,
  }),
}));

vi.mock("@/composables/useTableSelection", () => ({
  useTableSelection: () => ({
    selectedIds: computed(() => selectedIdsRef.value),
    allVisibleSelected: computed(() => false),
    isSelected: vi.fn(() => false),
    setSelectedIds: vi.fn(),
    select: vi.fn(),
    deselect: vi.fn(),
    toggle: vi.fn(),
    clear: vi.fn(),
    removeMany: removeManyMock,
    toggleVisible: vi.fn(),
    selectVisible: vi.fn(),
    batchUpdate: vi.fn(),
  }),
}));

import AccountsView from "../AccountsView.vue";

const AccountTableFiltersStub = defineComponent({
  name: "AccountTableFilters",
  props: {
    filters: {
      type: Object,
      required: true,
    },
  },
  emits: ["update:filters", "change"],
  template: `
    <button
      class="apply-concurrent-filter"
      @click="$emit('update:filters', { ...filters, capacity_status: 'concurrent' }); $emit('change')"
    >
      apply
    </button>
  `,
});

const AccountTestModalStub = defineComponent({
  name: "AccountTestModal",
  emits: ["tested", "close"],
  template: '<div class="account-test-modal-stub"></div>',
});

function mountView() {
  return shallowMount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: "<div><slot /></div>" },
        TablePageLayout: {
          template:
            '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
        },
        AccountBulkActionsBar: true,
        AccountTableActions: true,
        AccountTableFilters: AccountTableFiltersStub,
        DataTable: true,
        Pagination: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: AccountTestModalStub,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        AccountActionMenu: true,
        SyncFromCrsModal: true,
        ImportDataModal: true,
        BulkEditAccountModal: true,
        TempUnschedStatusModal: true,
        ConfirmDialog: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        AccountStatusIndicator: true,
        AccountUsageCell: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountCapacityCell: true,
        PlatformTypeBadge: true,
        Icon: true,
      },
    },
  });
}

describe("AccountsView capacity filter", () => {
  beforeEach(() => {
    accountsRef.value = [
      {
        id: 1,
        name: "Concurrent Account",
        current_concurrency: 2,
        current_window_cost: null,
        active_sessions: null,
      },
      {
        id: 2,
        name: "Idle Account",
        current_concurrency: 1,
        current_window_cost: null,
        active_sessions: null,
      },
    ];
    selectedIdsRef.value = [1, 2];
    tableParams.platform = "";
    tableParams.type = "";
    tableParams.status = "";
    tableParams.capacity_status = "";
    tableParams.privacy_mode = "";
    tableParams.network_status = "";
    tableParams.ip = "";
    tableParams.group = "";
    tableParams.search = "";
    tableParams.sort_by = "name";
    tableParams.sort_order = "asc";
    paginationState.page = 1;
    paginationState.page_size = 20;
    paginationState.total = accountsRef.value.length;
    paginationState.pages = 1;

    loadMock.mockReset();
    reloadMock.mockReset();
    debouncedReloadMock.mockReset();
    handlePageChangeMock.mockReset();
    handlePageSizeChangeMock.mockReset();
    getBatchTodayStatsMock.mockReset();
    getAllProxiesMock.mockReset();
    getIPOptionsMock.mockReset();
    getAllGroupsMock.mockReset();
    removeManyMock.mockReset();

    getBatchTodayStatsMock.mockResolvedValue({ stats: {} });
    getAllProxiesMock.mockResolvedValue([]);
    getIPOptionsMock.mockResolvedValue([]);
    getAllGroupsMock.mockResolvedValue([]);
  });

  it("启用正在并发筛选后，本地更新会移除并发数归零的账号", async () => {
    const wrapper = mountView();
    await flushPromises();

    await wrapper.get("button.apply-concurrent-filter").trigger("click");
    expect(tableParams.capacity_status).toBe("concurrent");

    const accountTestModal = wrapper.findComponent(AccountTestModalStub);
    await accountTestModal.vm.$emit("tested", {
      id: 1,
      name: "Concurrent Account",
      current_concurrency: 0,
      current_window_cost: null,
      active_sessions: null,
    });
    await flushPromises();

    expect(accountsRef.value).toEqual([
      {
        id: 2,
        name: "Idle Account",
        current_concurrency: 1,
        current_window_cost: null,
        active_sessions: null,
      },
    ]);
    expect(paginationState.total).toBe(1);
    expect(removeManyMock).toHaveBeenCalledWith([1]);
  });
});
