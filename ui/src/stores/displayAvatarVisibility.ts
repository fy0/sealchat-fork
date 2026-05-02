export type AvatarVisibilityScope = 'all' | 'ic' | 'ooc'

export interface AvatarRenderStateInput {
  avatarsEnabled: boolean
  avatarVisibilityScope: AvatarVisibilityScope
  icMode?: string | null
  mergedWithPrev?: boolean
}

export const normalizeAvatarVisibilityScope = (value: unknown): AvatarVisibilityScope => {
  if (value === 'ic' || value === 'ooc') {
    return value
  }
  return 'all'
}

export const normalizeMessageIcMode = (value: unknown): 'ic' | 'ooc' => {
  if (typeof value === 'string' && value.toLowerCase() === 'ooc') {
    return 'ooc'
  }
  return 'ic'
}

export const resolveAvatarRenderState = ({
  avatarsEnabled,
  avatarVisibilityScope,
  icMode,
  mergedWithPrev = false,
}: AvatarRenderStateInput) => {
  if (!avatarsEnabled) {
    return {
      showAvatar: false,
      hideAvatar: false,
    }
  }

  const normalizedScope = normalizeAvatarVisibilityScope(avatarVisibilityScope)
  const normalizedIcMode = normalizeMessageIcMode(icMode)
  const scopeMatched = normalizedScope === 'all' || normalizedScope === normalizedIcMode

  if (!scopeMatched) {
    return {
      showAvatar: true,
      hideAvatar: true,
    }
  }

  return {
    showAvatar: true,
    hideAvatar: Boolean(mergedWithPrev),
  }
}
