import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { Input } from "@/components/ui/input"
import {
  getLoadingOwnerAttributes,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"

interface SettingsPageSkeletonProps {
  owner?: string
  layer?: LoadingLayer
  className?: string
}

export function SettingsPageSkeleton({
  owner,
  layer = "route",
  className,
}: SettingsPageSkeletonProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("SettingsPageSkeleton requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer, intent: "route" }) : {})}
      data-slot="settings-page-skeleton"
      className={cn("space-y-6 px-4 py-4 md:py-6 lg:px-6", className)}
    >
      <div className="space-y-3">
        <Skeleton className="h-8 w-52" />
        <Skeleton className="h-4 w-80 max-w-full" />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="rounded-xl border border-border bg-card/70 p-5 shadow-2xs">
          <div className="space-y-4">
            <Skeleton className="h-5 w-32" />
            <Input aria-hidden="true" disabled tabIndex={-1} className="disabled:opacity-100" />
            <Skeleton className="h-24 w-full rounded-xl" />
            <ActionSkeleton size="lg" widthClassName="w-28" />
          </div>
        </div>
        <div className="space-y-4">
          <div className="rounded-xl border border-border bg-card/70 p-5 shadow-2xs">
            <div className="space-y-3">
              <Skeleton className="h-5 w-24" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-4/5" />
            </div>
          </div>
          <div className="rounded-xl border border-border bg-card/70 p-5 shadow-2xs">
            <div className="space-y-3">
              <Skeleton className="h-5 w-28" />
              <Input aria-hidden="true" disabled tabIndex={-1} className="disabled:opacity-100" />
              <ActionSkeleton size="lg" widthClassName="w-32" />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
