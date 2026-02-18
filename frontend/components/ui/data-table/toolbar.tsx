"use client"

import * as React from "react"
import { IconSearch, IconLoader2 } from "@/components/icons"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { SmartFilterInput } from "@/components/common/smart-filter-input"
import type { FilterField, ParsedFilter } from "@/components/common/smart-filter-input"
import { cn } from "@/lib/utils"

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
}: DataTableToolbarProps) {
  const t = useTranslations('common.actions')
  
  // Use translation as default placeholder
  const placeholder = searchPlaceholder ?? t('search')
  
  // Local search value state (simple mode)
  const [localSearchValue, setLocalSearchValue] = React.useState(searchValue)

  // Sync external search value
  React.useEffect(() => {
    setLocalSearchValue(searchValue)
  }, [searchValue])

  // Handle simple search submit
  const handleSimpleSearchSubmit = () => {
    onSearch?.(localSearchValue)
  }

  // Handle keyboard events
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSimpleSearchSubmit()
    }
  }

  // Handle smart filter search
  const handleSmartSearch = (_filters: ParsedFilter[], rawQuery: string) => {
    onSearch?.(rawQuery)
  }

  return (
    <div className={cn("flex items-center justify-between", className)}>
      {/* Left: Search/Filter */}
      <div className="flex items-center space-x-2 flex-1 max-w-xl">
        {leftContent ? (
          leftContent
        ) : searchMode === 'smart' ? (
          <SmartFilterInput
            fields={filterFields}
            examples={filterExamples}
            placeholder={placeholder}
            value={searchValue}
            onSearch={handleSmartSearch}
            className="flex-1"
          />
        ) : (
          <>
            <Input
              type="search"
              name="search"
              autoComplete="off"
              placeholder={placeholder}
              value={localSearchValue}
              onChange={(e) => setLocalSearchValue(e.target.value)}
              onKeyDown={handleKeyDown}
              className="h-8 flex-1"
            />
            <Button 
              variant="outline" 
              size="sm" 
              onClick={handleSimpleSearchSubmit} 
              disabled={isSearching}
            >
              {isSearching ? (
                <IconLoader2 className="h-4 w-4 animate-spin" />
              ) : (
                <IconSearch className="h-4 w-4" />
              )}
            </Button>
          </>
        )}
      </div>

      {/* Right: Action buttons */}
      {children && (
        <div className="flex items-center space-x-2">
          {children}
        </div>
      )}
    </div>
  )
}
