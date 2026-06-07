const getResponseMessage = (error: any): string => {
  const data = error?.response?.data;
  return data?.message || data?.error || error?.message || '';
};

const isGenericPermissionMessage = (message: string) => {
  const normalized = message.trim().toLowerCase();
  return normalized === '无权限访问'
    || normalized === 'permission denied'
    || normalized === 'forbidden'
    || normalized === 'unauthorized';
};

export const resolveActionErrorMessage = (
  error: any,
  fallback: string,
  permissionFallback?: string,
) => {
  const status = error?.response?.status;
  const responseMessage = getResponseMessage(error);
  if ((status === 401 || status === 403) && permissionFallback) {
    if (!responseMessage || isGenericPermissionMessage(responseMessage)) {
      return permissionFallback;
    }
    return `${permissionFallback}（${responseMessage}）`;
  }
  return responseMessage || fallback;
};
