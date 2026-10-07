export const DISPATCH_CONFIG_ERROR_PREFIX = "DispatchConfigError:";

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

export function requireDispatchBase(): string {
  const base = import.meta.env.VITE_DISPATCH_URL as string | undefined;
  if (!base || !String(base).trim()) {
    throw new DispatchConfigError(
      "VITE_DISPATCH_URL is required (dispatcher base URL, e.g. http://localhost:18090)",
    );
  }
  return String(base).replace(/\/$/, "");
}

export function nodeToWsUrl(node: string, dispatchBase: string): string {
  const https = dispatchBase.startsWith("https://");
  const scheme = https ? "wss" : "ws";
  return `${scheme}://${node}/ws`;
}

export async function resolveSignalingWsUrl(
  roomId: string,
  opts?: { baseUrl?: string; fetchImpl?: typeof fetch },
): Promise<string> {
  const base = opts?.baseUrl ?? requireDispatchBase();
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
