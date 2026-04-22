import { describe, expect, it, vi, beforeEach } from "vitest";

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  apiClient: {
    get: getMock,
  },
}));

import { exportData, list } from "../accounts";

describe("admin accounts api", () => {
  beforeEach(() => {
    getMock.mockReset();
  });

  it("list 会携带 ip 筛选参数", async () => {
    getMock.mockResolvedValue({
      data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 },
    });

    await list(1, 20, { ip: "203.0.113.10", search: "keyword" });

    expect(getMock).toHaveBeenCalledWith("/admin/accounts", {
      params: {
        page: 1,
        page_size: 20,
        ip: "203.0.113.10",
        search: "keyword",
      },
      signal: undefined,
    });
  });

  it("exportData 会携带 ip 筛选参数", async () => {
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
        platform: "openai",
        sort_by: "priority",
        sort_order: "desc",
      },
    });

    expect(getMock).toHaveBeenCalledWith("/admin/accounts/data", {
      params: {
        ip: "203.0.113.10",
        platform: "openai",
        sort_by: "priority",
        sort_order: "desc",
      },
    });
  });
});
