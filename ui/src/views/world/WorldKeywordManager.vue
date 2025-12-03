<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'
import { useWorldGlossaryStore } from '@/stores/worldGlossary'
import { useChatStore } from '@/stores/chat'
import { useMessage } from 'naive-ui'
import { triggerBlobDownload } from '@/utils/download'
import type { WorldKeywordItem } from '@/models/worldGlossary'

const glossary = useWorldGlossaryStore()
const chat = useChatStore()
const message = useMessage()

const drawerVisible = computed({
  get: () => glossary.managerVisible,
  set: (value: boolean) => glossary.setManagerVisible(value),
})

const currentWorldId = computed(() => chat.currentWorldId)
const keywordItems = computed(() => {
  const worldId = currentWorldId.value
  if (!worldId) return []
  const page = glossary.pages[worldId]
  return page?.items || []
})
const filterValue = computed({
  get: () => glossary.searchQuery,
  set: (value: string) => glossary.setSearchQuery(value),
})

const filteredKeywords = computed(() => {
  const q = filterValue.value.trim().toLowerCase()
  if (!q) return keywordItems.value
  return keywordItems.value.filter((item) => {
    const haystack = [item.keyword, ...(item.aliases || []), item.description || ''].join(' ').toLowerCase()
    return haystack.includes(q)
  })
})

const worldDetail = computed(() => {
  const worldId = currentWorldId.value
  if (!worldId) return null
  return chat.worldDetailMap[worldId] || null
})

const canEdit = computed(() => {
  const detail = worldDetail.value
  const role = detail?.memberRole
  return role === 'owner' || role === 'admin'
})

const formModel = reactive({
  keyword: '',
  aliases: '',
  matchMode: 'plain' as 'plain' | 'regex',
  description: '',
  display: 'standard' as 'standard' | 'minimal',
  isEnabled: true,
})

const importText = reactive({ content: '' })

function resetForm() {
  formModel.keyword = ''
  formModel.aliases = ''
  formModel.matchMode = 'plain'
  formModel.description = ''
  formModel.display = 'standard'
  formModel.isEnabled = true
}

function openCreate() {
  const worldId = currentWorldId.value
  if (!worldId) return
  resetForm()
  glossary.openEditor(worldId)
}

function openEdit(item: any) {
  const worldId = currentWorldId.value
  if (!worldId) return
  formModel.keyword = item.keyword
  formModel.aliases = (item.aliases || []).join(', ')
  formModel.matchMode = item.matchMode
  formModel.description = item.description
  formModel.display = item.display
  formModel.isEnabled = item.isEnabled
  glossary.openEditor(worldId, item)
}

async function submitEditor() {
  const worldId = glossary.editorState.worldId || currentWorldId.value
  if (!worldId) return
  const payload = {
    keyword: formModel.keyword.trim(),
    aliases: formModel.aliases
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean),
    matchMode: formModel.matchMode,
    description: formModel.description?.trim(),
    display: formModel.display,
    isEnabled: formModel.isEnabled,
  }
  try {
    if (glossary.editorState.keyword) {
      await glossary.editKeyword(worldId, glossary.editorState.keyword.id, payload)
      message.success('已更新术语')
    } else {
      await glossary.createKeyword(worldId, payload)
      message.success('已创建术语')
    }
    glossary.closeEditor()
  } catch (error: any) {
    message.error(error?.message || '保存失败')
  }
}

async function handleDelete(itemId: string) {
  const worldId = currentWorldId.value
  if (!worldId) return
  await glossary.removeKeyword(worldId, itemId)
  message.success('已删除')
}

async function handleToggle(item: WorldKeywordItem) {
  const worldId = currentWorldId.value
  if (!worldId) return
  await glossary.editKeyword(worldId, item.id, {
    keyword: item.keyword,
    aliases: item.aliases,
    matchMode: item.matchMode,
    description: item.description,
    display: item.display,
    isEnabled: !item.isEnabled,
  })
}

