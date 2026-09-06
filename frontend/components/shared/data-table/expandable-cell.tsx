"use client";
import { textRole } from "@/lib/typography";
import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ChevronDown, ChevronUp } from "@/components/icons";
// ============================================================================
// i18n Context for expandable components
// ============================================================================
interface ExpandableI18n {
    expand: string;
    collapse: string;
}
const defaultI18n: ExpandableI18n = {
    expand: "Expand",
    collapse: "Collapse",
};
const ExpandableI18nContext = React.createContext<ExpandableI18n>(defaultI18n);
/**
 * Provider for expandable component i18n
 * Wrap your table or page with this to provide translations
 */
export function ExpandableI18nProvider({ children, expand, collapse, }: {
    children: React.ReactNode;
    expand: string;
    collapse: string;
}) {
    const value = React.useMemo(() => ({ expand, collapse }), [expand, collapse]);
    return (<ExpandableI18nContext.Provider value={value}>
      {children}
    </ExpandableI18nContext.Provider>);
}
function useExpandableI18n() {
    return React.useContext(ExpandableI18nContext);
}
// ============================================================================
// ExpandableCell component
// ============================================================================
export interface ExpandableCellProps {
    /** Value to display */
    value: unknown;
    /** Display variant */
    variant?: "text" | "url" | "mono" | "muted";
    /** Shared typography role used by the cell content */
    textRoleName?: "body" | "tableCellPrimary" | "tableCellSecondary";
    /** Maximum display lines, default 3 */
    maxLines?: number;
    /** Additional CSS class name */
    className?: string;
    /** Placeholder when value is empty */
    placeholder?: string;
    /** Expand button text (overrides context) */
    expandLabel?: string;
    /** Collapse button text (overrides context) */
    collapseLabel?: string;
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
export function ExpandableCell({ value, variant = "text", textRoleName = "body", maxLines = 3, className, placeholder = "-", expandLabel, collapseLabel, }: ExpandableCellProps) {
    const i18n = useExpandableI18n();
    const [expanded, setExpanded] = React.useState(false);
    const [isOverflowing, setIsOverflowing] = React.useState(false);
    const contentRef = React.useRef<HTMLDivElement>(null);
    const expand = expandLabel ?? i18n.expand;
    const collapse = collapseLabel ?? i18n.collapse;
    const contentTextRole = textRole[textRoleName];
    const monoTextRole = textRoleName === "body" ? textRole.code : "font-mono tabular-nums";
    const placeholderTextRole = textRoleName.startsWith("tableCell") ? textRole.tableCellSecondary : textRole.bodySubtle;
    // Detect content overflow
    React.useEffect(() => {
        const el = contentRef.current;
        if (!el)
            return;
        const checkOverflow = () => {
            // Compare scrollHeight and clientHeight to determine overflow
            setIsOverflowing(el.scrollHeight > el.clientHeight + 1);
        };
        checkOverflow();
        // Listen for window size changes
        const resizeObserver = new ResizeObserver(checkOverflow);
        resizeObserver.observe(el);
        return () => resizeObserver.disconnect();
    }, [value, expanded]);
    if (value === null || value === undefined || value === "") {
        return <span className={placeholderTextRole}>{placeholder}</span>;
    }
    const displayValue = typeof value === "string" ? value : JSON.stringify(value, null, 2);
    const lineClampClass = {
        1: "line-clamp-1",
        2: "line-clamp-2",
        3: "line-clamp-3",
        4: "line-clamp-4",
        5: "line-clamp-5",
        6: "line-clamp-6",
    }[maxLines] || "line-clamp-3";
    return (<div className="flex flex-col gap-1">
      <div ref={contentRef} className={cn("break-all whitespace-pre-wrap", contentTextRole, variant === "mono" && monoTextRole, variant === "url" && "text-muted-foreground", variant === "muted" && "text-muted-foreground", !expanded && lineClampClass, className)}>
        {displayValue}
      </div>
      {(isOverflowing || expanded) && (<button type="button" onClick={() => setExpanded(!expanded)} className={cn("gap-0.5 hover:underline inline-flex items-center self-start text-primary", textRole.caption)}>
          {expanded ? (<>
              <ChevronUp className="h-3 w-3"/>
              <span>{collapse}</span>
            </>) : (<>
              <ChevronDown className="h-3 w-3"/>
              <span>{expand}</span>
            </>)}
        </button>)}
    </div>);
}
/**
 * URL-specific expandable cell
 */
export function ExpandableUrlCell(props: Omit<ExpandableCellProps, "variant">) {
    return <ExpandableCell {...props} variant="url"/>;
}
/**
 * Code/monospace font expandable cell
 */
export function ExpandableMonoCell(props: Omit<ExpandableCellProps, "variant">) {
    return <ExpandableCell {...props} variant="mono"/>;
}
// ============================================================================
// Badge list related components
// ============================================================================
export interface BadgeItem {
    id: number | string;
    name: string;
}
export interface ExpandableBadgeListProps {
    /** Badge item list */
    items: BadgeItem[] | null | undefined;
    /** Default display count upper bound, default 2 */
    maxVisible?: number;
    /** Keep preview on one line and only reveal badges that fit within the width budget */
    singleLinePreview?: boolean;
    /** Let the capped preview wrap when the route must expose each visible item */
    wrapPreview?: boolean;
    /** Badge variant */
    variant?: "default" | "secondary" | "outline" | "destructive";
    /** Placeholder when value is empty */
    placeholder?: string;
    /** Additional CSS class name */
    className?: string;
    /** Callback when Badge is clicked */
    onItemClick?: (item: BadgeItem) => void;
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
export function ExpandableBadgeList({ items, maxVisible = 2, singleLinePreview = false, wrapPreview = false, variant = "secondary", placeholder = "-", className, onItemClick, }: ExpandableBadgeListProps) {
    const i18n = useExpandableI18n();
    const [expanded, setExpanded] = React.useState(false);
    const [resolvedVisibleCount, setResolvedVisibleCount] = React.useState(maxVisible);
    const previewContainerRef = React.useRef<HTMLDivElement>(null);
    const measurementRef = React.useRef<HTMLDivElement>(null);
    const itemCount = items?.length ?? 0;
    const maxPreviewCount = Math.min(itemCount, maxVisible);
    React.useEffect(() => {
        if (itemCount === 0) {
            setResolvedVisibleCount(maxVisible);
            return;
        }
        if (!singleLinePreview || expanded) {
            setResolvedVisibleCount(maxPreviewCount);
            return;
        }
        const previewContainer = previewContainerRef.current;
        const measurement = measurementRef.current;
        if (!previewContainer || !measurement) {
            setResolvedVisibleCount(maxPreviewCount);
            return;
        }
        const measurePreview = () => {
            const availableWidth = previewContainer.clientWidth;
            const badgeNodes = Array.from(measurement.querySelectorAll<HTMLElement>('[data-badge-measure="item"]'));
            const expanderNode = measurement.querySelector<HTMLElement>('[data-badge-measure="expander"]');
            if (availableWidth <= 0 || badgeNodes.length === 0) {
                setResolvedVisibleCount(maxPreviewCount);
                return;
            }
            const computedStyle = window.getComputedStyle(measurement);
            const gap = Number.parseFloat(computedStyle.columnGap || computedStyle.gap || "0") || 0;
            const expanderWidth = expanderNode?.offsetWidth ?? 0;
            let usedWidth = 0;
            let nextVisibleCount = 0;
            for (let index = 0; index < badgeNodes.length; index += 1) {
                const badgeWidth = badgeNodes[index]?.offsetWidth ?? 0;
                const nextUsedWidth = usedWidth + (nextVisibleCount > 0 ? gap : 0) + badgeWidth;
                const hiddenItemsRemain = itemCount > index + 1;
                const reservedExpanderWidth = hiddenItemsRemain
                    ? expanderWidth + (index >= 0 ? gap : 0)
                    : 0;
                if (nextUsedWidth + reservedExpanderWidth <= availableWidth || nextVisibleCount === 0) {
                    usedWidth = nextUsedWidth;
                    nextVisibleCount += 1;
                    continue;
                }
                break;
            }
            setResolvedVisibleCount(Math.max(1, Math.min(nextVisibleCount, maxPreviewCount)));
        };
        measurePreview();
        const resizeObserver = new ResizeObserver(measurePreview);
        resizeObserver.observe(previewContainer);
        resizeObserver.observe(measurement);
        return () => resizeObserver.disconnect();
    }, [expanded, itemCount, maxPreviewCount, maxVisible, singleLinePreview]);
    if (!items || itemCount === 0) {
        return <span className={textRole.bodySubtle}>{placeholder}</span>;
    }
    const previewCount = expanded
        ? itemCount
        : singleLinePreview
            ? Math.max(1, Math.min(resolvedVisibleCount, maxPreviewCount))
            : maxPreviewCount;
    const hasMore = items.length > previewCount;
    const displayItems = expanded ? items : items.slice(0, previewCount);
    const renderBadge = (item: BadgeItem, inert = false) => (onItemClick && !inert ? (<Badge key={item.id} variant={variant} className="hover:bg-accent min-w-0 max-w-full text-xs" title={item.name} render={<button type="button" onClick={() => onItemClick(item)}/>}>
          <span className="block min-w-0 truncate text-left">{item.name}</span>
        </Badge>) : (<Badge key={item.id} variant={variant} className={cn(textRole.badgeSubtle, "min-w-0 max-w-full")} title={item.name}>
        <span className="block min-w-0 truncate text-left">{item.name}</span>
      </Badge>));
    return (<div className={cn("relative flex min-w-0 flex-col gap-1", className)}>
      <div ref={previewContainerRef} className={cn("min-w-0 items-center gap-1", expanded || wrapPreview ? "flex flex-wrap" : "flex")} style={{
            whiteSpace: expanded || wrapPreview ? "normal" : "nowrap",
            overflow: expanded || wrapPreview ? "visible" : "hidden",
        }}>
        {displayItems.map((item) => renderBadge(item))}
        {(hasMore || expanded) && (<button type="button" onClick={() => setExpanded(!expanded)} className={cn("gap-0.5 hover:text-foreground inline-flex shrink-0 items-center transition-colors", textRole.caption)}>
            {expanded ? (<>
                <ChevronUp className="h-3 w-3"/>
                <span>{i18n.collapse}</span>
              </>) : (<>
                <ChevronDown className="h-3 w-3"/>
                <span>{i18n.expand}</span>
              </>)}
          </button>)}
      </div>
      {singleLinePreview && !expanded && (<div ref={measurementRef} aria-hidden="true" className="pointer-events-none absolute left-0 top-0 flex items-center gap-1" style={{
                visibility: "hidden",
                whiteSpace: expanded ? "normal" : "nowrap",
                height: 0,
                overflow: expanded ? "visible" : "hidden",
            }}>
          {items.slice(0, maxPreviewCount).map((item) => (<div key={item.id} data-badge-measure="item" className="shrink-0">
              {renderBadge(item, true)}
            </div>))}
          {items.length > 1 && (<div data-badge-measure="expander" className="inline-flex shrink-0 items-center gap-0.5">
              <ChevronDown className="h-3 w-3"/>
              <span className={textRole.caption}>{i18n.expand}</span>
            </div>)}
        </div>)}
    </div>);
}
// ============================================================================
// String list related components
// ============================================================================
export interface ExpandableTagListProps {
    /** Tag list */
    items: string[] | null | undefined;
    /** Default display count upper bound when using single-line preview */
    maxVisible?: number;
    /** Keep preview on one line and fit within the current width budget */
    singleLinePreview?: boolean;
    /** Maximum display lines, default 2 */
    maxLines?: number;
    /** Badge variant */
    variant?: "default" | "secondary" | "outline" | "destructive";
    /** Placeholder when value is empty */
    placeholder?: string;
    /** Additional CSS class name */
    className?: string;
}
/**
 * Expandable tag list component (for string arrays)
 *
 * Features:
 * - Auto-detect overflow based on line count
 * - Responsive: shows more tags when container is wider
 * - Show expand/collapse button only when content overflows
 */
export function ExpandableTagList({ items, maxVisible = 2, singleLinePreview = false, maxLines = 2, variant = "outline", placeholder = "-", className, }: ExpandableTagListProps) {
    if (!items || items.length === 0) {
        return <span className={textRole.tableCellSecondary}>{placeholder}</span>;
    }
    if (singleLinePreview) {
        return (<ExpandableBadgeList items={items.map((item, index) => ({ id: `${item}-${index}`, name: item }))} maxVisible={maxVisible} singleLinePreview variant={variant} placeholder={placeholder} className={className}/>);
    }
    return (<ExpandableWrappedTagList items={items} maxLines={maxLines} variant={variant} className={className}/>);
}
function ExpandableWrappedTagList({ items, maxLines, variant, className, }: {
    items: string[];
    maxLines: number;
    variant: "default" | "secondary" | "outline" | "destructive";
    className?: string;
}) {
    const i18n = useExpandableI18n();
    const [expanded, setExpanded] = React.useState(false);
    const [isOverflowing, setIsOverflowing] = React.useState(false);
    const containerRef = React.useRef<HTMLDivElement>(null);
    // Detect content overflow
    React.useEffect(() => {
        const el = containerRef.current;
        if (!el || expanded)
            return;
        const checkOverflow = () => {
            setIsOverflowing(el.scrollHeight > el.clientHeight + 1);
        };
        checkOverflow();
        const resizeObserver = new ResizeObserver(checkOverflow);
        resizeObserver.observe(el);
        return () => resizeObserver.disconnect();
    }, [items, expanded]);
    // Line clamp class based on maxLines
    const lineClampClass = {
        1: "line-clamp-1",
        2: "line-clamp-2",
        3: "line-clamp-3",
        4: "line-clamp-4",
    }[maxLines] || "line-clamp-2";
    return (<div className="flex flex-col gap-1">
      <div ref={containerRef} className={cn("flex flex-wrap items-start gap-1", !expanded && lineClampClass, className)}>
        {items.map((item, index) => (<Badge key={`${item}-${index}`} variant={variant} className={textRole.badgeSubtle} title={item}>
            {item}
          </Badge>))}
      </div>
      {(isOverflowing || expanded) && (<button type="button" onClick={() => setExpanded(!expanded)} className={cn("gap-0.5 hover:text-foreground inline-flex items-center self-start transition-colors", textRole.caption)}>
          {expanded ? (<>
              <ChevronUp className="h-3 w-3"/>
              <span>{i18n.collapse}</span>
            </>) : (<>
              <ChevronDown className="h-3 w-3"/>
              <span>{i18n.expand}</span>
            </>)}
        </button>)}
    </div>);
}
