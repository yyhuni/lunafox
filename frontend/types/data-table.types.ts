import type { ColumnDef, SortingState, VisibilityState, Table, Column, RowData } from "@tanstack/react-table"
import type { CSSProperties, ReactNode } from "react"
import type { SelectedRowActionBarAction } from "@/components/shared/data-table/selected-row-action-bar"

export type DataTableColumnWidthPolicy =
  | { mode: "fixed" }
  | { mode: "flex"; flex: number; fill?: boolean }

/**
 * Extend TanStack Table's ColumnMeta type
 * Used to store column metadata such as title
 */
declare module '@tanstack/react-table' {
  interface ColumnMeta<TData extends RowData, TValue> {
  /** Column title, used for column header and column visibility control */
  title?: string
  /** Keep the column pinned to the right edge of the scroll container when horizontal overflow is present. */
  stickyRight?: boolean
  /** Reserve enough width for a single visible badge label to remain fully visible. */
  singleBadge?: boolean
  /** Derive the single-badge label from a row when the displayed badge text differs from the raw accessor value. */
  singleBadgeValue?: (row: TData) => string | null | undefined
  /** Complete rendered labels for a categorical single-badge column, including the first frame before rows arrive. */
  singleBadgeValues?: readonly string[]
  /** Opt a rendered summary/badge column out of content measurement auto-sizing. */
  enableAutoSize?: boolean
  /** Canonical backend orderBy field required before a server-paginated table may expose sorting. */
  orderBy?: string
  /** Review note for the index, query plan, search index, materialized view, or bounded scale supporting server sorting. */
  serverSortPerformance?: string
  /** First direction used when a server-sortable header is activated from the endpoint default order. */
  firstSortDirection?: 'asc' | 'desc'
  /** Semantic width behavior for shared fixed-layout business tables. */
  widthPolicy?: DataTableColumnWidthPolicy
  /** Type anchor to avoid unused generic warnings */
  __type?: { data: TData; value: TValue }
}
}

/**
 * Pagination state
 */
export interface PaginationState {
  pageIndex: number
  pageSize: number
}

/**
 * Server-side pagination info
 */
export interface PaginationInfo {
  total: number
  totalPages: number
  page: number
  pageSize: number
}

/**
 * Cursor-backed lists can only navigate to tokens returned for adjacent pages.
 * They must not expose numbered jumps that would imply an unavailable token.
 */
export interface CursorPaginationNavigation {
  mode: "cursor"
  /** Whether the current cursor query can be reset to its first response. */
  canFirstPage: boolean
  /** Clears the current query token cache and requests the first response. */
  onFirstPage?: () => void
  canPreviousPage: boolean
  canNextPage: boolean
}

/**
 * Cursor-backed responses may report a result total, but that total cannot
 * authorize a page transition because opaque tokens are only returned by the
 * service for adjacent pages.
 */
export interface CursorPaginationSummary {
  total: number
}

/**
 * Search mode type
 */
export type SearchMode = 'simple' | 'smart'
export type DataTableToolbarDensity = 'compact' | 'standard'
export type DataTableRowDensity = 'dense' | 'comfortable'
export type DataTableLoadingRowHeightEstimate = CSSProperties["height"] | ((rowIndex: number) => CSSProperties["height"])
export type DataTableSortingMode = 'client' | 'server' | 'none'

/**
 * Paired first-frame regions emitted by the shared table shell. A custom
 * contract must name all three regions so one ContentHandoff owner cannot
 * accidentally compare a mixed generic/feature-specific table surface.
 */
export interface DataTableLoadingSlots {
  toolbar: string
  body: string
  pagination: string
}

/**
 * Filter field definition
 * Note: description is required, consistent with SmartFilterInput component
 */
export interface FilterField {
  key: string
  label: string
  description: string
}

/**
 * Export option
 */
export interface ExportOption {
  key: string
  label: string
  icon?: ReactNode
  onClick: () => void
  disabled?: boolean | ((selectedCount: number) => boolean)
}

/**
 * Delete confirmation dialog configuration
 */
export interface DeleteConfirmationConfig {
  title?: string
  description?: string | ((count: number) => string)
  confirmLabel?: string
  cancelLabel?: string
}

/**
 * Grouped state config for UnifiedDataTable
 */
export interface UnifiedDataTableStateConfig<TData> {
  // Pagination
  pagination?: PaginationState
  setPagination?: React.Dispatch<React.SetStateAction<PaginationState>>
  paginationInfo?: PaginationInfo
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  onPaginationChange?: (pagination: PaginationState) => void

  // Search state
  searchValue?: string
  isSearching?: boolean

  // Selection
  rowSelection?: Record<string, boolean>
  onRowSelectionChange?: (selection: Record<string, boolean>) => void

  // Column control
  columnVisibility?: VisibilityState
  onColumnVisibilityChange?: (visibility: VisibilityState) => void

