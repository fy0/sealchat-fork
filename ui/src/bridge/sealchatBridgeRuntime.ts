import type {
  BridgeRoleSnapshot,
  SealChatBridgeHandshakeAck,
  SealChatBridgeHandshakeRequest,
  SealChatBridgeMessageEvent,
  SealChatBridgeMessagePayload,
  SealChatBridgeRequest,
  SealChatBridgeRolesSnapshot,
} from './sealchatBridgeProtocol'
import {
  markBridgeDisconnected,
  markBridgeError,
  markBridgeHandshake,
  markBridgeInbound,
  markBridgeMessagePublished,
  markBridgeRolesSnapshot,
} from './sealchatBridgeStatus'

type BridgeContext = {
  worldId: string
  channelId: string
}

type BridgeMessageEventLike = {
  source?: unknown
  origin: string
  data: unknown
}

type RuntimeDeps = {
  postMessage: (
    payload: SealChatBridgeHandshakeAck | SealChatBridgeRolesSnapshot | SealChatBridgeMessagePayload,
    origin: string,
  ) => void
  getCurrentContext: () => BridgeContext
  loadRoles: () => Promise<BridgeRoleSnapshot[]>
  isParentSource?: (source: unknown) => boolean
}

const isHandshakeRequest = (value: unknown): value is SealChatBridgeHandshakeRequest => {
  if (!value || typeof value !== 'object') {
    return false
  }
  const data = value as Record<string, unknown>
  return data.type === 'sealchat.bridge.handshake' && data.version === 1 && typeof data.nonce === 'string'
}

const isUnsubscribeRequest = (value: unknown): value is Extract<SealChatBridgeRequest, { type: 'sealchat.bridge.unsubscribe' }> => {
  if (!value || typeof value !== 'object') {
    return false
  }
  return (value as Record<string, unknown>).type === 'sealchat.bridge.unsubscribe'
}

export const createSealChatBridgeRuntime = (deps: RuntimeDeps) => {
  let active = false
  let targetOrigin = ''
  let targetSource: unknown = null

  const normalizeTargetOrigin = (origin: string): string => {
    const normalized = String(origin || '').trim()
    if (!normalized || normalized === 'null') {
      return '*'
    }
    return normalized
  }

  const reset = () => {
    active = false
    targetOrigin = ''
    targetSource = null
    markBridgeDisconnected()
  }

  const getCurrentContext = () => deps.getCurrentContext()

  const publishRolesSnapshot = async () => {
    if (!active || !targetOrigin) {
      return
    }
    const context = getCurrentContext()
    try {
      const roles = await deps.loadRoles()
      deps.postMessage({
        type: 'sealchat.bridge.roles.snapshot',
        worldId: context.worldId,
        channelId: context.channelId,
        generatedAt: Date.now(),
        roles,
      }, targetOrigin)
      markBridgeRolesSnapshot({
        worldId: context.worldId,
        channelId: context.channelId,
      })
    } catch (error) {
      markBridgeError(error)
      throw error
    }
  }

  const publishMessage = (payload: Omit<SealChatBridgeMessagePayload, 'type'>) => {
    if (!active || !targetOrigin) {
      return
    }
    try {
      deps.postMessage({
        type: 'sealchat.bridge.message',
        ...payload,
      }, targetOrigin)
      markBridgeMessagePublished({
        type: 'sealchat.bridge.message',
        worldId: payload.worldId,
        channelId: payload.channelId,
      })
    } catch (error) {
      markBridgeError(error)
      throw error
    }
  }

  const handleWindowMessage = async (event: BridgeMessageEventLike) => {
    if (deps.isParentSource && !deps.isParentSource(event.source)) {
      return
    }

    if (isUnsubscribeRequest(event.data)) {
      markBridgeInbound('sealchat.bridge.unsubscribe')
      reset()
      return
    }

    if (!isHandshakeRequest(event.data)) {
      return
    }

    active = true
    targetOrigin = normalizeTargetOrigin(event.origin)
    targetSource = event.source ?? null
    markBridgeInbound('sealchat.bridge.handshake')

    const context = getCurrentContext()
    try {
      deps.postMessage({
        type: 'sealchat.bridge.handshake.ack',
        version: 1,
        nonce: event.data.nonce,
        ok: true,
        worldId: context.worldId,
        channelId: context.channelId,
      }, targetOrigin)
      markBridgeHandshake({
        origin: targetOrigin,
        worldId: context.worldId,
        channelId: context.channelId,
      })
      await publishRolesSnapshot()
    } catch (error) {
      markBridgeError(error)
      throw error
    }
  }

  return {
    getCurrentContext,
    getTargetOrigin: () => targetOrigin,
    getTargetSource: () => targetSource,
    handleWindowMessage,
    isActive: () => active,
    publishMessage,
    publishRolesSnapshot,
    reset,
  }
}

export type SealChatBridgeRuntime = ReturnType<typeof createSealChatBridgeRuntime>

export const createBridgeMessagePayload = (
  event: SealChatBridgeMessageEvent,
  payload: Omit<SealChatBridgeMessagePayload, 'type' | 'event'>,
): Omit<SealChatBridgeMessagePayload, 'type'> => ({
  event,
  ...payload,
})
