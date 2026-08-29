import { defineComponent } from "vue";
import { mount, type VueWrapper } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";

import AccountTableFilters from "../AccountTableFilters.vue";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

const SelectStub = defineComponent({
  name: "SelectStub",
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: "",
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ["update:modelValue", "change"],
  template: '<div class="select-stub"></div>',
});

const SearchInputStub = defineComponent({
  name: "SearchInputStub",
  props: {
    modelValue: {
      type: String,
      default: "",
    },
  },
  emits: ["update:modelValue", "search"],
  template: '<div class="search-input-stub"></div>',
});

const IconStub = defineComponent({
  name: "IconStub",
  template: '<span class="icon-stub"></span>',
});

const findSelect = (wrapper: VueWrapper, filter: string) => {
  const select = wrapper
    .findAllComponents(SelectStub)
    .find((candidate) => candidate.attributes("data-filter") === filter);

  if (!select) {
    throw new Error(`未找到 ${filter} 筛选项`);
  }

  return select;
};

const expandAdvancedFilters = async (wrapper: VueWrapper) => {
  await wrapper.get('[data-testid="more-filters"]').trigger("click");
};

const baseFilters = {
  platform: "",
  type: "",
  status: "",
  rt_status: "",
  capacity_status: "",
  privacy_mode: "",
  network_status: "",
  ip: "",
  group: "",
};

const baseProps = {
  searchQuery: "",
  filters: baseFilters,
  groups: [],
  ipOptions: [
    {
      ip: "203.0.113.10",
      proxy_names: ["hk-node", "jp-node"],
      proxy_count: 2,
    },
  ],
};

const mountFilters = (filters = baseFilters) =>
  mount(AccountTableFilters, {
    props: {
      ...baseProps,
      filters: {
        ...baseProps.filters,
        ...filters,
      },
    },
    global: {
      stubs: {
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: IconStub,
      },
    },
  });