  // Sorting
  sortingMode?: DataTableSortingMode
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
  defaultSorting?: SortingState

  // Selection callback
  onSelectionChange?: (selectedRows: TData[]) => void

  // Optional external selected rows mirror. When a page clears this array after
  // confirmation or route-level state changes, the shared table resets its
  // internal row selection to keep checkboxes, pagination summary, and action
  // bars in sync.
  selectedRows?: TData[]
}

/**
 * Grouped UI config for UnifiedDataTable
 */
export interface UnifiedDataTableUIConfig {
  // Pagination UI
  hidePagination?: boolean
  pageSizeOptions?: number[]

  // Toolbar UI
  hideToolbar?: boolean
  showColumnVisibility?: boolean
  toolbarDensity?: DataTableToolbarDensity
  rowDensity?: DataTableRowDensity
  toolbarLeft?: ReactNode
  toolbarRight?: ReactNode

  // Search/filter UI
  searchPlaceholder?: string
  filterFields?: FilterField[]
  filterExamples?: string[]

  // Empty state
  emptyMessage?: string
  emptyComponent?: ReactNode

  // Loading presentation
  loading?: boolean
  loadingPresentation?: "rows" | "initial"
  initialLoadingToolbarFilterCount?: number
  loadingRowCount?: number
  loadingRowHeightEstimate?: DataTableLoadingRowHeightEstimate
  /**
   * Opt-in loading-only table-shell reservation for a first-screen handoff.
   * While `loading` is true, the value uses the actual shared header and row
   * rhythm. Resolved tables deliberately release that minimum height and use
   * their returned rows or the existing empty-state row. Virtual-scroll
   * measured-box estimates intentionally do not affect this reservation.
   */
  stableSurfaceRowCount?: number
  /**
   * Stable geometry names for a table inside a ContentHandoff owner. Routes
   * with more than one table must provide distinct complete slot sets.
   */
  loadingSlots?: DataTableLoadingSlots

  // Styling
  className?: string
  tableClassName?: string
}

/**
 * Grouped behavior config for UnifiedDataTable
 */
export interface UnifiedDataTableBehaviorConfig {
  // Search/filter behavior
  searchMode?: SearchMode
  onSearch?: (value: string) => void

  // Selection behavior
  enableRowSelection?: boolean

  // Auto column sizing behavior
  enableAutoColumnSizing?: boolean

  // Legacy compatibility input. New multi-flex business tables declare widthPolicy in column metadata.
  expandColumnIds?: string[]

  // Column layout strategy for semantic-width business tables
  columnLayout?: 'auto' | 'fixed'

  // Row click callback
  onRowClick?: (row: unknown) => void
  // Intent signal for bounded preloading before a row is activated
  onRowIntent?: (row: unknown) => void
  getRowActionLabel?: (row: unknown) => string
}

/**
 * Grouped action config for UnifiedDataTable
 */
export interface UnifiedDataTableActionConfig {
  // Selected-row actions
  selectedRowActions?: SelectedRowActionBarAction[]

  // Bulk operations
  onBulkDelete?: () => void
  bulkDeleteLabel?: string
  showBulkDelete?: boolean

  // Add operation
  onAddNew?: () => void
  onAddHover?: () => void
  addButtonLabel?: string
  showAddButton?: boolean

  // Bulk add operation
  onBulkAdd?: () => void
  bulkAddLabel?: string
  showBulkAdd?: boolean

  // Refresh operation
  onRefresh?: () => void
  refreshLabel?: string
  showRefresh?: boolean
  isRefreshing?: boolean

  // Export operations
  exportOptions?: ExportOption[]

  // Confirmation dialog
  deleteConfirmation?: DeleteConfirmationConfig
}

/**
 * Unified data table component props
 */
export interface UnifiedDataTableProps<TData> {
  // Core data
  data: TData[]
  columns: ColumnDef<TData, unknown>[]
  getRowId?: (row: TData, index: number) => string

  // Grouped config (single source of truth)
  state?: UnifiedDataTableStateConfig<TData>
  ui?: UnifiedDataTableUIConfig
  behavior?: UnifiedDataTableBehaviorConfig
  actions?: UnifiedDataTableActionConfig
}

/**
 * Toolbar component props
 */
export interface DataTableToolbarProps {
  // Search
  searchMode?: SearchMode
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  filterFields?: FilterField[]
  filterExamples?: string[]

  // Left side custom content
  leftContent?: ReactNode

  // Right side actions
  children?: ReactNode

  // Styling
  className?: string
}

/**
 * Pagination component props
 */
export interface DataTablePaginationProps<TData> {
  table: Table<TData>
  paginationInfo?: PaginationInfo
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
  pageSizeOptions?: number[]
  onPaginationChange?: (pagination: PaginationState) => void
  className?: string
}

/**
 * Column header component props
 */
export interface DataTableColumnHeaderProps<TData, TValue> {
  column: Column<TData, TValue>
  title: string
  className?: string
}
