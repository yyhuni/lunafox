/**
 * Table utility functions
 * Provides column width calculation and other table-related utilities
 */

import {
  UI_SANS_MEASURE_FONT_12_MEDIUM,
  UI_SANS_MEASURE_FONT_14,
  UI_SANS_MEASURE_FONT_14_MEDIUM,
} from "@/lib/font-stacks"

// Cache for text measurement context
let measureContext: CanvasRenderingContext2D | null | undefined

function shouldUseCanvasMeasurement(): boolean {
  if (typeof navigator !== "undefined" && /jsdom/i.test(navigator.userAgent)) {
    return false
  }

  return typeof document !== "undefined"
}

/**
 * Get or create a canvas context for measuring text width
 */
function getMeasureContext(): CanvasRenderingContext2D | null {
  if (!shouldUseCanvasMeasurement()) {
    return null
  }

  if (measureContext === undefined) {
    const canvas = document.createElement('canvas')
    measureContext = canvas.getContext('2d')
  }
  return measureContext
}

function estimateTextWidth(text: string): number {
  let width = 0

  for (const char of text) {
    width += /[^\u0000-\u00ff]/.test(char) ? 12 : 7
  }

  return width
}

/**
 * Measure text width using canvas
 * @param text - Text to measure
 * @param font - CSS font string (e.g., "400 14px system-ui, sans-serif")
 * @returns Text width in pixels
 */
export function measureTextWidth(text: string, font: string = UI_SANS_MEASURE_FONT_14): number {
  const ctx = getMeasureContext()
  if (!ctx) {
    return Math.ceil(estimateTextWidth(text))
  }
  ctx.font = font
  return ctx.measureText(text).width
}

/**
 * Options for calculating column widths
 */
export interface CalculateColumnWidthsOptions<TData> {
  /** Table data */
  data: TData[]
  /** Column definitions with accessorKey */
  columns: Array<{
    accessorKey?: string
    id?: string
    size?: number
    minSize?: number
    maxSize?: number
    /** If true, skip auto-sizing for this column */
    enableAutoSize?: boolean
    meta?: {
      /** If false, skip auto-sizing for this rendered column */
      enableAutoSize?: boolean
    }
    /** If false, header floor does not need to reserve sort icon width */
    enableSorting?: boolean
  }>
  /** Font to use for measurement */
  font?: string
  /** Padding to add to each cell (in pixels) */
  cellPadding?: number
  /** Header font (usually slightly different from cell font) */
  headerFont?: string
  /** Header labels for columns (keyed by accessorKey or id) */
  headerLabels?: Record<string, string>
  /** Maximum number of rows to sample (for performance) */
  maxSampleRows?: number
  /** Locale for date formatting */
  locale?: string
}

/**
 * Calculate optimal column widths based on content
 * Returns a map of column id -> calculated width
 */
/**
 * Check if a string looks like an ISO date
 */
function isISODateString(value: string): boolean {
  // Match ISO 8601 format: 2024-01-09T12:00:00.000Z or 2024-01-09T12:00:00
  return /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(value)
}

/**
 * Format date for display (matching the app's date format)
 */
function formatDateForMeasurement(dateString: string, locale: string): string {
  try {
    return new Date(dateString).toLocaleString(locale, {
      year: "numeric",
      month: "numeric",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    })
  } catch {
    return dateString
  }
}

export function calculateColumnWidths<TData extends Record<string, unknown>>({
  data,
  columns,
  font = UI_SANS_MEASURE_FONT_14,
  cellPadding = 32, // Default padding for cell content
  headerFont = UI_SANS_MEASURE_FONT_14_MEDIUM,
  headerLabels = {},
  maxSampleRows = 100,
  locale = 'zh-CN',
}: CalculateColumnWidthsOptions<TData>): Record<string, number> {
  const widths: Record<string, number> = {}
  
  // Sample data for performance (don't measure all rows if there are too many)
  const sampleData = data.slice(0, maxSampleRows)
  
  for (const column of columns) {
    const columnId = column.accessorKey || column.id
    if (!columnId) continue
    
    // Skip columns that explicitly disable auto-sizing
    if (column.enableAutoSize === false || column.meta?.enableAutoSize === false) {
      if (column.size) {
        widths[columnId] = column.size
      }
      continue
    }
    
    // Start with header width
    const headerLabel = headerLabels[columnId] || columnId
    let maxWidth = measureTextWidth(headerLabel, headerFont) + cellPadding
    
    // Measure content width for each row
    for (const row of sampleData) {
      const value = row[columnId]
      if (value == null) continue
      
      // Convert value to string for measurement
      let textValue: string
      if (typeof value === 'string') {
        // Check if it's a date string and format it
        if (isISODateString(value)) {
          textValue = formatDateForMeasurement(value, locale)
        } else {
          textValue = value
        }
      } else if (typeof value === 'number') {
        textValue = String(value)
      } else if (Array.isArray(value)) {
        // For arrays, join with comma (rough estimate)
        textValue = value.join(', ')
      } else if (typeof value === 'object') {
        // Skip complex objects - they need custom renderers
        continue
      } else {
        textValue = String(value)
      }
      
      const contentWidth = measureTextWidth(textValue, font) + cellPadding
      maxWidth = Math.max(maxWidth, contentWidth)
    }
    
    // Apply min/max constraints
    if (column.minSize) {
      maxWidth = Math.max(maxWidth, column.minSize)
    }
    if (column.maxSize) {
      maxWidth = Math.min(maxWidth, column.maxSize)
    }
    
    widths[columnId] = Math.ceil(maxWidth)
  }
  
  return widths
}

