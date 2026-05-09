<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

const SESSION_KEY = "session_id";

const sessionId = ref(localStorage.getItem(SESSION_KEY) ?? "");
const input = ref("");
const output = ref("");
const streaming = ref(false);
/** WebSocket 握手阶段 */
const wsConnecting = ref(false);
/** 已连接但尚未收到任何 delta（模型首 token 前的空窗） */
const awaitingFirstDelta = ref(false);
const llmProvider = ref("custom");
const llmApiURL = ref("https://open.bigmodel.cn/api/paas/v4/chat/completions");
const llmModel = ref("glm-5.1");
const llmBusy = ref(false);
const llmStatus = ref("");

const sessionLabel = computed(() => sessionId.value || "未创建");

const showStreamHint = computed(
  () =>
    streaming.value &&
    (wsConnecting.value || (awaitingFirstDelta.value && output.value === "")),
);
const streamHintText = computed(() =>
  wsConnecting.value ? "正在连接…" : "已连接，等待模型输出首字…",
);

async function loadLLMConfig() {
  llmBusy.value = true;
  llmStatus.value = "";
  try {
    const resp = await fetch("/api/llm/config");
    if (!resp.ok) throw new Error(`加载配置失败: ${resp.status}`);
    const data = (await resp.json()) as {
      config?: { provider?: string; api_url?: string; model?: string; has_api_key?: boolean };
    };
    const cfg = data.config ?? {};
    llmProvider.value = cfg.provider ?? "custom";
    llmApiURL.value = cfg.api_url ?? "https://open.bigmodel.cn/api/paas/v4/chat/completions";
    llmModel.value = cfg.model ?? "glm-5.1";
    llmStatus.value = cfg.has_api_key
      ? "已加载（密钥在服务端环境变量中）"
      : "已加载（尚未设置服务端密钥）";
  } catch (e) {
    llmStatus.value = String(e);
  } finally {
    llmBusy.value = false;
  }
}

async function applyLLMConfig() {
  if (llmBusy.value) return;
  llmBusy.value = true;
  llmStatus.value = "切换中…";
  try {
    const payload = {
      provider: llmProvider.value,
      api_url: llmApiURL.value,
      model: llmModel.value,
    };
    const resp = await fetch("/api/llm/config", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const data = await resp.json();
    if (!resp.ok) {
      throw new Error(data?.error || `切换失败: ${resp.status}`);
    }
    const cfg = data.config ?? {};
    llmProvider.value = cfg.provider ?? llmProvider.value;
    llmApiURL.value = cfg.api_url ?? llmApiURL.value;
    llmModel.value = cfg.model ?? llmModel.value;
    llmStatus.value = "已生效（密钥请在服务端环境变量配置）";
  } catch (e) {
    llmStatus.value = String(e);
  } finally {
    llmBusy.value = false;
  }
}

function persistSession(id: string) {
  if (!id || id === sessionId.value) return;
  sessionId.value = id;
  localStorage.setItem(SESSION_KEY, id);
}

function wsURL(): string {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  return proto + "//" + location.host + "/api/chat/ws";
}

/**
 * 与后端 api/stream_handler.go 约定一致。
 * delta 直接写入界面：LLM 吐字速度远低于 DOM，去掉固定批量间隔可避免「明明已有 token 却晚 80ms 才看见」的拖沓感。
 */
function streamChat() {
  const msg = input.value.trim();
  if (!msg || streaming.value) return;

  streaming.value = true;
  wsConnecting.value = true;
  awaitingFirstDelta.value = true;
  output.value = "";

  const ws = new WebSocket(wsURL());
  ws.onopen = () => {
    wsConnecting.value = false;
    ws.send(JSON.stringify({ msg, session_id: sessionId.value }));
  };

  ws.onmessage = (ev) => {
    let data: { type?: string; content?: string; message?: string; session_id?: string };
    try {
      data = JSON.parse(ev.data as string);
    } catch {
      output.value += "\n[解析失败] " + String(ev.data);
      ws.close();
      return;
    }

    if (data.type === "delta" && data.content) {
      awaitingFirstDelta.value = false;
      output.value += data.content;
    } else if (data.type === "meta" && data.session_id) {
      persistSession(data.session_id);
    } else if (data.type === "error") {
      awaitingFirstDelta.value = false;
      output.value += "\n[错误] " + (data.message ?? "");
      ws.close();
    } else if (data.type === "done") {
      if (data.session_id) persistSession(data.session_id);
      ws.close();
    }
  };

  ws.onerror = () => {
    awaitingFirstDelta.value = false;
    output.value += "\n[WebSocket 错误]";
  };

  ws.onclose = () => {
    wsConnecting.value = false;
    awaitingFirstDelta.value = false;
    streaming.value = false;
  };
}

function onInputKeydown(e: KeyboardEvent) {
  if (e.key !== "Enter") return;
  if (e.shiftKey) return;
  if (e.isComposing) return;
  e.preventDefault();
  streamChat();
}

async function httpChat() {
  const msg = input.value.trim();
  if (!msg || streaming.value) return;
  streaming.value = true;
  output.value = "请求中…";
  try {
    const resp = await fetch("/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ msg, session_id: sessionId.value }),
    });
    const data = await resp.json();
    if (data.session_id) persistSession(data.session_id);
    output.value = JSON.stringify(data, null, 2);
  } catch (e) {
    output.value = "请求失败: " + String(e);
  } finally {
    streaming.value = false;
  }
}

