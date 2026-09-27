import { Badge } from "@/components/ui/badge"
import {
  getHttpStatusBadgeClassName,
  getHttpStatusBadgeVariant,
} from "@/lib/http-status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export type HttpStatusBadgeSize = "table" | "default" | "overlay"

const sizeClassNames: Record<HttpStatusBadgeSize, string> = {
  table: "",
  default: "",
  overlay: "px-1.5 py-0.5 text-xs backdrop-blur-sm",
}

const sizeProps: Record<HttpStatusBadgeSize, { size: "tag" | "compact" }> = {
  table: { size: "compact" },
  default: { size: "tag" },
  overlay: { size: "compact" },
}

interface HttpStatusBadgeProps {
  statusCode: number | null | undefined
  size?: HttpStatusBadgeSize
  className?: string
}

export function HttpStatusBadge({
  statusCode,
  size = "table",
  className,
}: HttpStatusBadgeProps) {
  if (statusCode === null || statusCode === undefined) {
    return (
      <Badge
        {...sizeProps[size]}
        variant="outline"
        className={cn(
          sizeClassNames[size],
          "font-mono tabular-nums",
          textRole.tableCellSecondary,
          className
        )}
      >
        -
      </Badge>
    )
  }

  return (
    <Badge
      {...sizeProps[size]}
      variant={getHttpStatusBadgeVariant(statusCode)}
      className={cn(
        sizeClassNames[size],
        "font-mono tabular-nums",
        getHttpStatusBadgeClassName(statusCode),
        className
      )}
    >
      {statusCode}
    </Badge>
  )
}
