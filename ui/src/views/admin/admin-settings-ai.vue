<script setup lang="ts">
import { useUtilsStore } from '@/stores/utils'
import type { AIConfig, AIFeatureConfig, AIModelParams, AIProviderConfig } from '@/types'
import { cloneDeep } from 'lodash-es'
import { useMessage } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'

type BuiltinFeatureKey = 'polish' | 'battle_summary'

interface FeatureMeta {
  key: BuiltinFeatureKey
  label: string
  description: string
}

const FEATURE_LIST: FeatureMeta[] = [
  {
    key: 'polish',
    label: '润色',
    description: '用于输入框文本润色，用户侧显示画笔入口。',
  },
  {
    key: 'battle_summary',
    label: '战报总结',
    description: '用于顶部功能区和导出弹窗中的战报总结入口。',
  },
]

const utils = useUtilsStore()
const message = useMessage()

const defaultFeatureConfig = (featureKey: BuiltinFeatureKey): AIFeatureConfig => ({
  enabled: false,
  defaultPrompt: featureKey === 'battle_summary'
    ? '你是跑团战报助手。根据提供内容整理清晰、忠实原意的战报摘要。'
    : '你是中文文本润色助手。保持原意，修正病句，提升流畅度，不要增加无关信息。',
  defaultModel: 'deepseek-v4-flash',
  params: {},
  access: {
    mode: 'all',
    userIds: [],
    worldIds: [],
  },
})

