import type { DataTableColumnWidthPolicy } from "@/types/data-table.types"

const WIDTH_EPSILON = 0.01

export interface SemanticColumnWidthInput {
  id: string
  size: number
  minSize?: number
  maxSize?: number
  visible?: boolean
  widthPolicy?: DataTableColumnWidthPolicy
}

export interface SemanticColumnWidthAllocation {
  widthsById: Map<string, number>
  minimumWidth: number
}

type ResolvedColumn = {
  id: string
  width: number
  minimum: number
  maximum: number
  policy: DataTableColumnWidthPolicy
}

function toNonNegativeFiniteNumber(value: number | undefined, fallback: number) {
  return Number.isFinite(value) && (value ?? 0) >= 0 ? value as number : fallback
}

function resolveColumn(input: SemanticColumnWidthInput): ResolvedColumn {
  const size = toNonNegativeFiniteNumber(input.size, 0)
  const minimum = toNonNegativeFiniteNumber(input.minSize, size)
  const maximum = Math.max(minimum, toNonNegativeFiniteNumber(input.maxSize, Number.POSITIVE_INFINITY))
  const policy = input.widthPolicy ?? { mode: "fixed" as const }

  return {
    id: input.id,
    width: policy.mode === "flex" ? minimum : Math.min(Math.max(size, minimum), maximum),
    minimum,
    maximum,
    policy,
  }
}

/**
 * Allocates column widths from declarations and container geometry only, so
 * paging or filtering cannot make a business list reflow around current data.
 */
export function allocateSemanticColumnWidths(
  columns: SemanticColumnWidthInput[],
  containerWidth: number
): SemanticColumnWidthAllocation {
  const visibleColumns = columns
    .filter((column) => column.visible !== false)
    .map(resolveColumn)
  const minimumWidth = visibleColumns.reduce((total, column) => total + column.width, 0)
  const resolvedContainerWidth = Math.max(toNonNegativeFiniteNumber(containerWidth, 0), minimumWidth)
  let remainingWidth = resolvedContainerWidth - minimumWidth
  let activeFlexColumns = visibleColumns.filter((column) => (
    column.policy.mode === "flex" && column.maximum - column.width > WIDTH_EPSILON
  ))

  while (remainingWidth > WIDTH_EPSILON && activeFlexColumns.length > 0) {
    const totalFlex = activeFlexColumns.reduce((total, column) => (
      total + Math.max(column.policy.mode === "flex" ? column.policy.flex : 0, 0)
    ), 0)
    if (totalFlex <= 0) {
      break
    }

    let consumedWidth = 0
    for (const column of activeFlexColumns) {
      const flex = column.policy.mode === "flex" ? Math.max(column.policy.flex, 0) : 0
      const requestedWidth = remainingWidth * (flex / totalFlex)
      const growth = Math.min(requestedWidth, column.maximum - column.width)
      column.width += growth
      consumedWidth += growth
    }

    if (consumedWidth <= WIDTH_EPSILON) {
      break
    }

    remainingWidth -= consumedWidth
    activeFlexColumns = activeFlexColumns.filter((column) => column.maximum - column.width > WIDTH_EPSILON)
  }

  const fillColumn = visibleColumns.find((column) => (
    column.policy.mode === "flex" && column.policy.fill
  ))
  if (remainingWidth > WIDTH_EPSILON && fillColumn) {
    fillColumn.width += remainingWidth
  }

  return {
    widthsById: new Map(visibleColumns.map((column) => [column.id, column.width])),
    minimumWidth,
  }
}
