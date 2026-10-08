import { afterEach, describe, expect, it, vi } from "vitest";
import {
  DEFAULT_DISPATCH_BASE,
  DISPATCH_URL_STORAGE_KEY,
  getDispatchBase,
  isValidDispatchBase,
  normalizeDispatchBase,
  nodeToWsUrl,
  resolveSignalingWsUrl,
  setDispatchBaseOverride,
} from "./dispatch";

function memoryStorage(initial: Record<string, string> = {}): {
  store: Record<string, string>;
  storage: { getItem(k: string): string | null; setItem(k: string, v: string): void };
} {
  const store = { ...initial };
  return {
    store,
    storage: {
      getItem: (k) => store[k] ?? null,
      setItem: (k, v) => {
        store[k] = v;
      },
    },
  };
}

afterEach(() => {
  vi.unstubAllEnvs();
});

describe("normalizeDispatchBase / isValidDispatchBase", () => {
  it("trims and strips trailing slash", () => {
    expect(normalizeDispatchBase("  http://a:18090/  ")).toBe("http://a:18090");
  });

  it("lowercases http/https scheme", () => {
    expect(normalizeDispatchBase("HTTPS://example.com")).toBe(
      "https://example.com",
    );
    expect(normalizeDispatchBase("HTTP://127.0.0.1:18090/")).toBe(
      "http://127.0.0.1:18090",
    );
  });

  it("accepts http and https only", () => {
    expect(isValidDispatchBase("http://127.0.0.1:18090")).toBe(true);
    expect(isValidDispatchBase("HTTPS://example.com")).toBe(true);
    expect(isValidDispatchBase("")).toBe(false);
    expect(isValidDispatchBase("127.0.0.1:18090")).toBe(false);
    expect(isValidDispatchBase("ftp://x")).toBe(false);
  });
});

describe("getDispatchBase", () => {
  it("prefers localStorage over env", () => {
    const { storage } = memoryStorage({
      [DISPATCH_URL_STORAGE_KEY]: " http://override:9/ ",
    });
    vi.stubEnv("VITE_DISPATCH_URL", "http://from-env:18090");
    expect(getDispatchBase({ env: "http://from-env:18090", storage })).toBe(
      "http://override:9",
    );
  });

  it("uses env when no override", () => {
    const { storage } = memoryStorage();
    expect(getDispatchBase({ env: "http://from-env:18090/", storage })).toBe(
      "http://from-env:18090",
    );
  });

  it("falls back to DEFAULT_DISPATCH_BASE", () => {
    const { storage } = memoryStorage();
    expect(getDispatchBase({ env: "  ", storage })).toBe(DEFAULT_DISPATCH_BASE);
  });
});

describe("setDispatchBaseOverride", () => {
  it("writes normalized value", () => {
    const { store, storage } = memoryStorage();
    setDispatchBaseOverride(" http://saved:18090/ ", storage);
    expect(store[DISPATCH_URL_STORAGE_KEY]).toBe("http://saved:18090");
  });
});

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

  it("uses wss when dispatch base scheme is uppercase", () => {
    expect(nodeToWsUrl("n1.example:18080", "HTTPS://dispatch:18090")).toBe(
      "wss://n1.example:18080/ws",
    );
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
