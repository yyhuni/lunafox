"use client"

import * as React from "react"
import { Tabs as TabsPrimitive } from "@base-ui/react/tabs"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

type TabsVariant =
  | "default"
  | "underline"
  | "minimal"
  | "split"
  | "minimal-tab"
  | "filter"
  | "page-nav"
  | "content"
  | "metric"

function Tabs({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      className={cn("flex flex-col gap-2", className)}
      {...props}
    />
  )
}

interface TabsListProps extends React.ComponentProps<typeof TabsPrimitive.List> {
  variant?: TabsVariant
  size?: "sm" | "md"
}

interface TabsActiveIndicatorState {
  x: number
  width: number
  ready: boolean
}

function assignRef<T>(ref: React.Ref<T> | undefined, value: T | null) {
  if (typeof ref === "function") {
    ref(value)
  } else if (ref) {
    ref.current = value
  }
}

function resolveTabsVariant(variant: TabsVariant): Exclude<TabsVariant, "default" | "underline" | "minimal-tab"> {
  if (variant === "filter") return "filter"
  if (variant === "page-nav") return "page-nav"
  if (variant === "content") return "content"
  if (variant === "metric") return "metric"
  if (variant === "default") return "filter"
  if (variant === "underline") return "page-nav"
  if (variant === "minimal-tab") return "content"
  return variant
}

