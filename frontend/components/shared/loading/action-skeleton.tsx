import { buttonIconOnlySizes, buttonStructuralSizeClassNames, type ButtonStructuralSize } from "@/lib/ui/button-size-contract"
import { cn } from "@/lib/utils"

type ActionSkeletonSize = ButtonStructuralSize
type ActionSkeletonEmphasis = "outline" | "primary"

interface ActionSkeletonProps {
  widthClassName?: string
  size?: ActionSkeletonSize
  emphasis?: ActionSkeletonEmphasis
  className?: string
}

export function ActionSkeleton({
  widthClassName,
  size = "default",
  emphasis = "outline",
  className,
}: ActionSkeletonProps) {
  const emphasisClassName =
    emphasis === "primary"
      ? "border-border"
      : "border-input"

  return (
    <div
      data-slot="action-skeleton"
      data-action-emphasis={emphasis}
      aria-hidden="true"
      className={cn(
        "loading-skeleton radius-control shrink-0 border shadow-xs",
        buttonStructuralSizeClassNames[size],
        emphasisClassName,
        size !== "action-card" && !buttonIconOnlySizes.has(size) && (widthClassName ?? "w-24"),
        size === "action-card" && widthClassName,
        className
      )}
    />
  )
}
