import React from "react"

import { Textarea } from "@/components/ui/textarea"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export type LineNumberedTextareaHighlightTone = "error" | "warning"

export interface LineNumberedTextareaHighlight {
  lineNumber: number
  tone: LineNumberedTextareaHighlightTone
  label?: string
}

export const lineNumberedTextareaViewportClassName =
  "bg-background border flex h-70 overflow-hidden rounded-md"

export const lineNumberedTextareaCompactViewportClassName =
  "bg-background border flex h-45 overflow-hidden rounded-md"

export const lineNumberedTextareaResponsiveViewportClassName =
  "bg-background border flex h-45 overflow-hidden rounded-md md:h-70 xl:h-96"

export const lineNumberedTextareaFillViewportClassName =
  "h-full min-h-0 flex-1 md:h-full xl:h-full"

export const lineNumberedTextareaTallViewportClassName =
  "bg-background border overflow-hidden relative rounded-md [&_.line-numbered-textarea-row]:h-81"

export const lineNumberedTextareaGutterClassName =
  "bg-background border-r dark:bg-input/30 flex-shrink-0 font-mono text-sm"

export const lineNumberedTextareaMutedGutterClassName =
  "bg-background border-r dark:bg-input/30 flex-shrink-0 overflow-hidden select-none font-mono text-sm"

export function getLineNumberGutterWidth(lineCount: number) {
  const normalizedLineCount = Number.isFinite(lineCount)
    ? Math.max(1, Math.floor(lineCount))
    : 1
  const digitCount = Math.max(2, String(normalizedLineCount).length)

  return `calc(${digitCount}ch + 1rem)`
}

export const lineNumberedTextareaNumbersClassName =
  "font-mono h-full leading-5 overflow-y-auto py-3 scrollbar-hide text-muted-foreground text-right text-sm"

export const lineNumberedTextareaRowClassName = "h-5"

export const lineNumberedTextareaClassName =
  "border-0 focus-visible:ring-0 focus-visible:ring-offset-0 font-mono h-full leading-5 overflow-y-auto py-3 resize-none text-sm"

export const lineNumberedTextareaHighlightToneClassNames: Record<LineNumberedTextareaHighlightTone, string> = {
  error: "bg-error/10",
  warning: "bg-warning/10",
}

export const lineNumberedTextareaNumberToneClassNames: Record<LineNumberedTextareaHighlightTone, string> = {
  error: "text-error",
  warning: "text-warning",
}

export const lineNumberedTextareaBadgeToneClassNames: Record<LineNumberedTextareaHighlightTone, string> = {
  error: "border-error/30 bg-card text-error",
  warning: "border-warning/30 bg-card text-warning",
}

const LINE_NUMBERED_TEXTAREA_ROW_HEIGHT_PX = 20
const LINE_NUMBERED_TEXTAREA_OVERSCAN_ROWS = 8
const LINE_NUMBERED_TEXTAREA_DEFAULT_VIEWPORT_HEIGHT_PX = 320
const LINE_NUMBERED_TEXTAREA_VIRTUALIZATION_THRESHOLD = 200

export function countLineNumberedTextareaLines(value: string) {
  let lineCount = 1
  for (let index = 0; index < value.length; index += 1) {
    if (value.charCodeAt(index) === 10) lineCount += 1
  }
  return lineCount
}

function normalizeLineNumberedTextareaLineCount(lineCount: number) {
  if (!Number.isFinite(lineCount)) return 1
  return Math.max(1, Math.floor(lineCount))
}

function useTextareaViewportHeight(textareaRef: React.RefObject<HTMLTextAreaElement | null>) {
  const [viewportHeight, setViewportHeight] = React.useState(LINE_NUMBERED_TEXTAREA_DEFAULT_VIEWPORT_HEIGHT_PX)

  React.useEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return

    const updateViewportHeight = () => {
      const nextHeight = textarea.clientHeight
      if (nextHeight > 0) {
        setViewportHeight((currentHeight) => currentHeight === nextHeight ? currentHeight : nextHeight)
      }
    }

    updateViewportHeight()
    if (typeof ResizeObserver === "undefined") return

    const observer = new ResizeObserver(updateViewportHeight)
    observer.observe(textarea)
    return () => observer.disconnect()
  }, [textareaRef])

  return viewportHeight
}

interface LineNumberedTextareaProps
  extends Omit<React.ComponentProps<typeof Textarea>, "onChange"> {
  lineCount: number
  lineNumbersRef: React.RefObject<HTMLDivElement | null>
  textareaRef: React.RefObject<HTMLTextAreaElement | null>
  onChange?: (event: React.ChangeEvent<HTMLTextAreaElement>) => void
  onScroll?: (event: React.UIEvent<HTMLTextAreaElement>) => void
  onTextareaRef?: (element: HTMLTextAreaElement | null) => void
  viewportClassName?: string
  gutterClassName?: string
  numbersClassName?: string
  lineHighlights?: LineNumberedTextareaHighlight[]
}