function TabsList({
  className,
  variant = "default",
  size = "md",
  children,
  ref,
  onClickCapture,
  ...props
}: TabsListProps) {
  const resolvedVariant = resolveTabsVariant(variant)
  const listRef = React.useRef<HTMLDivElement>(null)
  const frameRef = React.useRef<number | null>(null)
  const pendingTriggerRef = React.useRef<HTMLElement | null>(null)
  const [activeIndicator, setActiveIndicator] = React.useState<TabsActiveIndicatorState>({
    x: 0,
    width: 0,
    ready: false,
  })

  const setListRef = React.useCallback((node: HTMLDivElement | null) => {
    listRef.current = node
    assignRef(ref, node)
  }, [ref])

  const updateActiveIndicatorForTrigger = React.useCallback((trigger: HTMLElement) => {
    const list = listRef.current

    if (!list) {
      setActiveIndicator((current) => current.ready ? { ...current, ready: false } : current)
      return
    }

    const listBounds = list.getBoundingClientRect()
    const triggerBounds = trigger.getBoundingClientRect()
    const nextIndicator = {
      x: triggerBounds.left - listBounds.left + list.scrollLeft,
      width: triggerBounds.width,
      ready: true,
    }

    setActiveIndicator((current) => (
      current.x === nextIndicator.x
      && current.width === nextIndicator.width
      && current.ready === nextIndicator.ready
        ? current
        : nextIndicator
    ))
  }, [])

  const updateActiveIndicator = React.useCallback(() => {
    const list = listRef.current
    const pendingTrigger = pendingTriggerRef.current
    const activeTrigger = pendingTrigger && list?.contains(pendingTrigger)
      ? pendingTrigger
      : list?.querySelector<HTMLElement>('[data-slot="tabs-trigger"][data-active]')

    if (!activeTrigger) {
      setActiveIndicator((current) => current.ready ? { ...current, ready: false } : current)
      return
    }

    updateActiveIndicatorForTrigger(activeTrigger)
  }, [updateActiveIndicatorForTrigger])

  const handleClickCapture = React.useCallback<NonNullable<TabsListProps["onClickCapture"]>>((event) => {
    onClickCapture?.(event)

    if (event.defaultPrevented || resolvedVariant !== "filter") return

    const target = event.target instanceof Element ? event.target : null
    const trigger = target?.closest<HTMLElement>('[data-slot="tabs-trigger"]')

    if (!trigger || trigger.matches("[data-disabled], :disabled") || !listRef.current?.contains(trigger)) return

    pendingTriggerRef.current = trigger
    updateActiveIndicatorForTrigger(trigger)
  }, [onClickCapture, resolvedVariant, updateActiveIndicatorForTrigger])

  React.useLayoutEffect(() => {
    if (resolvedVariant !== "filter") return

    const list = listRef.current
    if (!list) return

    const scheduleIndicatorUpdate = () => {
      if (frameRef.current !== null) {
        cancelAnimationFrame(frameRef.current)
      }

      frameRef.current = requestAnimationFrame(() => {
        frameRef.current = null
        updateActiveIndicator()
      })
    }
    const resizeObserver = "ResizeObserver" in window
      ? new ResizeObserver(scheduleIndicatorUpdate)
      : null
    const observeTriggers = () => {
      resizeObserver?.observe(list)
      list.querySelectorAll<HTMLElement>('[data-slot="tabs-trigger"]').forEach((trigger) => {
        resizeObserver?.observe(trigger)
      })
    }
    const mutationObserver = new MutationObserver(() => {
      const pendingTrigger = pendingTriggerRef.current
      const committedActiveTrigger = list.querySelector<HTMLElement>('[data-slot="tabs-trigger"][data-active]')

      // Route-backed tabs retain the clicked indicator until the URL-backed tab commits.
      if (pendingTrigger && committedActiveTrigger === pendingTrigger) {
        pendingTriggerRef.current = null
      }
      observeTriggers()
      scheduleIndicatorUpdate()
    })
    const fonts = document.fonts

    updateActiveIndicator()
    observeTriggers()
    mutationObserver.observe(list, {
      attributes: true,
      attributeFilter: ["data-active"],
      childList: true,
      subtree: true,
    })
    window.addEventListener("resize", scheduleIndicatorUpdate)
    void fonts?.ready.then(scheduleIndicatorUpdate)
    fonts?.addEventListener("loadingdone", scheduleIndicatorUpdate)

    return () => {
      if (frameRef.current !== null) {
        cancelAnimationFrame(frameRef.current)
        frameRef.current = null
      }
      resizeObserver?.disconnect()
      mutationObserver.disconnect()
      window.removeEventListener("resize", scheduleIndicatorUpdate)
      fonts?.removeEventListener("loadingdone", scheduleIndicatorUpdate)
    }
  }, [resolvedVariant, updateActiveIndicator])

  const activeIndicatorStyle = React.useMemo(() => ({
    "--tabs-active-indicator-x": `${activeIndicator.x}px`,
    "--tabs-active-indicator-width": `${activeIndicator.width}px`,
  }) as React.CSSProperties, [activeIndicator.width, activeIndicator.x])

  return (
    <TabsPrimitive.List
      ref={setListRef}
      onClickCapture={handleClickCapture}
      data-slot="tabs-list"
      data-variant={resolvedVariant}
      data-size={size}
      className={cn(
        "inline-flex w-fit max-w-full items-center justify-center",
        resolvedVariant === "filter" && "radius-surface relative isolate bg-muted text-muted-foreground",
        resolvedVariant === "filter" && size === "sm" && "h-8 p-[2px]",
        resolvedVariant === "filter" && size === "md" && "h-9 p-[3px]",
        resolvedVariant === "page-nav" && "gap-3 bg-transparent p-0",
        resolvedVariant === "page-nav" && size === "sm" && "h-8",
        resolvedVariant === "page-nav" && size === "md" && "h-9",
        resolvedVariant === "minimal" && size === "sm" && "h-8 gap-3 bg-transparent p-0",
        resolvedVariant === "minimal" && size === "md" && "h-9 gap-3 bg-transparent p-0",
        resolvedVariant === "split" && "h-8 gap-3 bg-transparent p-0",
        resolvedVariant === "content" && "h-8 w-full min-w-0 items-end justify-start gap-4 overflow-x-auto overflow-y-hidden bg-transparent border-b border-border p-0",
        resolvedVariant === "content" && size === "sm" && "gap-3",
        resolvedVariant === "content" && size === "md" && "h-9",
        resolvedVariant === "metric" && "radius-surface grid h-auto w-full grid-cols-3 items-stretch justify-center gap-0 overflow-hidden border border-border/70 bg-transparent p-0",
        className
      )}
      {...props}
    >
      {resolvedVariant === "filter" && (
        <span
          aria-hidden="true"
          data-slot="tabs-active-indicator"
          className={cn(
            "tabs-active-indicator pointer-events-none absolute left-0 z-0 radius-control bg-background shadow-sm transition-[transform,width] duration-[var(--motion-duration-fast)] ease-[var(--motion-ease-standard)] will-change-transform [transform:translate3d(var(--tabs-active-indicator-x),0,0)] [width:var(--tabs-active-indicator-width)] dark:border dark:border-input dark:bg-input/30",
            size === "sm" && "inset-y-[2px]",
            size === "md" && "inset-y-[3px]",
            activeIndicator.ready ? "visible" : "invisible"
          )}
          style={activeIndicatorStyle}
        />
      )}
      {children}
    </TabsPrimitive.List>
  )
}

interface TabsTriggerProps extends React.ComponentProps<typeof TabsPrimitive.Tab> {
  variant?: TabsVariant
  size?: "sm" | "md"
  activeIndicator?: "trigger" | "fixed"
}

