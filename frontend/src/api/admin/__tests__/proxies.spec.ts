import { describe, expect, it, vi, beforeEach } from "vitest";

const { getMock, putMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  apiClient: {
    get: getMock,
    put: putMock,
  },
}));

import { getIPOptions, getNetworkMonitorStatus, updateNetworkMonitor } from "../proxies";

describe("admin proxies api", () => {
  beforeEach(() => {
    getMock.mockReset();
    putMock.mockReset();
  });

  it("getIPOptions 请求 ip-options 接口", async () => {
    getMock.mockResolvedValue({ data: [] });

    await getIPOptions();

    expect(getMock).toHaveBeenCalledWith("/admin/proxies/ip-options");
  });

  it("getNetworkMonitorStatus 请求 network-monitor 接口", async () => {
    getMock.mockResolvedValue({ data: { enabled: true } });

    await getNetworkMonitorStatus();

    expect(getMock).toHaveBeenCalledWith("/admin/proxies/network-monitor");
  });

  it("updateNetworkMonitor 使用 enabled 请求体更新 network-monitor 接口", async () => {
    putMock.mockResolvedValue({ data: { enabled: false } });

    await updateNetworkMonitor(false);

    expect(putMock).toHaveBeenCalledWith("/admin/proxies/network-monitor", {
      enabled: false,
    });
  });
});
