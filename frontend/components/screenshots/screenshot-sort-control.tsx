"use client"

import {
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Sliders,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type ScreenshotSortField = "statusCode" | "createdAt"
type ScreenshotSort = { id: string; desc: boolean } | undefined

interface ScreenshotSortControlProps {
  activeSort: ScreenshotSort
  onSort: (field: ScreenshotSortField) => void
  statusCodeLabel: string
  createdAtLabel: string
}

function getSortLabel(activeSort: ScreenshotSort, field: ScreenshotSortField, label: string) {
  if (activeSort?.id !== field) return label
  return `${label} ${activeSort.desc ? "↓" : "↑"}`
}

function SortDirectionIcon({ activeSort, field }: { activeSort: ScreenshotSort; field: ScreenshotSortField }) {
  if (activeSort?.id !== field) return <ChevronsUpDown className="size-4" />
  return activeSort.desc ? <ChevronDown className="size-4" /> : <ChevronUp className="size-4" />
}

export function ScreenshotSortControl({
  activeSort,
  onSort,
  statusCodeLabel,
  createdAtLabel,
}: ScreenshotSortControlProps) {
  const activeLabel = activeSort?.id === "statusCode" ? statusCodeLabel : createdAtLabel
  const directionLabel = activeSort?.desc ? "降序" : "升序"

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button type="button" variant="outline" size="sm" />}>
        <Sliders className="size-4" />
        {activeLabel} · {directionLabel}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" width="content-fit">
        <DropdownMenuLabel>排序方式</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {(["statusCode", "createdAt"] as const).map((field) => {
          const label = field === "statusCode" ? statusCodeLabel : createdAtLabel
          return (
            <DropdownMenuItem key={field} onClick={() => onSort(field)}>
              <SortDirectionIcon activeSort={activeSort} field={field} />
              {getSortLabel(activeSort, field, label)}
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
