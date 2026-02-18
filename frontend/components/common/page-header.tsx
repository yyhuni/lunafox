"use client"

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
}: PageHeaderProps) {
  const displayCode = code ?? "PAGE"

  return (
    <div className={cn("px-4 lg:px-6 mb-2", className)}>
      {breadcrumbItems && breadcrumbItems.length > 0 ? (
        <div className="mb-2 text-xs text-muted-foreground flex items-center gap-2">
          {breadcrumbItems.map((item, index) => (
            <span key={item.href + item.label} className="flex items-center gap-2">
              {index > 0 ? <span>/</span> : null}
              <span>{item.label}</span>
            </span>
          ))}
        </div>
      ) : null}
      <div className="flex items-end gap-2 mb-2">
        <div className="flex items-baseline gap-3 border-b-2 border-primary pb-2">
          <h1 className="text-2xl font-bold tracking-tight uppercase">
            {title}
          </h1>
          <span className="font-mono text-xs text-muted-foreground font-medium tracking-wide">
            /{displayCode}
          </span>
        </div>
        <div className="flex-1 h-1.5 bg-[repeating-linear-gradient(45deg,transparent,transparent_4px,currentColor_4px,currentColor_5px)] text-primary/10" />
        {action}
      </div>
      {description && (
        <p className="text-sm text-muted-foreground">
          {description}
        </p>
      )}
    </div>
  )
}