onMounted(() => {
  sessionId.value = localStorage.getItem(SESSION_KEY) ?? "";
  void loadLLMConfig();
});

defineExpose({ streamChat, httpChat });
</script>

<template>
  <div class="chat">
    <header class="chat-head">
      <span class="title">AI</span>
      <span class="sess">会话 <code>{{ sessionLabel }}</code></span>
    </header>

    <details class="llm-config">
      <summary>模型配置（页面手动切换，安全模式）</summary>
      <div class="llm-grid">
        <label>
          提供商
          <select v-model="llmProvider" :disabled="llmBusy">
            <option value="qwen">qwen</option>
            <option value="ollama">ollama</option>
            <option value="deepseek">deepseek</option>
            <option value="custom">custom</option>
          </select>
        </label>
        <label>
          API URL
          <input v-model.trim="llmApiURL" :disabled="llmBusy" placeholder="https://.../v1/chat/completions" />
        </label>
        <label>
          Model
          <input v-model.trim="llmModel" :disabled="llmBusy" placeholder="deepseek-chat / qwen..." />
        </label>
      </div>
      <div class="llm-actions">
        <button type="button" class="btn primary" :disabled="llmBusy" @click="applyLLMConfig">应用配置</button>
        <button type="button" class="btn" :disabled="llmBusy" @click="loadLLMConfig">刷新</button>
      </div>
      <p v-if="llmStatus" class="llm-status">{{ llmStatus }}</p>
      <p class="llm-note">为避免明文泄露，页面不支持提交 API Key，请在服务端环境变量设置。</p>
    </details>

    <textarea
      v-model="input"
      class="msg"
      placeholder="Enter 发送（WS 流式），Shift+Enter 换行"
      rows="4"
      @keydown="onInputKeydown"
    />

    <div class="actions">
      <button type="button" class="btn primary" :disabled="streaming" @click="streamChat">
        流式 (WS)
      </button>
      <button type="button" class="btn" :disabled="streaming" @click="httpChat">HTTP JSON</button>
    </div>

    <div class="out-wrap">
      <div v-if="showStreamHint" class="stream-hint">{{ streamHintText }}</div>
      <pre class="out">{{ output || (!showStreamHint ? "回复显示在这里…" : "") }}</pre>
    </div>
  </div>
</template>

<style scoped>
.chat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: 100%;
  min-height: 0;
}
.chat-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: var(--muted);
}
.title {
  font-weight: 600;
  color: var(--fg);
}
.sess code {
  font-size: 11px;
}
.msg {
  width: 100%;
  flex: 0 0 auto;
  resize: vertical;
  min-height: 72px;
  padding: 8px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  box-sizing: border-box;
}
.llm-config {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 6px 8px;
  background: var(--panel2);
}
.llm-config summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--muted);
  user-select: none;
}
.llm-grid {
  margin-top: 8px;
  display: grid;
  gap: 8px;
}
.llm-grid label {
  display: grid;
  gap: 4px;
  font-size: 12px;
  color: var(--muted);
}
.llm-grid input,
.llm-grid select {
  width: 100%;
  box-sizing: border-box;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--fg);
  padding: 6px 8px;
  font-size: 12px;
}
.llm-actions {
  margin-top: 8px;
  display: flex;
  gap: 8px;
}
.llm-status {
  margin: 6px 2px 2px;
  font-size: 12px;
  color: var(--muted);
}
.llm-note {
  margin: 4px 2px 2px;
  font-size: 12px;
  color: #f9b77e;
}
.actions {
  display: flex;
  gap: 8px;
}
.btn {
  padding: 6px 12px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  cursor: pointer;
  font-size: 13px;
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn.primary {
  background: var(--accent);
  border-color: var(--accent);
  color: #0b0d12;
}
.out-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 120px;
  min-width: 0;
}
.stream-hint {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--muted);
  padding: 4px 2px 8px;
  animation: pulse-opacity 1.2s ease-in-out infinite;
}
@keyframes pulse-opacity {
  0%,
  100% {
    opacity: 0.65;
  }
  50% {
    opacity: 1;
  }
}
.out {
  flex: 1;
  min-height: 80px;
  margin: 0;
  padding: 10px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--fg);
  white-space: pre-wrap;
  word-break: break-word;
  overflow: auto;
  font-size: 13px;
}
</style>