function TabsTrigger({
  children,
  className,
  variant = "default",
  size = "md",
  activeIndicator = "trigger",
  render,
  nativeButton,
  ...props
}: TabsTriggerProps) {
  const resolvedVariant = resolveTabsVariant(variant)
  const resolvedNativeButton = nativeButton ?? (render ? false : undefined)

  return (
    <TabsPrimitive.Tab
      data-slot="tabs-trigger"
      data-variant={resolvedVariant}
      data-size={size}
      className={cn(
        "inline-flex items-center justify-center gap-1.5 whitespace-nowrap cursor-pointer transition-[background-color,border-color,color,box-shadow] focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        textRole.tab,
        resolvedVariant === "filter" && "radius-control relative z-10 bg-transparent text-foreground/60 h-[calc(100%-1px)] flex-1 border border-transparent px-2 py-1 focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-1 hover:text-foreground data-active:bg-transparent data-active:text-foreground data-active:shadow-none dark:text-muted-foreground dark:hover:text-foreground dark:data-active:border-transparent dark:data-active:bg-transparent dark:data-active:text-foreground",
        resolvedVariant === "filter" && size === "sm" && "text-xs",
        resolvedVariant === "page-nav" && "radius-control-top text-muted-foreground data-active:text-foreground px-2 border-b-2 border-transparent data-active:border-primary bg-transparent",
        resolvedVariant === "page-nav" && size === "sm" && "h-8 py-1 text-xs",
        resolvedVariant === "page-nav" && size === "md" && "h-9 py-1.5",
        resolvedVariant === "minimal" && activeIndicator === "trigger" && "radius-control-top text-muted-foreground data-active:text-foreground px-2 py-1 text-[11px] font-medium leading-4 border-b-2 border-transparent data-active:border-primary bg-transparent hover:text-foreground hover:border-primary/50 transition-colors",
        resolvedVariant === "minimal" && activeIndicator === "fixed" && "relative radius-control-top text-muted-foreground data-active:text-foreground px-2 py-1 text-[11px] font-medium leading-4 border-b-0 bg-transparent hover:text-foreground transition-colors after:pointer-events-none after:absolute after:top-1/2 after:left-1/2 after:h-0.5 after:w-2 after:-translate-x-1/2 after:translate-y-2 after:rounded-full after:bg-transparent after:content-[''] data-active:after:bg-primary",
        resolvedVariant === "minimal" && size === "sm" && "h-8",
        resolvedVariant === "minimal" && size === "md" && "h-9",
        resolvedVariant === "split" && "radius-control-top text-muted-foreground data-active:text-foreground px-2 py-1 text-[11px] font-medium leading-4 border-b border-border bg-transparent hover:text-foreground transition-colors data-active:font-semibold focus-visible:ring-0 shadow-none",
        resolvedVariant === "content" && "radius-control-top -mb-px text-muted-foreground data-active:bg-transparent data-active:text-foreground data-active:border-primary border-b-[2.5px] border-transparent bg-transparent hover:text-foreground hover:border-primary/50 transition-colors focus-visible:ring-0 focus-visible:ring-offset-0",
        resolvedVariant === "content" && size === "sm" && "h-8 px-2 py-1 text-[11px] font-medium leading-4",
        resolvedVariant === "content" && size === "md" && "h-9 px-3 py-1.5 text-[11px] font-medium leading-4",
        resolvedVariant === "metric" && "h-auto min-h-20 min-w-0 flex-1 flex-col items-stretch justify-center gap-1 rounded-none border-0 border-r border-border/50 bg-transparent px-3 py-3 text-left text-muted-foreground transition-colors hover:bg-muted/30 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 data-active:bg-muted/50 data-active:text-foreground data-active:shadow-none last:border-r-0",
        className
      )}
      render={render}
      nativeButton={resolvedNativeButton}
      {...props}
    >
      {children}
    </TabsPrimitive.Tab>
  )
}

function TabsContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Panel>) {
  return (
    <TabsPrimitive.Panel
      data-slot="tabs-content"
      className={cn("flex-1 outline-none", className)}
      {...props}
    />
  )
}

function TabsCountBadge({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="tabs-count-badge"
      className={cn(
        "radius-pill inline-flex h-5 min-w-5 shrink-0 items-center justify-center border border-transparent bg-foreground/10 px-1.5 tabular-nums text-foreground",
        textRole.badge,
        className
      )}
      {...props}
    />
  )
}

export { Tabs, TabsList, TabsTrigger, TabsContent, TabsCountBadge }
