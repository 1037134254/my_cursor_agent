const ACCESS_KEY = "access_token";
const REFRESH_KEY = "refresh_token";

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_KEY);
}

/** HTTP API 用的 Authorization 头（未登录返回空对象）。 */
export function authHeaders(): Record<string, string> {
  const t = getAccessToken();
  if (!t) return {};
  return { Authorization: `Bearer ${t}` };
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export async function login(username: string, password: string): Promise<void> {
  const r = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  const text = await r.text();
  if (!r.ok) throw new Error(text || `登录失败 ${r.status}`);
  const j = JSON.parse(text) as {
    access_token?: string;
    refresh_token?: string;
  };
  if (j.access_token) localStorage.setItem(ACCESS_KEY, j.access_token);
  if (j.refresh_token) localStorage.setItem(REFRESH_KEY, j.refresh_token);
}

export async function logout(): Promise<void> {
  const rt = localStorage.getItem(REFRESH_KEY);
  clearTokens();
  await fetch("/api/auth/logout", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: rt ?? "" }),
  }).catch(() => {});
}
