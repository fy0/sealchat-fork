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

  const resp = await api.post('/api/v1/upload', formData, { headers });
  const filesField = resp.data?.files;
  let rawId = '';
  if (Array.isArray(filesField)) {
    rawId = filesField[0];
  } else if (typeof filesField === 'string') {
    rawId = filesField;
  } else if (filesField && typeof filesField === 'object') {
    rawId = (filesField as any)[0];
  }

  if (!rawId) {
    throw new Error('上传失败，请稍后重试');
  }

  try {
    await db.thumbs.add({
      id: rawId,
      recentUsed: Number(Date.now()),
      filename: file.name,
      mimeType: file.type,
      data: await blobToArrayBuffer(file),
    });
  } catch (error) {
    console.warn('缓存上传文件失败', error);
  }

  return {
    attachmentId: rawId.startsWith('id:') ? rawId : `id:${rawId}`,
    response: resp.data,
  };
};
