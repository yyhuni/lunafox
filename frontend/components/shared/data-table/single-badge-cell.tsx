"use client"

import type { ComponentProps, ReactNode } from "react"

import { Badge } from "@/components/ui/badge"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface SingleBadgeCellProps {
  value: ReactNode | null | undefined
  placeholder?: string
  variant?: ComponentProps<typeof Badge>["variant"]
  className?: string
  badgeClassName?: string
  dataBadgeType?: string
}

export function SingleBadgeCell({
  value,
  placeholder = "-",
  variant = "outline",
  className,
  badgeClassName,
  dataBadgeType,
}: SingleBadgeCellProps) {
  const hasValue = value !== null && value !== undefined && value !== ""
  const title =
    typeof value === "string" || typeof value === "number" || typeof value === "boolean"
      ? String(value)
      : undefined

  return (
    <div className={cn("flex min-h-5 min-w-0 max-w-full items-center whitespace-nowrap", className)}>
      {hasValue ? (
        <Badge
          variant={variant}
          className={cn("max-w-full truncate", badgeClassName)}
          data-badge-type={dataBadgeType}
          title={title}
        >
          {value}
        </Badge>
      ) : (
        <span className={textRole.tableCellSecondary}>{placeholder}</span>
      )}
    </div>
  )
}
