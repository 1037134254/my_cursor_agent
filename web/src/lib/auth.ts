import { computed, ref } from "vue";

const ACCESS_KEY = "access_token";
const REFRESH_KEY = "refresh_token";

const accessTokenRef = ref<string | null>(localStorage.getItem(ACCESS_KEY));

export const isLoggedIn = computed(() => !!accessTokenRef.value);

export function getAccessToken(): string | null {
  return accessTokenRef.value;
}

export function authHeaders(): Record<string, string> {
  const t = accessTokenRef.value;
  if (!t) return {};
  return { Authorization: `Bearer ${t}` };
}

function setTokens(access: string | null, refresh?: string | null): void {
  if (access) {
    localStorage.setItem(ACCESS_KEY, access);
    accessTokenRef.value = access;
  } else {
    localStorage.removeItem(ACCESS_KEY);
    accessTokenRef.value = null;
  }
  if (refresh === null) {
    localStorage.removeItem(REFRESH_KEY);
  } else if (refresh) {
    localStorage.setItem(REFRESH_KEY, refresh);
  }
}

export function clearTokens(): void {
  setTokens(null, null);
}

export interface CurrentUser {
  username: string;
  tenantId: string;
  roles: string[];
}

/** 从 JWT payload 解出用户信息，未登录或解析失败返回 null。 */
export function currentUser(): CurrentUser | null {
  const t = accessTokenRef.value;
  if (!t) return null;
  const parts = t.split(".");
  if (parts.length < 2) return null;
  try {
    const json = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"));
    const p = JSON.parse(json) as {
      uid?: string;
      tid?: string;
      roles?: string[];
    };
    return {
      username: p.uid ?? "",
      tenantId: p.tid ?? "",
      roles: p.roles ?? [],
    };
  } catch {
    return null;
  }
}

export async function login(username: string, password: string): Promise<void> {
  const r = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  const text = await r.text();
  if (!r.ok) {
    let msg = `登录失败 ${r.status}`;
    try {
      const errBody = JSON.parse(text) as { error?: string };
      if (errBody.error) msg = errBody.error;
    } catch {
      // 服务端没返回 JSON，使用默认状态码消息
    }
    throw new Error(msg);
  }
  const j = JSON.parse(text) as {
    access_token?: string;
    refresh_token?: string;
  };
  setTokens(j.access_token ?? null, j.refresh_token ?? null);
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

/** 微信回调 OAuth：消费 URL 上的 access_token / refresh_token / oauth_error。 */
export function consumeOAuthRedirect(): { ok: boolean; error?: string } {
  const q = new URLSearchParams(window.location.search);
  const err = q.get("oauth_error");
  const at = q.get("access_token");
  const rt = q.get("refresh_token");
  if (at) {
    setTokens(at, rt ?? null);
    window.history.replaceState({}, "", window.location.pathname + window.location.hash);
    return { ok: true };
  }
  if (err) {
    window.history.replaceState({}, "", window.location.pathname + window.location.hash);
    return { ok: false, error: err };
  }
  return { ok: false };
}

/** 用 refresh_token 静默换新令牌；成功返回 true 并刷新内存中的 access_token。 */
export async function refreshAccessToken(): Promise<boolean> {
  const rt = localStorage.getItem(REFRESH_KEY);
  if (!rt) return false;
  try {
    const r = await fetch("/api/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: rt }),
    });
    if (!r.ok) {
      clearTokens();
      return false;
    }
    const j = (await r.json()) as { access_token?: string; refresh_token?: string };
    if (!j.access_token) {
      clearTokens();
      return false;
    }
    setTokens(j.access_token, j.refresh_token ?? rt);
    return true;
  } catch {
    clearTokens();
    return false;
  }
}
