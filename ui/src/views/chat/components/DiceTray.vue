<template>
  <div class="dice-tray">
    <div class="dice-tray__header">
      <div>
        默认骰：<strong>{{ currentDefaultDice }}</strong>
      </div>
      <n-button v-if="canEditDefault" size="tiny" text type="primary" @click="drawerVisible = true">
        修改
      </n-button>
    </div>
    <div class="dice-tray__body">
      <div class="dice-tray__column dice-tray__column--quick">
        <div class="dice-tray__section-title">快捷骰</div>
        <div class="dice-tray__quick-grid">
          <n-button
            v-for="faces in quickFaces"
            :key="faces"
            size="small"
            quaternary
            @click="handleQuickInsert(faces)"
          >
            d{{ faces }}
          </n-button>
        </div>
      </div>
      <div class="dice-tray__column dice-tray__column--form">
        <div class="dice-tray__section-title">自定义</div>
        <div class="dice-tray__form">
          <n-form-item label="数量">
            <n-input-number v-model:value="count" :min="1" size="small" />
          </n-form-item>
          <n-form-item label="面数">
            <n-input-number v-model:value="sides" :min="1" size="small" />
          </n-form-item>
          <n-form-item label="修正">
            <n-input-number v-model:value="modifier" size="small" />
          </n-form-item>
          <n-form-item label="理由">
            <n-input v-model:value="reason" size="small" placeholder="可选，例如攻击" />
          </n-form-item>
          <div class="dice-tray__actions">
            <n-button size="small" :disabled="!canSubmit" @click="handleInsert">
              插入到输入框
            </n-button>
            <n-button type="primary" size="small" :disabled="!canSubmit" @click="handleRoll">
              立即掷骰
            </n-button>
          </div>
        </div>
      </div>
    </div>
  </div>
  <n-drawer v-model:show="drawerVisible" placement="right" width="320">
    <n-drawer-content title="修改默认骰">
      <n-form size="small" label-placement="left" :show-feedback="false">
        <n-form-item label="面数">
          <n-input v-model:value="defaultDiceInput" placeholder="例如 d20" />
        </n-form-item>
        <n-alert v-if="defaultDiceError" type="warning" :show-icon="false">
          {{ defaultDiceError }}
        </n-alert>
        <div class="dice-tray__settings-actions">
          <n-button @click="drawerVisible = false">取消</n-button>
          <n-button type="primary" :disabled="!!defaultDiceError" @click="handleSaveDefault">
            保存
          </n-button>
        </div>
      </n-form>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ensureDefaultDiceExpr, isValidDefaultDiceExpr } from '@/utils/dice';

const props = withDefaults(defineProps<{
  defaultDice?: string
  canEditDefault?: boolean
}>(), {
  defaultDice: 'd20',
  canEditDefault: false,
});

const emit = defineEmits<{
  (event: 'insert', expr: string): void
  (event: 'roll', expr: string): void
  (event: 'update-default', expr: string): void
}>();

const quickFaces = [2, 4, 6, 8, 10, 12, 20, 100];
const count = ref(1);
const sides = ref<number | null>(null);
const modifier = ref(0);
const reason = ref('');
const drawerVisible = ref(false);
const defaultDiceInput = ref(ensureDefaultDiceExpr(props.defaultDice));

const currentDefaultDice = computed(() => ensureDefaultDiceExpr(props.defaultDice));

watch(() => props.defaultDice, (value) => {
  defaultDiceInput.value = ensureDefaultDiceExpr(value);
  if (!sides.value) {
    sides.value = parseInt(defaultDiceInput.value.slice(1), 10) || 20;
  }
}, { immediate: true });

const sanitizedReason = computed(() => reason.value.trim());

const expression = computed(() => {
  if (!count.value || !sides.value) {
    return '';
  }
  const amount = Math.max(1, Math.floor(count.value));
  const face = Math.max(1, Math.floor(sides.value));
  const parts = [`.r${amount}d${face}`];
  if (modifier.value) {
    const delta = Math.trunc(modifier.value);
    if (delta > 0) {
      parts.push(`+${delta}`);
    } else {
      parts.push(`${delta}`);
    }
  }
  if (sanitizedReason.value) {
    parts.push(`#${sanitizedReason.value}`);
  }
  return parts.join(' ');
});

const canSubmit = computed(() => !!expression.value);

const handleQuickInsert = (faces: number) => {
  const expr = `.r1d${faces}`;
  emit('insert', expr);
};

const handleInsert = () => {
  if (canSubmit.value) {
    emit('insert', expression.value);
  }
};

const handleRoll = () => {
  if (canSubmit.value) {
    emit('roll', expression.value);
  }
};

const defaultDiceError = computed(() => {
  if (!defaultDiceInput.value) {
    return '请输入默认骰，例如 d20';
  }
  if (!isValidDefaultDiceExpr(defaultDiceInput.value)) {
    return '格式不正确，示例：d20';
  }
  return '';
});

const handleSaveDefault = () => {
  if (defaultDiceError.value) {
    return;
  }
  emit('update-default', ensureDefaultDiceExpr(defaultDiceInput.value));
  drawerVisible.value = false;
};
</script>

<style scoped>
.dice-tray {
  min-width: 320px;
  max-width: 480px;
  padding: 12px;
  background: var(--sc-bg-elevated, #fff);
  border: 1px solid var(--sc-border-strong, #e5e7eb);
  border-radius: 12px;
  color: var(--sc-fg-primary, #111);
}

.dice-tray__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 13px;
}

.dice-tray__body {
  display: flex;
  gap: 12px;
}

.dice-tray__column {
  flex: 1;
  padding: 8px;
  border-radius: 10px;
  background: var(--sc-bg-layer, #fafafa);
}

.dice-tray__column--quick {
  flex: 0 0 140px;
}

.dice-tray__section-title {
  font-size: 12px;
  color: var(--sc-fg-muted, #666);
  margin-bottom: 6px;
}

.dice-tray__quick-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.dice-tray__form :deep(.n-form-item) {
  margin-bottom: 8px;
}

.dice-tray__actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 8px;
}

.dice-tray__settings-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

:global([data-display-palette='night']) .dice-tray {
  background: var(--sc-bg-elevated, #2a282a);
  border-color: var(--sc-border-strong, rgba(255, 255, 255, 0.12));
  color: var(--sc-fg-primary, #eee);
}

:global([data-display-palette='night']) .dice-tray__column {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

:global([data-display-palette='night']) .dice-tray__column--quick {
  background: rgba(255, 255, 255, 0.03);
}

:global([data-display-palette='night']) .dice-tray__column--form {
  background: rgba(255, 255, 255, 0.06);
}
</style>
