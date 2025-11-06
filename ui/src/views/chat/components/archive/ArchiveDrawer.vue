<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'

interface ArchivedMessage {
  id: string
  content: string
  createdAt: string
  archivedAt: string
  archivedBy: string
  sender: {
    name: string
    avatar?: string
  }
}

interface Props {
  visible: boolean
  messages: ArchivedMessage[]
  loading?: boolean
}

interface Emits {
  (e: 'update:visible', visible: boolean): void
  (e: 'unarchive', messageIds: string[]): void
  (e: 'delete', messageIds: string[]): void
  (e: 'refresh'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const selectedIds = ref<string[]>([])

const allSelected = computed({
  get: () => selectedIds.value.length === props.messages.length && props.messages.length > 0,
  set: (value: boolean) => {
    selectedIds.value = value ? props.messages.map(m => m.id) : []
  }
})

const hasSelection = computed(() => selectedIds.value.length > 0)

const formatContent = (content: string) => {
  // 简单的内容预览，移除HTML标签
  return content.replace(/<[^>]*>/g, '').slice(0, 100) + (content.length > 100 ? '...' : '')
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString()
}

const handleUnarchive = async () => {
  if (!hasSelection.value) {
    message.warning('请选择要恢复的消息')
    return
  }

  try {
    emit('unarchive', selectedIds.value)
    selectedIds.value = []
    message.success('消息已恢复')
  } catch (error) {
    message.error('恢复失败')
  }
}

const handleDelete = async () => {
  if (!hasSelection.value) {
    message.warning('请选择要删除的消息')
    return
  }

  // 这里应该有确认对话框
  try {
    emit('delete', selectedIds.value)
    selectedIds.value = []
    message.success('消息已删除')
  } catch (error) {
    message.error('删除失败')
  }
}

const handleClose = () => {
  emit('update:visible', false)
  selectedIds.value = []
}
</script>

<template>
  <n-drawer
    :show="visible"
    @update:show="emit('update:visible', $event)"
    placement="right"
    :width="400"
  >
    <n-drawer-content>
      <template #header>
        <div class="archive-header">
          <span>归档消息管理</span>
          <n-button text @click="emit('refresh')">
            <template #icon>
              <n-icon component="ReloadOutlined" />
            </template>
          </n-button>
        </div>
      </template>

      <div class="archive-content">
        <div v-if="loading" class="archive-loading">
          <n-spin size="large" />
          <p>加载中...</p>
        </div>

        <div v-else-if="messages.length === 0" class="archive-empty">
          <n-empty description="暂无归档消息" />
        </div>

        <div v-else class="archive-list">
          <div class="archive-controls">
            <n-checkbox
              v-model:checked="allSelected"
              :indeterminate="hasSelection && !allSelected"
            >
              全选 ({{ messages.length }})
            </n-checkbox>

            <div class="control-actions">
              <n-button
                size="small"
                :disabled="!hasSelection"
                @click="handleUnarchive"
              >
                恢复选中
              </n-button>
              <n-button
                size="small"
                type="error"
                :disabled="!hasSelection"
                @click="handleDelete"
              >
                删除选中
              </n-button>
            </div>
          </div>

          <div class="message-list">
            <div
              v-for="msg in messages"
              :key="msg.id"
              class="message-item"
              :class="{ 'selected': selectedIds.includes(msg.id) }"
            >
              <n-checkbox
                :checked="selectedIds.includes(msg.id)"
                @update:checked="(checked) => {
                  if (checked) {
                    selectedIds.push(msg.id)
                  } else {
                    const index = selectedIds.indexOf(msg.id)
                    if (index > -1) selectedIds.splice(index, 1)
                  }
                }"
              />

              <div class="message-content">
                <div class="message-header">
                  <span class="sender-name">{{ msg.sender.name }}</span>
                  <span class="message-date">{{ formatDate(msg.createdAt) }}</span>
                </div>
                <div class="message-text">{{ formatContent(msg.content) }}</div>
                <div class="archive-info">
                  <span class="archive-date">归档于 {{ formatDate(msg.archivedAt) }}</span>
                  <span class="archive-by">by {{ msg.archivedBy }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </n-drawer-content>
  </n-drawer>
</template>

<style lang="scss" scoped>
.archive-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.archive-content {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.archive-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  gap: 1rem;
}

.archive-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
}

.archive-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.archive-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: rgba(248, 250, 252, 0.8);
  border-radius: 0.5rem;
  border: 1px solid rgba(148, 163, 184, 0.2);
}

.control-actions {
  display: flex;
  gap: 0.5rem;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.message-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 0.5rem;
  background: #ffffff;
  transition: all 0.2s ease;
}

.message-item:hover {
  border-color: rgba(59, 130, 246, 0.3);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.1);
}

.message-item.selected {
  border-color: rgba(59, 130, 246, 0.5);
  background: rgba(59, 130, 246, 0.05);
}

.message-content {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.sender-name {
  font-weight: 600;
  color: #1f2937;
}

.message-date {
  font-size: 0.75rem;
  color: #6b7280;
}

.message-text {
  color: #374151;
  line-height: 1.5;
  margin-bottom: 0.5rem;
  word-break: break-word;
}

.archive-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: #9ca3af;
}

.archive-date {
  color: #f59e0b;
}

.archive-by {
  color: #6b7280;
}
</style>