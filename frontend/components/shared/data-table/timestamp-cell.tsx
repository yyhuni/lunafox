"use client"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface TimestampCellProps {
  value: string | null | undefined
  placeholder?: string
  className?: string
}

export function TimestampCell({
  value,
  placeholder = "-",
  className,
}: TimestampCellProps) {
  return (
    <div className={cn("flex min-h-5 max-w-full items-center whitespace-nowrap", textRole.tableCellSecondary, className)}>
      {value || placeholder}
    </div>
  )
}
