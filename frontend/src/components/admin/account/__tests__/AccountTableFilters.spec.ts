import { defineComponent } from "vue";
import { mount } from "@vue/test-utils";
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

describe("AccountTableFilters", () => {
  const baseProps = {
    searchQuery: "",
    filters: {
      platform: "",
      type: "",
      status: "",
      capacity_status: "",
      privacy_mode: "",
      network_status: "",
      ip: "",
      group: "",
    },
    groups: [],
    ipOptions: [
      {
        ip: "203.0.113.10",
        proxy_names: ["hk-node", "jp-node"],
        proxy_count: 2,
      },
    ],
  };

  it("状态筛选项包含 disabled", () => {
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const statusOptions = selects[2]?.props("options") as Array<{
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
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const statusSelect = selects[2];

    await statusSelect.vm.$emit("update:modelValue", "disabled");
    await statusSelect.vm.$emit("change", "disabled");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "disabled",
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

  it("网络状态筛选项包含 online/offline", () => {
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const networkOptions = selects[5]?.props("options") as Array<{
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
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const ipOptions = selects[6]?.props("options") as Array<{
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
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const ipSelect = selects[6];

    await ipSelect.vm.$emit("update:modelValue", "203.0.113.10");
    await ipSelect.vm.$emit("change", "203.0.113.10");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "",
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
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const capacityOptions = selects[3]?.props("options") as Array<{
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
    const wrapper = mount(AccountTableFilters, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub,
        },
      },
    });

    const selects = wrapper.findAllComponents(SelectStub);
    const capacitySelect = selects[3];

    await capacitySelect.vm.$emit("update:modelValue", "concurrent");
    await capacitySelect.vm.$emit("change", "concurrent");

    expect(wrapper.emitted("update:filters")).toEqual([
      [
        {
          platform: "",
          type: "",
          status: "",
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
});
