"use client"

import React from "react"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface MonoValueCellProps {
  value: React.ReactNode | null | undefined
  placeholder?: React.ReactNode
  className?: string
  contentClassName?: string
}

export function MonoValueCell({
  value,
  placeholder = "-",
  className,
  contentClassName,
}: MonoValueCellProps) {
  const displayValue = value === null || value === undefined || value === "" ? placeholder : value

  return (
    <div
      className={cn(
        "flex min-h-5 max-w-full items-center whitespace-nowrap",
        textRole.tableCellSecondary,
        className
      )}
    >
      <span className={cn("font-mono", contentClassName)}>{displayValue}</span>
    </div>
  )
}
