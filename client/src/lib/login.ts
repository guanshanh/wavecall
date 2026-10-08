import { getDispatchBase } from "./dispatch";

export type LoginResult =
  | { ok: true; token: string; account: string }
  | { ok: false; message: string };

const FILL_MESSAGE = "请填写账号和密码";
const DENIED_MESSAGE = "账号或密码错误";
const DISPATCH_MESSAGE = "无法联系调度服务";

export async function login(
  account: string,
  password: string,
  opts?: { baseUrl?: string; fetchImpl?: typeof fetch },
): Promise<LoginResult> {
  const base = opts?.baseUrl ?? getDispatchBase();
  const fetchImpl = opts?.fetchImpl ?? fetch;
  let res: Response;
  try {
    res = await fetchImpl(`${base}/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ account, password }),
    });
  } catch {
    return { ok: false, message: DISPATCH_MESSAGE };
  }
  if (res.status === 400) {
    return { ok: false, message: FILL_MESSAGE };
  }
  if (res.status === 401) {
    return { ok: false, message: DENIED_MESSAGE };
  }
  if (!res.ok) {
    return { ok: false, message: DISPATCH_MESSAGE };
  }
  let body: { token?: string; account?: string };
  try {
    body = (await res.json()) as { token?: string; account?: string };
  } catch {
    return { ok: false, message: DISPATCH_MESSAGE };
  }
  if (!body.token || !body.account) {
    return { ok: false, message: DISPATCH_MESSAGE };
  }
  return { ok: true, token: body.token, account: body.account };
}
