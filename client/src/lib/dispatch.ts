export const DISPATCH_CONFIG_ERROR_PREFIX = "DispatchConfigError:";

export const DISPATCH_URL_STORAGE_KEY = "wavecall.dispatchUrl";
export const DEFAULT_DISPATCH_BASE = "http://127.0.0.1:18090";

export type DispatchStorage = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
};

export class DispatchConfigError extends Error {
  constructor(message: string) {
    super(`${DISPATCH_CONFIG_ERROR_PREFIX}${message}`);
    this.name = "DispatchConfigError";
  }
}

export function isDispatchConfigError(err: unknown): boolean {
  if (err instanceof DispatchConfigError) {
    return true;
  }
  return (
    err instanceof Error && err.message.startsWith(DISPATCH_CONFIG_ERROR_PREFIX)
  );
}

export function normalizeDispatchBase(raw: string): string {
  const trimmed = String(raw).trim().replace(/\/$/, "");
  return trimmed.replace(/^https?:\/\//i, (scheme) => scheme.toLowerCase());
}

export function isValidDispatchBase(base: string): boolean {
  return /^https?:\/\//i.test(normalizeDispatchBase(base));
}

function defaultStorage(): DispatchStorage | undefined {
  try {
    if (typeof localStorage === "undefined") return undefined;
    return localStorage;
  } catch {
    return undefined;
  }
}

export function getDispatchBase(opts?: {
  env?: string;
  storage?: DispatchStorage;
}): string {
  const storage = opts?.storage ?? defaultStorage();
  const fromStorage = storage?.getItem(DISPATCH_URL_STORAGE_KEY);
  if (fromStorage && fromStorage.trim()) {
    return normalizeDispatchBase(fromStorage);
  }
  const env =
    opts?.env !== undefined
      ? opts.env
      : (import.meta.env.VITE_DISPATCH_URL as string | undefined);
  if (env && String(env).trim()) {
    return normalizeDispatchBase(String(env));
  }
  return DEFAULT_DISPATCH_BASE;
}

export function setDispatchBaseOverride(
  base: string,
  storage?: DispatchStorage,
): void {
  const target = storage ?? defaultStorage();
  if (!target) return;
  target.setItem(DISPATCH_URL_STORAGE_KEY, normalizeDispatchBase(base));
}

/** @deprecated Prefer getDispatchBase — kept as alias; never throws for missing env. */
export function requireDispatchBase(): string {
  return getDispatchBase();
}

export function nodeToWsUrl(node: string, dispatchBase: string): string {
  const https = /^https:\/\//i.test(dispatchBase);
  const scheme = https ? "wss" : "ws";
  return `${scheme}://${node}/ws`;
}

export async function resolveSignalingWsUrl(
  roomId: string,
  opts?: { baseUrl?: string; fetchImpl?: typeof fetch },
): Promise<string> {
  const base = opts?.baseUrl ?? getDispatchBase();
  const fetchImpl = opts?.fetchImpl ?? fetch;
  const url = `${base}/dispatch?room=${encodeURIComponent(roomId)}`;
  const res = await fetchImpl(url, { method: "GET" });
  if (!res.ok) {
    throw new Error(`dispatch failed: ${res.status} ${res.statusText}`);
  }
  const body = (await res.json()) as { node?: string };
  if (!body.node) {
    throw new Error("dispatch response missing node");
  }
  return nodeToWsUrl(body.node, base);
}
