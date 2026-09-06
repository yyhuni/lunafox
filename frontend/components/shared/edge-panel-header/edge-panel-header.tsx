import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

export type EdgePanelHeaderVariant = "workbench" | "form" | "detail" | "compact"

interface EdgePanelHeaderProps {
  variant: EdgePanelHeaderVariant
  title: ReactNode
  description?: ReactNode
  leading?: ReactNode
  titleMeta?: ReactNode
  headerMeta?: ReactNode
  actions?: ReactNode
  progress?: ReactNode
  className?: string
}

const rowClassNames: Record<EdgePanelHeaderVariant, string> = {
  workbench: "items-start justify-between gap-4",
  form: "justify-between gap-4",
  detail: "items-center gap-3",
  compact: "items-center justify-between gap-2",
}

const leadingClassNames: Record<EdgePanelHeaderVariant, string> = {
  workbench: "radius-surface flex size-10 shrink-0 items-center justify-center bg-muted text-muted-foreground",
  form: "radius-surface flex size-10 shrink-0 items-center justify-center bg-muted text-muted-foreground [&>svg]:size-7",
  detail: "radius-surface flex size-7 shrink-0 items-center justify-center bg-muted text-muted-foreground",
  compact: "radius-surface flex size-6 shrink-0 items-center justify-center bg-muted text-muted-foreground",
}

export function EdgePanelHeader({
  variant,
  title,
  description,
  leading,
  titleMeta,
  headerMeta,
  actions,
  progress,
  className,
}: EdgePanelHeaderProps) {
  const hasDescription = description !== undefined && description !== null
  const rowClassName =
    variant === "form"
      ? cn(rowClassNames.form, hasDescription ? "items-start" : "items-center")
      : rowClassNames[variant]

  return (
    <div
      data-slot="edge-panel-header"
      data-variant={variant}
      className={cn("min-w-0", variant === "workbench" && "flex min-w-0 flex-col", className)}
    >
      <div className={cn("flex min-h-8 min-w-0", rowClassName)}>
        {leading ? (
          <span data-slot="edge-panel-leading" className={leadingClassNames[variant]}>
            {leading}
          </span>
        ) : null}

        <div className={cn("min-w-0 flex-1", (variant === "workbench" || variant === "form") && "space-y-1")}>
          <div data-slot="edge-panel-title-row" className="flex min-w-0 items-center gap-2">
            <div data-slot="edge-panel-title" className="min-w-0 flex-1">
              {title}
            </div>
            {titleMeta ? <div className="shrink-0">{titleMeta}</div> : null}
          </div>

          {hasDescription ? (
            <div data-slot="edge-panel-description" className="min-w-0">
              {description}
            </div>
          ) : null}
        </div>

        {actions ? (
          <div data-slot="edge-panel-actions" className="flex shrink-0 items-center gap-2">
            {actions}
          </div>
        ) : null}
      </div>

      {headerMeta ? <div className="min-w-0">{headerMeta}</div> : null}

      {progress ? (
        <div className={cn(variant === "workbench" && "pt-4")}>{progress}</div>
      ) : null}
    </div>
  )
}
