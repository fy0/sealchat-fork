import { reactive, readonly } from 'vue'

export interface SealChatBridgeStatusState {
  active: boolean
  targetOrigin: string
  worldId: string
  channelId: string
  lastHandshakeAt: number
  lastRolesSnapshotAt: number
  lastMessageAt: number
  lastInboundType: string
  lastOutboundType: string
  lastError: string
  connectState: string
}

type BridgeContextInput = {
  worldId?: string
  channelId?: string
}

type BridgeHandshakeInput = BridgeContextInput & {
  origin?: string
  at?: number
}

type BridgePublishInput = BridgeContextInput & {
  at?: number
  type?: string
}

const createDefaultBridgeStatusState = (): SealChatBridgeStatusState => ({
  active: false,
  targetOrigin: '',
  worldId: '',
  channelId: '',
  lastHandshakeAt: 0,
  lastRolesSnapshotAt: 0,
  lastMessageAt: 0,
  lastInboundType: '',
  lastOutboundType: '',
  lastError: '',
  connectState: '',
})

export const createSealChatBridgeStatusState = (): SealChatBridgeStatusState => (
  reactive(createDefaultBridgeStatusState()) as SealChatBridgeStatusState
)

const mutableSealChatBridgeStatusState = createSealChatBridgeStatusState()

export const sealChatBridgeStatusState = readonly(mutableSealChatBridgeStatusState) as Readonly<SealChatBridgeStatusState>

const resolveStateAndInput = <T extends object>(
  stateOrInput: SealChatBridgeStatusState | T,
  maybeInput?: T,
): { state: SealChatBridgeStatusState; input: T } => {
  if (maybeInput) {
    return {
      state: stateOrInput as SealChatBridgeStatusState,
      input: maybeInput,
    }
  }
  return {
    state: mutableSealChatBridgeStatusState,
    input: stateOrInput as T,
  }
}

const normalizeTimestamp = (value?: number) => {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
    return Math.trunc(value)
  }
  return Date.now()
}

const stringifyBridgeError = (error: unknown) => {
  if (error instanceof Error) {
    return error.message || error.name || '未知错误'
  }
  if (typeof error === 'string') {
    return error.trim() || '未知错误'
  }
  if (error && typeof error === 'object') {
    try {
      return JSON.stringify(error)
    } catch {
      return '未知错误'
    }
  }
  return '未知错误'
}

export const resetSealChatBridgeStatus = (state: SealChatBridgeStatusState = mutableSealChatBridgeStatusState) => {
  Object.assign(state, createDefaultBridgeStatusState())
}

export function syncSealChatBridgeContext(context: BridgeContextInput): void
export function syncSealChatBridgeContext(state: SealChatBridgeStatusState, context: BridgeContextInput): void
export function syncSealChatBridgeContext(
  stateOrContext: SealChatBridgeStatusState | BridgeContextInput,
  maybeContext?: BridgeContextInput,
) {
  const { state, input } = resolveStateAndInput(stateOrContext, maybeContext)
  state.worldId = String(input.worldId || '').trim()
  state.channelId = String(input.channelId || '').trim()
}

export function markBridgeInbound(type: string): void
export function markBridgeInbound(state: SealChatBridgeStatusState, type: string): void
export function markBridgeInbound(
  stateOrType: SealChatBridgeStatusState | string,
  maybeType?: string,
) {
  const state = typeof stateOrType === 'string' ? mutableSealChatBridgeStatusState : stateOrType
  const type = typeof stateOrType === 'string' ? stateOrType : (maybeType || '')
  state.lastInboundType = String(type || '').trim()
}

export function markBridgeHandshake(input: BridgeHandshakeInput): void
export function markBridgeHandshake(state: SealChatBridgeStatusState, input: BridgeHandshakeInput): void
export function markBridgeHandshake(
  stateOrInput: SealChatBridgeStatusState | BridgeHandshakeInput,
  maybeInput?: BridgeHandshakeInput,
) {
  const { state, input } = resolveStateAndInput(stateOrInput, maybeInput)
  state.active = true
  state.targetOrigin = String(input.origin || '').trim()
  state.lastHandshakeAt = normalizeTimestamp(input.at)
  state.lastError = ''
  syncSealChatBridgeContext(state, input)
}

export function markBridgeRolesSnapshot(input: BridgePublishInput): void
export function markBridgeRolesSnapshot(state: SealChatBridgeStatusState, input: BridgePublishInput): void
export function markBridgeRolesSnapshot(
  stateOrInput: SealChatBridgeStatusState | BridgePublishInput,
  maybeInput?: BridgePublishInput,
) {
  const { state, input } = resolveStateAndInput(stateOrInput, maybeInput)
  state.lastRolesSnapshotAt = normalizeTimestamp(input.at)
  state.lastOutboundType = String(input.type || 'sealchat.bridge.roles.snapshot').trim()
  state.lastError = ''
  syncSealChatBridgeContext(state, input)
}

export function markBridgeMessagePublished(input: BridgePublishInput): void
export function markBridgeMessagePublished(state: SealChatBridgeStatusState, input: BridgePublishInput): void
export function markBridgeMessagePublished(
  stateOrInput: SealChatBridgeStatusState | BridgePublishInput,
  maybeInput?: BridgePublishInput,
) {
  const { state, input } = resolveStateAndInput(stateOrInput, maybeInput)
  state.lastMessageAt = normalizeTimestamp(input.at)
  state.lastOutboundType = String(input.type || 'sealchat.bridge.message').trim()
  state.lastError = ''
  syncSealChatBridgeContext(state, input)
}

export function markBridgeDisconnected(): void
export function markBridgeDisconnected(state: SealChatBridgeStatusState): void
export function markBridgeDisconnected(state: SealChatBridgeStatusState = mutableSealChatBridgeStatusState) {
  state.active = false
  state.targetOrigin = ''
}

export function markBridgeError(error: unknown): void
export function markBridgeError(state: SealChatBridgeStatusState, error: unknown): void
export function markBridgeError(
  stateOrError: SealChatBridgeStatusState | unknown,
  maybeError?: unknown,
) {
  const state = maybeError === undefined ? mutableSealChatBridgeStatusState : stateOrError as SealChatBridgeStatusState
  const error = maybeError === undefined ? stateOrError : maybeError
  state.lastError = stringifyBridgeError(error)
}

export function setSealChatBridgeConnectState(connectState: string): void
export function setSealChatBridgeConnectState(state: SealChatBridgeStatusState, connectState: string): void
export function setSealChatBridgeConnectState(
  stateOrConnectState: SealChatBridgeStatusState | string,
  maybeConnectState?: string,
) {
  const state = typeof stateOrConnectState === 'string' ? mutableSealChatBridgeStatusState : stateOrConnectState
  const connectState = typeof stateOrConnectState === 'string' ? stateOrConnectState : (maybeConnectState || '')
  state.connectState = String(connectState || '').trim()
}
