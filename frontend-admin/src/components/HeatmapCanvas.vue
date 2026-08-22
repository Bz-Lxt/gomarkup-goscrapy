<script setup lang="ts">
import { computed } from 'vue'
import type { Snapshot, SnapshotNode } from '@/api/types'

const props = defineProps<{
  snapshot: Snapshot
  imageSrc: string
  selectedId?: number | null
}>()

const emit = defineEmits<{
  select: [node: SnapshotNode]
}>()

const layers = computed(() => {
  return [...(props.snapshot.nodes || [])]
    .filter((n) => n.box && n.box.w > 0 && n.box.h > 0)
    .sort((a, b) => b.box.w * b.box.h - a.box.w * a.box.h)
})

function styleOf(node: SnapshotNode) {
  return {
    left: `${node.box.x}px`,
    top: `${node.box.y}px`,
    width: `${node.box.w}px`,
    height: `${node.box.h}px`,
  }
}
</script>

<template>
  <div class="heatmap-frame">
    <div
      class="heatmap-stage"
      :style="{ width: snapshot.width + 'px', height: snapshot.height + 'px' }"
    >
      <img
        :src="imageSrc"
        alt="页面快照"
        class="heatmap-img"
        :width="snapshot.width"
        :height="snapshot.height"
      />
      <button
        v-for="node in layers"
        :key="node.node_id"
        type="button"
        class="heat-zone"
        :class="{ selected: selectedId === node.node_id }"
        :style="styleOf(node)"
        :title="`${node.tag} · ${node.text || node.node_id}`"
        @click="emit('select', node)"
      />
    </div>
  </div>
</template>

<style scoped>
.heatmap-frame {
  width: 100%;
  max-height: min(72vh, 820px);
  overflow: auto;
  background: #050b10;
  border: 1px solid var(--line);
  border-radius: 10px;
}
.heatmap-stage {
  position: relative;
}
.heatmap-img {
  display: block;
  object-fit: none;
  max-width: none;
}
.heat-zone {
  position: absolute;
  padding: 0;
  margin: 0;
  border: 1px solid rgba(62, 224, 197, 0.18);
  background: rgba(62, 224, 197, 0.04);
  cursor: pointer;
  box-sizing: border-box;
}
.heat-zone:hover {
  border-color: #3ee0c5;
  background: rgba(62, 224, 197, 0.12);
  box-shadow: 0 0 8px rgba(62, 224, 197, 0.45);
  z-index: 3;
}
.heat-zone.selected {
  border-color: #f5b942;
  background: rgba(245, 185, 66, 0.16);
  box-shadow: 0 0 8px rgba(245, 185, 66, 0.5);
  z-index: 4;
}
</style>
