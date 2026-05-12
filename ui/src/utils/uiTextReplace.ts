import type { UITextReplaceConfig, UITextReplaceRule } from '../types'

export type PreparedUITextReplaceRule = {
  id: string
  searchText: string
  replaceText: string
}

export type UITextReplaceIgnoredContext = {
  tagName?: string
  classNames?: string[]
  ancestorSelectors?: string[]
  isContentEditable?: boolean
}

type IdleWindow = Window & {
  requestIdleCallback?: (callback: IdleRequestCallback, options?: IdleRequestOptions) => number
  cancelIdleCallback?: (handle: number) => void
}

const DEFAULT_RULES: UITextReplaceRule[] = [
  { id: 'default-world-lobby', searchText: '世界大厅', replaceText: '世界大厅', enabled: true },
  { id: 'default-world-manage', searchText: '世界管理', replaceText: '世界管理', enabled: true },
  { id: 'default-glossary-manage', searchText: '术语管理', replaceText: '术语管理', enabled: true },
  { id: 'default-announcement', searchText: '公告', replaceText: '公告', enabled: true },
]

const IGNORE_SELECTOR = [
  '[data-ui-text-replace-ignore]',
  '[data-message-id]',
  '.message-row__grid',
  '.message-list',
  '.message-item',
  '.chat-input-wrapper',
  '.chat-input-container',
  '.chat-input-area',
  '.chat-input-editor-main',
  '.chat-input-plain-wrapper',
  '.hybrid-input',
  '.tiptap-editor',
  '.announcement-rich-html',
  '.sticky-note-editor__wrapper',
  '.sticky-note__rich-input',
  '.ProseMirror',
  'input',
  'textarea',
  '[contenteditable="true"]',
].join(', ')

const IGNORE_CLASS_NAMES = new Set([
  'chat-input-wrapper',
  'chat-input-container',
  'chat-input-area',
  'chat-input-editor-main',
  'chat-input-plain-wrapper',
  'hybrid-input',
  'tiptap-editor',
  'announcement-rich-html',
  'sticky-note-editor__wrapper',
  'sticky-note__rich-input',
  'message-list',
  'message-item',
  'message-row__grid',
])

const IGNORE_ANCESTOR_MATCHES = new Set([
  '[data-message-id]',
  '.chat-input-wrapper',
  '.chat-input-container',
  '.chat-input-area',
  '.chat-input-editor-main',
  '.chat-input-plain-wrapper',
  '.hybrid-input',
  '.tiptap-editor',
  '.announcement-rich-html',
  '.sticky-note-editor__wrapper',
  '.sticky-note__rich-input',
])

const cloneDefaultRules = () => DEFAULT_RULES.map((item) => ({ ...item }))

export const normalizeUITextReplaceConfig = (value?: UITextReplaceConfig | null): UITextReplaceConfig => {
  const sourceRules = Array.isArray(value?.rules) && value!.rules.length > 0 ? value!.rules : cloneDefaultRules()
  const rules = sourceRules
    .map((item, index) => ({
      id: String(item?.id || '').trim() || `ui-text-replace-${index + 1}`,
      searchText: String(item?.searchText || '').trim(),
      replaceText: String(item?.replaceText || '').trim(),
      enabled: item?.enabled !== false,
    }))
    .filter((item) => item.searchText.length > 0)
  return {
    enabled: value?.enabled === true,
    rules: rules.length > 0 ? rules : cloneDefaultRules(),
  }
}

export const prepareUITextReplaceRules = (config?: UITextReplaceConfig | null): PreparedUITextReplaceRule[] => {
  const normalized = normalizeUITextReplaceConfig(config)
  return normalized.rules
    .filter((item) => item.enabled && item.searchText && item.searchText !== item.replaceText)
    .map((item) => ({
      id: item.id,
      searchText: item.searchText,
      replaceText: item.replaceText,
    }))
    .sort((a, b) => b.searchText.length - a.searchText.length)
}

export const applyUITextReplaceRules = (text: string, rules: PreparedUITextReplaceRule[]): string => {
  let nextText = text
  for (const rule of rules) {
    if (!rule.searchText || nextText.includes(rule.searchText) === false) continue
    nextText = nextText.split(rule.searchText).join(rule.replaceText)
  }
  return nextText
}

export const isUITextReplaceIgnoredContext = (context: UITextReplaceIgnoredContext): boolean => {
  const tagName = String(context.tagName || '').toUpperCase()
  if (tagName === 'INPUT' || tagName === 'TEXTAREA' || tagName === 'SCRIPT' || tagName === 'STYLE') {
    return true
  }
  if (context.isContentEditable) {
    return true
  }
  if ((context.classNames || []).some((item) => IGNORE_CLASS_NAMES.has(item))) {
    return true
  }
  return (context.ancestorSelectors || []).some((item) => IGNORE_ANCESTOR_MATCHES.has(item))
}

const getRootElement = (): HTMLElement | null => {
  if (typeof document === 'undefined') return null
  return document.getElementById('app') || document.body
}

const getIdleWindow = (): IdleWindow | null => {
  if (typeof window === 'undefined') return null
  return window as IdleWindow
}

const matchesIgnoredDomContext = (element: Element | null): boolean => {
  if (!element) return true
  if (element instanceof HTMLElement && isUITextReplaceIgnoredContext({
    tagName: element.tagName,
    classNames: Array.from(element.classList || []),
    ancestorSelectors: [],
    isContentEditable: element.isContentEditable,
  })) {
    return true
  }
  return Boolean(element.closest(IGNORE_SELECTOR))
}

