<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { ModelProfile, TenantBinding } from "../composables/useModels";
import {
  deleteProfile,
  listBindings,
  listProfiles,
  probeProfile,
  putBindings,
  upsertProfile,
} from "../composables/useModels";

const tab = ref<"profiles" | "bindings">("profiles");
const busy = ref(false);
const err = ref("");
const storeKind = ref("");

const profiles = ref<ModelProfile[]>([]);
const editing = ref<Partial<ModelProfile> | null>(null);
const creating = ref(false);

const tenantId = ref("default");
const bindings = ref<TenantBinding[]>([]);
const draftBindings = ref<
  Array<{
    profile_id: string;
    purpose: "chat" | "embed" | "rerank";
    is_default: boolean;
    rpm_override: number;
  }>
>([]);

const emptyForm = (): Partial<ModelProfile> & { id: string } => ({
  id: "",
  name: "",
  purpose: "chat",
  provider: "custom",
  endpoint: "",
  model: "",
  api_key_ref: "",
  timeout_seconds: 180,
  max_retries: 2,
  retry_backoff_ms: 400,
  rpm: 0,
  stream_timeout_seconds: 0,
  enabled: true,
});

const isReadOnly = computed(() => storeKind.value === "env");

function startCreate() {
  creating.value = true;
  editing.value = emptyForm();
}

function edit(p: ModelProfile) {
  creating.value = false;
  editing.value = { ...p };
}

function cancelEdit() {
  editing.value = null;
  creating.value = false;
}