async function handleExport() {
  const worldId = currentWorldId.value
  if (!worldId) return
  const items = await glossary.exportKeywords(worldId)
  const blob = new Blob([JSON.stringify(items, null, 2)], { type: 'application/json' })
  const worldName = chat.worldMap[worldId]?.name || 'world'
  triggerBlobDownload(blob, `${worldName}-keywords.json`)
  message.success('已导出词库')
}

async function handleImport(replace = false) {
  const worldId = glossary.importState.worldId || currentWorldId.value
  if (!worldId) return
  try {
    const parsed = JSON.parse(importText.content || '[]')
    if (!Array.isArray(parsed)) {
      message.error('JSON 格式错误，需要数组')
      return
    }
    await glossary.importKeywords(worldId, parsed, replace)
    message.success('导入完成')
  } catch (error: any) {
    message.error(error?.message || '导入失败')
  }
}

watch(
  () => drawerVisible.value,
  (visible) => {
    if (visible) {
      if (currentWorldId.value) {
        glossary.ensureKeywords(currentWorldId.value, { force: true })
        chat.worldDetail(currentWorldId.value)
      }
    }
  },
)

watch(
  () => currentWorldId.value,
  (worldId) => {
    if (worldId && drawerVisible.value) {
      glossary.ensureKeywords(worldId, { force: true })
    }
  },
)

onMounted(() => {
  if (currentWorldId.value) {
    glossary.ensureKeywords(currentWorldId.value)
  }
})

watch(
  () => ({
    visible: glossary.editorState.visible,
    keyword: glossary.editorState.keyword,
    prefill: glossary.editorState.prefill,
  }),
  (state) => {
    if (!state.visible) {
      resetForm()
      return
    }
    if (state.keyword) {
      const keyword = state.keyword
      formModel.keyword = keyword.keyword
      formModel.aliases = (keyword.aliases || []).join(', ')
      formModel.matchMode = keyword.matchMode
      formModel.description = keyword.description
      formModel.display = keyword.display
      formModel.isEnabled = keyword.isEnabled
    } else {
      resetForm()
      if (state.prefill) {
        formModel.keyword = state.prefill
        glossary.editorState.prefill = null
      }
    }
  },
)

const isEditing = computed(() => Boolean(glossary.editorState.keyword))
const editorVisible = computed({
  get: () => glossary.editorState.visible,
  set: (value: boolean) => {
    if (!value) glossary.closeEditor()
  },
})
const importVisible = computed({
  get: () => glossary.importState.visible,
  set: (value: boolean) => {
    if (!value) glossary.closeImport()
  },
})

watch(
  () => importVisible.value,
  (visible) => {
    if (!visible) {
      importText.content = ''
    }
  },
)
</script>

