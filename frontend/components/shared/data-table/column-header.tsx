"use client"

import type { Column } from "@tanstack/react-table"
import { useTranslations } from "next-intl"
import { ChevronUp, ChevronDown, ChevronsUpDown, IconEyeOff } from "@/components/icons"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { textRole } from "@/lib/typography"

interface DataTableColumnHeaderProps<TData, TValue> {
  column: Column<TData, TValue>
  title: string
  className?: string
}

/**
 * Unified column header component
 * 
 * Shared status-menu owner for sortable business-list columns. The table's
 * existing column capability remains authoritative so server-paginated routes
 * cannot expose sort or hide operations they do not support.
 */
export function DataTableColumnHeader<TData, TValue>({
  column,
  title,
  className,
}: DataTableColumnHeaderProps<TData, TValue>) {
  const tDataTable = useTranslations("dataTable")

  if (!column.getCanSort()) {
    return <div className={cn("text-left", textRole.tableHeader, className)}>{title}</div>
  }

  const sorted = column.getIsSorted()
  const SortIcon = sorted === "desc"
    ? ChevronDown
    : sorted === "asc"
      ? ChevronUp
      : ChevronsUpDown
  const setSort = (direction: "asc" | "desc") => {
    if (sorted !== direction) {
      column.toggleSorting(direction === "desc")
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            layout="tableHeaderInline"
            className={className}
            aria-label={`${title}${tDataTable("columnActions")}`}
          />
        }
      >
        <span className="min-w-0 truncate">{title}</span>
        <SortIcon className="shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" width="compact">
        <DropdownMenuLabel>{tDataTable("sortMethod")}</DropdownMenuLabel>
        <DropdownMenuItem onClick={() => setSort("asc")}>
          <ChevronUp />
          {tDataTable("sortAsc")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setSort("desc")}>
          <ChevronDown />
          {tDataTable("sortDesc")}
        </DropdownMenuItem>
        {column.getCanHide() ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => column.toggleVisibility(false)}>
              <IconEyeOff />
              {tDataTable("hideColumn")}
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
