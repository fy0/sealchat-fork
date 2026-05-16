import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const galleryGridPath = resolve(scriptDir, '../src/components/gallery/GalleryGrid.vue');
const source = readFileSync(galleryGridPath, 'utf8');

test('GalleryGrid uses sentinel-based load-more observer', () => {
  assert.match(source, /ref="loadMoreSentinelRef"/, 'missing load-more sentinel ref');
  assert.match(source, /useRobustInfiniteScroll\(/, 'missing robust infinite scroll composable');
  assert.doesNotMatch(source, /useInfiniteScroll\(/, 'should not depend on inner scroll container');
  assert.match(source, /@scroll="handleContentScroll"/, 'missing raw scroll fallback');
});

test('GalleryGrid can shrink into a real scroll container inside flex layouts', () => {
  const styleStart = source.indexOf('<style scoped>');
  assert.notEqual(styleStart, -1, 'missing scoped styles');
  const styleSource = source.slice(styleStart);
  assert.match(styleSource, /\.gallery-grid\s*\{[\s\S]*flex:\s*1\s*;/, 'gallery grid should grow within panel');
  assert.match(styleSource, /\.gallery-grid\s*\{[\s\S]*min-height:\s*0\s*;/, 'gallery grid should allow inner scrolling');
});

test('GalleryGrid auto-fills short content with more pages', () => {
  assert.match(source, /scrollFallback: true/, 'missing scroll fallback');
  assert.match(source, /observeResize: true/, 'missing resize observer fallback');
  assert.match(source, /requestAnimationFrameCheck: true/, 'missing raf recheck');
});
