<template>
  <div class="gallery-collection-tree">
    <div class="gallery-collection-tree__header">
      <slot name="header">分类</slot>
    </div>
    <div class="gallery-collection-tree__list">
      <div
        v-for="collection in collections"
        :key="collection.id"
        class="gallery-collection-tree__item-wrapper"
      >
        <n-button
          text
          block
          class="gallery-collection-tree__item"
          :type="collection.id === activeId ? 'primary' : 'default'"
          @click="$emit('select', collection.id)"
        >
          <span class="gallery-collection-tree__name">{{ collection.name }}</span>
          <span class="gallery-collection-tree__meta" v-if="collection.quotaUsed">
            {{ formatSize(collection.quotaUsed) }}
          </span>
        </n-button>
        <n-dropdown
          v-if="collection.id === activeId"
          trigger="click"
          :options="contextMenuOptions"
          @select="(key) => $emit('context-action', key, collection)"
        >
          <n-button text size="tiny" class="gallery-collection-tree__menu">⋮</n-button>
        </n-dropdown>
      </div>
    </div>
    <div class="gallery-collection-tree__actions">
      <slot name="actions"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton, NDropdown } from 'naive-ui';
import type { GalleryCollection } from '@/types';

const props = defineProps<{ collections: GalleryCollection[]; activeId: string | null }>();

const contextMenuOptions = [
  { label: '重命名', key: 'rename' },
  { label: '删除', key: 'delete' }
];

function formatSize(size: number) {
  if (!size) return '';
  if (size < 1024) return `${size}B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)}KB`;
  return `${(size / (1024 * 1024)).toFixed(1)}MB`;
}
</script>

<style scoped>
.gallery-collection-tree {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gallery-collection-tree__item-wrapper {
  display: flex;
  align-items: center;
  gap: 4px;
}

.gallery-collection-tree__item-wrapper .gallery-collection-tree__item {
  flex: 1;
}

.gallery-collection-tree__menu {
  opacity: 0.6;
}

.gallery-collection-tree__menu:hover {
  opacity: 1;
}

.gallery-collection-tree__list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 320px;
  overflow-y: auto;
}

.gallery-collection-tree__item {
  justify-content: space-between;
  text-align: left;
}

.gallery-collection-tree__name {
  flex: 1;
}

.gallery-collection-tree__meta {
  font-size: 12px;
  color: var(--text-color-3);
  margin-left: 8px;
}
</style>
