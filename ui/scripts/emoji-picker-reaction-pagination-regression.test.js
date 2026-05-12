import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const modalPath = resolve(scriptDir, '../src/views/chat/components/EmojiPickerModal.vue');
const source = readFileSync(modalPath, 'utf8');

test('EmojiPickerModal uses sentinel observer for reaction emoji pagination', () => {
  assert.match(source, /ref="customGridSentinelRef"/, 'missing reaction grid sentinel');
  assert.match(source, /useRobustInfiniteScroll\(/, 'missing robust infinite scroll composable');
  assert.match(source, /@scroll="handleCustomGridScroll"/, 'missing raw scroll fallback');
  assert.match(source, /v-if="props\.mode !== 'emoji-only' && activeTab === 'reaction'"/, 'reaction tab should render only when active');
});

test('EmojiPickerModal can request more reaction emoji pages from gallery store', () => {
  assert.match(source, /const canLoadMoreCustomEmoji = computed\(/, 'missing server pagination computed');
  assert.match(source, /await gallery\.loadItems\(collectionId,\s*\{[\s\S]*append:\s*true/, 'missing append pagination request');
  assert.match(source, /reactionPagination\.value\.total > customEmojiItems\.value\.length/, 'should compare loaded count against server total');
});

test('EmojiPickerModal auto-fills short reaction grids', () => {
  assert.match(source, /scrollFallback: true/, 'missing scroll fallback');
  assert.match(source, /observeResize: true/, 'missing resize observer fallback');
  assert.match(source, /requestAnimationFrameCheck: true/, 'missing raf recheck');
});
