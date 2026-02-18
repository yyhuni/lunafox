"use client"

import React from "react"
import { 
  IconClock, 
  IconCircleCheck, 
  IconCircleX, 
  IconLoader2, 
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { ScanStatus } from "@/types/scan.types"

// Mock Data
const MOCK_DATA: Array<{ status: ScanStatus; progress: number; label: string }> = [
  { status: "running", progress: 45, label: "Scanning Ports" },
  { status: "running", progress: 78, label: "Analyzing Content" },
  { status: "completed", progress: 100, label: "Done" },
  { status: "failed", progress: 23, label: "Connection Error" },
  { status: "pending", progress: 0, label: "Queued" },
  { status: "cancelled", progress: 12, label: "Stopped" },
]

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

// --- Variant 1: Circular Progress with Center Icon ---
// A donut chart with the status icon in the center. 
// Hover shows progress percentage? Or progress is the ring.
const CircularProgressVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
  const Icon = getStatusIcon(status)
  const color = getStatusColor(status)
  const size = 32
  const strokeWidth = 3
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (progress / 100) * circumference
  
  return (
    <div className="flex items-center gap-3">
      <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
        {/* Background Ring */}
        <svg width={size} height={size} className="rotate-[-90deg]">
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="transparent"
            stroke="currentColor"
            strokeWidth={strokeWidth}
            className="text-muted/20"
          />
          {/* Progress Ring */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="transparent"
            stroke={color}
            strokeWidth={strokeWidth}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            className="transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
          />
        </svg>
        {/* Center Icon */}
        <div className="absolute inset-0 flex items-center justify-center">
           <Icon className={cn("w-3.5 h-3.5", status === "running" && "animate-spin")} style={{ color }} />
        </div>
      </div>
      <div className="flex flex-col">
        <span className="text-sm font-medium leading-none" style={{ color }}>
          {status === "completed" ? "Completed" : status === "running" ? "Scanning..." : status}
        </span>
        {status === "running" && (
            <span className="text-xs text-muted-foreground font-mono mt-0.5">
                {progress}%
            </span>
        )}
      </div>
    </div>
  )
}

// --- Variant 2: Compact Pill with Background Fill ---
// A badge-like pill that fills up like a progress bar.
const CompactPillVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
  const color = getStatusColor(status)
  
  return (
    <div 
      className="relative overflow-hidden rounded-full border h-7 min-w-[100px] flex items-center justify-center"
      style={{ borderColor: status === "running" ? color : "transparent" }}
    >
      {/* Background Fill */}
      <div 
        className="absolute inset-0 opacity-20 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
        style={{ 
          backgroundColor: color,
          width: `${status === "completed" ? 100 : progress}%` 
        }}
      />
      
      {/* Content */}
      <div className="relative z-10 flex items-center gap-1.5 px-3">
        <span className="text-xs font-medium uppercase tracking-wider" style={{ color }}>
          {status}
        </span>
        {status === "running" && (
           <span className="text-[10px] font-mono opacity-80" style={{ color }}>
             {progress}%
           </span>
        )}
      </div>
    </div>
  )
}

// --- Variant 3: Split Badge (Status Left, Progress Right) ---
const SplitBadgeVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
  const Icon = getStatusIcon(status)
  const color = getStatusColor(status)

  return (
    <div className={cn(
        "inline-flex items-center border rounded-md overflow-hidden bg-background h-7",
        status === "running" ? "border-muted-foreground/20" : "border-transparent"
    )}>
      {/* Icon Section */}
      <div 
        className="h-full px-2 flex items-center justify-center"
        style={{ backgroundColor: `${color}15` }}
      >
        <Icon className={cn("w-3.5 h-3.5", status === "running" && "animate-spin")} style={{ color }} />
      </div>
      
      {/* Text/Progress Section */}
      <div className="px-2.5 flex items-center gap-2 min-w-[80px]">
        <div className="flex flex-col gap-0.5 w-full">
            <div className="flex items-center justify-between gap-2">
                <span className="text-[11px] font-medium uppercase" style={{ color }}>{status}</span>
                {status !== "completed" && status !== "pending" && (
                    <span className="text-[10px] text-muted-foreground font-mono">{progress}%</span>
                )}
            </div>
            {status === "running" && (
                <div className="h-0.5 w-full bg-muted/50 rounded-full overflow-hidden">
                    <div className="h-full bg-current transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500" style={{ width: `${progress}%`, color }} />
                </div>
            )}
        </div>
      </div>
    </div>
  )
}

