import { describe, it, expect, vi } from "vitest";
import {
  DispatchConfigError,
  nodeToWsUrl,
  requireDispatchBase,
  resolveSignalingWsUrl,
} from "./dispatch";

describe("nodeToWsUrl", () => {
  it("uses ws for http dispatch base", () => {
    expect(nodeToWsUrl("n1.example:18080", "http://dispatch:18090")).toBe(
      "ws://n1.example:18080/ws",
    );
  });
  it("uses wss for https dispatch base", () => {
    expect(nodeToWsUrl("n1.example:18080", "https://dispatch:18090")).toBe(
      "wss://n1.example:18080/ws",
    );
  });
});

describe("requireDispatchBase", () => {
  it("throws DispatchConfigError when unset", () => {
    vi.stubEnv("VITE_DISPATCH_URL", "");
    expect(() => requireDispatchBase()).toThrow(DispatchConfigError);
    vi.unstubAllEnvs();
  });
});

describe("resolveSignalingWsUrl", () => {
  it("fetches node and builds ws url", async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ node: "127.0.0.1:18080" }),
    });
    const url = await resolveSignalingWsUrl("room-1", {
      baseUrl: "http://localhost:18090",
      fetchImpl: fetchImpl as unknown as typeof fetch,
    });
    expect(fetchImpl).toHaveBeenCalledWith(
      "http://localhost:18090/dispatch?room=room-1",
      expect.any(Object),
    );
    expect(url).toBe("ws://127.0.0.1:18080/ws");
  });

  it("throws on non-OK", async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      statusText: "Unavailable",
    });
    await expect(
      resolveSignalingWsUrl("room-1", {
        baseUrl: "http://localhost:18090",
        fetchImpl: fetchImpl as unknown as typeof fetch,
      }),
    ).rejects.toThrow(/503/);
  });
});
