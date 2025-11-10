import { defineStore } from 'pinia'

export type DisplayLayout = 'bubble' | 'compact'
export type DisplayPalette = 'day' | 'night'

export interface DisplaySettings {
  layout: DisplayLayout
  palette: DisplayPalette
  showAvatar: boolean
  mergeNeighbors: boolean
  maxExportMessages: number
  maxExportConcurrency: number
}

const STORAGE_KEY = 'sealchat_display_settings'

const SLICE_LIMIT_DEFAULT = 5000
const SLICE_LIMIT_MIN = 1000
const SLICE_LIMIT_MAX = 20000
const CONCURRENCY_DEFAULT = 2
const CONCURRENCY_MIN = 1
const CONCURRENCY_MAX = 8

const coerceLayout = (value?: string): DisplayLayout => (value === 'compact' ? 'compact' : 'bubble')
const coercePalette = (value?: string): DisplayPalette => (value === 'night' ? 'night' : 'day')
const coerceBoolean = (value: any): boolean => value !== false
const coerceNumberInRange = (value: any, fallback: number, min: number, max: number): number => {
  const num = Number(value)
  if (!Number.isFinite(num)) return fallback
  if (num < min) return min
  if (num > max) return max
  return Math.round(num)
}

const defaultSettings = (): DisplaySettings => ({
  layout: 'bubble',
  palette: 'day',
  showAvatar: true,
  mergeNeighbors: true,
  maxExportMessages: SLICE_LIMIT_DEFAULT,
  maxExportConcurrency: CONCURRENCY_DEFAULT,
})

const loadSettings = (): DisplaySettings => {
  if (typeof window === 'undefined') {
    return defaultSettings()
  }
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return defaultSettings()
    }
    const parsed = JSON.parse(raw) as Partial<DisplaySettings>
    return {
      layout: coerceLayout(parsed.layout),
      palette: coercePalette(parsed.palette),
      showAvatar: coerceBoolean(parsed.showAvatar),
      mergeNeighbors: coerceBoolean(parsed.mergeNeighbors),
      maxExportMessages: coerceNumberInRange(
        parsed.maxExportMessages,
        SLICE_LIMIT_DEFAULT,
        SLICE_LIMIT_MIN,
        SLICE_LIMIT_MAX,
      ),
      maxExportConcurrency: coerceNumberInRange(
        parsed.maxExportConcurrency,
        CONCURRENCY_DEFAULT,
        CONCURRENCY_MIN,
        CONCURRENCY_MAX,
      ),
    }
  } catch (error) {
    console.warn('加载显示模式设置失败，使用默认值', error)
    return defaultSettings()
  }
}

const normalizeWith = (base: DisplaySettings, patch?: Partial<DisplaySettings>): DisplaySettings => ({
  layout: patch && patch.layout ? coerceLayout(patch.layout) : base.layout,
  palette: patch && patch.palette ? coercePalette(patch.palette) : base.palette,
  showAvatar:
    patch && Object.prototype.hasOwnProperty.call(patch, 'showAvatar')
      ? coerceBoolean(patch.showAvatar)
      : base.showAvatar,
  mergeNeighbors:
    patch && Object.prototype.hasOwnProperty.call(patch, 'mergeNeighbors')
      ? coerceBoolean(patch.mergeNeighbors)
      : base.mergeNeighbors,
  maxExportMessages:
    patch && Object.prototype.hasOwnProperty.call(patch, 'maxExportMessages')
      ? coerceNumberInRange(patch.maxExportMessages, SLICE_LIMIT_DEFAULT, SLICE_LIMIT_MIN, SLICE_LIMIT_MAX)
      : base.maxExportMessages,
  maxExportConcurrency:
    patch && Object.prototype.hasOwnProperty.call(patch, 'maxExportConcurrency')
      ? coerceNumberInRange(
          patch.maxExportConcurrency,
          CONCURRENCY_DEFAULT,
          CONCURRENCY_MIN,
          CONCURRENCY_MAX,
        )
      : base.maxExportConcurrency,
})

export const useDisplayStore = defineStore('display', {
  state: () => ({
    settings: loadSettings(),
  }),
  getters: {
    layout: (state) => state.settings.layout,
    palette: (state) => state.settings.palette,
    showAvatar: (state) => state.settings.showAvatar,
  },
  actions: {
    updateSettings(patch: Partial<DisplaySettings>) {
      this.settings = normalizeWith(this.settings, patch)
      this.persist()
      this.applyTheme()
    },
    reset() {
      this.settings = defaultSettings()
      this.persist()
      this.applyTheme()
    },
    persist() {
      if (typeof window === 'undefined') return
      try {
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(this.settings))
      } catch (error) {
        console.warn('显示模式设置写入失败', error)
      }
    },
    applyTheme(target?: DisplaySettings) {
      if (typeof document === 'undefined') return
      const effective = target || this.settings
      const root = document.documentElement
      root.dataset.displayPalette = effective.palette
      root.dataset.displayLayout = effective.layout
    },
  },
})
