<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import { useChatStore } from '@/stores/chat';
import { useMessage } from 'naive-ui';

const props = defineProps<{ worldId: string }>();
const chat = useChatStore();
const message = useMessage();
const inviteMap = ref<Record<string, any>>({});
const loading = ref(false);

const loadInvites = async () => {
  if (!props.worldId) return;
  loading.value = true;
  try {
    const resp = await chat.loadWorldSections(props.worldId, ['invites']);
    const list = Array.isArray(resp.invites) ? resp.invites : [];
    const map: Record<string, any> = {};
    for (const item of list) {
      const role = item.role || 'member';
      if (!map[role]) {
        map[role] = item;
      }
    }
    inviteMap.value = map;
  } catch (e) {
    message.error('加载邀请失败');
  } finally {
    loading.value = false;
  }
};

const showCreateModal = ref(false);
const inviteForm = ref({ ttlMinutes: 0, maxUse: 0, memo: '', role: 'member' });

const inviteCards = [
  { role: 'member', title: '成员邀请', desc: '加入后可在频道发言' },
  { role: 'spectator', title: '旁观邀请', desc: '加入后可查看所有频道但不可发言' },
];

const inviteByRole = computed(() => inviteMap.value);

const resetForm = (role: string = 'member') => {
  inviteForm.value = { ttlMinutes: 0, maxUse: 0, memo: '', role };
};

const openCreateModal = (role: string) => {
  resetForm(role);
  showCreateModal.value = true;
};

const saveInvite = async () => {
  if (!props.worldId) return;
  try {
    const ttl = Math.max(0, Number(inviteForm.value.ttlMinutes) || 0);
    const maxUse = Math.max(0, Number(inviteForm.value.maxUse) || 0);
    const payload: any = {
      ttlMinutes: ttl,
      maxUse: maxUse,
      memo: inviteForm.value.memo?.trim() || undefined,
      role: inviteForm.value.role,
    };
    const resp = await chat.createWorldInvite(props.worldId, payload);
    inviteMap.value = {
      ...inviteMap.value,
      [resp.invite?.role || 'member']: resp.invite,
    };
    showCreateModal.value = false;
    message.success('已创建邀请');
    await loadInvites();
  } catch (e: any) {
    message.error(e?.response?.data?.message || '创建邀请失败');
  }
};

const copySlug = async (slug: string) => {
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(slug);
    } else {
      const textarea = document.createElement('textarea');
      textarea.value = slug;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
    }
    message.success('已复制邀请码');
  } catch (e) {
    message.error('复制失败，请手动选择后复制');
  }
};

const buildInviteLink = (slug: string) => {
  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  return `${origin}/#/invite/${slug}`;
};

const latestInviteByRole = (role: string) => inviteByRole.value[role] || null;

onMounted(loadInvites);
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <h3 class="font-bold">邀请链接</h3>
      <n-button size="small" quaternary @click="() => openCreateModal('member')">创建成员邀请</n-button>
    </div>
    <n-spin :show="loading">
      <div class="invite-grid">
        <div v-for="card in inviteCards" :key="card.role" class="invite-card">
          <div class="invite-card-header">
            <div>
              <div class="invite-card-title">{{ card.title }}</div>
              <div class="invite-card-desc">{{ card.desc }}</div>
            </div>
            <n-button size="tiny" type="primary" @click="openCreateModal(card.role)">创建</n-button>
          </div>
          <div v-if="latestInviteByRole(card.role)" class="invite-card-body">
            <n-input readonly size="small" :value="buildInviteLink(latestInviteByRole(card.role).slug)" />
            <div class="invite-meta">邀请码：{{ latestInviteByRole(card.role).slug }}</div>
            <div class="invite-meta">
              使用 {{ latestInviteByRole(card.role).usedCount }} / {{ latestInviteByRole(card.role).maxUse || '∞' }}
            </div>
            <n-button block size="tiny" secondary @click="copySlug(buildInviteLink(latestInviteByRole(card.role).slug))">
              复制邀请链接
            </n-button>
          </div>
          <n-empty v-else size="small" description="暂无有效邀请" />
        </div>
      </div>
    </n-spin>
    <n-modal v-model:show="showCreateModal" preset="dialog" title="创建世界邀请" style="max-width:520px">
      <n-form label-placement="left" label-width="96">
        <n-form-item label="邀请身份">
          <n-radio-group v-model:value="inviteForm.role" size="small">
            <n-space>
              <n-radio-button value="member">成员</n-radio-button>
              <n-radio-button value="spectator">旁观者</n-radio-button>
            </n-space>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="有效期(分钟)">
          <n-space>
            <n-input-number v-model:value="inviteForm.ttlMinutes" :min="0" :step="30" placeholder="0 表示永久" />
            <n-radio-group v-model:value="inviteForm.ttlMinutes" size="small">
              <n-space>
                <n-radio-button :value="0">永久</n-radio-button>
                <n-radio-button :value="30">30 分钟</n-radio-button>
                <n-radio-button :value="60">1 小时</n-radio-button>
                <n-radio-button :value="60 * 24">1 天</n-radio-button>
              </n-space>
            </n-radio-group>
          </n-space>
        </n-form-item>
        <n-form-item label="可用次数">
          <n-space>
            <n-input-number v-model:value="inviteForm.maxUse" :min="0" :step="1" placeholder="0 表示无限" />
            <n-radio-group v-model:value="inviteForm.maxUse" size="small">
              <n-space>
                <n-radio-button :value="0">无限</n-radio-button>
                <n-radio-button :value="1">1 次</n-radio-button>
                <n-radio-button :value="5">5 次</n-radio-button>
                <n-radio-button :value="10">10 次</n-radio-button>
              </n-space>
            </n-radio-group>
          </n-space>
        </n-form-item>
        <n-form-item label="备注">
          <n-input v-model:value="inviteForm.memo" type="textarea" autosize placeholder="可选，方便区分用途" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-space>
          <n-button quaternary @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" @click="saveInvite">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.invite-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}

.invite-card {
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.invite-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.invite-card-title {
  font-weight: 600;
}

.invite-card-desc {
  font-size: 12px;
  color: #94a3b8;
}

.invite-card-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.invite-meta {
  font-size: 12px;
  color: #94a3b8;
}
</style>
