<script setup lang="ts">
import TreeRows from "./TreeRows.vue";

export interface TreeNode {
  name: string;
  path?: string;
  children?: TreeNode[];
}

defineProps<{ nodes: TreeNode[] }>();
const emit = defineEmits<{ select: [path: string] }>();
</script>

<template>
  <ul class="tree-ul">
    <li v-for="n in nodes" :key="n.path ?? n.name + '-' + (n.children?.length ?? 0)">
      <template v-if="n.children?.length">
        <details open class="folder-wrap">
          <summary class="folder">{{ n.name }}</summary>
          <TreeRows :nodes="n.children" @select="emit('select', $event)" />
        </details>
      </template>
      <button v-else type="button" class="file" @click="emit('select', n.path!)">
        {{ n.name }}
      </button>
    </li>
  </ul>
</template>

<style scoped>
.tree-ul {
  list-style: none;
  margin: 0;
  padding-left: 0.65rem;
}
.folder-wrap {
  margin: 0.15rem 0;
}
.folder {
  cursor: pointer;
  user-select: none;
  color: var(--muted);
  font-size: 12px;
}
.file {
  display: block;
  width: 100%;
  text-align: left;
  border: none;
  background: transparent;
  color: var(--fg);
  cursor: pointer;
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.file:hover {
  background: var(--hover);
}
</style>
