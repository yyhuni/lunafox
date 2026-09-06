"use client"

import * as React from "react"
import { useTranslations } from 'next-intl'
import { SmartFilterInput } from "@/components/common/smart-filter-input"
import type { FilterField, ParsedFilter } from "@/components/common/smart-filter-input"
import type { DataTableToolbarDensity } from "@/types/data-table.types"
import { cn } from "@/lib/utils"
import { SimpleSearchToolbar } from "./simple-search-toolbar"
import { useSimpleSearchState } from "./use-simple-search"

interface DataTableToolbarProps {
  // Search mode
  searchMode?: 'simple' | 'smart'
  searchPlaceholder?: string
  searchValue?: string
  onSearch?: (value: string) => void
  isSearching?: boolean
  filterFields?: FilterField[]
  filterExamples?: string[]
  
  // Left custom content
  leftContent?: React.ReactNode
  
  // Right actions
  children?: React.ReactNode
  
  // Styles
  className?: string
  toolbarDensity?: DataTableToolbarDensity
}

/**
 * Unified toolbar component
 * 
 * Features:
 * - Supports both simple search and smart filter modes
 * - Left side search/filter, right side action buttons
 * - Supports custom content slots
 */
export function DataTableToolbar({
  searchMode = 'simple',
  searchPlaceholder,
  searchValue = "",
  onSearch,
  isSearching = false,
  filterFields,
  filterExamples,
  leftContent,
  children,
  className,
  toolbarDensity = "compact",
}: DataTableToolbarProps) {
  const t = useTranslations('common.actions')
  
  // Use translation as default placeholder
  const placeholder = searchPlaceholder ?? t('search')
  
  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  // Handle smart filter search
  const handleSmartSearch = (_filters: ParsedFilter[], rawQuery: string) => {
    onSearch?.(rawQuery)
  }

  const leftContentClassName = cn(
    "flex w-full min-w-0 flex-wrap items-center gap-2 sm:flex-1",
    !leftContent && "sm:max-w-xl"
  )

  return (
    <div className={cn("flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between", className)}>
      {/* Left: Search/Filter */}
      <div className={leftContentClassName}>
        {leftContent ? (
          leftContent
        ) : searchMode === 'smart' ? (
          <SmartFilterInput
            fields={filterFields}
            examples={filterExamples}
            placeholder={placeholder}
            value={searchValue}
            onSearch={handleSmartSearch}
            toolbarDensity={toolbarDensity}
            className="flex-1"
          />
        ) : (
          <SimpleSearchToolbar
            value={localSearchValue}
            onChange={handleSearchInputChange}
            onSubmit={commitSearch}
            loading={isSearching}
            placeholder={placeholder}
            toolbarDensity={toolbarDensity}
            className="sm:flex-1"
            groupClassName="sm:flex-1"
            inputWidthMode="fill"
          />
        )}
      </div>

      {/* Right: Action buttons */}
      {children && (
        <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
          {children}
        </div>
      )}
    </div>
  )
}
