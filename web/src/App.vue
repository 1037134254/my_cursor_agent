<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import ChatPanel from "./components/ChatPanel.vue";
import EditorPane from "./components/EditorPane.vue";
import FileTree from "./components/FileTree.vue";
import LoginPage from "./components/LoginPage.vue";
import ModelAdmin from "./components/ModelAdmin.vue";
import {
  consumeOAuthRedirect,
  currentUser,
  isLoggedIn,
  logout as doLogout,
} from "./lib/auth";
import { getCurrentBinding } from "./composables/useModels";
import { fetchWorkspaceFile, fetchWorkspaceFiles, putWorkspaceFile } from "./composables/useWorkspace";

const paths = ref<string[]>([]);
const currentPath = ref<string | null>(null);
const editorText = ref("");
const dirty = ref(false);
const status = ref("");
const loadingTree = ref(false);
const oauthError = ref("");

const me = computed(() => currentUser());

const isAdmin = computed(() => {
  const u = me.value;
  return !!u?.roles?.includes("admin");
});

const showModelAdmin = ref(false);
const chatModelHint = ref("");

async function refreshChatModelHint() {
  if (!isLoggedIn.value) {
    chatModelHint.value = "";
    return;
  }
  const { profile, error } = await getCurrentBinding("chat");
  if (profile) {
    chatModelHint.value = `chat: ${profile.id} · ${profile.model}`;
  } else {
    chatModelHint.value = error ? `chat: 未就绪（${error}）` : "chat: 未就绪";
  }
}

// ---------- 三栏可拖拽宽度（持久化到 localStorage） ----------
const LAYOUT_KEY = "layout_widths_v1";
const MIN_SIDE = 160;
const MAX_LEFT = 480;
const MAX_RIGHT = 560;

function loadLayout(): { left: number; right: number } {
  try {
    const raw = localStorage.getItem(LAYOUT_KEY);
    if (raw) {
      const j = JSON.parse(raw) as { left?: number; right?: number };
      return {
        left: clamp(j.left ?? 240, MIN_SIDE, MAX_LEFT),
        right: clamp(j.right ?? 320, MIN_SIDE, MAX_RIGHT),
      };
    }
  } catch {
    /* ignore */
  }
  return { left: 240, right: 320 };
}

function clamp(n: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, n));
}

const initial = loadLayout();
const leftWidth = ref<number>(initial.left);
const rightWidth = ref<number>(initial.right);

const bodyStyle = computed(() => ({
  gridTemplateColumns: `${leftWidth.value}px 6px minmax(0, 1fr) 6px ${rightWidth.value}px`,
}));

function persistLayout() {
  localStorage.setItem(
    LAYOUT_KEY,
    JSON.stringify({ left: leftWidth.value, right: rightWidth.value }),
  );
}

function beginDrag(side: "left" | "right", e: PointerEvent) {
  e.preventDefault();
  const startX = e.clientX;
  const startW = side === "left" ? leftWidth.value : rightWidth.value;

  function onMove(ev: PointerEvent) {
    const delta = ev.clientX - startX;
    if (side === "left") {
      leftWidth.value = clamp(startW + delta, MIN_SIDE, MAX_LEFT);
    } else {
      // 右分隔条：向左拖应让右栏变宽
      rightWidth.value = clamp(startW - delta, MIN_SIDE, MAX_RIGHT);
    }
  }
  function onUp() {
    window.removeEventListener("pointermove", onMove);
    window.removeEventListener("pointerup", onUp);
    document.body.style.cursor = "";
    persistLayout();
  }
  document.body.style.cursor = "col-resize";
  window.addEventListener("pointermove", onMove);
  window.addEventListener("pointerup", onUp);
}

// ---------- 新建文件 ----------
const newFilePrompting = ref(false);
const newFilePath = ref("");
const newFileBusy = ref(false);

function startNewFile() {
  newFilePath.value = "";
  newFilePrompting.value = true;
}

async function confirmNewFile() {
  const p = newFilePath.value.trim().replace(/^[/\\]+/, "");
  if (!p) {
    newFilePrompting.value = false;
    return;
  }
  newFileBusy.value = true;
  status.value = "";
  try {
    await putWorkspaceFile(p, "");
    await refreshTree();
    await openFile(p);
    newFilePrompting.value = false;
    status.value = `已新建 ${p}`;
  } catch (e) {
    status.value = e instanceof Error ? e.message : String(e);
  } finally {
    newFileBusy.value = false;
  }
}

