"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import type { Table } from "@tanstack/react-table"
import {
  IconChevronDown,
  IconLayoutColumns,
  IconDownload,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DataTableToolbar } from "./toolbar"
import type { DownloadOption, SearchMode, FilterField } from "@/types/data-table.types"

export interface TableToolbarProps<TData> {
  table: Table<TData>

  // Search/Filter
  searchMode?: SearchMode
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  filterFields?: FilterField[]
  filterExamples?: string[]

  // Toolbar content
  toolbarLeft?: React.ReactNode
  toolbarRight?: React.ReactNode

  // Download
  downloadOptions?: DownloadOption[]
  selectedCount: number

  // Children (additional toolbar actions)
  children?: React.ReactNode
}

export function TableToolbar<TData>({
  table,
  searchMode = 'simple',
  searchPlaceholder,
  searchValue,
  onSearch,
  isSearching,
  filterFields,
  filterExamples,
  toolbarLeft,
  toolbarRight,
  downloadOptions,
  selectedCount,
  children,
}: TableToolbarProps<TData>) {
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")

  // Get column label - only use meta.title, force developers to explicitly define
  const getColumnLabel = (column: { id: string; columnDef: { meta?: { title?: string } } }) => {
    // Only use meta.title, return column.id if not defined (to help discover omissions)
    return column.columnDef.meta?.title ?? column.id
  }

  // Render download button
  const renderDownloadButton = () => {
    if (!downloadOptions || downloadOptions.length === 0) return null

    if (downloadOptions.length === 1) {
      const option = downloadOptions[0]
      const isDisabled = typeof option.disabled === 'function'
        ? option.disabled(selectedCount)
        : option.disabled
      return (
        <Button
          variant="outline"
          size="sm"
          onClick={option.onClick}
          disabled={isDisabled}
        >
          {option.icon || <IconDownload className="h-4 w-4" />}
          {option.label}
        </Button>
      )
    }

    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm">
            <IconDownload className="h-4 w-4" />
            {tActions("download")}
            <IconChevronDown className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {downloadOptions.map((option) => {
            const isDisabled = typeof option.disabled === 'function'
              ? option.disabled(selectedCount)
              : option.disabled
            return (
              <DropdownMenuItem
                key={option.key}
                onClick={option.onClick}
                disabled={isDisabled}
              >
                {option.icon || <IconDownload className="h-4 w-4" />}
                {option.label}
              </DropdownMenuItem>
            )
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }

  return (
    <DataTableToolbar
      searchMode={searchMode}
      searchPlaceholder={searchPlaceholder}
      searchValue={searchValue}
      onSearch={onSearch}
      isSearching={isSearching}
      filterFields={filterFields}
      filterExamples={filterExamples}
      leftContent={toolbarLeft}
    >
      {/* Column visibility control */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm">
            <IconLayoutColumns className="h-4 w-4" />
            {tDataTable("showColumns")}
            <IconChevronDown className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {table
            .getAllColumns()
            .filter((column) => typeof column.accessorFn !== "undefined" && column.getCanHide())
            .map((column) => (
              <DropdownMenuCheckboxItem
                key={column.id}
                className="capitalize"
                checked={column.getIsVisible()}
                onCheckedChange={(value) => column.toggleVisibility(!!value)}
              >
                {getColumnLabel(column)}
              </DropdownMenuCheckboxItem>
            ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {toolbarRight}

      {/* Download button */}
      {renderDownloadButton()}

      {/* Additional actions passed as children */}
      {children}
    </DataTableToolbar>
  )
}
