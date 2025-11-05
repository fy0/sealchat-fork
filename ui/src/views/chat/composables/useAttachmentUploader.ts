import { api } from '@/stores/_config';
import { useUserStore } from '@/stores/user';
import { useChatStore } from '@/stores/chat';
import { blobToArrayBuffer } from '@/utils/tools';
import { db } from '@/models';

interface UploadImageOptions {
  channelId?: string;
}

interface UploadImageResult {
  attachmentId: string;
  response: any;
}

export const uploadImageAttachment = async (file: File, options?: UploadImageOptions): Promise<UploadImageResult> => {
  const user = useUserStore();
  const chat = useChatStore();
  const channelId = options?.channelId || chat.curChannel?.id || '';

  const formData = new FormData();
  formData.append('file', file);

  const headers: Record<string, string> = {
    Authorization: `${user.token}`,
  };
  if (channelId) {
    headers.ChannelId = channelId;
  }

  const resp = await api.post('/api/v1/attachment-upload', formData, { headers });
  const idsField = resp.data?.ids;
  const filesField = resp.data?.files;

  const extractFirst = (value: unknown): string => {
    if (!value) return '';
    if (Array.isArray(value) && value.length) return String(value[0] ?? '');
    if (typeof value === 'string') return value;
    if (typeof value === 'object') {
      const firstKey = Object.keys(value as Record<string, unknown>)[0];
      if (firstKey) {
        return String((value as Record<string, unknown>)[firstKey] ?? '');
      }
    }
    return '';
  };

  const rawId = extractFirst(idsField);

  if (!rawId) {
    // 兼容旧结构：尝试从 files 字段回退一次
    const legacyToken = extractFirst(filesField);
    if (legacyToken) {
      throw new Error('服务端未返回附件ID，已停止兼容旧数据，请升级后端接口');
    }
    throw new Error('上传失败，请稍后重试');
  }

  const cacheKey = rawId;

  if (cacheKey) {
    try {
      await db.thumbs.put({
        id: cacheKey,
        recentUsed: Number(Date.now()),
        filename: file.name,
        mimeType: file.type,
        data: await blobToArrayBuffer(file),
      });
    } catch (error) {
      console.warn('缓存上传文件失败', error);
    }
  }

  const attachmentRef = `id:${rawId}`;

  return {
    attachmentId: attachmentRef as string,
    response: resp.data,
  };
};
