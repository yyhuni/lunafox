"use client"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface PageHeaderProps {
  /** Page code, such as "TGT-01" */
  code?: string
  /** Main title */
  title: string
  /** Description text (optional) */
  description?: string
  /** Breadcrumbs (compatible with old page parameters) */
  breadcrumbItems?: Array<{ label: string; href: string }>
  /** Custom class for outer container */
  className?: string
  /** Right action area (optional) */
  action?: React.ReactNode
  /** Action aligned with the description; wraps below it on narrow screens. */
  descriptionAction?: React.ReactNode
  /** Compact supplementary content shown immediately after the description. */
  descriptionSupplement?: React.ReactNode
  /** Centered context for command-style headers on wide screens. */
  middle?: React.ReactNode
  /** Header density; compact keeps workbench pages tighter without changing the default page style. */
  density?: "normal" | "compact"
}

/**
 * Industrial-style page header using the minimalist underline variant (Option C).
 * Keeps only the bottom accent line and removes extra background/border decoration.
 */
export function PageHeader({
  code,
  title,
  description,
  breadcrumbItems,
  className,
  action,
  descriptionAction,
  descriptionSupplement,
  middle,
  density = "normal",
}: PageHeaderProps) {
  const displayCode = code ?? "PAGE"
  const compact = density === "compact"
  const hasMiddle = Boolean(middle)

  return (
    <div className={cn("px-4 lg:px-6", className)}>
      {breadcrumbItems && breadcrumbItems.length > 0 ? (
        <div className={cn("flex gap-2 items-center mb-2", textRole.caption)}>
          {breadcrumbItems.map((item, index) => (
            <span key={item.href + item.label} className="flex gap-2 items-center">
              {index > 0 ? <span>/</span> : null}
              <span>{item.label}</span>
            </span>
          ))}
        </div>
      ) : null}
      <div className={cn(
        hasMiddle
          ? "grid grid-cols-[minmax(0,1fr)_auto] items-end gap-2 xl:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] xl:gap-6"
          : "flex gap-2 items-end",
        description ? (compact ? "mb-1" : "mb-2") : "mb-0"
      )}>
        <div className={cn(
          "flex gap-3 items-baseline",
          hasMiddle ? "min-w-0" : "border-b-2 border-primary",
          hasMiddle ? "" : compact ? "pb-1" : "pb-2"
        )}>
          <h1 className={textRole.pageTitleDisplay}>
            {title}
          </h1>
          <span className={textRole.monoLabel}>
            /{displayCode}
          </span>
        </div>
        {hasMiddle ? (
          <>
            <div className="hidden min-w-0 self-end items-center justify-self-center xl:flex">
              {middle}
            </div>
            {action ? <div className="col-start-2 self-end justify-self-end xl:col-start-3">{action}</div> : null}
          </>
        ) : (
          <>
            <div className={cn(
              "bg-[repeating-linear-gradient(45deg,transparent,transparent_4px,currentColor_4px,currentColor_5px)] flex-1 text-primary/10",
              compact ? "h-1" : "h-1.5"
            )} />
            {action}
          </>
        )}
      </div>
      {description || descriptionSupplement || descriptionAction ? (
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
          {description || descriptionSupplement ? (
            <div className="flex min-w-0 items-center gap-1.5">
              {description ? (
                <p className={cn("min-w-0", textRole.pageDescription)}>
                  {description}
                </p>
              ) : null}
              {descriptionSupplement}
            </div>
          ) : null}
          {descriptionAction ? (
            <div className="ml-auto shrink-0">
              {descriptionAction}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}