describe("AccountTableFilters", () => {
  it("状态筛选项包含 disabled", () => {
    const wrapper = mountFilters();

    const statusOptions = findSelect(wrapper, "status").props("options") as Array<{
      value: string;
      label: string;
    }>;

    expect(statusOptions).toEqual(
      expect.arrayContaining([
        { value: "disabled", label: "admin.accounts.status.disabled" },
      ]),
    );
  });

  it("选择 disabled 时会发出对应状态值", async () => {
    const wrapper = mountFilters();

    const statusSelect = findSelect(wrapper, "status");

    await statusSelect.vm.$emit("update:modelValue", "disabled");
    await statusSelect.vm.$emit("change", "disabled");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "disabled",
          rt_status: "",
          capacity_status: "",
          privacy_mode: "",
          network_status: "",
          ip: "",
          group: "",
        },
      ],
    ]);
    expect(wrapper.emitted("change")).toHaveLength(1);
  });


  it("RT 筛选项包含全部Token/有RT/无RT", async () => {
    const wrapper = mountFilters();

    await expandAdvancedFilters(wrapper);

    const rtOptions = findSelect(wrapper, "rt_status").props("options") as Array<{
      value: string;
      label: string;
    }>;

    expect(rtOptions).toEqual(
      expect.arrayContaining([
        { value: "", label: "admin.accounts.allTokenStatus" },
        { value: "has_rt", label: "admin.accounts.hasRT" },
        { value: "no_rt", label: "admin.accounts.noRT" },
      ]),
    );
  });

  it("选择无RT时会发出对应筛选值", async () => {
    const wrapper = mountFilters();

    await expandAdvancedFilters(wrapper);

    const rtSelect = findSelect(wrapper, "rt_status");

    await rtSelect.vm.$emit("update:modelValue", "no_rt");
    await rtSelect.vm.$emit("change", "no_rt");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "",
          rt_status: "no_rt",
          capacity_status: "",
          privacy_mode: "",
          network_status: "",
          ip: "",
          group: "",
        },
      ],
    ]);
    expect(wrapper.emitted("change")).toHaveLength(1);
  });

  it("网络状态筛选项包含 online/offline", async () => {
    const wrapper = mountFilters();

    await expandAdvancedFilters(wrapper);

    const networkOptions = findSelect(wrapper, "network_status").props("options") as Array<{
      value: string;
      label: string;
    }>;

    expect(networkOptions).toEqual(
      expect.arrayContaining([
        { value: "online", label: "admin.accounts.networkStatus.online" },
        { value: "offline", label: "admin.accounts.networkStatus.offline" },
      ]),
    );
  });

  it("IP 筛选项包含全部 IP 与代理名描述", () => {
    const wrapper = mountFilters();

    const ipOptions = findSelect(wrapper, "ip").props("options") as Array<{
      value: string;
      label: string;
      description?: string;
    }>;

    expect(ipOptions).toEqual(
      expect.arrayContaining([
        { value: "", label: "admin.accounts.allIPs" },
        {
          value: "203.0.113.10",
          label: "203.0.113.10",
          description: "hk-node jp-node",
        },
      ]),
    );
  });

  it("选择 IP 时会发出对应筛选值", async () => {
    const wrapper = mountFilters();

    const ipSelect = findSelect(wrapper, "ip");

    await ipSelect.vm.$emit("update:modelValue", "203.0.113.10");
    await ipSelect.vm.$emit("change", "203.0.113.10");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "",
          rt_status: "",
          capacity_status: "",
          privacy_mode: "",
          network_status: "",
          ip: "203.0.113.10",
          group: "",
        },
      ],
    ]);
    expect(wrapper.emitted("change")).toHaveLength(1);
  });

  it("容量筛选项包含全部容量与正在并发", () => {
    const wrapper = mountFilters();

    const capacityOptions = findSelect(wrapper, "capacity_status").props("options") as Array<{
      value: string;
      label: string;
    }>;

    expect(capacityOptions).toEqual(
      expect.arrayContaining([
        { value: "", label: "admin.accounts.allCapacity" },
        {
          value: "concurrent",
          label: "admin.accounts.capacityConcurrent",
        },
      ]),
    );
  });

  it("选择正在并发时会发出对应容量筛选值", async () => {
    const wrapper = mountFilters();

    const capacitySelect = findSelect(wrapper, "capacity_status");

    await capacitySelect.vm.$emit("update:modelValue", "concurrent");
    await capacitySelect.vm.$emit("change", "concurrent");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "",
          rt_status: "",
          capacity_status: "concurrent",
          privacy_mode: "",
          network_status: "",
          ip: "",
          group: "",
        },
      ],
    ]);
    expect(wrapper.emitted("change")).toHaveLength(1);
  });

  it("默认收起低频筛选，点击更多筛选后展开", async () => {
    const wrapper = mountFilters();

    expect(wrapper.find('[data-testid="more-filters"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="more-filters"]').attributes("aria-expanded")).toBe("false");
    expect(wrapper.find('[data-testid="advanced-filter-count"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="advanced-filters"]').exists()).toBe(false);
    expect(
      wrapper.findAllComponents(SelectStub).map((select) => select.attributes("data-filter")),
    ).toEqual(["status", "capacity_status", "ip", "group"]);

    await expandAdvancedFilters(wrapper);

    expect(wrapper.find('[data-testid="more-filters"]').attributes("aria-expanded")).toBe("true");
    expect(wrapper.find('[data-testid="advanced-filter-count"]').exists()).toBe(false);
    expect(
      wrapper
        .find('[data-testid="advanced-filters"]')
        .findAllComponents(SelectStub)
        .map((select) => select.attributes("data-filter")),
    ).toEqual(["platform", "type", "privacy_mode", "rt_status", "network_status"]);

    await wrapper.find('[data-testid="more-filters"]').trigger("click");

    expect(wrapper.find('[data-testid="more-filters"]').attributes("aria-expanded")).toBe("false");
    expect(wrapper.find('[data-testid="advanced-filters"]').exists()).toBe(false);
  });

  it("低频筛选已生效时默认展开并显示启用数量", () => {
    const wrapper = mountFilters({
      platform: "openai",
      network_status: "offline",
    });

    expect(wrapper.find('[data-testid="more-filters"]').attributes("aria-expanded")).toBe("true");
    expect(wrapper.find('[data-testid="advanced-filters"]').exists()).toBe(true);
    expect(wrapper.get('[data-testid="advanced-filter-count"]').text()).toBe("2");
  });
});