function cancelNewFile() {
  newFilePrompting.value = false;
  newFilePath.value = "";
}

async function refreshTree() {
  if (!isLoggedIn.value) {
    paths.value = [];
    return;
  }
  loadingTree.value = true;
  status.value = "";
  try {
    paths.value = await fetchWorkspaceFiles();
  } catch (e) {
    status.value = String(e);
  } finally {
    loadingTree.value = false;
  }
}

async function openFile(path: string) {
  status.value = "";
  try {
    const text = await fetchWorkspaceFile(path);
    currentPath.value = path;
    editorText.value = text;
    dirty.value = false;
  } catch (e) {
    status.value = String(e);
  }
}

async function saveCurrent() {
  if (!currentPath.value) {
    status.value = "请先打开或新建路径";
    return;
  }
  status.value = "保存中…";
  try {
    await putWorkspaceFile(currentPath.value, editorText.value);
    dirty.value = false;
    status.value = "已保存";
    await refreshTree();
  } catch (e) {
    status.value = String(e);
  }
}

function onEditorDirty() {
  dirty.value = true;
}

function onKeyDown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === "s") {
    e.preventDefault();
    void saveCurrent();
  }
}

async function handleLogout() {
  await doLogout();
  paths.value = [];
  currentPath.value = null;
  editorText.value = "";
  dirty.value = false;
  status.value = "已退出登录";
}

watch(
  isLoggedIn,
  (val) => {
    if (val) {
      void refreshTree();
      void refreshChatModelHint();
    } else {
      chatModelHint.value = "";
    }
  },
  { immediate: false },
);

onMounted(() => {
  const r = consumeOAuthRedirect();
  if (r.error) oauthError.value = "微信登录失败：" + r.error;
  window.addEventListener("keydown", onKeyDown);
  if (isLoggedIn.value) {
    void refreshTree();
    void refreshChatModelHint();
  }
});

onUnmounted(() => {
  window.removeEventListener("keydown", onKeyDown);
});
</script>

<template>
  <LoginPage v-if="!isLoggedIn" :initial-notice="oauthError" />

  <div v-else class="app">
    <header class="top-bar">
      <span class="brand">my_cursor</span>
      <span class="user-info" v-if="me">
        <span class="user-name">{{ me.username }}</span>
        <span class="tenant">@{{ me.tenantId }}</span>
        <span v-if="me.roles.length" class="roles">{{ me.roles.join(",") }}</span>
        <span v-if="chatModelHint" class="model-hint">{{ chatModelHint }}</span>
      </span>
      <button v-if="isAdmin" type="button" class="btn-models" @click="showModelAdmin = true">模型</button>
      <button type="button" class="btn-logout" @click="handleLogout">退出</button>
    </header>

    <div v-if="showModelAdmin" class="model-overlay">
      <div class="model-overlay-inner">
        <header class="model-overlay-head">
          <span>模型注册表</span>
          <button type="button" class="btn-close" @click="showModelAdmin = false">关闭</button>
        </header>
        <ModelAdmin />
      </div>
    </div>

    <div class="body" :style="bodyStyle">
      <aside class="sidebar">
        <div class="side-head">
          <span>文件</span>
          <span class="side-actions">
            <button type="button" class="linkish" :disabled="loadingTree || newFileBusy" @click="startNewFile">+ 新建</button>
            <button type="button" class="linkish" :disabled="loadingTree" @click="refreshTree">刷新</button>
          </span>
        </div>
        <form v-if="newFilePrompting" class="new-file" @submit.prevent="confirmNewFile">
          <input
            v-model="newFilePath"
            type="text"
            placeholder="路径，例如 notes/today.md"
            autofocus
            :disabled="newFileBusy"
            @keydown.esc.prevent="cancelNewFile"
          />
          <div class="new-file-actions">
            <button type="submit" class="btn-mini primary" :disabled="newFileBusy || !newFilePath.trim()">建</button>
            <button type="button" class="btn-mini" :disabled="newFileBusy" @click="cancelNewFile">取消</button>
          </div>
        </form>
        <FileTree :paths="paths" @select="openFile" />
      </aside>

      <div class="splitter" @pointerdown="beginDrag('left', $event)" />

      <main class="editor-wrap">
        <header class="editor-bar">
          <span class="path">{{ currentPath || "未打开文件" }}</span>
          <span v-if="dirty" class="dot">●</span>
          <button type="button" class="btn-save" @click="saveCurrent">保存</button>
        </header>
        <EditorPane :file-path="currentPath" :model-value="editorText" @update:model-value="editorText = $event" @dirty="onEditorDirty" />
        <p v-if="status" class="status">{{ status }}</p>
      </main>

      <div class="splitter" @pointerdown="beginDrag('right', $event)" />

      <aside class="chat-wrap">
        <ChatPanel />
      </aside>
    </div>
  </div>