// --- Variant 4: Minimalist Dot & Line ---
const MinimalistVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    
    return (
        <div className="flex flex-col gap-1.5 min-w-[120px]">
            <div className="flex items-center justify-between text-xs">
                <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />
                    <span className="font-medium capitalize text-foreground/80">{status}</span>
                </div>
                <span className="font-mono text-muted-foreground">{displayProgress}%</span>
            </div>
            <div className="h-1 w-full bg-muted rounded-full overflow-hidden">
                 <div 
                    className="h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
                    style={{ backgroundColor: color, width: `${displayProgress}%` }}
                 />
            </div>
        </div>
    )
}

// --- Variant 5: Integrated Cell (Block) ---
const IntegratedBlockVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
    const color = getStatusColor(status)
    const Icon = getStatusIcon(status)
    
    return (
        <div className="relative w-full h-10 rounded-md overflow-hidden group border border-transparent hover:border-border/50 transition-colors">
            {/* Background Progress Bar */}
             <div 
                className="absolute inset-0 opacity-10 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
                style={{ 
                  backgroundColor: color,
                  width: `${status === "completed" ? 100 : progress}%` 
                }}
              />
              
              <div className="relative z-10 h-full flex items-center justify-between px-3">
                 <div className="flex items-center gap-2">
                    <Icon className={cn("w-4 h-4", status === "running" && "animate-spin")} style={{ color }} />
                    <span className="text-sm font-medium capitalize" style={{ color }}>{status}</span>
                 </div>
                 {status === "running" && (
                     <span className="text-xs font-mono font-medium opacity-70" style={{ color }}>{progress}%</span>
                 )}
              </div>
        </div>
    )
}

// --- Variant 6: Segmented Bar (Industrial) ---
const SegmentedBarVariant = ({ 
    status, 
    progress, 
    variant = "default" 
}: { 
    status: ScanStatus, 
    progress: number, 
    variant?: "default" | "inline" | "matrix" | "ruler" 
}) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    const Icon = getStatusIcon(status)
    
    // F1: Default (Classic Stacked)
    if (variant === "default") {
        const segments = 10
        const activeSegments = Math.ceil((displayProgress / 100) * segments)
        return (
            <div className="flex flex-col gap-1.5 w-[140px]">
                <div className="flex items-center justify-between text-[10px] uppercase font-mono tracking-tight leading-none">
                    <div className="flex items-center gap-1.5">
                        <Icon className={cn("w-3 h-3", status === "running" && "animate-spin")} style={{ color }} />
                        <span style={{ color }}>{status}</span>
                    </div>
                    <span className="opacity-70">{displayProgress}%</span>
                </div>
                <div className="flex gap-0.5 h-2">
                    {Array.from({ length: segments }).map((_, i) => (
                        <div 
                            key={i}
                            className="flex-1 rounded-[1px] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300"
                            style={{ 
                                backgroundColor: i < activeSegments ? color : "transparent",
                                opacity: i < activeSegments ? 1 : 0.15,
                                border: `1px solid ${color}`,
                            }}
                        />
                    ))}
                </div>
            </div>
        )
    }

    // F2: Inline (Single Line)
    if (variant === "inline") {
        const segments = 8
        const activeSegments = Math.ceil((displayProgress / 100) * segments)
        return (
            <div className="flex items-center gap-3 w-[160px] h-8 bg-muted/10 px-2 rounded-sm border border-transparent hover:border-border/40 transition-colors">
                <Icon className={cn("w-3.5 h-3.5 flex-shrink-0", status === "running" && "animate-spin")} style={{ color }} />
                <div className="flex flex-col flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                        <span className="text-[9px] uppercase font-bold tracking-wider leading-none" style={{ color }}>{status}</span>
                    </div>
                    <div className="flex gap-[2px] h-1.5 w-full">
                         {Array.from({ length: segments }).map((_, i) => (
                            <div 
                                key={i}
                                className="flex-1 rounded-[1px] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300"
                                style={{ 
                                    backgroundColor: i < activeSegments ? color : `${color}15`, // dim background for empty
                                }}
                            />
                        ))}
                    </div>
                </div>
            </div>
        )
    }

    // F3: Matrix (Dot Grid)
    if (variant === "matrix") {
        // 2 rows of 10 dots = 20 dots total resolution
        const cols = 10
        const rows = 2
        const totalDots = cols * rows
        const activeDots = Math.ceil((displayProgress / 100) * totalDots)
        
        return (
            <div className="flex items-center gap-3">
                 <div className="flex flex-col gap-0.5">
                     <span className="text-[10px] font-mono font-bold uppercase leading-none" style={{ color }}>{status}</span>
                     <span className="text-[9px] font-mono opacity-60 leading-none">{displayProgress.toString().padStart(3, '0')}%</span>
                 </div>
                 <div className="grid gap-[2px]" style={{ gridTemplateColumns: `repeat(${cols}, 1fr)` }}>
                     {Array.from({ length: totalDots }).map((_, i) => {
                         // Map 1D index to 2D logic if needed, but simple linear fill is fine for progress
                         // To fill vertical first or horizontal first? Horizontal is standard.
                         return (
                            <div 
                                key={i}
                                className="w-1 h-1 rounded-[1px] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300"
                                style={{ 
                                    backgroundColor: i < activeDots ? color : `${color}20`,
                                }}
                            />
                         )
                     })}
                 </div>
            </div>
        )
    }

    // F4: Ruler (Technical Scale)
    if (variant === "ruler") {
        return (
            <div className="flex flex-col w-[140px] gap-0.5">
                <div className="flex items-baseline justify-between">
                    <span className="text-[10px] uppercase font-bold tracking-tighter" style={{ color }}>{status}</span>
                    <span className="text-[9px] font-mono">{displayProgress}%</span>
                </div>
                <div className="relative h-2 w-full flex items-end">
                     {/* The Scale Marks */}
                     <div className="absolute inset-0 flex justify-between w-full h-full pointer-events-none">
                        {Array.from({ length: 11 }).map((_, i) => (
                            <div 
                                key={i} 
                                className="w-[1px] bg-foreground/20" 
                                style={{ height: i % 5 === 0 ? '100%' : '50%' }}
                            />
                        ))}
                     </div>
                     {/* The Progress Fill */}
                     <div className="absolute bottom-0 left-0 h-1 bg-current z-10 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 opacity-80" 
                          style={{ width: `${displayProgress}%`, color }} 
                     />
                     {/* Bottom Border */}
                     <div className="absolute bottom-0 left-0 w-full h-[1px] bg-foreground/20" />
                </div>
            </div>
        )
    }

    return null
}

