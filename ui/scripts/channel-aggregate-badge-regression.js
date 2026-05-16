const assert = require('node:assert/strict');

function collectVisiblePublicChannelIds(nodes, bucket = new Set()) {
  if (!Array.isArray(nodes)) return bucket;
  nodes.forEach((node) => {
    if (!node || typeof node.id !== 'string') return;
    bucket.add(node.id);
    if (Array.isArray(node.children) && node.children.length) {
      collectVisiblePublicChannelIds(node.children, bucket);
    }
  });
  return bucket;
}

function formatAggregateBadgeCount(count) {
  if (!Number.isFinite(count) || count <= 0) return '';
  return count > 99 ? '99+' : String(Math.trunc(count));
}

function resolveOtherChannelUnreadAggregate(args) {
  const currentChannelId = typeof args.currentChannelId === 'string' ? args.currentChannelId : '';
  const validIds = collectVisiblePublicChannelIds(args.channelTree);
  let total = 0;
  for (const [channelId, rawCount] of Object.entries(args.unreadCountMap || {})) {
    if (!validIds.has(channelId) || channelId === currentChannelId) {
      continue;
    }
    const count = Number(rawCount) || 0;
    if (count > 0) {
      total += count;
    }
  }
  return total;
}

const tree = [
  {
    id: 'root',
    children: [
      { id: 'child-a', children: [] },
      { id: 'child-b', children: [] },
    ],
  },
];

assert.equal(
  resolveOtherChannelUnreadAggregate({
    currentChannelId: 'root',
    unreadCountMap: { root: 4, 'child-a': 2, 'child-b': 1 },
    channelTree: tree,
  }),
  3,
  '应排除当前频道，只统计其他频道未读',
);

assert.equal(
  resolveOtherChannelUnreadAggregate({
    currentChannelId: 'child-a',
    unreadCountMap: { root: 4, 'child-a': 2, 'child-b': 1, 'friend:1': 99, unknown: 5 },
    channelTree: tree,
  }),
  5,
  '应只统计当前世界可见公开频道，忽略私聊和未知频道',
);

assert.equal(formatAggregateBadgeCount(0), '', '0 未读不显示 badge');
assert.equal(formatAggregateBadgeCount(7), '7', '普通计数直接显示');
assert.equal(formatAggregateBadgeCount(120), '99+', '超过 99 应折叠为 99+');

console.log('channel aggregate badge regressions passed');
