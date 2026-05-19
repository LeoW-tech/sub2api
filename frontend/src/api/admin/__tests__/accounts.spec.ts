import { describe, expect, it, vi, beforeEach } from "vitest";

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  apiClient: {
    get: getMock,
    post: postMock,
  },
}));

import { completeOpenAIPendingCreate, exportData, list } from "../accounts";

describe("admin accounts api", () => {
  beforeEach(() => {
    getMock.mockReset();
    postMock.mockReset();
  });

  it("list 会携带 ip 与 capacity_status 筛选参数", async () => {
    getMock.mockResolvedValue({
      data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 },
    });

    await list(1, 20, {
      ip: "203.0.113.10",
      capacity_status: "concurrent",
      rt_status: "no_rt",
      search: "keyword",
    });

    expect(getMock).toHaveBeenCalledWith("/admin/accounts", {
      params: {
        page: 1,
        page_size: 20,
        ip: "203.0.113.10",
        capacity_status: "concurrent",
        rt_status: "no_rt",
        search: "keyword",
      },
      signal: undefined,
    });
  });

  it("exportData 会携带 ip 与 capacity_status 筛选参数", async () => {
    getMock.mockResolvedValue({
      data: {
        type: "sub2api-data",
        version: 1,
        exported_at: "",
        proxies: [],
        accounts: [],
      },
    });

    await exportData({
      filters: {
        ip: "203.0.113.10",
        capacity_status: "concurrent",
        rt_status: "has_rt",
        platform: "openai",
        sort_by: "priority",
        sort_order: "desc",
      },
    });

    expect(getMock).toHaveBeenCalledWith("/admin/accounts/data", {
      params: {
        ip: "203.0.113.10",
        capacity_status: "concurrent",
        rt_status: "has_rt",
        platform: "openai",
        sort_by: "priority",
        sort_order: "desc",
      },
    });
  });

  it("completeOpenAIPendingCreate 会调用 OpenAI pending create 完成接口", async () => {
    postMock.mockResolvedValue({
      data: { id: 10, name: "OpenAI Plus" },
    });

    const account = await completeOpenAIPendingCreate({
      code: "oauth-code",
      state: "oauth-state",
    });

    expect(postMock).toHaveBeenCalledWith(
      "/auth/openai/complete-pending-create",
      {
        code: "oauth-code",
        state: "oauth-state",
      },
      { skipAuthRedirect: true },
    );
    expect(account).toEqual({ id: 10, name: "OpenAI Plus" });
  });
});
