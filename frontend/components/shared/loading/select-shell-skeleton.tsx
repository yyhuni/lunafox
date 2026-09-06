import { Skeleton } from "@/components/ui/skeleton"
import { Select, SelectTrigger } from "@/components/ui/select"
import { cn } from "@/lib/utils"

interface SelectShellSkeletonProps {
  size?: "sm" | "default"
  widthClassName?: string
  valueWidthClassName?: string
  leadingSpaceClassName?: string
  className?: string
}

export function SelectShellSkeleton({
  size = "default",
  widthClassName = "w-28",
  valueWidthClassName = "w-10",
  leadingSpaceClassName,
  className,
}: SelectShellSkeletonProps) {
  return (
    <Select disabled>
      <SelectTrigger
        size={size}
        disabled
        aria-hidden="true"
        data-slot="select-shell-skeleton"
        className={cn("disabled:cursor-default disabled:opacity-100", widthClassName, className)}
      >
        {leadingSpaceClassName ? (
          <span aria-hidden="true" className={cn("shrink-0 opacity-0", leadingSpaceClassName)} />
        ) : null}
        <Skeleton className={cn("h-4 rounded-full", valueWidthClassName)} />
      </SelectTrigger>
    </Select>
  )
}