const TABLE_HEADER_TEXT_FONT = UI_SANS_MEASURE_FONT_12_MEDIUM
const TABLE_HEADER_CELL_HORIZONTAL_PADDING_PX = 16
const TABLE_HEADER_TRIGGER_HORIZONTAL_PADDING_PX = 16
const TABLE_HEADER_SORT_ICON_WIDTH_PX = 16
const TABLE_HEADER_SORT_ICON_GAP_PX = 4
const TABLE_HEADER_MEASUREMENT_BUFFER_PX = 8
const TABLE_BADGE_TEXT_FONT = UI_SANS_MEASURE_FONT_12_MEDIUM
const TABLE_BADGE_HORIZONTAL_PADDING_PX = 18
const TABLE_BADGE_CELL_HORIZONTAL_PADDING_PX = 16
const TABLE_BADGE_MEASUREMENT_BUFFER_PX = 48

export function getColumnHeaderMinWidthPx(
  title: string,
  options?: {
    headerFont?: string
    cellHorizontalPaddingPx?: number
    triggerHorizontalPaddingPx?: number
    horizontalPaddingPx?: number
    includeSortIcon?: boolean
    sortIconWidthPx?: number
    sortIconGapPx?: number
    measurementBufferPx?: number
  }
): number {
  const measuredTitleWidth = measureTextWidth(title, options?.headerFont ?? TABLE_HEADER_TEXT_FONT)
  const cellHorizontalPaddingPx =
    options?.cellHorizontalPaddingPx ?? TABLE_HEADER_CELL_HORIZONTAL_PADDING_PX
  const triggerHorizontalPaddingPx =
    options?.triggerHorizontalPaddingPx ??
    options?.horizontalPaddingPx ??
    TABLE_HEADER_TRIGGER_HORIZONTAL_PADDING_PX
  const includeSortIcon = options?.includeSortIcon ?? true
  const sortIconWidthPx = options?.sortIconWidthPx ?? TABLE_HEADER_SORT_ICON_WIDTH_PX
  const sortIconGapPx = options?.sortIconGapPx ?? TABLE_HEADER_SORT_ICON_GAP_PX
  const measurementBufferPx = options?.measurementBufferPx ?? TABLE_HEADER_MEASUREMENT_BUFFER_PX

  return Math.ceil(
    measuredTitleWidth +
      cellHorizontalPaddingPx +
      triggerHorizontalPaddingPx +
      (includeSortIcon ? sortIconWidthPx + sortIconGapPx : 0) +
      measurementBufferPx
  )
}

export function getBadgeMinWidthPx(
  label: string,
  options?: {
    font?: string
    horizontalPaddingPx?: number
    cellHorizontalPaddingPx?: number
    measurementBufferPx?: number
  }
): number {
  return Math.ceil(
    measureTextWidth(label, options?.font ?? TABLE_BADGE_TEXT_FONT) +
      (options?.horizontalPaddingPx ?? TABLE_BADGE_HORIZONTAL_PADDING_PX) +
      (options?.cellHorizontalPaddingPx ?? TABLE_BADGE_CELL_HORIZONTAL_PADDING_PX) +
      (options?.measurementBufferPx ?? TABLE_BADGE_MEASUREMENT_BUFFER_PX)
  )
}

/**
 * Hook-friendly version that returns initial column sizing state
 */
export function getInitialColumnSizing<TData extends Record<string, unknown>>(
  options: CalculateColumnWidthsOptions<TData>
): Record<string, number> {
  // Only run on client side
  if (typeof window === 'undefined') {
    return {}
  }
  return calculateColumnWidths(options)
}
