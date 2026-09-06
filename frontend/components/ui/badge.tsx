import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { PolymorphicSlot } from "@/components/ui/polymorphic"
import { getSeverityStyle } from "@/lib/severity-config"
import {
  getScanStatusBadgeVariant,
  getScanStatusColorVar,
  getStatusToneBadgeClass,
  isSharedScanStatus,
} from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const badgeVariants = cva(
  cn(
    "radius-badge inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden whitespace-nowrap border border-border bg-transparent px-2 py-1 text-foreground [&>svg]:pointer-events-none [&>svg]:size-3 focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 transition-[color,box-shadow,border-color] dark:aria-invalid:ring-destructive/40 data-[badge-type=subdomain]:text-muted-foreground data-[badge-type=website]:text-muted-foreground data-[badge-type=ip]:text-muted-foreground data-[badge-type=endpoint]:text-muted-foreground data-[badge-type=vulnerability]:text-muted-foreground data-[badge-type=workflow]:text-muted-foreground data-[badge-type=engine]:text-muted-foreground",
    textRole.badge
  ),
  {
    variants: {
      variant: {
        default: "",
        secondary: "",
        destructive: "",
        outline: "",
        count: "radius-pill border-border/60 bg-background/50 tabular-nums",
        filterCount: "radius-pill border-0 bg-background/50 tabular-nums",
        warning: getStatusToneBadgeClass("warning"),
        error: getStatusToneBadgeClass("error"),
        success: getStatusToneBadgeClass("success"),
        info: getStatusToneBadgeClass("info"),
        severityCritical: getSeverityStyle("critical").className,
        severityHigh: getSeverityStyle("high").className,
        severityMedium: getSeverityStyle("medium").className,
        severityLow: getSeverityStyle("low").className,
        severityInfo: getSeverityStyle("info").className,
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function resolveBadgeVariant(
  variant: VariantProps<typeof badgeVariants>["variant"],
  badgeType?: string
): VariantProps<typeof badgeVariants>["variant"] {
  if (!badgeType) return variant
  if (!isSharedScanStatus(badgeType)) return variant

  return getScanStatusBadgeVariant(badgeType)
}

function resolveBadgeTypeStatusClassName(badgeType?: string): string {
  if (!badgeType) return ""
  if (!isSharedScanStatus(badgeType)) return ""

  return "border-l-[3px] pl-2"
}

function resolveBadgeTypeStyle(
  badgeType: string | undefined,
  style: React.CSSProperties | undefined
): React.CSSProperties | undefined {
  if (!badgeType || !isSharedScanStatus(badgeType)) return style

  return { ...style, borderLeftColor: getScanStatusColorVar(badgeType) }
}

function Badge({
  className,
  variant,
  "data-badge-type": badgeType,
  render,
  style,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & {
    render?: React.ReactElement
    "data-badge-type"?: string
  }) {
  const resolvedVariant = resolveBadgeVariant(variant, badgeType)
  const badgeTypeClassName = resolveBadgeTypeStatusClassName(badgeType)
  const badgeTypeStyle = resolveBadgeTypeStyle(badgeType, style)

  return (
    <PolymorphicSlot
      defaultTagName="span"
      render={render}
      data-slot="badge"
      data-badge-type={badgeType}
      className={cn(badgeVariants({ variant: resolvedVariant }), badgeTypeClassName, className)}
      style={badgeTypeStyle}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
