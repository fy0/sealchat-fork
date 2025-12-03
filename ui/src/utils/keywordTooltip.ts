interface TooltipContent {
  title: string
  description: string
  editable?: boolean
  onEdit?: () => void
}

type ContentResolver = (keywordId: string) => TooltipContent | null | undefined

export function createKeywordTooltip(resolver: ContentResolver) {
  if (typeof document === 'undefined') {
    return {
      show() {},
      hide() {},
    }
  }
  const tooltip = document.createElement('div')
  tooltip.className = 'keyword-tooltip'
  tooltip.style.display = 'none'
  document.body.appendChild(tooltip)

  const hide = () => {
    tooltip.style.display = 'none'
  }

  const show = (target: HTMLElement, keywordId: string) => {
    const data = resolver(keywordId)
    if (!data) {
      hide()
      return
    }
    const title = data.title || '术语'
    const description = data.description || ''
    tooltip.innerHTML = ''
    const header = document.createElement('div')
    header.className = 'keyword-tooltip__header'
    header.textContent = title
    tooltip.appendChild(header)
    if (description) {
      const body = document.createElement('div')
      body.className = 'keyword-tooltip__body'
      body.textContent = description
      tooltip.appendChild(body)
    }
    if (data.editable && typeof data.onEdit === 'function') {
      const action = document.createElement('button')
      action.type = 'button'
      action.className = 'keyword-tooltip__action'
      action.textContent = '编辑'
      action.addEventListener('click', (event) => {
        event.stopPropagation()
        data.onEdit?.()
        hide()
      })
      tooltip.appendChild(action)
    }
    const rect = target.getBoundingClientRect()
    const top = Math.max(8, rect.top - tooltip.offsetHeight - 8)
    const left = Math.min(window.innerWidth - 260, Math.max(8, rect.left + rect.width / 2 - 130))
    tooltip.style.display = 'block'
    tooltip.style.top = `${top + window.scrollY}px`
    tooltip.style.left = `${left + window.scrollX}px`
  }

  return { show, hide }
}
