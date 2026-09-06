"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import type { Table } from "@tanstack/react-table"
import {
  IconFileExport,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu"
import {
  ColumnVisibilityMenu,
  ToolbarActionMenu,
} from "./menu-owners"
import { DataTableToolbar } from "./toolbar"
import type { DataTableToolbarDensity, ExportOption, SearchMode, FilterField } from "@/types/data-table.types"

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
  showColumnVisibility?: boolean

  // Export
  exportOptions?: ExportOption[]
  selectedCount: number
  toolbarDensity?: DataTableToolbarDensity

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
  showColumnVisibility = true,
  exportOptions,
  selectedCount,
  toolbarDensity = "compact",
  children,
}: TableToolbarProps<TData>) {
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const actionButtonSize = toolbarDensity === "standard" ? "default" : "sm"

  // Render export button
  const renderExportButton = () => {
    if (!exportOptions || exportOptions.length === 0) return null

    if (exportOptions.length === 1) {
      const option = exportOptions[0]
      const isDisabled = typeof option.disabled === 'function'
        ? option.disabled(selectedCount)
        : option.disabled
      return (
        <Button
          variant="outline"
          size={actionButtonSize}
          onClick={option.onClick}
          disabled={isDisabled}
        >
          {option.icon || <IconFileExport className="h-4 w-4" />}
          {option.label}
        </Button>
      )
    }

    return (
      <ToolbarActionMenu
        label={tActions("export")}
        icon={<IconFileExport className="h-4 w-4" />}
        buttonSize={actionButtonSize}
      >
        {exportOptions.map((option) => {
          const isDisabled = typeof option.disabled === 'function'
            ? option.disabled(selectedCount)
            : option.disabled
          return (
            <DropdownMenuItem
              key={option.key}
              onClick={option.onClick}
              disabled={isDisabled}
            >
              {option.label}
            </DropdownMenuItem>
          )
        })}
      </ToolbarActionMenu>
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
      toolbarDensity={toolbarDensity}
    >
      {showColumnVisibility ? (
        <ColumnVisibilityMenu
          table={table}
          label={tDataTable("showColumns")}
          buttonSize={actionButtonSize}
        />
      ) : null}

      {toolbarRight}

      {/* Export button */}
      {renderExportButton()}

      {/* Additional actions passed as children */}
      {children}
    </DataTableToolbar>
  )
}
