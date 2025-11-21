<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue';
import { useChatStore } from '@/stores/chat';
import { useDialog, useMessage, NButton, NTag, NSpace } from 'naive-ui';
import dayjs from 'dayjs';

const props = defineProps<{
  worldId: string;
  visible: boolean;
}>();

const visible = defineModel<boolean>('visible', { default: false });
const chat = useChatStore();
const message = useMessage();
const dialog = useDialog();

interface MemberRow {
  id: string;
  worldId: string;
  userId: string;
  role: string;
  joinedAt: string;
  username: string;
  nickname: string;
}

const loading = ref(false);
const keyword = ref('');
const pagination = ref({ page: 1, pageSize: 20, total: 0 });
const rows = ref<MemberRow[]>([]);

const roleLabel = (role: string) => {
  switch (role) {
    case 'owner':
      return '拥有者';
    case 'admin':
      return '管理员';
    case 'spectator':
      return '旁观者';
    default:
      return '成员';
  }
};

const columns = computed(() => [
  {
    title: '用户',
    key: 'user',
    render(row: MemberRow) {
      return h('div', { class: 'member-user-cell' }, [
        h('div', { class: 'member-nick' }, row.nickname || row.username || '（未设置昵称）'),
        h('div', { class: 'member-id' }, row.userId),
      ]);
    },
  },
  {
    title: '角色',
    key: 'role',
    render(row: MemberRow) {
      const type = row.role === 'owner' ? 'warning' : row.role === 'admin' ? 'info' : 'default';
      return h(NTag, { type, size: 'small' }, { default: () => roleLabel(row.role) });
    },
  },
  {
    title: '加入时间',
    key: 'joinedAt',
    render(row: MemberRow) {
      return dayjs(row.joinedAt).format('YYYY-MM-DD HH:mm');
    },
  },
  {
    title: '操作',
    key: 'actions',
    render(row: MemberRow) {
      const disabled = row.role === 'owner';
      return h(
        NSpace,
        { size: 'small' },
        () => [
          h(
            NButton,
            {
              size: 'tiny',
              tertiary: true,
              disabled: disabled || row.role === 'admin',
              onClick: () => handleRoleChange(row, 'admin'),
            },
            { default: () => '设为管理员' },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              tertiary: true,
              disabled: disabled || row.role === 'member',
              onClick: () => handleRoleChange(row, 'member'),
            },
            { default: () => '设为成员' },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              tertiary: true,
              disabled: disabled || row.role === 'spectator',
              onClick: () => handleRoleChange(row, 'spectator'),
            },
            { default: () => '设为旁观者' },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              tertiary: true,
              type: 'error',
              disabled,
              onClick: () => handleRemove(row),
            },
            { default: () => '移除' },
          ),
        ],
      );
    },
  },
]);

const loadMembers = async () => {
  if (!props.worldId || !visible.value) return;
  loading.value = true;
  try {
    const resp = await chat.worldMemberList(props.worldId, {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keyword: keyword.value.trim(),
    });
    rows.value = resp.items || [];
    pagination.value.total = resp.total || 0;
  } catch (err: any) {
    message.error(err?.response?.data?.message || '加载成员失败');
  } finally {
    loading.value = false;
  }
};

const handleSearch = () => {
  pagination.value.page = 1;
  loadMembers();
};

const handleRoleChange = async (row: MemberRow, role: string) => {
  if (!props.worldId || row.role === role) return;
  try {
    await chat.worldMemberSetRole(props.worldId, row.userId, role);
    message.success('角色已更新');
    await loadMembers();
  } catch (err: any) {
    message.error(err?.response?.data?.message || '角色更新失败');
  }
};

const handleRemove = (row: MemberRow) => {
  if (!props.worldId) return;
  dialog.warning({
    title: '移除成员',
    content: `确定要移除「${row.nickname || row.userId}」吗？`,
    positiveText: '确认移除',
    negativeText: '取消',
    maskClosable: false,
    onPositiveClick: async () => {
      try {
        await chat.worldMemberRemove(props.worldId, row.userId);
        message.success('已移除成员');
        await loadMembers();
      } catch (err: any) {
        message.error(err?.response?.data?.message || '移除失败');
      }
    },
  });
};

const handlePageChange = (page: number) => {
  pagination.value.page = page;
  loadMembers();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
  loadMembers();
};

watch(
  () => visible.value,
  (val) => {
    if (val) {
      pagination.value.page = 1;
      loadMembers();
    }
  },
);

watch(
  () => props.worldId,
  () => {
    if (visible.value) {
      pagination.value.page = 1;
      loadMembers();
    }
  },
);

onMounted(() => {
  if (visible.value) {
    loadMembers();
  }
});
</script>

<template>
  <n-modal
    :show="visible"
    preset="dialog"
    title="成员管理"
    style="max-width: 760px"
    @update:show="val => visible = val"
  >
    <div class="member-manager-body">
      <div class="member-toolbar">
        <n-input
          v-model:value="keyword"
          placeholder="搜索用户ID / 昵称"
          size="small"
          clearable
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <n-button size="small" @click="handleSearch">搜索</n-button>
      </div>
      <n-data-table
        :columns="columns"
        :data="rows"
        :loading="loading"
        :bordered="false"
        :pagination="false"
        size="small"
      />
      <div class="member-pagination">
        <n-pagination
          :page="pagination.page"
          :page-size="pagination.pageSize"
          :item-count="pagination.total"
          show-size-picker
          :page-sizes="[10, 20, 50]"
          size="small"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </div>
  </n-modal>
</template>

<style scoped>
.member-manager-body {
  max-height: 70vh;
  overflow: auto;
  padding-right: 4px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.member-toolbar {
  display: flex;
  gap: 8px;
}
.member-pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 4px;
}
.member-user-cell {
  display: flex;
  flex-direction: column;
}
.member-nick {
  font-weight: 600;
}
.member-id {
  font-size: 12px;
  color: #94a3b8;
}
</style>