async function reloadProfiles() {
  err.value = "";
  busy.value = true;
  try {
    const r = await listProfiles();
    profiles.value = r.profiles;
    storeKind.value = r.store;
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

async function saveProfile() {
  if (!editing.value?.id?.trim()) {
    err.value = "id 不能为空";
    return;
  }
  err.value = "";
  busy.value = true;
  try {
    await upsertProfile(editing.value as Partial<ModelProfile> & { id: string }, {
      create: creating.value,
    });
    cancelEdit();
    await reloadProfiles();
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

async function remove(id: string) {
  if (!confirm(`确定删除 profile「${id}」？`)) return;
  err.value = "";
  busy.value = true;
  try {
    await deleteProfile(id);
    await reloadProfiles();
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

async function probe(id: string) {
  err.value = "";
  busy.value = true;
  try {
    const reply = await probeProfile(id);
    alert(`probe OK，截断回复：\n${reply.slice(0, 400)}`);
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

async function reloadBindings() {
  err.value = "";
  busy.value = true;
  try {
    bindings.value = await listBindings(tenantId.value.trim() || "default");
    draftBindings.value = bindings.value.map((b) => ({
      profile_id: b.profile_id,
      purpose: b.purpose,
      is_default: b.is_default,
      rpm_override: b.rpm_override,
    }));
    if (draftBindings.value.length === 0) {
      draftBindings.value.push({
        profile_id: "",
        purpose: "chat",
        is_default: true,
        rpm_override: 0,
      });
    }
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

function addBindingRow() {
  draftBindings.value.push({
    profile_id: "",
    purpose: "chat",
    is_default: false,
    rpm_override: 0,
  });
}

function removeBindingRow(i: number) {
  draftBindings.value.splice(i, 1);
  if (draftBindings.value.length === 0) {
    draftBindings.value.push({
      profile_id: "",
      purpose: "chat",
      is_default: true,
      rpm_override: 0,
    });
  }
}

async function saveBindings() {
  err.value = "";
  busy.value = true;
  try {
    const tid = tenantId.value.trim() || "default";
    const cleaned = draftBindings.value.filter((b) => b.profile_id.trim() !== "");
    await putBindings(tid, cleaned);
    await reloadBindings();
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

onMounted(() => {
  void reloadProfiles();
});
</script>

<template>
  <div class="wrap">
    <header class="head">
      <h2>模型注册表</h2>
      <span v-if="storeKind" class="pill">store={{ storeKind }}</span>
      <span v-if="isReadOnly" class="warn">只读 env 模式：请在 .env 配 LLM_*，或连 MySQL 后建 profile</span>
    </header>

    <nav class="tabs">
      <button type="button" :class="{ on: tab === 'profiles' }" @click="tab = 'profiles'">Profiles</button>
      <button type="button" :class="{ on: tab === 'bindings' }" @click="tab = 'bindings'">Bindings</button>
    </nav>

    <p v-if="err" class="err">{{ err }}</p>

    <!-- Profiles -->
    <section v-show="tab === 'profiles'" class="panel">
      <div class="toolbar">
        <button type="button" :disabled="busy || isReadOnly" @click="reloadProfiles">刷新</button>
        <button type="button" :disabled="busy || isReadOnly" @click="startCreate">新建</button>
      </div>

      <table class="tbl">
        <thead>
          <tr>
            <th>id</th>
            <th>name</th>
            <th>purpose</th>
            <th>provider</th>
            <th>model</th>
            <th>key_ref</th>
            <th>ok?</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in profiles" :key="p.id">
            <td class="mono">{{ p.id }}</td>
            <td>{{ p.name }}</td>
            <td>{{ p.purpose }}</td>
            <td>{{ p.provider }}</td>
            <td class="mono sm">{{ p.model }}</td>
            <td class="mono sm">{{ p.api_key_ref || "—" }}</td>
            <td>{{ p.api_key_resolved ? "✓" : "✗" }}</td>
            <td class="acts">
              <button type="button" :disabled="busy" @click="probe(p.id)">probe</button>
              <button type="button" :disabled="busy || isReadOnly" @click="edit(p)">编辑</button>
              <button type="button" :disabled="busy || isReadOnly" @click="remove(p.id)">删</button>
            </td>
          </tr>
        </tbody>
      </table>

      <form v-if="editing" class="form" @submit.prevent="saveProfile">
        <h3>{{ creating ? "新建" : "编辑" }} {{ editing.id }}</h3>
        <label>id<input v-model="editing.id" class="mono" :disabled="!creating" required /></label>
        <label>name<input v-model="editing.name" required /></label>
        <label
          >purpose
          <select v-model="editing.purpose">
            <option value="chat">chat</option>
            <option value="embed">embed</option>
            <option value="rerank">rerank</option>
          </select>
        </label>
        <label
          >provider
          <select v-model="editing.provider">
            <option value="custom">custom</option>
            <option value="qwen">qwen</option>
            <option value="ollama">ollama</option>
            <option value="deepseek">deepseek</option>
          </select>
        </label>
        <label>endpoint<input v-model="editing.endpoint" class="mono wide" required /></label>
        <label>model<input v-model="editing.model" class="mono" required /></label>
        <label>api_key_ref<input v-model="editing.api_key_ref" class="mono" placeholder="如 GLM_KEY_A" /></label>
        <div class="row2">
          <label>timeout<input v-model.number="editing.timeout_seconds" type="number" min="1" /></label>
          <label>retries<input v-model.number="editing.max_retries" type="number" min="0" /></label>
        </div>
        <div class="row2">
          <label>backoff ms<input v-model.number="editing.retry_backoff_ms" type="number" min="0" /></label>
          <label>rpm<input v-model.number="editing.rpm" type="number" min="0" /></label>
        </div>
        <label
          >enabled
          <input v-model="editing.enabled" type="checkbox" />
        </label>
        <div class="form-actions">
          <button type="submit" class="primary" :disabled="busy || isReadOnly">保存</button>
          <button type="button" :disabled="busy" @click="cancelEdit">取消</button>
        </div>
      </form>
    </section>

    <!-- Bindings -->
    <section v-show="tab === 'bindings'" class="panel">
      <div class="toolbar">
        <label class="inline"
          >tenant
          <input v-model="tenantId" class="mono" />
        </label>
        <button type="button" :disabled="busy" @click="reloadBindings">加载</button>
        <button type="button" :disabled="busy || isReadOnly" @click="addBindingRow">+ 行</button>
        <button type="button" class="primary" :disabled="busy || isReadOnly" @click="saveBindings">保存绑定</button>
      </div>

      <table class="tbl">
        <thead>
          <tr>
            <th>profile_id</th>
            <th>purpose</th>
            <th>default</th>
            <th>rpm_override</th>
            <th />
          </tr>
        </thead>
        <tbody>
          <tr v-for="(b, i) in draftBindings" :key="i">
            <td><input v-model="b.profile_id" class="mono wide" /></td>
            <td>
              <select v-model="b.purpose">
                <option value="chat">chat</option>
                <option value="embed">embed</option>
                <option value="rerank">rerank</option>
              </select>
            </td>
            <td><input v-model="b.is_default" type="checkbox" /></td>
            <td><input v-model.number="b.rpm_override" type="number" min="0" /></td>
            <td><button type="button" :disabled="busy || isReadOnly" @click="removeBindingRow(i)">删行</button></td>
          </tr>
        </tbody>
      </table>
      <p class="hint">每个 purpose 仅允许一行 is_default=true；rpm_override=0 表示沿用 profile 默认 RPM。</p>
    </section>
  </div>
</template>

<style scoped>
.wrap {
  padding: 12px 16px 24px;
  color: var(--fg, #e8eaed);
  font-size: 13px;
  max-height: calc(100vh - 44px);
  overflow: auto;
}
.head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}
.head h2 {
  margin: 0;
  font-size: 16px;
}
.pill {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: #1c212b;
  border: 1px solid #2a3140;
}
.warn {
  font-size: 12px;
  color: #fbbf24;
}
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 10px;
}
.tabs button {
  padding: 4px 12px;
  border-radius: 6px;
  border: 1px solid #2a3140;
  background: #161a22;
  color: inherit;
  cursor: pointer;
}
.tabs button.on {
  border-color: #7dd3fc;
  color: #7dd3fc;
}
.err {
  color: #f87171;
  margin: 0 0 8px;
}
.panel {
  border: 1px solid #2a3140;
  border-radius: 8px;
  padding: 10px;
  background: #0f1115;
}
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
}
.toolbar button.primary,
.form-actions button.primary {
  border-color: #7dd3fc;
  color: #7dd3fc;
}
.tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.tbl th,
.tbl td {
  border-bottom: 1px solid #2a3140;
  padding: 6px 4px;
  text-align: left;
  vertical-align: middle;
}
.mono {
  font-family: ui-monospace, monospace;
}
.sm {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.wide {
  width: 100%;
  min-width: 200px;
}
.acts {
  white-space: nowrap;
}
.acts button {
  margin-right: 4px;
  font-size: 11px;
}
.form {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed #2a3140;
  display: grid;
  gap: 8px;
  max-width: 640px;
}
.form h3 {
  margin: 0 0 4px;
  font-size: 14px;
}
.form label {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.form input,
.form select {
  padding: 4px 6px;
  border-radius: 4px;
  border: 1px solid #2a3140;
  background: #161a22;
  color: inherit;
}
.row2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.form-actions {
  display: flex;
  gap: 8px;
}
.hint {
  font-size: 11px;
  color: #8b939e;
  margin-top: 8px;
}
.inline {
  display: flex;
  align-items: center;
  gap: 6px;
}
.inline input {
  width: 140px;
}
</style>