<template>
  <n-drawer v-model:show="drawerVisible" :width="520" placement="right" :mask-closable="true">
    <template #header>
      <div class="flex items-center justify-between">
        <span>术语词库</span>
        <div class="space-x-2">
          <n-button size="tiny" @click="currentWorldId && glossary.ensureKeywords(currentWorldId, { force: true })">刷新</n-button>
          <n-button size="tiny" tertiary :disabled="!canEdit || !currentWorldId" @click="openCreate">新增</n-button>
          <n-button size="tiny" tertiary :disabled="!canEdit || !currentWorldId" @click="glossary.openImport(currentWorldId || '')">导入</n-button>
          <n-button size="tiny" tertiary @click="handleExport">导出</n-button>
        </div>
      </div>
    </template>
    <div class="space-y-4">
      <n-input
        v-model:value="filterValue"
        placeholder="搜索关键词或描述"
        clearable
        size="small"
      />
      <n-alert v-if="!canEdit" type="info" title="仅可查看">
        该世界仅管理员可编辑术语，您当前没有编辑权限。
      </n-alert>
      <n-spin :show="glossary.loadingMap[currentWorldId || '']">
        <n-table :single-line="false" size="small">
          <thead>
            <tr>
              <th>关键词</th>
              <th>匹配</th>
              <th>显示</th>
              <th>状态</th>
              <th style="width: 120px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredKeywords" :key="item.id">
              <td>
                <div class="font-medium">{{ item.keyword }}</div>
                <div class="text-xs text-gray-500" v-if="item.aliases?.length">别名：{{ item.aliases.join(', ') }}</div>
                <div class="text-xs text-gray-500" v-if="item.description">{{ item.description }}</div>
              </td>
              <td>{{ item.matchMode === 'regex' ? '正则' : '文本' }}</td>
              <td>{{ item.display === 'minimal' ? '极简下划线' : '标准' }}</td>
              <td>
                <n-tag size="small" :type="item.isEnabled ? 'success' : 'default'">
                  {{ item.isEnabled ? '启用' : '关闭' }}
                </n-tag>
              </td>
              <td>
                <n-space size="small">
                  <n-button size="tiny" text :disabled="!canEdit" @click="openEdit(item)">编辑</n-button>
                  <n-button size="tiny" text :disabled="!canEdit" @click="handleToggle(item)">
                    {{ item.isEnabled ? '停用' : '启用' }}
                  </n-button>
                  <n-popconfirm v-if="canEdit" @positive-click="handleDelete(item.id)">
                    <template #trigger>
                      <n-button size="tiny" text type="error">删除</n-button>
                    </template>
                    确认删除该术语？
                  </n-popconfirm>
                </n-space>
              </td>
            </tr>
            <tr v-if="!filteredKeywords.length">
              <td colspan="5" class="text-center text-gray-400">暂无数据</td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>
  </n-drawer>

  <n-modal v-model:show="editorVisible" preset="card" :title="isEditing ? '编辑术语' : '新增术语'" style="width: 520px">
    <n-form label-placement="top">
      <n-form-item label="关键词" required>
        <n-input v-model:value="formModel.keyword" placeholder="必填" />
      </n-form-item>
      <n-form-item label="别名（用逗号分隔）">
        <n-input v-model:value="formModel.aliases" placeholder="可选" />
      </n-form-item>
      <n-form-item label="匹配模式">
        <n-select
          v-model:value="formModel.matchMode"
          :options="[
            { label: '文本匹配', value: 'plain' },
            { label: '正则表达式', value: 'regex' },
          ]"
        />
      </n-form-item>
      <n-form-item label="描述">
        <n-input type="textarea" v-model:value="formModel.description" :autosize="{ minRows: 3, maxRows: 6 }" />
      </n-form-item>
      <n-form-item label="显示样式">
        <n-select
          v-model:value="formModel.display"
          :options="[
            { label: '标准高亮（背景+下划线）', value: 'standard' },
            { label: '极简下划线', value: 'minimal' },
          ]"
        />
      </n-form-item>
      <n-form-item>
        <n-switch v-model:value="formModel.isEnabled">
          <template #checked>已启用</template>
          <template #unchecked>已停用</template>
        </n-switch>
      </n-form-item>
    </n-form>
    <template #action>
      <n-space>
        <n-button @click="glossary.closeEditor()">取消</n-button>
        <n-button type="primary" @click="submitEditor">保存</n-button>
      </n-space>
    </template>
  </n-modal>

  <n-modal v-model:show="importVisible" preset="card" title="导入术语" style="width: 520px">
    <n-alert type="info" class="mb-3">
      请输入 JSON 数组，每个元素包含 `keyword`、`aliases`、`matchMode`、`description` 等字段。
    </n-alert>
    <n-input
      v-model:value="importText.content"
      type="textarea"
      :autosize="{ minRows: 8 }"
      placeholder='[\n  { "keyword": "阿瓦隆", "description": "古老之城" }\n]'
    />
    <template #action>
      <n-space>
        <n-button text @click="glossary.closeImport()">取消</n-button>
        <n-button :loading="glossary.importState.processing" @click="handleImport(false)">追加</n-button>
        <n-button type="primary" :loading="glossary.importState.processing" @click="handleImport(true)">替换</n-button>
      </n-space>
    </template>
    <div v-if="glossary.importState.lastStats" class="text-xs text-gray-500 mt-2">
      导入结果：新增 {{ glossary.importState.lastStats.created }}，更新 {{ glossary.importState.lastStats.updated }}，跳过 {{ glossary.importState.lastStats.skipped }}
    </div>
  </n-modal>
</template>
