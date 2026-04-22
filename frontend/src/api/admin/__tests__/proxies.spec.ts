import { describe, expect, it, vi, beforeEach } from "vitest";

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  apiClient: {
    get: getMock,
  },
}));

import { getIPOptions } from "../proxies";

describe("admin proxies api", () => {
  beforeEach(() => {
    getMock.mockReset();
  });

  it("getIPOptions 请求 ip-options 接口", async () => {
    getMock.mockResolvedValue({ data: [] });

    await getIPOptions();

    expect(getMock).toHaveBeenCalledWith("/admin/proxies/ip-options");
  });
});
