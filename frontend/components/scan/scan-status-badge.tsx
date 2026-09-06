import React from "react"
import { 
  IconClock, 
  IconLoader2, 
  semanticIcons,
} from "@/components/icons"
import { getScanStatusColorVar } from "@/lib/status-config"
import { cn } from "@/lib/utils"
import { ScanStatus } from "@/types/scan.types"

const getStatusIcon = (status: ScanStatus) => {
  switch (status) {
    case "succeeded": return semanticIcons.status.success
    case "failed": return semanticIcons.status.failed
    case "running": return IconLoader2
    case "pending": return IconClock
    case "cancelled": return semanticIcons.status.cancelled
    default: return IconClock
  }
}

interface ScanStatusBadgeProps {
  status: ScanStatus
  progress?: number
  variant?: "default" | "icon" | "icon-only" | "filled" | "sharp" | "inline" // Added "inline" for F2
  className?: string
  label?: string
  labels?: Record<ScanStatus, string>
}

export const ScanStatusBadge = React.memo(function ScanStatusBadge({ 
  status, 
  progress = 0, 
  variant = "inline", // Defaulting to F2
  className,
  label: labelOverride,
  labels,
}: ScanStatusBadgeProps) {
    const color = getScanStatusColorVar(status)
    const displayProgress = status === "succeeded" ? 100 : progress
    const Icon = getStatusIcon(status)
    const label = labelOverride ?? (labels ? labels[status] : status)

    if (variant === "icon-only") {
        return (
            <span
                role="img"
                aria-label={label}
                title={label}
                className={cn("inline-flex size-4 shrink-0 items-center justify-center", className)}
            >
                <Icon aria-hidden="true" className={cn("size-3.5", status === "running" && "animate-spin")} style={{ color }} />
            </span>
        )
    }
    
    // --- Variant F2: Inline Block (Segmented) ---
    if (variant === "inline") {
        const segments = 8
        const rawActiveSegments = Math.ceil((displayProgress / 100) * segments)
        const hasProgressBar = status !== "pending"
        const activeSegments = hasProgressBar ? Math.max(1, rawActiveSegments) : rawActiveSegments
        
        return (
            <div className={cn(
                "flex items-center gap-2.5 h-8 rounded-lg transition-colors max-w-40",
                className
            )}>
                <Icon className={cn("w-3.5 h-3.5 flex-shrink-0", status === "running" && "animate-spin")} style={{ color }} />
                <div className="flex flex-1 flex-col h-full justify-center min-w-0 py-1">
                    <div className={cn("flex items-center justify-between leading-none", status !== "pending" && "mb-1.5")}>
                        <span className="font-bold text-[10px] tracking-wider uppercase" style={{ color }}>{label}</span>
                    </div>
                    {status !== "pending" && (
                         <div className="flex gap-0.5 h-1 w-full">
                             {Array.from({ length: segments }).map((_, i) => (
                                <div 
                                    key={i}
                                    className="duration-300 flex-1 rounded-none transition-[background-color]"
                                    style={{ 
                                        backgroundColor: i < activeSegments ? color : `${color}20`, // Increased opacity for empty segments slightly for visibility on transparent bg
                                    }}
                                />
                            ))}
                        </div>
                    )}
                </div>
            </div>
        )
    }

    // --- Fallback / Previous Variant J (Compact Dual-Line) ---
    // Kept for backward compatibility if needed, though we primarily use F2 now
    const baseClasses = "flex flex-col justify-center px-2 border-l-2 relative overflow-hidden transition-colors hover:bg-muted/30"
    const sizeClasses = "h-9 w-32" 
    const borderStyle = { borderLeftColor: color }
    const roundedClass = "rounded-lg"

    return (
        <div 
            className={cn(baseClasses, sizeClasses, roundedClass, variant === "filled" ? "" : "bg-muted/20", className)} 
            style={{ ...borderStyle, ...(variant === "filled" ? { backgroundColor: `${color}15` } : {}) }}
        >
            <div className="flex items-center justify-between relative w-full z-10">
                <div className="flex gap-1.5 items-center">
                    {(variant === "icon" || variant === "filled") && (
                        <Icon className={cn("w-3 h-3", status === "running" && "animate-spin")} style={{ color }} />
                    )}
                    <span className="font-bold leading-none text-[10px] tracking-tight uppercase" style={{ color }}>
                        {label}
                    </span>
                </div>
                {status !== "pending" && status !== "cancelled" && (
                     <span className="font-mono leading-none opacity-70 text-[10px]">{displayProgress}%</span>
                )}
            </div>
            
            {status !== "succeeded" && status !== "pending" && status !== "cancelled" && (
                <div className="bg-foreground/5 h-1 mt-1.5 overflow-hidden relative rounded-md w-full z-10">
                     <div 
                        className="duration-500 h-full w-full origin-left rounded-md transition-[transform,background-color]"
                        style={{ backgroundColor: color, transform: `scaleX(${displayProgress / 100})` }}
                    />
                </div>
             )}
             {variant === "sharp" && status === "running" && (
                 <div className="absolute bg-foreground/10 bottom-0 h-px left-0 w-full">
                    <div 
                        className="animate-pulse bg-current duration-500 h-full w-full origin-left transition-[transform,background-color]"
                         style={{ backgroundColor: color, transform: `scaleX(${displayProgress / 100})` }}
                    />
                 </div>
             )}
        </div>
    )
})
