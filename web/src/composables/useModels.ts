import { authHeaders } from "../lib/auth";

export interface ModelProfile {
  id: string;
  name: string;
  purpose: "chat" | "embed" | "rerank";
  provider: "custom" | "qwen" | "ollama" | "deepseek";
  endpoint: string;
  model: string;
  api_key_ref: string;
  api_key_resolved: boolean;
  timeout_seconds: number;
  max_retries: number;
  retry_backoff_ms: number;
  rpm: number;
  stream_timeout_seconds: number;
  enabled: boolean;
  created_by: string;
  created_at: number;
  updated_at: number;
}

export interface TenantBinding {
  tenant_id: string;
  profile_id: string;
  purpose: "chat" | "embed" | "rerank";
  is_default: boolean;
  rpm_override: number;
  created_at: number;
}

async function asError(r: Response): Promise<never> {
  let msg = `${r.status}`;
  try {
    const j = (await r.json()) as { error?: string };
    if (j.error) msg = j.error;
  } catch {
    /* 非 JSON 响应，沿用状态码 */
  }
  throw new Error(msg);
}

export async function listProfiles(): Promise<{ profiles: ModelProfile[]; store: string }> {
  const r = await fetch("/api/llm/profiles", { headers: { ...authHeaders() } });
  if (!r.ok) await asError(r);
  return (await r.json()) as { profiles: ModelProfile[]; store: string };
}

export async function upsertProfile(
  p: Partial<ModelProfile> & { id: string },
  opts?: { create?: boolean },
): Promise<ModelProfile> {
  const create = opts?.create ?? false;
  const url = create ? "/api/llm/profiles" : `/api/llm/profiles/${encodeURIComponent(p.id)}`;
  const method = create ? "POST" : "PUT";
  const r = await fetch(url, {
    method,
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(p),
  });
  if (!r.ok) await asError(r);
  const j = (await r.json()) as { profile: ModelProfile };
  return j.profile;
}

export async function deleteProfile(id: string): Promise<void> {
  const r = await fetch(`/api/llm/profiles/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: { ...authHeaders() },
  });
  if (!r.ok) await asError(r);
}

export async function probeProfile(id: string): Promise<string> {
  const r = await fetch(`/api/llm/profiles/${encodeURIComponent(id)}/probe`, {
    method: "POST",
    headers: { ...authHeaders() },
  });
  if (!r.ok) await asError(r);
  const j = (await r.json()) as { reply?: string };
  return j.reply ?? "";
}

export async function listBindings(tenantId: string): Promise<TenantBinding[]> {
  const r = await fetch(`/api/tenants/${encodeURIComponent(tenantId)}/models`, {
    headers: { ...authHeaders() },
  });
  if (!r.ok) await asError(r);
  const j = (await r.json()) as { bindings: TenantBinding[] };
  return j.bindings ?? [];
}

export async function putBindings(
  tenantId: string,
  bindings: Array<{
    profile_id: string;
    purpose: "chat" | "embed" | "rerank";
    is_default: boolean;
    rpm_override: number;
  }>,
): Promise<TenantBinding[]> {
  const r = await fetch(`/api/tenants/${encodeURIComponent(tenantId)}/models`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify({ bindings }),
  });
  if (!r.ok) await asError(r);
  const j = (await r.json()) as { bindings: TenantBinding[] };
  return j.bindings ?? [];
}

export async function getCurrentBinding(purpose: "chat" | "embed" | "rerank" = "chat"): Promise<{
  profile: ModelProfile | null;
  error: string | null;
}> {
  const r = await fetch(`/api/llm/current?purpose=${purpose}`, { headers: { ...authHeaders() } });
  if (!r.ok) {
    let err = `${r.status}`;
    try {
      const j = (await r.json()) as { error?: string };
      if (j.error) err = j.error;
    } catch {
      /* 沿用状态码 */
    }
    return { profile: null, error: err };
  }
  const j = (await r.json()) as { profile: ModelProfile };
  return { profile: j.profile, error: null };
}
