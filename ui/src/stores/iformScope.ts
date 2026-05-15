import type { SChannel } from '@/types';

export interface ResolveIFormVisibleChannelIdOptions {
  currentChannelId: string | null;
  currentChannelIsPrivate: boolean | null;
  lastNonPrivateChannelId: string | null;
}

export const isPrivateIFormChannel = (
  channel?: Pick<SChannel, 'isPrivate' | 'friendInfo' | 'permType'> & { id?: string; type?: unknown } | null,
) => {
  if (!channel) {
    return false;
  }
  if (channel.isPrivate) {
    return true;
  }
  if (channel.friendInfo) {
    return true;
  }
  const permType = typeof channel.permType === 'string' ? channel.permType.toLowerCase() : '';
  if (permType === 'private') {
    return true;
  }
  return typeof channel.type === 'number' && channel.type === 3;
};

export const resolveIFormVisibleChannelId = ({
  currentChannelId,
  currentChannelIsPrivate,
  lastNonPrivateChannelId,
}: ResolveIFormVisibleChannelIdOptions) => {
  if (!currentChannelId) {
    return null;
  }
  if (currentChannelIsPrivate && lastNonPrivateChannelId) {
    return lastNonPrivateChannelId;
  }
  return currentChannelId;
};