// --- Variant 7: Border Progress (Subtle Frame) ---
const BorderProgressVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    const Icon = getStatusIcon(status)
    
    return (
        <div 
            className="relative px-3 py-1.5 rounded-full border bg-background flex items-center gap-2 w-fit overflow-hidden"
            style={{ borderColor: `${color}40` }}
        >
            <div 
                className="absolute bottom-0 left-0 h-[2px] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
                style={{ backgroundColor: color, width: `${displayProgress}%` }}
            />
            
            <Icon className={cn("w-3.5 h-3.5", status === "running" && "animate-spin")} style={{ color }} />
            <span className="text-xs font-medium uppercase tracking-wide" style={{ color }}>
                {status}
            </span>
            {status === "running" && (
                <span className="text-[10px] font-mono opacity-60 ml-1">
                    {progress}%
                </span>
            )}
        </div>
    )
}

// --- Variant 8: Typographic Focus (Big Number) ---
const TypographicFocusVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    
    return (
        <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold font-mono leading-none tracking-tighter" style={{ color }}>
                {displayProgress}<span className="text-sm opacity-50">%</span>
            </span>
            <div className="flex flex-col">
                 <span className="text-[10px] font-semibold uppercase tracking-wider opacity-70" style={{ color }}>
                    {status}
                 </span>
                 <div className="h-1 w-12 bg-muted rounded-full mt-1 overflow-hidden">
                    <div 
                        className="h-full rounded-full"
                        style={{ backgroundColor: color, width: `${displayProgress}%` }}
                    />
                 </div>
            </div>
        </div>
    )
}

