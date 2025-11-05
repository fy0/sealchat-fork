<template>
  <n-upload
    :show-file-list="false"
    multiple
    :max="12"
    accept="image/*"
    :disabled="disabled"
    @change="handleChange"
  >
    <n-upload-dragger>
      <div class="gallery-upload-zone">
        <slot>拖拽图片到此处或点击上传</slot>
      </div>
    </n-upload-dragger>
  </n-upload>
</template>

<script setup lang="ts">
import type { UploadFileInfo } from 'naive-ui';
import { NUpload, NUploadDragger } from 'naive-ui';

const props = defineProps<{ disabled?: boolean }>();
const emit = defineEmits<{ (e: 'select', files: UploadFileInfo[]): void }>();

function handleChange(options: { fileList: UploadFileInfo[] }) {
  emit('select', options.fileList);
}
</script>

<style scoped>
.gallery-upload-zone {
  padding: 24px 16px;
  text-align: center;
  color: var(--text-color-3);
}
</style>
