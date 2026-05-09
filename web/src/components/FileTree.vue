<script setup lang="ts">
import { computed } from "vue";
import TreeRows, { type TreeNode } from "./TreeRows.vue";

const props = defineProps<{ paths: string[] }>();
const emit = defineEmits<{ select: [path: string] }>();

const nodes = computed(() => buildTree(props.paths));

function buildTree(paths: string[]): TreeNode[] {
  const sorted = [...paths].sort((a, b) => a.localeCompare(b));
  const root: TreeNode = { name: "", children: [] };

  for (const p of sorted) {
    const parts = p.split("/").filter(Boolean);
    let cur = root;
    for (let i = 0; i < parts.length; i++) {
      const seg = parts[i];
      const isLeaf = i === parts.length - 1;
      if (!cur.children) cur.children = [];
      let next = cur.children.find((c) => c.name === seg);
      if (!next) {
        next = {
          name: seg,
          path: isLeaf ? p : undefined,
          children: isLeaf ? undefined : [],
        };
        cur.children.push(next);
      } else if (isLeaf) {
        next.path = p;
      } else if (!next.children) {
        next.children = [];
      }
      cur = next;
    }
  }

  sortNodes(root.children ?? []);
  return root.children ?? [];
}

function sortNodes(list: TreeNode[]) {
  list.sort((a, b) => {
    const fa = a.children?.length ? 0 : 1;
    const fb = b.children?.length ? 0 : 1;
    if (fa !== fb) return fa - fb;
    return a.name.localeCompare(b.name);
  });
  for (const n of list) {
    if (n.children?.length) sortNodes(n.children);
  }
}
</script>

<template>
  <div class="file-tree">
    <TreeRows :nodes="nodes" @select="emit('select', $event)" />
    <p v-if="paths.length === 0" class="empty">workspace 下暂无文件<br />可在编辑器保存新文件创建目录</p>
  </div>
</template>

<style scoped>
.file-tree {
  min-height: 40px;
}
.empty {
  font-size: 12px;
  color: var(--muted);
  line-height: 1.5;
}
</style>
