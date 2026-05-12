import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const chatPath = resolve(scriptDir, '../src/views/chat/chat.vue');
const source = readFileSync(chatPath, 'utf8');

test('chat emoji panel uses sentinel-based pagination', () => {
  assert.match(source, /const emojiPanelContentRef = ref<HTMLElement \| null>\(null\);/, 'missing emoji panel content ref');
  assert.match(source, /const emojiPanelLoadMoreSentinelRef = ref<HTMLElement \| null>\(null\);/, 'missing emoji panel sentinel ref');
  assert.match(source, /const emojiPanelRenderKey = ref\(0\);/, 'missing panel render key');
  assert.match(source, /useRobustInfiniteScroll\(/, 'missing robust infinite scroll composable');
  assert.match(source, /@scroll="handleEmojiPanelContentScroll"/, 'missing raw scroll fallback');
});

test('chat emoji panel can append more collection pages', () => {
  assert.match(source, /const loadMoreEmojiPanelItems = async \(\) =>/, 'missing emoji panel load-more helper');
  assert.match(source, /await gallery\.loadItems\(tabId,\s*\{[\s\S]*append:\s*true/, 'missing append page request');
  assert.match(source, /const tabId = activeEmojiTab\.value;/, 'missing active tab guard');
});

test('chat emoji panel auto-fills short content', () => {
  assert.match(source, /scrollFallback: true/, 'missing scroll fallback');
  assert.match(source, /observeResize: true/, 'missing resize observer fallback');
  assert.match(source, /requestAnimationFrameCheck: true/, 'missing raf recheck');
  assert.match(source, /refreshEmojiPanelRender\(\);/, 'missing render refresh on open or tab switch');
});