// --- Variant 9: Neon Glow (Dark Mode Specialized) ---
const NeonGlowVariant = ({ status, progress }: { status: ScanStatus, progress: number }) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    const Icon = getStatusIcon(status)

    return (
        <div className="relative group">
            <div className="flex items-center justify-between gap-4 mb-1">
                <div className="flex items-center gap-1.5">
                    <Icon className={cn("w-3.5 h-3.5", status === "running" && "animate-spin")} style={{ color, filter: `drop-shadow(0 0 2px ${color})` }} />
                    <span 
                        className="text-xs font-medium uppercase tracking-wider" 
                        style={{ 
                            color,
                            textShadow: status === "running" ? `0 0 10px ${color}60` : "none"
                        }}
                    >
                        {status}
                    </span>
                </div>
                <span className="text-xs font-mono" style={{ color }}>{displayProgress}%</span>
            </div>
            
            <div className="h-1.5 w-full bg-black/20 rounded-full overflow-hidden border border-white/5">
                <div 
                    className="h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 relative"
                    style={{ 
                        backgroundColor: color, 
                        width: `${displayProgress}%`,
                        boxShadow: `0 0 8px ${color}, 0 0 4px ${color}`
                    }}
                >
                    <div className="absolute inset-0 bg-white/20 animate-pulse" />
                </div>
            </div>
        </div>
    )
}

// --- Variant 10: Compact Dual-Line ---
const CompactDualLineVariant = ({ status, progress, variant = "default" }: { status: ScanStatus, progress: number, variant?: "default" | "icon" | "filled" | "sharp" }) => {
    const color = getStatusColor(status)
    const displayProgress = status === "completed" ? 100 : progress
    const Icon = getStatusIcon(status)
    
    // Base styles
    const baseClasses = "flex flex-col justify-center px-2 border-l-2 relative overflow-hidden transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:bg-muted/30"
    const sizeClasses = "h-9 w-32" // Slightly larger for better readability
    
    // Variant specifics
    const borderStyle = { borderLeftColor: color }
    const roundedClass = variant === "sharp" ? "rounded-none" : "rounded-sm"

    return (
        <div 
            className={cn(baseClasses, sizeClasses, roundedClass, variant === "filled" ? "" : "bg-muted/20")} 
            style={{ ...borderStyle, ...(variant === "filled" ? { backgroundColor: `${color}15` } : {}) }}
        >
            <div className="flex justify-between items-center w-full relative z-10">
                <div className="flex items-center gap-1.5">
                    {variant === "icon" && (
                        <Icon className={cn("w-3 h-3", status === "running" && "animate-spin")} style={{ color }} />
                    )}
                    <span className="text-[10px] font-bold uppercase tracking-tight leading-none" style={{ color }}>
                        {status}
                    </span>
                </div>
                {status !== "pending" && status !== "cancelled" && (
                     <span className="text-[10px] font-mono opacity-70 leading-none">{displayProgress}%</span>
                )}
            </div>
            
            {status !== "completed" && status !== "pending" && status !== "cancelled" && (
                <div className="h-1 w-full bg-foreground/5 mt-1.5 rounded-sm overflow-hidden relative z-10">
                     <div 
                        className="h-full rounded-sm transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
                        style={{ backgroundColor: color, width: `${displayProgress}%` }}
                    />
                </div>
            )}
            
            {/* Optional: Subtle background progress for 'filled' variant or just 'completed' state */}
             {variant === "sharp" && status === "running" && (
                 <div className="absolute bottom-0 left-0 h-[1px] w-full bg-foreground/10">
                    <div 
                        className="h-full bg-current transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 animate-pulse"
                         style={{ backgroundColor: color, width: `${displayProgress}%` }}
                    />
                 </div>
             )}
        </div>
    )
}

