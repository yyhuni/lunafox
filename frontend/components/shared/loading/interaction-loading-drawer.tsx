"use client"

import type { ReactNode } from "react"

import { DetailDrawer } from "@/components/shared/detail-drawer"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { Spinner } from "@/components/shared/loading/spinner"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface InteractionLoadingDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  owner: string
  title: ReactNode
  description?: ReactNode
}

export function InteractionLoadingDrawer({
  open,
  onOpenChange,
  owner,
  title,
  description,
}: InteractionLoadingDrawerProps) {
  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
    >
      <div
        {...getLoadingOwnerAttributes({ owner, layer: "interaction", intent: "interaction" })}
        data-loading-phase="loading"
        aria-busy="true"
        className="min-h-0 flex-1 overflow-y-auto py-5"
      >
        <div className="space-y-5 px-4 lg:px-6">
          <div className="flex items-center gap-2">
            <Spinner className="text-muted-foreground" />
            <div className="min-w-0">
              <div className={cn("truncate", textRole.panelTitle)}>{title}</div>
              {description ? (
                <div className={cn("truncate", textRole.bodySubtle)}>{description}</div>
              ) : null}
            </div>
          </div>

          <Skeleton className="h-32 w-full rounded-xl" />
          <div className="grid gap-4 sm:grid-cols-2">
            <Skeleton className="h-28 w-full rounded-xl" />
            <Skeleton className="h-28 w-full rounded-xl" />
          </div>
          <Skeleton className="h-72 w-full rounded-xl" />
        </div>
      </div>
    </DetailDrawer>
  )
}
