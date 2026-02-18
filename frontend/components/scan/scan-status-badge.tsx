import React from "react"
import { 
  IconClock, 
  IconCircleCheck, 
  IconCircleX, 
  IconLoader2, 
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { ScanStatus } from "@/types/scan.types"

// --- Helper for Colors ---
const getStatusColor = (status: ScanStatus) => {
  switch (status) {
    case "completed": return "var(--success)"
    case "failed": return "var(--error)"
    case "running": return "var(--warning)"
    case "pending": return "var(--info)"
    case "cancelled": return "var(--muted-foreground)"
    default: return "var(--muted-foreground)"
  }
}

const getStatusIcon = (status: ScanStatus) => {
  switch (status) {
    case "completed": return IconCircleCheck
    case "failed": return IconCircleX
    case "running": return IconLoader2
    case "pending": return IconClock
    case "cancelled": return IconCircleX
    default: return IconClock
  }
}

interface ScanStatusBadgeProps {
  status: ScanStatus
  progress?: number
  variant?: "default" | "icon" | "filled" | "sharp" | "inline" // Added "inline" for F2
  className?: string
  labels?: Record<ScanStatus, string>
}

export const ScanStatusBadge = ({ 
  status, 
  progress = 0, 
  variant = "inline", // Defaulting to F2
  className,
  labels,
}: ScanStatusBadgeProps) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    const Icon = getStatusIcon(status)
    const label = labels ? labels[status] : status
    
    // --- Variant F2: Inline Block (Segmented) ---
    if (variant === "inline") {
        const segments = 8
        const rawActiveSegments = Math.ceil((displayProgress / 100) * segments)
        const hasProgressBar = status !== "pending"
        const activeSegments = hasProgressBar ? Math.max(1, rawActiveSegments) : rawActiveSegments
        
        return (
            <div className={cn(
                "flex items-center gap-2.5 h-8 rounded-sm transition-colors max-w-[160px]",
                className
            )}>
                <Icon className={cn("w-3.5 h-3.5 flex-shrink-0", status === "running" && "animate-spin")} style={{ color }} />
                <div className="flex flex-col flex-1 min-w-0 justify-center h-full py-1">
                    <div className={cn("flex items-center justify-between leading-none", status !== "pending" && "mb-1.5")}>
                        <span className="text-[10px] uppercase font-bold tracking-wider" style={{ color }}>{label}</span>
                    </div>
                    {status !== "pending" && (
                         <div className="flex gap-[2px] h-1 w-full">
                             {Array.from({ length: segments }).map((_, i) => (
                                <div 
                                    key={i}
                                    className="flex-1 rounded-[1px] transition-[background-color] duration-300"
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
    const roundedClass = variant === "sharp" ? "rounded-none" : "rounded-sm"

    return (
        <div 
            className={cn(baseClasses, sizeClasses, roundedClass, variant === "filled" ? "" : "bg-muted/20", className)} 
            style={{ ...borderStyle, ...(variant === "filled" ? { backgroundColor: `${color}15` } : {}) }}
        >
            <div className="flex justify-between items-center w-full relative z-10">
                <div className="flex items-center gap-1.5">
                    {(variant === "icon" || variant === "filled") && (
                        <Icon className={cn("w-3 h-3", status === "running" && "animate-spin")} style={{ color }} />
                    )}
                    <span className="text-[10px] font-bold uppercase tracking-tight leading-none" style={{ color }}>
                        {label}
                    </span>
                </div>
                {status !== "pending" && status !== "cancelled" && (
                     <span className="text-[10px] font-mono opacity-70 leading-none">{displayProgress}%</span>
                )}
            </div>
            
            {status !== "completed" && status !== "pending" && status !== "cancelled" && (
                <div className="h-1 w-full bg-foreground/5 mt-1.5 rounded-sm overflow-hidden relative z-10">
                     <div 
                        className="h-full rounded-sm transition-[width] duration-500"
                        style={{ backgroundColor: color, width: `${displayProgress}%` }}
                    />
                </div>
            )}
             {variant === "sharp" && status === "running" && (
                 <div className="absolute bottom-0 left-0 h-[1px] w-full bg-foreground/10">
                    <div 
                        className="h-full bg-current transition-[width,background-color] duration-500 animate-pulse"
                         style={{ backgroundColor: color, width: `${displayProgress}%` }}
                    />
                 </div>
             )}
        </div>
    )
}
