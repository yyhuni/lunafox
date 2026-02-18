/**
 * Table utility functions
 * Provides column width calculation and other table-related utilities
 */

// Cache for text measurement context
let measureContext: CanvasRenderingContext2D | null = null

/**
 * Get or create a canvas context for measuring text width
 */
function getMeasureContext(): CanvasRenderingContext2D {
  if (!measureContext) {
    const canvas = document.createElement('canvas')
    measureContext = canvas.getContext('2d')!
  }
  return measureContext
}

/**
 * Measure text width using canvas
 * @param text - Text to measure
 * @param font - CSS font string (e.g., "14px Inter, sans-serif")
 * @returns Text width in pixels
 */
export function measureTextWidth(text: string, font: string = '14px Inter, system-ui, sans-serif'): number {
  const ctx = getMeasureContext()
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
  font = '14px Inter, system-ui, sans-serif',
  cellPadding = 32, // Default padding for cell content
  headerFont = '500 14px Inter, system-ui, sans-serif',
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
    if (column.enableAutoSize === false) {
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
