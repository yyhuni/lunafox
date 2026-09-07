import * as React from "react"

import { cn } from "@/lib/utils"

export const SUPPORT_PAGE_ROUTE_SURFACE_CLASS =
  "mx-auto flex min-h-[max(70vh,calc(100vh-4rem))] w-full max-w-[1200px] flex-col items-center justify-start overflow-x-clip p-4 py-8 md:p-8"

export const SUPPORT_PAGE_MAIN_CLASS =
  "relative z-10 mx-auto flex w-full max-w-5xl flex-col justify-start md:min-h-0 md:flex-1 md:justify-center"

export const SUPPORT_PAGE_CONTENT_STACK_CLASS = "flex w-full flex-col gap-10 md:gap-12"

export const SUPPORT_PAGE_HEADER_CLASS = "flex flex-col items-center gap-5 text-center"

export const SUPPORT_PAGE_HEADER_CONTENT_CLASS = "flex max-w-3xl flex-col items-center gap-5"

export const SUPPORT_PAGE_VALUE_BAND_LIST_CLASS =
  "grid border-y border-border md:grid-cols-3 md:divide-x md:divide-border"

export const SUPPORT_PAGE_VALUE_BAND_ITEM_CLASS = "flex gap-4 px-2 py-5 text-left md:px-6"

export const SUPPORT_PAGE_ACTIONS_CLASS = "flex flex-col items-center gap-3 sm:flex-row sm:justify-center"

export const SUPPORT_PAGE_FOOTER_CLASS =
  "flex w-full flex-col items-center gap-3 text-center text-xs text-muted-foreground/70"

export const SUPPORT_TIER_CARD_CLASS =
  "group relative flex h-full min-h-[168px] flex-col items-start justify-between rounded-lg border p-5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background"

export const SUPPORT_PAGE_TIER_OPTIONS_LOADING_SLOT = {
  "data-loading-slot": "support-page-tier-options",
} as const

interface SupportPageLayoutProps extends React.ComponentProps<"div"> {
  decoration?: React.ReactNode
  footer?: React.ReactNode
}

export function SupportPageLayout({ children, className, decoration, footer, ...props }: SupportPageLayoutProps) {
  return (
    <div
      {...props}
      className={cn(SUPPORT_PAGE_ROUTE_SURFACE_CLASS, "relative isolate", className)}
    >
      {decoration}
      <main className={SUPPORT_PAGE_MAIN_CLASS}>
        {children}
      </main>
      {footer ? <div className="relative z-10 mx-auto mt-10 w-full max-w-5xl md:mt-0">{footer}</div> : null}
    </div>
  )
}

export function SupportPageContentStack({ className, ...props }: React.ComponentProps<"div">) {
  return <div {...props} className={cn(SUPPORT_PAGE_CONTENT_STACK_CLASS, className)} />
}

export function SupportPageHeader({ children, className, ...props }: React.ComponentProps<"header">) {
  return (
    <header {...props} data-loading-slot="support-page-header" className={cn(SUPPORT_PAGE_HEADER_CLASS, className)}>
      <div className={SUPPORT_PAGE_HEADER_CONTENT_CLASS}>{children}</div>
    </header>
  )
}

export function SupportPageValueBand({ className, ...props }: React.ComponentProps<"section">) {
  return <section {...props} data-loading-slot="support-page-value-band" className={className} />
}

export function SupportPageValueBandList({ className, ...props }: React.ComponentProps<"ul">) {
  return <ul {...props} className={cn(SUPPORT_PAGE_VALUE_BAND_LIST_CLASS, className)} />
}

interface SupportPageValueBandItemProps extends React.ComponentProps<"li"> {
  index: number
}

export function SupportPageValueBandItem({ index, className, ...props }: SupportPageValueBandItemProps) {
  return (
    <li
      {...props}
      className={cn(
        SUPPORT_PAGE_VALUE_BAND_ITEM_CLASS,
        index > 0 && "border-t border-border md:border-t-0",
        className,
      )}
    />
  )
}

export function SupportPageActions({ className, ...props }: React.ComponentProps<"div">) {
  return <div {...props} data-loading-slot="support-page-actions" className={cn(SUPPORT_PAGE_ACTIONS_CLASS, className)} />
}

export function SupportPageFooter({ className, ...props }: React.ComponentProps<"footer">) {
  return <footer {...props} className={cn(SUPPORT_PAGE_FOOTER_CLASS, className)} />
}