</template>

<style>
:root {
  --bg: #0f1115;
  --panel: #161a22;
  --panel2: #1c212b;
  --border: #2a3140;
  --fg: #e8eaed;
  --muted: #8b939e;
  --hover: #252b36;
  --accent: #7dd3fc;
}
html,
body {
  margin: 0;
  height: 100%;
}
#app {
  height: 100%;
}
</style>

<style scoped>
.app {
  display: grid;
  grid-template-rows: 40px 1fr;
  height: 100vh;
  background: var(--bg);
  color: var(--fg);
  font-family: ui-sans-serif, system-ui, sans-serif;
}
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  border-bottom: 1px solid var(--border);
  background: var(--panel);
  font-size: 13px;
}
.brand {
  font-weight: 600;
  letter-spacing: 0.3px;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  margin-left: 8px;
}
.user-name {
  color: var(--fg);
}
.tenant {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 10px;
  background: var(--panel2);
}
.roles {
  font-size: 11px;
  color: var(--accent);
}
.model-hint {
  font-size: 11px;
  color: var(--muted);
  margin-left: 6px;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.btn-models {
  margin-left: auto;
  padding: 4px 12px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--accent);
  cursor: pointer;
  font-size: 12px;
}
.btn-models:hover {
  background: var(--hover);
}
.btn-logout {
  padding: 4px 12px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  cursor: pointer;
  font-size: 12px;
}
.btn-logout:hover {
  background: var(--hover);
}
.model-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: stretch;
  justify-content: center;
  padding: 24px 12px;
  box-sizing: border-box;
}
.model-overlay-inner {
  width: min(960px, 100%);
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  max-height: 100%;
  overflow: hidden;
}
.model-overlay-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
  font-weight: 600;
}
.btn-close {
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  cursor: pointer;
  font-size: 12px;
}
.btn-close:hover {
  background: var(--hover);
}
.body {
  display: grid;
  min-height: 0;
}
.splitter {
  cursor: col-resize;
  background: var(--border);
  position: relative;
  z-index: 1;
  transition: background 0.15s;
}
.splitter:hover,
.splitter:active {
  background: var(--accent);
}
.sidebar {
  border-right: 1px solid var(--border);
  padding: 10px 8px;
  overflow: auto;
  background: var(--panel);
  min-width: 0;
}
.side-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--muted);
}
.side-actions {
  display: flex;
  gap: 10px;
}
.linkish {
  border: none;
  background: none;
  color: var(--accent);
  cursor: pointer;
  font-size: 12px;
  padding: 0;
}
.linkish:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.new-file {
  display: grid;
  gap: 4px;
  padding: 6px 0 10px;
  border-bottom: 1px dashed var(--border);
  margin-bottom: 8px;
}
.new-file input {
  width: 100%;
  box-sizing: border-box;
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  font-size: 12px;
  outline: none;
}
.new-file input:focus {
  border-color: var(--accent);
}
.new-file-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
}
.btn-mini {
  padding: 3px 10px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  font-size: 12px;
  cursor: pointer;
}
.btn-mini.primary {
  border-color: var(--accent);
  background: var(--accent);
  color: #0b0d12;
}
.btn-mini:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.editor-wrap {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.editor-bar {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  background: var(--panel);
}
.path {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--muted);
}
.dot {
  color: #fbbf24;
}
.btn-save {
  padding: 4px 12px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--panel2);
  color: var(--fg);
  cursor: pointer;
  font-size: 12px;
}
.btn-save:hover {
  background: var(--hover);
}
.status {
  flex: 0 0 auto;
  margin: 0;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--muted);
}
.chat-wrap {
  border-left: 1px solid var(--border);
  padding: 10px;
  overflow: auto;
  background: var(--panel);
  min-height: 0;
  min-width: 0;
}
</style>
