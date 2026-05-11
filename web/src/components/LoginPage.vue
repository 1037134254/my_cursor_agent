<script setup lang="ts">
import { onMounted, ref } from "vue";
import { login } from "../lib/auth";

const username = ref("");
const password = ref("");
const busy = ref(false);
const error = ref("");
const notice = ref("");

interface Provider {
  id: string;
  name: string;
}
const providers = ref<Provider[]>([]);

async function loadProviders() {
  try {
    const r = await fetch("/api/auth/oauth/providers");
    if (!r.ok) return;
    const j = (await r.json()) as { providers?: Provider[] };
    providers.value = j.providers ?? [];
  } catch {
    // 静默：第三方登录是可选项
  }
}

async function doLogin() {
  if (busy.value) return;
  error.value = "";
  busy.value = true;
  try {
    await login(username.value.trim(), password.value);
    password.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

interface OAuthMeta {
  id: string;
  name: string;
  /** 后端路由前缀（/api/auth/oauth/{slug}/start） */
  slug: string;
  /** 跳转中的提示语 */
  notice: string;
  /** 按钮 CSS 类后缀（颜色风格） */
  variant: "wechat" | "github" | "default";
}

const KNOWN_PROVIDERS: Record<string, OAuthMeta> = {
  wechat_web: { id: "wechat_web", name: "微信扫码登录", slug: "wechat", notice: "正在跳转微信扫码…", variant: "wechat" },
  github: { id: "github", name: "GitHub 登录", slug: "github", notice: "正在跳转 GitHub 授权…", variant: "github" },
};

function metaOf(id: string, fallbackName: string): OAuthMeta {
  return (
    KNOWN_PROVIDERS[id] ?? {
      id,
      name: fallbackName || id,
      slug: id,
      notice: `正在跳转 ${fallbackName || id}…`,
      variant: "default",
    }
  );
}

async function startOAuth(id: string, name: string) {
  error.value = "";
  const meta = metaOf(id, name);
  notice.value = meta.notice;
  try {
    const r = await fetch(`/api/auth/oauth/${meta.slug}/start`);
    const j = (await r.json()) as { url?: string; error?: string };
    if (!r.ok || !j.url) {
      throw new Error(j.error || `状态码 ${r.status}`);
    }
    window.location.href = j.url;
  } catch (e) {
    notice.value = "";
    error.value = e instanceof Error ? e.message : String(e);
  }
}

const props = defineProps<{ initialNotice?: string }>();

onMounted(() => {
  if (props.initialNotice) notice.value = props.initialNotice;
  void loadProviders();
});
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="title">my_cursor</h1>
      <p class="subtitle">企业 AI 工作台 · 请先登录</p>

      <form class="form" @submit.prevent="doLogin">
        <label>
          <span>账号</span>
          <input
            v-model="username"
            type="text"
            autocomplete="username"
            autofocus
            :disabled="busy"
            placeholder="用户名"
          />
        </label>
        <label>
          <span>密码</span>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            :disabled="busy"
            placeholder="密码"
          />
        </label>

        <button class="primary" type="submit" :disabled="busy || !username || !password">
          {{ busy ? "登录中…" : "登 录" }}
        </button>
      </form>

      <p v-if="error" class="error">{{ error }}</p>
      <p v-else-if="notice" class="notice">{{ notice }}</p>

      <div v-if="providers.length" class="oauth-row">
        <div class="divider"><span>或使用第三方登录</span></div>
        <button
          v-for="p in providers"
          :key="p.id"
          class="oauth"
          :class="`oauth-${metaOf(p.id, p.name).variant}`"
          type="button"
          :disabled="busy"
          @click="startOAuth(p.id, p.name)"
        >
          {{ metaOf(p.id, p.name).name }}
        </button>
      </div>

      <p class="hint">
        账号由管理员在 <code>AUTH_USERS</code> 中配置；初次部署可使用 <code>admin / 123456</code>。
      </p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  height: 100vh;
  display: grid;
  place-items: center;
  background: radial-gradient(1100px 600px at 20% -10%, #1d2a40 0%, transparent 60%),
    radial-gradient(900px 500px at 110% 110%, #2a1d3a 0%, transparent 60%),
    var(--bg, #0f1115);
  color: var(--fg, #e8eaed);
  font-family: ui-sans-serif, system-ui, sans-serif;
}
.login-card {
  width: 360px;
  padding: 28px 28px 22px;
  border: 1px solid var(--border, #2a3140);
  border-radius: 14px;
  background: var(--panel, #161a22);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.45);
}
.title {
  margin: 0 0 4px;
  font-size: 22px;
  letter-spacing: 0.5px;
}
.subtitle {
  margin: 0 0 22px;
  font-size: 13px;
  color: var(--muted, #8b939e);
}
.form {
  display: grid;
  gap: 12px;
}
.form label {
  display: grid;
  gap: 6px;
  font-size: 12px;
  color: var(--muted, #8b939e);
}
.form input {
  height: 38px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid var(--border, #2a3140);
  background: var(--panel2, #1c212b);
  color: var(--fg, #e8eaed);
  font-size: 14px;
  outline: none;
}
.form input:focus {
  border-color: var(--accent, #7dd3fc);
}
.primary {
  height: 40px;
  margin-top: 4px;
  border: 1px solid var(--accent, #7dd3fc);
  background: var(--accent, #7dd3fc);
  color: #0b0d12;
  font-weight: 600;
  font-size: 14px;
  border-radius: 8px;
  cursor: pointer;
}
.primary:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.oauth-row {
  margin-top: 18px;
  display: grid;
  gap: 8px;
}
.divider {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--muted, #8b939e);
  font-size: 12px;
}
.divider::before,
.divider::after {
  content: "";
  flex: 1;
  height: 1px;
  background: var(--border, #2a3140);
}
.oauth {
  width: 100%;
  height: 38px;
  border-radius: 8px;
  border: 1px solid var(--border, #2a3140);
  background: var(--panel2, #1c212b);
  color: var(--fg, #e8eaed);
  font-size: 13px;
  cursor: pointer;
}
.oauth:hover {
  filter: brightness(1.1);
}
.oauth-wechat {
  border-color: #2ec06d;
  background: rgba(46, 192, 109, 0.1);
  color: #6ee29c;
}
.oauth-github {
  border-color: #6e7681;
  background: rgba(110, 118, 129, 0.18);
  color: #e6edf3;
}
.error {
  margin: 14px 0 0;
  color: #f87171;
  font-size: 12px;
}
.notice {
  margin: 14px 0 0;
  color: var(--muted, #8b939e);
  font-size: 12px;
}
.hint {
  margin: 16px 0 0;
  font-size: 12px;
  color: var(--muted, #8b939e);
}
.hint code {
  background: var(--panel2, #1c212b);
  padding: 1px 6px;
  border-radius: 4px;
}
</style>
