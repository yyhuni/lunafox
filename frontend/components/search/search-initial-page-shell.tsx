import type { ReactNode } from "react"

import { Search } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export const QUICK_SEARCH_TAGS = [
  { label: 'statusCode=="200"', query: 'statusCode=="200"' },
  { label: 'tech="nginx"', query: 'tech="nginx"' },
  { label: 'tech="php"', query: 'tech="php"' },
  { label: 'tech="vue"', query: 'tech="vue"' },
  { label: 'tech="react"', query: 'tech="react"' },
  { label: 'statusCode=="403"', query: 'statusCode=="403"' },
]

export function SearchAssetBarShell({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        "radius-control-prominent flex h-9 w-full min-w-0 overflow-hidden border border-input bg-background shadow-xs",
        className
      )}
    >
      {children}
    </div>
  )
}

export function SearchInitialPageShell({
  children,
  animated = false,
}: {
  children: ReactNode
  animated?: boolean
}) {
  return (
    <div
      className={cn(
        "relative flex flex-1 flex-col items-center justify-center overflow-hidden px-4",
        animated && "animate-in fade-in slide-in-from-bottom-4 duration-300"
      )}
    >
      <div className="-mt-16 flex w-full max-w-4xl flex-col items-center gap-6">
        {children}
      </div>
    </div>
  )
}

export function SearchInitialHeading({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="flex flex-col items-center gap-1">
      <div className="flex items-center gap-2">
        <Search className="h-6 w-6 text-primary" />
        <h1 className={textRole.pageTitle}>{title}</h1>
      </div>
      <p className={textRole.bodySubtle}>{hint}</p>
    </div>
  )
}

export function SearchQuickTags({ onTagClick }: { onTagClick?: (query: string) => void }) {
  return (
    <div className="flex flex-wrap justify-center gap-2">
      {QUICK_SEARCH_TAGS.map((tag) => (
        <Badge
          key={tag.query}
          variant="outline"
          className="px-3 py-1 transition-colors hover:bg-accent"
          render={(
            <button
              type="button"
              disabled={!onTagClick}
              onClick={onTagClick ? () => onTagClick(tag.query) : undefined}
            />
          )}
        >
          {tag.label}
        </Badge>
      ))}
    </div>
  )
}
