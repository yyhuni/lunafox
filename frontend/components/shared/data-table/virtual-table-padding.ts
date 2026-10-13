export type VirtualTableScrollBox = {
  start: number
  end: number
  size: number
}

// `start` includes the header scroll margin. Padding is the unrendered
// prefix and suffix inside the tbody, so the margin is removed once.
export function virtualTableScrollPadding(
  virtualRows: readonly VirtualTableScrollBox[],
  totalSize: number,
  scrollMargin: number,
) {
  if (virtualRows.length === 0 || totalSize <= 0) {
    return { paddingTop: 0, paddingBottom: 0 }
  }

  const first = virtualRows[0]
  const last = virtualRows[virtualRows.length - 1]
  return {
    paddingTop: Math.max(0, first.start - scrollMargin),
    paddingBottom: Math.max(0, totalSize - (last.end - scrollMargin)),
  }
}