export function LineNumberedTextarea({
  lineCount,
  lineNumbersRef,
  textareaRef,
  className,
  viewportClassName,
  gutterClassName,
  numbersClassName,
  lineHighlights = [],
  onScroll,
  onTextareaRef,
  ...props
}: LineNumberedTextareaProps) {
  const highlightRowsRef = React.useRef<HTMLDivElement | null>(null)
  const [scrollTop, setScrollTop] = React.useState(0)
  const normalizedLineCount = normalizeLineNumberedTextareaLineCount(lineCount)
  const isVirtualized = normalizedLineCount > LINE_NUMBERED_TEXTAREA_VIRTUALIZATION_THRESHOLD
  const viewportHeight = useTextareaViewportHeight(textareaRef)
  const highlightByLine = React.useMemo(
    () => new Map(lineHighlights.map((highlight) => [highlight.lineNumber, highlight])),
    [lineHighlights]
  )
  const visibleLineRange = React.useMemo(() => {
    if (!isVirtualized) {
      return { startIndex: 0, endIndex: normalizedLineCount }
    }

    const startIndex = Math.max(
      0,
      Math.floor(scrollTop / LINE_NUMBERED_TEXTAREA_ROW_HEIGHT_PX) - LINE_NUMBERED_TEXTAREA_OVERSCAN_ROWS
    )
    const endIndex = Math.min(
      normalizedLineCount,
      Math.ceil((scrollTop + viewportHeight) / LINE_NUMBERED_TEXTAREA_ROW_HEIGHT_PX) + LINE_NUMBERED_TEXTAREA_OVERSCAN_ROWS
    )
    return { startIndex, endIndex: Math.max(startIndex + 1, endIndex) }
  }, [isVirtualized, normalizedLineCount, scrollTop, viewportHeight])
  const visibleRows = React.useMemo(() => {
    const rows: Array<{
      lineNumber: number
      offset: number
      highlight: LineNumberedTextareaHighlight | undefined
    }> = []

    for (let index = visibleLineRange.startIndex; index < visibleLineRange.endIndex; index += 1) {
      const lineNumber = index + 1
      rows.push({
        lineNumber,
        offset: index * LINE_NUMBERED_TEXTAREA_ROW_HEIGHT_PX,
        highlight: highlightByLine.get(lineNumber),
      })
    }

    return rows
  }, [highlightByLine, visibleLineRange])
  const contentHeight = normalizedLineCount * LINE_NUMBERED_TEXTAREA_ROW_HEIGHT_PX

  React.useEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return
    setScrollTop((currentScrollTop) => currentScrollTop === textarea.scrollTop ? currentScrollTop : textarea.scrollTop)
  }, [normalizedLineCount, textareaRef])

  const handleScroll = React.useCallback((event: React.UIEvent<HTMLTextAreaElement>) => {
    const nextScrollTop = event.currentTarget.scrollTop
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = nextScrollTop
    }
    if (highlightRowsRef.current) {
      highlightRowsRef.current.scrollTop = nextScrollTop
    }
    setScrollTop((currentScrollTop) => currentScrollTop === nextScrollTop ? currentScrollTop : nextScrollTop)
    onScroll?.(event)
  }, [lineNumbersRef, onScroll])

  return (
    <div className={cn(lineNumberedTextareaViewportClassName, viewportClassName)}>
      <div
        className={cn(lineNumberedTextareaGutterClassName, gutterClassName)}
        style={{ width: getLineNumberGutterWidth(normalizedLineCount) }}
      >
        <div ref={lineNumbersRef} data-slot="line-numbered-textarea-numbers" className={cn(lineNumberedTextareaNumbersClassName, numbersClassName)}>
          <div className="relative" style={{ height: contentHeight }}>
            {visibleRows.map(({ lineNumber, offset, highlight }) => (
              <div
                key={lineNumber}
                data-slot="line-numbered-textarea-number-row"
                className={cn(
                  lineNumberedTextareaRowClassName,
                  "absolute left-0 right-0 px-2",
                  highlight && lineNumberedTextareaNumberToneClassNames[highlight.tone]
                )}
                style={{ transform: `translateY(${offset}px)` }}
              >
                {lineNumber}
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className="bg-background flex-1 overflow-hidden relative">
        <div
          ref={highlightRowsRef}
          aria-hidden="true"
          className="absolute inset-0 z-20 overflow-hidden py-3 pointer-events-none"
        >
          <div className="relative" style={{ height: contentHeight }}>
            {visibleRows.map(({ lineNumber, offset, highlight }) => (
              <div
                key={lineNumber}
                data-slot="line-numbered-textarea-highlight-row"
                className={cn(
                  lineNumberedTextareaRowClassName,
                  "absolute left-0 right-0",
                  highlight && lineNumberedTextareaHighlightToneClassNames[highlight.tone]
                )}
                style={{ transform: `translateY(${offset}px)` }}
              >
                {highlight?.label && (
                  <span
                    className={cn(
                      "absolute right-2 top-0 inline-flex h-5 items-center rounded border px-1.5",
                      textRole.badge,
                      lineNumberedTextareaBadgeToneClassNames[highlight.tone]
                    )}
                  >
                    {highlight.label}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
        <Textarea
          ref={(element) => {
            textareaRef.current = element
            onTextareaRef?.(element)
          }}
          className={cn(
            lineNumberedTextareaClassName,
            "bg-transparent relative z-10",
            lineHighlights.length > 0 && "pr-20",
            className
          )}
          onScroll={handleScroll}
          style={{ lineHeight: "20px", ...props.style }}
          {...props}
        />
      </div>
    </div>
  )
}
