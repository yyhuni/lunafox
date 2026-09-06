export const UI_SANS_FONT_STACK =
  "system-ui, -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans CJK SC', sans-serif"

export const UI_MONO_FONT_STACK = "'JetBrains Mono', 'Fira Code', Consolas, monospace"

export function getUiSansCanvasFont({
  sizePx,
  weight = 400,
}: {
  sizePx: number
  weight?: number | string
}): string {
  return `${weight} ${sizePx}px ${UI_SANS_FONT_STACK}`
}

export const UI_SANS_MEASURE_FONT_14 = getUiSansCanvasFont({ sizePx: 14 })
export const UI_SANS_MEASURE_FONT_14_MEDIUM = getUiSansCanvasFont({ sizePx: 14, weight: 500 })
export const UI_SANS_MEASURE_FONT_12_MEDIUM = getUiSansCanvasFont({ sizePx: 12, weight: 500 })