class UITextReplaceRuntime {
  private observer: MutationObserver | null = null
  private idleHandle: number | null = null
  private flushTimer: number | null = null
  private pendingNodes = new Set<Node>()
  private trackedNodes = new Set<Text>()
  private originalTextMap = new WeakMap<Text, string>()
  private selfMutatingNodes = new WeakSet<Text>()
  private activeRules: PreparedUITextReplaceRule[] = []
  private configSignature = ''

  apply(config?: UITextReplaceConfig | null) {
    const normalized = normalizeUITextReplaceConfig(config)
    const nextSignature = JSON.stringify(normalized)
    if (nextSignature === this.configSignature) return

    this.restoreTrackedNodes()
    this.configSignature = nextSignature
    this.activeRules = prepareUITextReplaceRules(normalized)

    if (!normalized.enabled || this.activeRules.length === 0) {
      this.stopObserver()
      return
    }

    this.startObserver()
    this.scheduleFullScan()
  }

  private scheduleFullScan() {
    const root = getRootElement()
    if (!root) return
    this.cancelIdleWork()
    const idleWindow = getIdleWindow()
    if (idleWindow?.requestIdleCallback) {
      this.idleHandle = idleWindow.requestIdleCallback(() => {
        this.idleHandle = null
        this.queueNode(root)
      }, { timeout: 400 })
      return
    }
    this.idleHandle = window.setTimeout(() => {
      this.idleHandle = null
      this.queueNode(root)
    }, 180)
  }

  private startObserver() {
    const root = getRootElement()
    if (!root || this.observer) return
    this.observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (mutation.type === 'characterData') {
          const target = mutation.target
          if (!(target instanceof Text)) continue
          if (this.selfMutatingNodes.has(target)) {
            this.selfMutatingNodes.delete(target)
            continue
          }
          if (this.trackedNodes.has(target)) {
            this.originalTextMap.set(target, target.nodeValue || '')
          }
          this.queueNode(target)
          continue
        }
        mutation.addedNodes.forEach((node) => this.queueNode(node))
      }
    })
    this.observer.observe(root, {
      childList: true,
      characterData: true,
      subtree: true,
    })
  }

  private stopObserver() {
    this.cancelIdleWork()
    this.cancelFlushWork()
    this.pendingNodes.clear()
    if (this.observer) {
      this.observer.disconnect()
      this.observer = null
    }
  }

  private cancelIdleWork() {
    if (this.idleHandle === null) return
    const idleWindow = getIdleWindow()
    if (idleWindow?.cancelIdleCallback) {
      idleWindow.cancelIdleCallback(this.idleHandle)
    } else {
      window.clearTimeout(this.idleHandle)
    }
    this.idleHandle = null
  }

  private cancelFlushWork() {
    if (this.flushTimer === null) return
    window.clearTimeout(this.flushTimer)
    this.flushTimer = null
  }

  private queueNode(node: Node | null) {
    if (!node || this.activeRules.length === 0) return
    this.pendingNodes.add(node)
    if (this.flushTimer !== null) return
    this.flushTimer = window.setTimeout(() => {
      this.flushTimer = null
      this.flushPendingNodes()
    }, 32)
  }

  private flushPendingNodes() {
    if (this.activeRules.length === 0 || this.pendingNodes.size === 0) return
    const nodes = Array.from(this.pendingNodes)
    this.pendingNodes.clear()
    for (const node of nodes) {
      this.processNode(node)
    }
  }

  private processNode(node: Node) {
    if (node instanceof Text) {
      this.processTextNode(node)
      return
    }
    if (!(node instanceof Element) && !(node instanceof DocumentFragment)) {
      return
    }
    if (node instanceof Element && matchesIgnoredDomContext(node)) {
      return
    }
    const root = getRootElement()
    if (!root) return
    const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT)
    let current = walker.nextNode()
    while (current) {
      if (current instanceof Text) {
        this.processTextNode(current)
      }
      current = walker.nextNode()
    }
  }

  private processTextNode(node: Text) {
    const parentElement = node.parentElement
    if (!parentElement || matchesIgnoredDomContext(parentElement)) return

    const currentText = node.nodeValue || ''
    if (!currentText.trim()) return

    const baseText = this.originalTextMap.get(node) ?? currentText
    const nextText = applyUITextReplaceRules(baseText, this.activeRules)

    if (nextText === baseText) {
      if (currentText !== baseText) {
        this.writeTextNode(node, baseText)
      }
      this.trackedNodes.delete(node)
      return
    }

    this.originalTextMap.set(node, baseText)
    this.trackedNodes.add(node)
    if (currentText !== nextText) {
      this.writeTextNode(node, nextText)
    }
  }

  private writeTextNode(node: Text, value: string) {
    this.selfMutatingNodes.add(node)
    node.nodeValue = value
  }

  private restoreTrackedNodes() {
    for (const node of Array.from(this.trackedNodes)) {
      if (!node.isConnected) {
        this.trackedNodes.delete(node)
        continue
      }
      const original = this.originalTextMap.get(node)
      if (typeof original === 'string' && node.nodeValue !== original) {
        this.writeTextNode(node, original)
      }
      this.trackedNodes.delete(node)
    }
  }
}

const runtime = new UITextReplaceRuntime()

export const applyUITextReplaceConfig = (config?: UITextReplaceConfig | null) => {
  runtime.apply(config)
}
