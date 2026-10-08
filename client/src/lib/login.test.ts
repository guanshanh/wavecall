import { describe, expect, it } from "vitest";
import { login } from "./login";

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("login", () => {
  it("returns the token and account on 200", async () => {
    const result = await login("achi", "pw", {
      baseUrl: "http://dispatch.example",
      fetchImpl: async (input, init) => {
        expect(String(input)).toBe("http://dispatch.example/login");
        expect(init?.method).toBe("POST");
        expect(init?.body).toBe(JSON.stringify({ account: "achi", password: "pw" }));
        return jsonResponse(200, { token: "tok", account: "achi" });
      },
    });
    expect(result).toEqual({ ok: true, token: "tok", account: "achi" });
  });

  it("maps 400 and 401 to the fixed copy", async () => {
    const bad = await login("", "pw", {
      baseUrl: "http://dispatch.example",
      fetchImpl: async () => jsonResponse(400, { error: "invalid" }),
    });
    expect(bad).toEqual({ ok: false, message: "请填写账号和密码" });

    const denied = await login("achi", "no", {
      baseUrl: "http://dispatch.example",
      fetchImpl: async () => jsonResponse(401, { error: "unauthorized" }),
    });
    expect(denied).toEqual({ ok: false, message: "账号或密码错误" });
  });

  it("maps network and other statuses to the dispatch failure copy", async () => {
    const network = await login("achi", "pw", {
      baseUrl: "http://dispatch.example",
      fetchImpl: async () => {
        throw new Error("offline");
      },
    });
    expect(network).toEqual({ ok: false, message: "无法联系调度服务" });

    const other = await login("achi", "pw", {
      baseUrl: "http://dispatch.example",
      fetchImpl: async () => jsonResponse(500, { error: "nope" }),
    });
    expect(other).toEqual({ ok: false, message: "无法联系调度服务" });
  });
});
