"use client"

import * as React from "react"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { ChevronDown, ChevronUp } from "@/components/icons"

// ============================================================================
// i18n Context for expandable components
// ============================================================================

interface ExpandableI18n {
  expand: string
  collapse: string
}

const defaultI18n: ExpandableI18n = {
  expand: "Expand",
  collapse: "Collapse",
}

const ExpandableI18nContext = React.createContext<ExpandableI18n>(defaultI18n)

/**
 * Provider for expandable component i18n
 * Wrap your table or page with this to provide translations
 */
export function ExpandableI18nProvider({
  children,
  expand,
  collapse,
}: {
  children: React.ReactNode
  expand: string
  collapse: string
}) {
  const value = React.useMemo(() => ({ expand, collapse }), [expand, collapse])
  return (
    <ExpandableI18nContext.Provider value={value}>
      {children}
    </ExpandableI18nContext.Provider>
  )
}

function useExpandableI18n() {
  return React.useContext(ExpandableI18nContext)
}

// ============================================================================
// ExpandableCell component
// ============================================================================

export interface ExpandableCellProps {
  /** Value to display */
  value: string | null | undefined
  /** Display variant */
  variant?: "text" | "url" | "mono" | "muted"
  /** Maximum display lines, default 3 */
  maxLines?: number
  /** Additional CSS class name */
  className?: string
  /** Placeholder when value is empty */
  placeholder?: string
  /** Expand button text (overrides context) */
  expandLabel?: string
  /** Collapse button text (overrides context) */
  collapseLabel?: string
}

/**
 * Unified expandable cell component
 * 
 * Features:
 * - Default display up to 3 lines (configurable)
 * - Auto-detect content overflow
 * - Show expand/collapse button only when content overflows
 * - Supports text, url, mono, muted variants
 */
export function ExpandableCell({
  value,
  variant = "text",
  maxLines = 3,
  className,
  placeholder = "-",
  expandLabel,
  collapseLabel,
}: ExpandableCellProps) {
  const i18n = useExpandableI18n()
  const [expanded, setExpanded] = React.useState(false)
  const [isOverflowing, setIsOverflowing] = React.useState(false)
  const contentRef = React.useRef<HTMLDivElement>(null)

  const expand = expandLabel ?? i18n.expand
  const collapse = collapseLabel ?? i18n.collapse

  // Detect content overflow
  React.useEffect(() => {
    const el = contentRef.current
    if (!el) return

    const checkOverflow = () => {
      // Compare scrollHeight and clientHeight to determine overflow
      setIsOverflowing(el.scrollHeight > el.clientHeight + 1)
    }

    checkOverflow()

    // Listen for window size changes
    const resizeObserver = new ResizeObserver(checkOverflow)
    resizeObserver.observe(el)

    return () => resizeObserver.disconnect()
  }, [value, expanded])

  if (!value) {
    return <span className="text-muted-foreground text-sm">{placeholder}</span>
  }

  const lineClampClass = {
    1: "line-clamp-1",
    2: "line-clamp-2",
    3: "line-clamp-3",
    4: "line-clamp-4",
    5: "line-clamp-5",
    6: "line-clamp-6",
  }[maxLines] || "line-clamp-3"

  return (
    <div className="flex flex-col gap-1">
      <div
        ref={contentRef}
        className={cn(
          "text-sm break-all leading-relaxed whitespace-pre-wrap",
          variant === "mono" && "font-mono text-xs text-muted-foreground",
          variant === "url" && "text-muted-foreground",
          variant === "muted" && "text-muted-foreground",
          !expanded && lineClampClass,
          className
        )}
      >
        {value}
      </div>
      {(isOverflowing || expanded) && (
        <button type="button"
          onClick={() => setExpanded(!expanded)}
          className="inline-flex items-center gap-0.5 text-xs text-primary hover:underline self-start"
        >
          {expanded ? (
            <>
              <ChevronUp className="h-3 w-3" />
              <span>{collapse}</span>
            </>
          ) : (
            <>
              <ChevronDown className="h-3 w-3" />
              <span>{expand}</span>
            </>
          )}
        </button>
      )}
    </div>
  )
}

/**
 * URL-specific expandable cell
 */
export function ExpandableUrlCell(props: Omit<ExpandableCellProps, "variant">) {
  return <ExpandableCell {...props} variant="url" />
}

/**
 * Code/monospace font expandable cell
 */
export function ExpandableMonoCell(props: Omit<ExpandableCellProps, "variant">) {
  return <ExpandableCell {...props} variant="mono" />
}

// ============================================================================
// Badge list related components
// ============================================================================

export interface BadgeItem {
  id: number | string
  name: string
}

