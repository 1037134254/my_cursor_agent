<script setup lang="ts">
import * as monaco from "monaco-editor";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from "vue";
import { pathToMonacoLanguage } from "../lib/language";

const rootEl = ref<HTMLDivElement | null>(null);
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null);

const props = withDefaults(
  defineProps<{
    filePath: string | null;
    modelValue: string;
  }>(),
  { filePath: null, modelValue: "" },
);

const emit = defineEmits<{ "update:modelValue": [v: string]; dirty: [] }>();

let configured = false;

function setupMonacoWorkers() {
  if (configured) return;
  configured = true;
  (window as unknown as { MonacoEnvironment?: { getWorker: () => Worker } }).MonacoEnvironment =
    {
      getWorker() {
        return new editorWorker();
      },
    };
}

onMounted(() => {
  setupMonacoWorkers();
  if (!rootEl.value) return;
  const lang = pathToMonacoLanguage(props.filePath ?? "");
  editor.value = monaco.editor.create(rootEl.value, {
    value: props.modelValue,
    language: lang,
    theme: "vs-dark",
    automaticLayout: true,
    fontSize: 14,
    minimap: { enabled: true },
    scrollBeyondLastLine: false,
  });
  editor.value.onDidChangeModelContent(() => {
    emit("update:modelValue", editor.value!.getValue());
    emit("dirty");
  });
});

watch(
  () => props.filePath,
  (p, prev) => {
    const ed = editor.value;
    if (!ed) return;
    const m = ed.getModel();
    if (!m) return;
    const lang = pathToMonacoLanguage(p ?? "");
    monaco.editor.setModelLanguage(m, lang);
    if (p !== prev) {
      ed.setValue(props.modelValue);
    }
  },
);

onBeforeUnmount(() => {
  editor.value?.dispose();
  editor.value = null;
});
</script>

<template>
  <div ref="rootEl" class="monaco-root" />
</template>

<style scoped>
.monaco-root {
  height: 100%;
  min-height: 200px;
}
</style>
