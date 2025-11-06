<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useMessage } from 'naive-ui'

interface ExportParams {
  format: string
  timeRange: [string, string] | null
  includeOoc: boolean
  includeArchived: boolean
}

interface Props {
  visible: boolean
  channelId?: string
}

interface Emits {
  (e: 'update:visible', visible: boolean): void
  (e: 'export', params: ExportParams): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const loading = ref(false)

const form = reactive<ExportParams>({
  format: 'txt',
  timeRange: null,
  includeOoc: true,
  includeArchived: false,
})

const formatOptions = [
  { label: '纯文本 (.txt)', value: 'txt' },
  { label: 'HTML (.html)', value: 'html' },
  { label: 'JSON (.json)', value: 'json' },
]

const handleExport = async () => {
  if (!props.channelId) {
    message.error('未选择频道')
    return
  }

  loading.value = true
  try {
    emit('export', { ...form })
    message.info('导出请求已提交，功能开发中...')
  } catch (error) {
    message.error('导出失败')
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('update:visible', false)
  // 重置表单
  form.format = 'txt'
  form.timeRange = null
  form.includeOoc = true
  form.includeArchived = false
}

const shortcuts = {
  '最近7天': () => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - 7)
    return [start.getTime(), end.getTime()]
  },
  '最近30天': () => {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - 30)
    return [start.getTime(), end.getTime()]
  },
  '最近3个月': () => {
    const end = new Date()
    const start = new Date()
    start.setMonth(start.getMonth() - 3)
    return [start.getTime(), end.getTime()]
  },
}
</script>

<template>
  <n-modal
    :show="visible"
    @update:show="emit('update:visible', $event)"
    preset="card"
    title="导出聊天记录"
    class="export-dialog"
    :auto-focus="false"
  >
    <div class="export-notice">
      <n-alert type="info" :show-icon="false">
        <template #header>
          <n-icon component="InfoCircleOutlined" />
          功能开发中
        </template>
        当前为测试接口，仅用于前端联调，不会生成实际文件。
      </n-alert>
    </div>

    <n-form :model="form" label-width="100px" label-placement="left">
      <n-form-item label="导出格式">
        <n-select
          v-model:value="form.format"
          :options="formatOptions"
          placeholder="选择导出格式"
        />
      </n-form-item>

      <n-form-item label="时间范围">
        <n-date-picker
          v-model:value="form.timeRange"
          type="datetimerange"
          clearable
          :shortcuts="shortcuts"
          format="yyyy-MM-dd HH:mm:ss"
          placeholder="选择时间范围，留空表示全部"
          style="width: 100%"
        />
      </n-form-item>

      <n-form-item label="包含内容">
        <n-space vertical>
          <n-checkbox v-model:checked="form.includeOoc">
            包含场外 (OOC) 消息
          </n-checkbox>
          <n-checkbox v-model:checked="form.includeArchived">
            包含已归档消息
          </n-checkbox>
        </n-space>
      </n-form-item>
    </n-form>

    <template #footer>
      <n-space justify="end">
        <n-button @click="handleClose">取消</n-button>
        <n-button
          type="primary"
          :loading="loading"
          @click="handleExport"
        >
          开始导出
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style lang="scss" scoped>
.export-dialog {
  width: 500px;
  max-width: 90vw;
}

.export-notice {
  margin-bottom: 1.5rem;
}

:deep(.n-alert) {
  .n-alert__header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
}
</style>