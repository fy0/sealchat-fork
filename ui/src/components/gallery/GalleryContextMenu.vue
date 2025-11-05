<template>
  <n-dropdown
    trigger="manual"
    :x="position.x"
    :y="position.y"
    :options="options"
    :show="visible"
    @select="handleSelect"
    @clickoutside="$emit('update:visible', false)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { NDropdown, type DropdownOption } from 'naive-ui';

const props = defineProps<{
  x: number;
  y: number;
  visible: boolean;
  options: DropdownOption[];
}>();

const emit = defineEmits<{ (e: 'select', key: string): void; (e: 'update:visible', value: boolean): void }>();

const position = computed(() => ({ x: props.x, y: props.y }));

function handleSelect(key: string) {
  emit('select', key);
  emit('update:visible', false);
}
</script>
