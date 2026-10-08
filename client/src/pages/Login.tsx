import { useState } from "react";
import {
  getDispatchBase,
  isValidDispatchBase,
  normalizeDispatchBase,
  setDispatchBaseOverride,
} from "../lib/dispatch";
import { login } from "../lib/login";
import { useAuthStore } from "../stores/authStore";

const INVALID_SERVER_MESSAGE = "请填写有效的服务器地址";

export default function Login() {
  const setSession = useAuthStore((s) => s.setSession);
  const [serverUrl, setServerUrl] = useState(() => getDispatchBase());
  const [account, setAccount] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (pending) return;
    setError("");
    const base = normalizeDispatchBase(serverUrl);
    if (!isValidDispatchBase(base)) {
      setError(INVALID_SERVER_MESSAGE);
      return;
    }
    setPending(true);
    const result = await login(account, password, { baseUrl: base });
    setPending(false);
    if (!result.ok) {
      setError(result.message);
      return;
    }
    setDispatchBaseOverride(base);
    setServerUrl(base);
    setPassword("");
    setSession(result.token, result.account);
  };

  return (
    <div className="min-h-screen bg-gray-900 flex items-center justify-center">
      <div className="bg-gray-800 rounded-2xl p-8 w-full max-w-md shadow-2xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-white mb-2">🌊 Wavecall</h1>
          <p className="text-gray-400">使用已登记的账号登录</p>
        </div>
        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">服务器地址</label>
            <input
              type="url"
              inputMode="url"
              spellCheck={false}
              value={serverUrl}
              onChange={(e) => setServerUrl(e.target.value)}
              className="w-full px-4 py-3 bg-gray-700 text-white rounded-lg outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              required
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">账号</label>
            <input
              type="text"
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              autoComplete="username"
              className="w-full px-4 py-3 bg-gray-700 text-white rounded-lg outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">密码</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              className="w-full px-4 py-3 bg-gray-700 text-white rounded-lg outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>
          {error ? <p className="text-sm text-red-400">{error}</p> : null}
          <button
            type="submit"
            disabled={pending}
            className="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white font-semibold rounded-lg transition-colors cursor-pointer disabled:opacity-60"
          >
            登录
          </button>
        </form>
      </div>
    </div>
  );
}