const createDefaultProvider = (): AIProviderConfig => ({
  id: `provider-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
  name: 'DeepSeek',
  enabled: true,
  baseUrl: 'https://api.deepseek.com/v1',
  apiKey: '',
  models: ['deepseek-v4-flash'],
  weight: 1,
})

const defaultConfig = (): AIConfig => ({
  enabled: false,
  routing: { mode: 'round_robin' },
  retry: {
    maxAttempts: 2,
    initialDelayMs: 300,
    maxDelayMs: 3000,
  },
  providers: [
    {
      id: 'deepseek-default',
      name: 'DeepSeek',
      enabled: true,
      baseUrl: 'https://api.deepseek.com/v1',
      apiKey: '',
      models: ['deepseek-v4-flash'],
      weight: 1,
    },
  ],
  features: {
    polish: defaultFeatureConfig('polish'),
    battle_summary: defaultFeatureConfig('battle_summary'),
  },
})

const model = ref<AIConfig>(defaultConfig())
const originalSnapshot = ref('')
const loading = ref(false)
const saving = ref(false)
const testingProviderId = ref('')
const featureEditorVisible = ref(false)
const editingFeatureKey = ref<BuiltinFeatureKey>('polish')
const featureEditorDraft = ref<AIFeatureConfig>(defaultFeatureConfig('polish'))

const snapshotOf = (value: AIConfig) => JSON.stringify(value)
const isModified = computed(() => snapshotOf(model.value) !== originalSnapshot.value)
const currentFeatureMeta = computed(() => FEATURE_LIST.find((item) => item.key === editingFeatureKey.value) || FEATURE_LIST[0])

const normalizeFeatureMap = (features?: Partial<Record<BuiltinFeatureKey, AIFeatureConfig>>): Record<BuiltinFeatureKey, AIFeatureConfig> => ({
  polish: {
    ...defaultFeatureConfig('polish'),
    ...(features?.polish || {}),
    params: { ...defaultFeatureConfig('polish').params, ...(features?.polish?.params || {}) },
    access: { ...defaultFeatureConfig('polish').access, ...(features?.polish?.access || {}) },
  },
  battle_summary: {
    ...defaultFeatureConfig('battle_summary'),
    ...(features?.battle_summary || {}),
    params: { ...defaultFeatureConfig('battle_summary').params, ...(features?.battle_summary?.params || {}) },
    access: { ...defaultFeatureConfig('battle_summary').access, ...(features?.battle_summary?.access || {}) },
  },
})

const normalizeProvider = (provider: AIProviderConfig, index: number): AIProviderConfig => ({
  id: provider.id?.trim() || `provider-${index + 1}`,
  name: provider.name?.trim() || `Provider ${index + 1}`,
  enabled: provider.enabled !== false,
  baseUrl: provider.baseUrl?.trim() || 'https://api.deepseek.com/v1',
  apiKey: provider.apiKey || '',
  models: Array.isArray(provider.models) && provider.models.length ? provider.models : ['deepseek-v4-flash'],
  weight: Number.isFinite(provider.weight) && provider.weight > 0 ? provider.weight : 1,
})

const mergeConfig = (incoming?: Partial<AIConfig>): AIConfig => ({
  ...defaultConfig(),
  ...(incoming || {}),
  routing: { ...defaultConfig().routing, ...(incoming?.routing || {}) },
  retry: { ...defaultConfig().retry, ...(incoming?.retry || {}) },
  providers: Array.isArray(incoming?.providers) && incoming.providers.length
    ? incoming.providers.map((provider, index) => normalizeProvider(provider, index))
    : defaultConfig().providers,
  features: normalizeFeatureMap(incoming?.features as Partial<Record<BuiltinFeatureKey, AIFeatureConfig>> | undefined),
})

const parseCommaList = (value: string): string[] => value
  .split(',')
  .map((item: string) => item.trim())
  .filter((item: string) => item.length > 0)

const updateProviderModels = (provider: AIProviderConfig, value: string) => {
  provider.models = parseCommaList(value)
}

const updateFeatureUserIds = (feature: AIFeatureConfig, value: string) => {
  feature.access.userIds = parseCommaList(value)
}

const updateFeatureWorldIds = (feature: AIFeatureConfig, value: string) => {
  feature.access.worldIds = parseCommaList(value)
}

const load = async () => {
  loading.value = true
  try {
    const resp = await utils.adminAIConfigGet()
    model.value = mergeConfig(resp.data?.config || {})
    originalSnapshot.value = snapshotOf(model.value)
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '加载 AI 配置失败')
  } finally {
    loading.value = false
  }
}

const save = async () => {
  saving.value = true
  try {
    const payload = cloneDeep(model.value)
    const resp = await utils.adminAIConfigUpdate(payload)
    model.value = mergeConfig(resp.data?.config || {})
    originalSnapshot.value = snapshotOf(model.value)
    message.success('AI 配置已保存')
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || '保存 AI 配置失败')
  } finally {
    saving.value = false
  }
}

const testProvider = async (providerId: string) => {
  testingProviderId.value = providerId
  try {
    const provider = model.value.providers.find((item) => item.id === providerId)
    const resp = await utils.adminAIProviderTest({
      providerId,
      model: provider?.models?.[0] || '',
      prompt: '连通性测试',
    })
    message.success(`测试成功：${resp.data?.model || provider?.models?.[0] || 'unknown'}`)
  } catch (error: any) {
    message.error(error?.response?.data?.message || error?.message || 'AI provider 测试失败')
  } finally {
    testingProviderId.value = ''
  }
}

const addProvider = () => {
  model.value.providers.push(createDefaultProvider())
}

const removeProvider = (providerId: string) => {
  if (model.value.providers.length <= 1) {
    message.warning('至少保留一个 provider')
    return
  }
  model.value.providers = model.value.providers.filter((item) => item.id !== providerId)
}

const openFeatureEditor = (featureKey: BuiltinFeatureKey) => {
  editingFeatureKey.value = featureKey
  featureEditorDraft.value = cloneDeep(model.value.features[featureKey] || defaultFeatureConfig(featureKey))
  featureEditorVisible.value = true
}

const closeFeatureEditor = () => {
  featureEditorVisible.value = false
}

const applyFeatureEditor = () => {
  model.value.features[editingFeatureKey.value] = cloneDeep(featureEditorDraft.value)
  featureEditorVisible.value = false
}

const formatParamsSummary = (params: AIModelParams) => {
  const summary: string[] = []
  if (typeof params.temperature === 'number') {
    summary.push(`temperature ${params.temperature}`)
  }
  if (typeof params.topP === 'number') {
    summary.push(`topP ${params.topP}`)
  }
  if (typeof params.maxTokens === 'number' && params.maxTokens > 0) {
    summary.push(`maxTokens ${params.maxTokens}`)
  }
  return summary.length ? summary.join(' / ') : '默认'
}

onMounted(load)

defineExpose({
  save,
  isModified: () => isModified.value,
})
</script>

<template>
  <div class="admin-settings-scroll overflow-y-auto pr-2" style="max-height: 61vh; margin-top: 0;">
    <n-spin :show="loading || saving">
      <n-alert type="info" title="平台 AI 能力" class="admin-ai__notice">
        默认提供一组 DeepSeek 配置：`https://api.deepseek.com/v1` + `deepseek-v4-flash`。API Key 留空表示保留旧值或等待管理员填写。
      </n-alert>

      <n-form label-placement="left" label-width="120">
        <n-collapse class="settings-collapse" :default-expanded-names="['general', 'providers', 'features']">
          <n-collapse-item title="总开关与重试" name="general">
            <n-form-item label="启用平台 AI">
              <n-switch v-model:value="model.enabled" />
            </n-form-item>
            <n-form-item label="路由模式">
              <n-tag type="info" size="small">轮询</n-tag>
            </n-form-item>
            <n-form-item label="重试次数">
              <n-input-number v-model:value="model.retry.maxAttempts" :min="1" />
            </n-form-item>
            <n-form-item label="初始延迟(ms)">
              <n-input-number v-model:value="model.retry.initialDelayMs" :min="50" />
            </n-form-item>
            <n-form-item label="最大延迟(ms)">
              <n-input-number v-model:value="model.retry.maxDelayMs" :min="100" />
            </n-form-item>
          </n-collapse-item>

          <n-collapse-item title="API Provider" name="providers">
            <n-form-item label="Provider 列表" feedback="多个 endpoint + key 组合将按轮询方式依次尝试。">
              <div class="admin-ai__provider-toolbar">
                <n-button size="small" tertiary @click="addProvider">新增 Provider</n-button>
              </div>
            </n-form-item>
            <div
              v-for="provider in model.providers"
              :key="provider.id"
              class="admin-ai__provider"
            >
              <n-grid :cols="2" x-gap="16">
                <n-gi>
                  <n-form-item label="ID">
                    <n-input v-model:value="provider.id" />
                  </n-form-item>
                </n-gi>
                <n-gi>
                  <n-form-item label="名称">
                    <n-input v-model:value="provider.name" />
                  </n-form-item>
                </n-gi>
                <n-gi>
                  <n-form-item label="启用">
                    <n-switch v-model:value="provider.enabled" />
                  </n-form-item>
                </n-gi>
                <n-gi>
                  <n-form-item label="权重">
                    <n-input-number v-model:value="provider.weight" :min="1" />
                  </n-form-item>
                </n-gi>
                <n-gi span="2">
                  <n-form-item label="Base URL">
                    <n-input v-model:value="provider.baseUrl" />
                  </n-form-item>
                </n-gi>
                <n-gi span="2">
                  <n-form-item label="API Key" feedback="留空表示保留旧值；后端不会回显已保存密钥。">
                    <n-input v-model:value="provider.apiKey" type="password" show-password-on="click" />
                  </n-form-item>
                </n-gi>
                <n-gi span="2">
                  <n-form-item label="模型列表" feedback="英文逗号分隔，首个模型作为默认测试模型。">
                    <n-input
                      :value="provider.models.join(', ')"
                      @update:value="(value: string) => updateProviderModels(provider, value)"
                    />
                  </n-form-item>
                </n-gi>
              </n-grid>
              <div class="admin-ai__provider-actions">
                <n-button
                  size="small"
                  tertiary
                  :loading="testingProviderId === provider.id"
                  @click="testProvider(provider.id)"
                >
                  测试连通性
                </n-button>
                <n-button size="small" tertiary type="error" @click="removeProvider(provider.id)">
                  删除
                </n-button>
              </div>
            </div>
          </n-collapse-item>

          <n-collapse-item title="功能配置" name="features">
            <div
              v-for="feature in FEATURE_LIST"
              :key="feature.key"
              class="admin-ai__feature"
            >
              <div class="admin-ai__feature-header">
                <div>
                  <h3 class="admin-ai__feature-title">{{ feature.label }}</h3>
                  <p class="admin-ai__feature-desc">{{ feature.description }}</p>
                </div>
                <n-button size="small" tertiary @click="openFeatureEditor(feature.key)">编辑</n-button>
              </div>

              <n-descriptions label-placement="left" :column="1" size="small">
                <n-descriptions-item label="启用">
                  <n-switch v-model:value="model.features[feature.key].enabled" />
                </n-descriptions-item>
                <n-descriptions-item label="默认模型">
                  {{ model.features[feature.key].defaultModel }}
                </n-descriptions-item>
                <n-descriptions-item label="开放范围">
                  {{ model.features[feature.key].access.mode }}
                </n-descriptions-item>
                <n-descriptions-item label="模型参数">
                  {{ formatParamsSummary(model.features[feature.key].params) }}
                </n-descriptions-item>
              </n-descriptions>
            </div>
          </n-collapse-item>
        </n-collapse>
      </n-form>
    </n-spin>

    <n-modal
      v-model:show="featureEditorVisible"
      preset="card"
      class="sc-fluid-modal sc-fluid-modal--xwide"
      :title="`${currentFeatureMeta.label}配置`"
      :auto-focus="false"
    >
      <div class="admin-ai__modal-body">
        <n-grid :cols="2" x-gap="20" y-gap="12" responsive="screen">
          <n-gi>
            <n-form label-placement="top">
              <n-form-item label="启用功能">
                <n-switch v-model:value="featureEditorDraft.enabled" />
              </n-form-item>
              <n-form-item label="默认模型">
                <n-input v-model:value="featureEditorDraft.defaultModel" />
              </n-form-item>
              <n-form-item label="开放范围">
                <n-select
                  v-model:value="featureEditorDraft.access.mode"
                  :options="[
                    { label: '所有用户', value: 'all' },
                    { label: '指定用户', value: 'users' },
                    { label: '指定世界', value: 'worlds' },
                    { label: '用户或世界', value: 'users_or_worlds' },
                  ]"
                />
              </n-form-item>
              <n-form-item label="用户 ID">
                <n-input
                  :value="featureEditorDraft.access.userIds.join(', ')"
                  @update:value="(value: string) => updateFeatureUserIds(featureEditorDraft, value)"
                />
              </n-form-item>
              <n-form-item label="世界 ID">
                <n-input
                  :value="featureEditorDraft.access.worldIds.join(', ')"
                  @update:value="(value: string) => updateFeatureWorldIds(featureEditorDraft, value)"
                />
              </n-form-item>
            </n-form>
          </n-gi>

          <n-gi>
            <n-form label-placement="top">
              <n-form-item label="Temperature">
                <n-input-number v-model:value="featureEditorDraft.params.temperature" :min="0" :max="2" :step="0.1" />
              </n-form-item>
              <n-form-item label="Top P">
                <n-input-number v-model:value="featureEditorDraft.params.topP" :min="0" :max="1" :step="0.1" />
              </n-form-item>
              <n-form-item label="Max Tokens">
                <n-input-number v-model:value="featureEditorDraft.params.maxTokens" :min="0" />
              </n-form-item>
            </n-form>
          </n-gi>

          <n-gi span="2">
            <n-form label-placement="top">
              <n-form-item label="默认 Prompt">
                <n-input
                  v-model:value="featureEditorDraft.defaultPrompt"
                  type="textarea"
                  :rows="10"
                />
              </n-form-item>
            </n-form>
          </n-gi>
        </n-grid>
      </div>

      <template #footer>
        <n-space justify="end">
          <n-button @click="closeFeatureEditor">取消</n-button>
          <n-button type="primary" @click="applyFeatureEditor">应用</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.admin-settings-scroll {
  overflow-x: hidden;
  overflow-y: scroll;
  scrollbar-gutter: stable;
}

.settings-collapse {
  width: 100%;
}

.admin-ai__notice {
  margin-bottom: 16px;
}

.admin-ai__provider-toolbar {
  width: 100%;
  display: flex;
  justify-content: flex-end;
}

.admin-ai__provider,
.admin-ai__feature {
  padding: 12px 0;
  border-bottom: 1px solid rgba(148, 163, 184, 0.16);
}

.admin-ai__provider:last-child,
.admin-ai__feature:last-child {
  border-bottom: 0;
}

.admin-ai__provider-actions,
.admin-ai__feature-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.admin-ai__provider-actions {
  margin-top: 8px;
  justify-content: flex-end;
}

.admin-ai__feature-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.admin-ai__feature-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: rgba(100, 116, 139, 0.92);
}

.admin-ai__modal-body {
  min-width: 0;
}
</style>
