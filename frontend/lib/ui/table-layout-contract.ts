export interface FixedTableColumnLayout {
  key: string
  size: number
  minSize?: number
  maxSize?: number
  stickyRight?: boolean
}

export function getFixedTableLayoutTotalWidth(
  columns: readonly FixedTableColumnLayout[]
) {
  if (columns.length === 0) {
    throw new Error("getFixedTableLayoutTotalWidth requires at least one column.")
  }

  return columns.reduce((totalWidth, column) => {
    if (!Number.isFinite(column.size) || column.size <= 0) {
      throw new Error(`Table column "${column.key}" requires a positive finite size.`)
    }

    return totalWidth + column.size
  }, 0)
}