export interface ExpandableBadgeListProps {
  /** Badge item list */
  items: BadgeItem[] | null | undefined
  /** Default display count, default 2 */
  maxVisible?: number
  /** Badge variant */
  variant?: "default" | "secondary" | "outline" | "destructive"
  /** Placeholder when value is empty */
  placeholder?: string
  /** Additional CSS class name */
  className?: string
  /** Callback when Badge is clicked */
  onItemClick?: (item: BadgeItem) => void
}

/**
 * Expandable Badge list component
 * 
 * Features:
 * - Default display first N Badges (configurable)
 * - Show expand button when exceeding count
 * - Click expand button to show all Badges
 * - Show collapse button after expansion
 */
export function ExpandableBadgeList({
  items,
  maxVisible = 2,
  variant = "secondary",
  placeholder = "-",
  className,
  onItemClick,
}: ExpandableBadgeListProps) {
  const i18n = useExpandableI18n()
  const [expanded, setExpanded] = React.useState(false)

  if (!items || items.length === 0) {
    return <span className="text-sm text-muted-foreground">{placeholder}</span>
  }

  const hasMore = items.length > maxVisible
  const displayItems = expanded ? items : items.slice(0, maxVisible)

  return (
    <div className={cn("flex flex-wrap items-center gap-1", className)}>
      {displayItems.map((item) => (
        onItemClick ? (
          <Badge
            asChild
            key={item.id}
            variant={variant}
            className="text-xs hover:bg-accent"
            title={item.name}
          >
            <button
              type="button"
              onClick={() => onItemClick(item)}
            >
              {item.name}
            </button>
          </Badge>
        ) : (
          <Badge
            key={item.id}
            variant={variant}
            className="text-xs"
            title={item.name}
          >
            {item.name}
          </Badge>
        )
      ))}
      {hasMore && (
        <button type="button"
          onClick={() => setExpanded(!expanded)}
          className="inline-flex items-center gap-0.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          {expanded ? (
            <>
              <ChevronUp className="h-3 w-3" />
              <span>{i18n.collapse}</span>
            </>
          ) : (
            <>
              <ChevronDown className="h-3 w-3" />
              <span>{i18n.expand}</span>
            </>
          )}
        </button>
      )}
    </div>
  )
}

// ============================================================================
// String list related components
// ============================================================================

export interface ExpandableTagListProps {
  /** Tag list */
  items: string[] | null | undefined
  /** Maximum display lines, default 2 */
  maxLines?: number
  /** Badge variant */
  variant?: "default" | "secondary" | "outline" | "destructive"
  /** Placeholder when value is empty */
  placeholder?: string
  /** Additional CSS class name */
  className?: string
}

/**
 * Expandable tag list component (for string arrays)
 * 
 * Features:
 * - Auto-detect overflow based on line count
 * - Responsive: shows more tags when container is wider
 * - Show expand/collapse button only when content overflows
 */
export function ExpandableTagList({
  items,
  maxLines = 2,
  variant = "outline",
  placeholder = "-",
  className,
}: ExpandableTagListProps) {
  const i18n = useExpandableI18n()
  const [expanded, setExpanded] = React.useState(false)
  const [isOverflowing, setIsOverflowing] = React.useState(false)
  const containerRef = React.useRef<HTMLDivElement>(null)

  // Detect content overflow
  React.useEffect(() => {
    const el = containerRef.current
    if (!el || expanded) return

    const checkOverflow = () => {
      setIsOverflowing(el.scrollHeight > el.clientHeight + 1)
    }

    checkOverflow()

    const resizeObserver = new ResizeObserver(checkOverflow)
    resizeObserver.observe(el)

    return () => resizeObserver.disconnect()
  }, [items, expanded])

  if (!items || items.length === 0) {
    return <span className="text-sm text-muted-foreground">{placeholder}</span>
  }

  // Line clamp class based on maxLines
  const lineClampClass = {
    1: "line-clamp-1",
    2: "line-clamp-2",
    3: "line-clamp-3",
    4: "line-clamp-4",
  }[maxLines] || "line-clamp-2"

  return (
    <div className="flex flex-col gap-1">
      <div
        ref={containerRef}
        className={cn(
          "flex flex-wrap items-start gap-1",
          !expanded && lineClampClass,
          className
        )}
      >
        {items.map((item, index) => (
          <Badge
            key={`${item}-${index}`}
            variant={variant}
            className="text-xs"
            title={item}
          >
            {item}
          </Badge>
        ))}
      </div>
      {(isOverflowing || expanded) && (
        <button type="button"
          onClick={() => setExpanded(!expanded)}
          className="inline-flex items-center gap-0.5 text-xs text-muted-foreground hover:text-foreground transition-colors self-start"
        >
          {expanded ? (
            <>
              <ChevronUp className="h-3 w-3" />
              <span>{i18n.collapse}</span>
            </>
          ) : (
            <>
              <ChevronDown className="h-3 w-3" />
              <span>{i18n.expand}</span>
            </>
          )}
        </button>
      )}
    </div>
  )
}
