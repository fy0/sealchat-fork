<script setup lang="ts">
import { reactive, watch, computed } from 'vue'
import type { DisplaySettings } from '@/stores/display'

interface Props {
  visible: boolean
  settings: DisplaySettings
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'save', value: DisplaySettings): void
}>()

const draft = reactive<DisplaySettings>({
  layout: 'bubble',
  palette: 'day',
  showAvatar: true,
  mergeNeighbors: true,
  maxExportMessages: 5000,
  maxExportConcurrency: 2,
})

watch(
  () => props.settings,
  (value) => {
    if (!value) return
    draft.layout = value.layout
    draft.palette = value.palette
    draft.showAvatar = value.showAvatar
    draft.mergeNeighbors = value.mergeNeighbors
    draft.maxExportMessages = value.maxExportMessages
    draft.maxExportConcurrency = value.maxExportConcurrency
  },
  { deep: true, immediate: true },
)

const previewClasses = computed(() => [
  'display-preview',
  `display-preview--${draft.palette}`,
  `display-preview--${draft.layout}`,
])

const handleClose = () => emit('update:visible', false)
const handleConfirm = () => emit('save', { ...draft })
</script>

<template>
  <n-modal
    preset="card"
    :show="props.visible"
    title="显示模式"
    :style="{ width: '520px' }"
    @update:show="emit('update:visible', $event)"
  >
    <div class="display-settings">
      <section class="display-settings__section">
        <header>
          <div>
            <p class="section-title">版式</p>
            <p class="section-desc">气泡模式强调对话气泡，紧凑模式更接近论坛流</p>
          </div>
        </header>
        <n-radio-group v-model:value="draft.layout" size="large">
          <n-radio-button value="bubble">气泡模式</n-radio-button>
          <n-radio-button value="compact">紧凑模式</n-radio-button>
        </n-radio-group>
      </section>

      <section class="display-settings__section">
        <header>
          <div>
            <p class="section-title">主题</p>
            <p class="section-desc">在日间/夜间之间切换沉浸背景</p>
          </div>
        </header>
        <n-radio-group v-model:value="draft.palette" size="large">
          <n-radio-button value="day">日间模式</n-radio-button>
          <n-radio-button value="night">夜间模式</n-radio-button>
        </n-radio-group>
      </section>

      <section class="display-settings__section">
        <header>
          <div>
            <p class="section-title">头像显示</p>
            <p class="section-desc">隐藏头像可获得更紧凑的布局</p>
          </div>
        </header>
        <n-switch v-model:value="draft.showAvatar">
          <template #checked>显示头像</template>
          <template #unchecked>隐藏头像</template>
        </n-switch>
      </section>

      <section class="display-settings__section">
        <header>
          <div>
            <p class="section-title">合并连续消息</p>
            <p class="section-desc">相邻同角色消息视作一段，拖动可拆分</p>
          </div>
        </header>
        <n-switch v-model:value="draft.mergeNeighbors">
          <template #checked>已启用</template>
          <template #unchecked>已关闭</template>
        </n-switch>
      </section>

      <section class="display-settings__section">
        <header class="preview-header">
          <div>
            <p class="section-title">实时预览</p>
            <p class="section-desc">预览不同主题下的消息背景</p>
          </div>
        </header>
        <div :class="previewClasses">
          <div class="preview-card">
            <div class="preview-avatar" />
            <div>
              <p class="preview-name">晨星角色 · 场内</p>
              <p class="preview-body">采用 {{ draft.layout === 'bubble' ? '气泡' : '紧凑' }} 模式展示。</p>
            </div>
          </div>
          <div class="preview-card preview-card--ooc">
            <div class="preview-avatar" />
            <div>
              <p class="preview-name">旁白 · 场外</p>
              <p class="preview-body">日夜模式在此同步变化。</p>
            </div>
          </div>
          <div class="preview-card preview-card--preview">
            <div>
              <p class="preview-name">实时预览</p>
              <p class="preview-body">无气泡，使用密排圆点背景。</p>
            </div>
          </div>
        </div>
      </section>

      <n-collapse class="display-settings__section" default-expanded-value="[]">
        <n-collapse-item title="更多控件（敬请期待）" name="more">
          <p class="section-desc">
            将在后续版本开放行距、字号、输入区布局等高级能力。
          </p>
        </n-collapse-item>
      </n-collapse>

      <n-space justify="space-between" align="center" class="display-settings__footer">
        <n-button quaternary size="small" @click="handleClose">取消</n-button>
        <n-button type="primary" size="small" @click="handleConfirm">应用设置</n-button>
      </n-space>
    </div>
  </n-modal>
</template>

<style scoped lang="scss">
:deep(.n-card) {
  background-color: var(--sc-bg-elevated);
  border: 1px solid var(--sc-border-strong);
  color: var(--sc-text-primary);
}

.display-settings {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  color: var(--sc-text-primary);
}

.display-settings__section header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.45rem;
}

.section-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--sc-text-primary);
}

.section-desc {
  font-size: 0.8rem;
  color: var(--sc-text-secondary);
  margin-top: 0.15rem;
}

.display-preview {
  border-radius: 0.9rem;
  padding: 0.9rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  border: 1px solid var(--sc-border-mute);
  background: linear-gradient(135deg, var(--sc-bg-surface), var(--sc-bg-elevated));
}

.display-preview--night {
  background: linear-gradient(135deg, var(--sc-bg-header), var(--sc-bg-elevated));
  border-color: var(--sc-border-strong);
}

.display-preview .preview-card {
  display: flex;
  gap: 0.75rem;
  padding: 0.65rem 0.75rem;
  border-radius: var(--preview-radius, 1rem);
  background-color: var(--sc-bg-surface);
  border: 1px solid var(--sc-border-mute);
}

.display-preview--night .preview-card {
  background-color: var(--sc-bg-input);
  color: var(--sc-text-primary);
}

.display-preview--night .preview-card--ooc {
  background-color: var(--chat-ooc-bg);
}

.display-preview--night .preview-card--preview {
  background-image: radial-gradient(var(--chat-preview-dot) 1px, transparent 1px);
  background-color: var(--chat-preview-bg);
  background-size: 6px 6px;
}

.display-preview--night .preview-name {
  color: var(--sc-text-primary);
}

.display-preview--night .preview-body {
  color: var(--sc-text-secondary);
}

.preview-card--ooc {
  background-color: var(--chat-ooc-bg);
}

.preview-card--preview {
  flex-direction: column;
  background-color: var(--chat-preview-bg);
  background-image: radial-gradient(var(--chat-preview-dot) 1px, transparent 1px);
  background-size: 6px 6px;
}

.preview-avatar {
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 0.75rem;
  background: linear-gradient(135deg, #f87171, #fbbf24);
  border: 1px solid var(--sc-border-mute);
}

.preview-name {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--sc-text-primary);
}

.preview-body {
  font-size: 0.78rem;
  color: var(--sc-text-secondary);
}

.display-preview--compact {
  --preview-radius: 0.75rem;
}

.display-settings__footer {
  margin-top: 0.5rem;
}
</style>
