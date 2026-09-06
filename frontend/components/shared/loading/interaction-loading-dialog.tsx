"use client"

import type { ReactNode } from "react"

import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { Spinner } from "@/components/shared/loading/spinner"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"

interface InteractionLoadingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  owner: string
  title: ReactNode
  description?: ReactNode
  className?: string
}

export function InteractionLoadingDialog({
  open,
  onOpenChange,
  owner,
  title,
  description,
  className,
}: InteractionLoadingDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={className}>
        <div
          {...getLoadingOwnerAttributes({ owner, layer: "interaction", intent: "interaction" })}
          data-loading-phase="loading"
          aria-busy="true"
          className="space-y-6"
        >
          <DialogHeader className="text-left">
            <div className="flex items-center gap-2">
              <Spinner className="text-muted-foreground" />
              <DialogTitle>{title}</DialogTitle>
            </div>
            {description ? (
              <DialogDescription>{description}</DialogDescription>
            ) : (
              <Skeleton className="h-4 w-64 max-w-full rounded-full" />
            )}
          </DialogHeader>

          <div className="space-y-3">
            <Input aria-hidden="true" disabled tabIndex={-1} className="disabled:opacity-100" />
            <Input aria-hidden="true" disabled tabIndex={-1} className="disabled:opacity-100" />
            <Skeleton className="h-28 w-full rounded-xl" />
            <Skeleton className="h-20 w-full rounded-xl" />
          </div>

          <div className="flex justify-end gap-2">
            <ActionSkeleton widthClassName="w-20" />
            <ActionSkeleton widthClassName="w-28" emphasis="primary" />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
