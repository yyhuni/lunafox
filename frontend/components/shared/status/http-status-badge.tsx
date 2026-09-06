import { Badge } from "@/components/ui/badge"
import {
  getHttpStatusBadgeClassName,
  getHttpStatusBadgeVariant,
} from "@/lib/http-status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export type HttpStatusBadgeSize = "table" | "default" | "overlay"

const sizeClassNames: Record<HttpStatusBadgeSize, string> = {
  table: "h-5 px-2 py-0",
  default: "px-2 py-1",
  overlay: "px-1.5 py-0.5 text-xs backdrop-blur-sm",
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
