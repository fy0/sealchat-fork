import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const composablePath = resolve(scriptDir, '../src/composables/useRobustInfiniteScroll.ts');
const source = readFileSync(composablePath, 'utf8');

test('useRobustInfiniteScroll combines observer, scroll fallback, and resize checks', () => {
  assert.match(source, /useIntersectionObserver\(/, 'missing intersection observer');
  assert.match(source, /useEventListener\(\s*containerRef,\s*'scroll'/, 'missing scroll event fallback');
  assert.match(source, /useResizeObserver\(/, 'missing resize observer fallback');
});

test('useRobustInfiniteScroll protects load-more with gate checks', () => {
  assert.match(source, /isEnabled = computed/, 'missing enabled gate');
  assert.match(source, /isLoadBlocked = computed/, 'missing blocked gate');
  assert.match(source, /loadPending\.value/, 'missing load dedupe guard');
});

test('useRobustInfiniteScroll supports nextTick and requestAnimationFrame rechecks', () => {
  assert.match(source, /await nextTick\(\)/, 'missing nextTick recheck');
  assert.match(source, /requestAnimationFrame/, 'missing requestAnimationFrame recheck');
  assert.match(source, /scrollHeight <= container\.clientHeight \+/, 'missing short-content detection');
});
