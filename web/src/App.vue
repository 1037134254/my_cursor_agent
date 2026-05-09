<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import ChatPanel from "./components/ChatPanel.vue";
import EditorPane from "./components/EditorPane.vue";
import FileTree from "./components/FileTree.vue";
import { fetchWorkspaceFile, fetchWorkspaceFiles, putWorkspaceFile } from "./composables/useWorkspace";

const paths = ref<string[]>([]);
const currentPath = ref<string | null>(null);
const editorText = ref("");
const dirty = ref(false);
const status = ref("");
const loadingTree = ref(false);

async function refreshTree() {
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

onMounted(() => {
  window.addEventListener("keydown", onKeyDown);
  void refreshTree();
});

onUnmounted(() => {
  window.removeEventListener("keydown", onKeyDown);
});
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="side-head">
        <span>文件</span>
        <button type="button" class="linkish" :disabled="loadingTree" @click="refreshTree">刷新</button>
      </div>
      <FileTree :paths="paths" @select="openFile" />
    </aside>

    <main class="editor-wrap">
      <header class="editor-bar">
        <span class="path">{{ currentPath || "未打开文件" }}</span>
        <span v-if="dirty" class="dot">●</span>
        <button type="button" class="btn-save" @click="saveCurrent">保存</button>
      </header>
      <EditorPane :file-path="currentPath" :model-value="editorText" @update:model-value="editorText = $event" @dirty="onEditorDirty" />
      <p v-if="status" class="status">{{ status }}</p>
    </main>

    <aside class="chat-wrap">
      <ChatPanel />
    </aside>
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
  grid-template-columns: 240px minmax(0, 1fr) 320px;
  height: 100vh;
  background: var(--bg);
  color: var(--fg);
  font-family: ui-sans-serif, system-ui, sans-serif;
}
.sidebar {
  border-right: 1px solid var(--border);
  padding: 10px 8px;
  overflow: auto;
  background: var(--panel);
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
}
</style>