export function StatusProgressDemos() {
  return (
    <div className="space-y-12 pb-20">
      
      {/* Focus on Variant F */}
      <section className="space-y-6">
        <div>
            <h2 className="text-2xl font-bold mb-2">Focus: Variant F (Industrial Segments)</h2>
            <p className="text-muted-foreground">Exploring different segmented and data-driven displays.</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            
            {/* F1: Default */}
            <div className="space-y-3">
                <h4 className="font-medium text-sm">F1: Classic Stacked</h4>
                <div className="rounded-lg border p-4 bg-card space-y-2">
                    {MOCK_DATA.slice(0, 3).map((item, i) => (
                        <div key={i} className="flex items-center justify-between">
                             <span className="text-xs text-muted-foreground w-20">{item.label}</span>
                             <SegmentedBarVariant {...item} variant="default" />
                        </div>
                    ))}
                </div>
            </div>

            {/* F2: Inline */}
            <div className="space-y-3">
                <h4 className="font-medium text-sm">F2: Inline Block</h4>
                <div className="rounded-lg border p-4 bg-card space-y-2">
                    {MOCK_DATA.slice(0, 3).map((item, i) => (
                        <div key={i} className="flex items-center justify-between">
                             <span className="text-xs text-muted-foreground w-20">{item.label}</span>
                             <SegmentedBarVariant {...item} variant="inline" />
                        </div>
                    ))}
                </div>
            </div>

            {/* F3: Matrix */}
            <div className="space-y-3">
                <h4 className="font-medium text-sm">F3: Data Matrix</h4>
                <div className="rounded-lg border p-4 bg-card space-y-2">
                    {MOCK_DATA.slice(0, 3).map((item, i) => (
                        <div key={i} className="flex items-center justify-between">
                             <span className="text-xs text-muted-foreground w-20">{item.label}</span>
                             <SegmentedBarVariant {...item} variant="matrix" />
                        </div>
                    ))}
                </div>
            </div>

            {/* F4: Ruler */}
            <div className="space-y-3">
                <h4 className="font-medium text-sm">F4: Technical Ruler</h4>
                <div className="rounded-lg border p-4 bg-card space-y-2">
                    {MOCK_DATA.slice(0, 3).map((item, i) => (
                        <div key={i} className="flex items-center justify-between">
                             <span className="text-xs text-muted-foreground w-20">{item.label}</span>
                             <SegmentedBarVariant {...item} variant="ruler" />
                        </div>
                    ))}
                </div>
            </div>
        </div>
      </section>

      {/* Focus on Variant J (Previous) */}
      <section className="space-y-6 opacity-50 hover:opacity-100 transition-opacity border-t pt-8">
        <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold">Previous Variants (A-I)</h3>
            <span className="text-xs text-muted-foreground">Scroll down to see previous explorations</span>
        </div>
      </section>


      {/* Variant 1 */}
      <section className="space-y-4 hidden">
        <div>
            <h3 className="text-lg font-semibold">Variant A: Circular Indicator</h3>
            <p className="text-sm text-muted-foreground">Focuses on icon with a progress ring. Clean and modern.</p>
        </div>
        <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <CircularProgressVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 2 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant B: Compact Pill</h3>
            <p className="text-sm text-muted-foreground">Combines status badge and progress bar into a single pill.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <CompactPillVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 3 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant C: Split Badge</h3>
            <p className="text-sm text-muted-foreground">Separates icon and status/progress for clearer hierarchy.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <SplitBadgeVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>
      
      {/* Variant 4 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant D: Minimalist</h3>
            <p className="text-sm text-muted-foreground">Standard text with a thin progress line below. Very clean.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <MinimalistVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 5 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant E: Integrated Cell</h3>
            <p className="text-sm text-muted-foreground">Fills the cell background to show progress.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <IntegratedBlockVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 6 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant F: Segmented Bar</h3>
            <p className="text-sm text-muted-foreground">Industrial/Cyberpunk style with segmented progress blocks.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <SegmentedBarVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 7 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant G: Border Progress</h3>
            <p className="text-sm text-muted-foreground">Subtle progress indicator integrated into the pill border/bottom.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <BorderProgressVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 8 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant H: Typographic Focus</h3>
            <p className="text-sm text-muted-foreground">Emphasizes the percentage number over the status text.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <TypographicFocusVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 9 */}
      <section className="space-y-4">
        <div>
            <h3 className="text-lg font-semibold">Variant I: Neon Glow</h3>
            <p className="text-sm text-muted-foreground">High contrast glowing effect, best for dark themes.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card bg-zinc-950 text-zinc-50 dark">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b border-white/10 last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <NeonGlowVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>

      {/* Variant 10 */}
      <section className="space-y-4 hidden">
        <div>
            <h3 className="text-lg font-semibold">Variant J: Compact Dual-Line</h3>
            <p className="text-sm text-muted-foreground">Very dense information display with left border accent.</p>
        </div>
         <div className="rounded-lg border p-4 bg-card">
            <div className="grid grid-cols-1 gap-4">
                {MOCK_DATA.map((item, i) => (
                    <div key={i} className="flex items-center border-b last:border-0 py-3">
                         <div className="w-1/3 text-sm">{item.label}</div>
                         <div className="w-2/3">
                             <CompactDualLineVariant {...item} />
                         </div>
                    </div>
                ))}
            </div>
        </div>
      </section>
    </div>
  )
}
