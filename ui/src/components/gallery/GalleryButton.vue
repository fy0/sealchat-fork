<template>
  <n-tooltip trigger="hover">
    <template #trigger>
      <n-button quaternary circle :disabled="!userId" @click="openGallery">
        <template #icon>
          <n-icon :component="ImageOutline" size="18" />
        </template>
      </n-button>
    </template>
    打开画廊
  </n-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { NButton, NIcon, NTooltip } from 'naive-ui';
import { ImageOutline } from '@vicons/ionicons5';
import { useGalleryStore } from '@/stores/gallery';
import { useUserStore } from '@/stores/user';

const gallery = useGalleryStore();
const user = useUserStore();

const userId = computed(() => user.info?.id || '');

async function openGallery() {
  if (!userId.value) return;
  gallery.loadEmojiPreference(userId.value);
  await gallery.openPanel(userId.value);
}
</script>
