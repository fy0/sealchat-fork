import type { SChannel } from '@/types';

export interface ChannelAggregateBadgeState {
  visible: boolean;
  count: number;
  label: string;
}

const collectVisiblePublicChannelIds = (nodes?: SChannel[], bucket: Set<string> = new Set()): Set<string> => {
  if (!Array.isArray(nodes)) {
    return bucket;
  }
  nodes.forEach((node) => {
    const id = typeof node?.id === 'string' ? node.id : '';
    if (!id) {
      return;
    }
    bucket.add(id);
    if (Array.isArray(node.children) && node.children.length > 0) {
      collectVisiblePublicChannelIds(node.children as SChannel[], bucket);
    }
  });
  return bucket;
};

export const formatChannelAggregateBadgeCount = (count: number): string => {
  if (!Number.isFinite(count) || count <= 0) {
    return '';
  }
  return count > 99 ? '99+' : String(Math.trunc(count));
};

export const resolveOtherChannelUnreadAggregate = (args: {
  currentChannelId?: string | null;
  unreadCountMap?: Record<string, number>;
  channelTree?: SChannel[];
}): ChannelAggregateBadgeState => {
  const currentChannelId = typeof args.currentChannelId === 'string' ? args.currentChannelId.trim() : '';
  const validIds = collectVisiblePublicChannelIds(args.channelTree);
  let count = 0;

  Object.entries(args.unreadCountMap || {}).forEach(([channelId, rawCount]) => {
    if (!validIds.has(channelId) || channelId === currentChannelId) {
      return;
    }
    const unread = Number(rawCount) || 0;
    if (unread > 0) {
      count += unread;
    }
  });

  const label = formatChannelAggregateBadgeCount(count);
  return {
    visible: label.length > 0,
    count,
    label,
  };
};
