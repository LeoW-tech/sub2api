import { computed, defineComponent, h, reactive, ref } from "vue";
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
    removeMany: vi.fn(),
    toggleVisible: vi.fn(),
    selectVisible: vi.fn(),
    batchUpdate: vi.fn(),
  }),
}));

import AccountsView from "../AccountsView.vue";

const DataTableStub = defineComponent({
  name: "DataTable",
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () =>
      h("div", { class: "data-table-stub" }, [
        props.data[0]
          ? slots["cell-platform_type"]?.({
              row: props.data[0],
              value: props.data[0]?.platform,
            })
          : null,
      ]);
  },
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
        AccountTableFilters: true,
        DataTable: DataTableStub,
        Pagination: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
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
        PlatformTypeBadge: {
          template: '<span class="platform-badge-stub">platform</span>',
        },
        Icon: true,
      },
    },
  });
}

describe("AccountsView compact badge", () => {
  beforeEach(() => {
    accountsRef.value = [
      {
        id: 1,
        name: "OpenAI Account",
        platform: "openai",
        type: "apikey",
        credentials: {},
        extra: {
          openai_compact_mode: " FORCE_ON ",
          openai_compact_checked_at: "2026-04-25T09:00:00Z",
        },
      },
    ];
    selectedIdsRef.value = [];
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

    getBatchTodayStatsMock.mockResolvedValue({ stats: {} });
    getAllProxiesMock.mockResolvedValue([]);
    getIPOptionsMock.mockResolvedValue([]);
    getAllGroupsMock.mockResolvedValue([]);
  });

  it("normalizes compact mode before rendering the supported badge", async () => {
    const wrapper = mountView();
    await flushPromises();

    const badge = wrapper
      .findAll("span")
      .find((node) => node.text() === "admin.accounts.openai.compactSupported");

    expect(badge).toBeDefined();
    expect(badge?.attributes("title")).toContain(
      "admin.accounts.openai.compactLastChecked",
    );
  });
});
