"use client"

import React from "react"
import type { Icon as CarbonIcon } from "@/components/icons"
import {
  IconServer,
  IconActivity,
  IconCpu,
  IconDatabase,
  IconClock,
  IconSettings,
  IconDotsVertical,
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"

// --- Mock Data ---
interface AgentNode {
  id: string
  name: string
  hostname: string
  status: "online" | "offline" | "maintenance"
  version: string
  ip: string
  lastHeartbeat: string
  metrics: {
    cpu: number
    mem: number
    disk: number
    tasks: number
    uptime: string
  }
  config: {
    cpuThreshold: number
    memThreshold: number
    diskThreshold: number
    maxTasks: number
  }
}

const MOCK_NODES: AgentNode[] = [
  {
    id: "1",
    name: "worker-prod-01",
    hostname: "hk-node-alpha",
    status: "online",
    version: "v1.2.0",
    ip: "10.0.1.15",
    lastHeartbeat: "2s ago",
    metrics: { cpu: 45, mem: 62, disk: 28, tasks: 12, uptime: "4d 12h" },
    config: { cpuThreshold: 90, memThreshold: 90, diskThreshold: 85, maxTasks: 20 }
  },
  {
    id: "2",
    name: "worker-prod-02",
    hostname: "sg-node-bravo",
    status: "online",
    version: "v1.2.0",
    ip: "10.0.1.16",
    lastHeartbeat: "5s ago",
    metrics: { cpu: 88, mem: 75, disk: 45, tasks: 8, uptime: "12d 5h" },
    config: { cpuThreshold: 90, memThreshold: 90, diskThreshold: 85, maxTasks: 20 }
  },
  {
    id: "3",
    name: "worker-dev-01",
    hostname: "local-dev-x1",
    status: "offline",
    version: "v1.1.9",
    ip: "192.168.1.50",
    lastHeartbeat: "1h ago",
    metrics: { cpu: 0, mem: 0, disk: 0, tasks: 0, uptime: "-" },
    config: { cpuThreshold: 90, memThreshold: 90, diskThreshold: 85, maxTasks: 10 }
  },
  {
    id: "4",
    name: "worker-gpu-01",
    hostname: "us-west-gpu-01",
    status: "maintenance",
    version: "v1.2.1-beta",
    ip: "10.0.2.10",
    lastHeartbeat: "10m ago",
    metrics: { cpu: 12, mem: 34, disk: 15, tasks: 2, uptime: "1h 20m" },
    config: { cpuThreshold: 95, memThreshold: 95, diskThreshold: 90, maxTasks: 4 }
  }
]

// --- Helper Components ---

function StatusBadge({ status }: { status: string }) {
  const styles = {
    online: "bg-[var(--success)]/15 text-[var(--success)] border-[var(--success)]/20",
    offline: "bg-muted text-muted-foreground border-border",
    maintenance: "bg-[var(--warning)]/15 text-[var(--warning)] border-[var(--warning)]/20"
  }
  
  return (
    <div className={cn("px-2 py-0.5 rounded-full text-[10px] uppercase font-bold border", styles[status as keyof typeof styles])}>
      {status}
    </div>
  )
}

function getMetricColor(value: number) {
  if (value >= 90) return "bg-[var(--error)]"
  if (value >= 75) return "bg-[var(--warning)]"
  return "bg-[var(--success)]"
}

// --- Metric Progress Variant A: Slim Line ---
function MetricSlim({
  label,
  value,
  icon: IconComp,
}: {
  label: string
  value: number
  icon: CarbonIcon
}) {
  return (
    <div className="space-y-1">
      <div className="flex justify-between items-center text-xs">
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <IconComp className="w-3.5 h-3.5" />
          <span>{label}</span>
        </div>
        <span className="font-mono font-medium">{value}%</span>
      </div>
      <div className="h-1 w-full bg-muted rounded-full overflow-hidden">
        <div 
          className={cn("h-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500", getMetricColor(value))} 
          style={{ width: `${value}%` }}
        />
      </div>
    </div>
  )
}

// --- Metric Progress Variant B: Thick Bar with Inside Text ---
function MetricThick({ label, value }: { label: string, value: number }) {
  const colorClass = getMetricColor(value)
  return (
    <div className="relative h-6 w-full bg-muted/50 rounded overflow-hidden">
      <div 
        className={cn("absolute inset-y-0 left-0 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 opacity-20", colorClass)}
        style={{ width: `${value}%` }}
      />
      <div 
        className={cn("absolute inset-y-0 left-0 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 w-1", colorClass)}
        style={{ left: `${value}%` }}
      />
      <div className="absolute inset-0 flex items-center justify-between px-2 text-xs font-medium">
        <span>{label}</span>
        <span className="font-mono">{value}%</span>
      </div>
    </div>
  )
}

// --- Metric Progress Variant C: Segmented ---
function MetricSegmented({ label, value }: { label: string, value: number }) {
  const segments = 10
  const active = Math.ceil((value / 100) * segments)
  const colorClass = getMetricColor(value)
  
  return (
    <div className="flex items-center justify-between gap-3 text-xs">
      <span className="w-8 text-muted-foreground uppercase tracking-wider text-[10px]">{label}</span>
      <div className="flex-1 flex gap-0.5 h-3">
        {Array.from({ length: segments }).map((_, i) => (
          <div 
            key={i}
            className={cn(
              "flex-1 rounded-[1px] transition-[color,background-color,border-color,opacity,transform,box-shadow]",
              i < active ? colorClass : "bg-muted/30"
            )}
          />
        ))}
      </div>
      <span className="w-8 text-right font-mono text-muted-foreground">{value}%</span>
    </div>
  )
}

// --- Metric Progress Variant D: Circular ---
function MetricCircular({
  label,
  value,
  icon: IconComp,
}: {
  label: string
  value: number
  icon: CarbonIcon
}) {
  const size = 36
  const stroke = 3
  const radius = (size - stroke) / 2
  const circumference = 2 * Math.PI * radius
  const offset = circumference - (value / 100) * circumference
  
  const colorVar = value >= 90 ? "var(--error)" : value >= 75 ? "var(--warning)" : "var(--success)"

  return (
    <div className="flex flex-col items-center gap-1">
      <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="rotate-[-90deg]">
          <circle cx={size/2} cy={size/2} r={radius} fill="none" stroke="currentColor" strokeWidth={stroke} className="text-muted/20" />
          <circle 
            cx={size/2} cy={size/2} r={radius} fill="none" stroke={colorVar} strokeWidth={stroke}
            strokeDasharray={circumference} strokeDashoffset={offset} strokeLinecap="round"
            className="transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <IconComp className="w-3.5 h-3.5 text-muted-foreground" />
        </div>
      </div>
      <div className="flex flex-col items-center leading-none">
        <span className="text-[10px] font-bold">{value}%</span>
        <span className="text-[8px] text-muted-foreground uppercase">{label}</span>
      </div>
    </div>
  )
}


// --- Card Variant 1: Compact List Item ---
function CardCompact({ node }: { node: AgentNode }) {
  return (
    <div className="group flex items-center justify-between p-3 border rounded-lg bg-card hover:shadow-sm transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:border-primary/20">
      <div className="flex items-center gap-3">
        <div className={cn(
          "w-2 h-2 rounded-full",
          node.status === "online" ? "bg-[var(--success)] shadow-[0_0_8px_var(--success)]" : "bg-muted-foreground"
        )} />
        <div>
          <h4 className="text-sm font-medium leading-none mb-1">{node.name}</h4>
          <p className="text-xs text-muted-foreground font-mono">{node.ip}</p>
        </div>
        <Badge variant="outline" className="ml-2 text-[10px] h-5">{node.version}</Badge>
      </div>

      <div className="hidden md:flex items-center gap-8">
        <div className="w-24">
           <MetricSlim label="CPU" value={node.metrics.cpu} icon={IconCpu} />
        </div>
        <div className="w-24">
           <MetricSlim label="MEM" value={node.metrics.mem} icon={IconDatabase} />
        </div>
      </div>

      <div className="flex items-center gap-2">
         <span className="text-xs text-muted-foreground mr-2">{node.metrics.uptime}</span>
         <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground" aria-label="Open settings"><IconSettings className="w-4 h-4" /></Button>
      </div>
    </div>
  )
}

// --- Card Variant 2: Modern Grid Card ---
function CardModern({ node }: { node: AgentNode }) {
  return (
    <Card className="overflow-hidden hover:border-primary/50 transition-colors">
      <CardHeader className="p-4 pb-2 flex flex-row items-center justify-between space-y-0">
        <div className="flex items-center gap-2">
            <div className={cn(
              "p-1.5 rounded-md",
              node.status === "online" ? "bg-[var(--success)]/10 text-[var(--success)]" : "bg-muted text-muted-foreground"
            )}>
              <IconServer className="w-4 h-4" />
            </div>
            <div>
              <CardTitle className="text-sm font-medium">{node.name}</CardTitle>
              <CardDescription className="text-[10px] font-mono">{node.ip}</CardDescription>
            </div>
        </div>
        <StatusBadge status={node.status} />
      </CardHeader>
      <CardContent className="p-4 pt-2 space-y-4">
        <div className="grid grid-cols-2 gap-2 mt-2">
            <div className="bg-muted/30 p-2 rounded flex flex-col items-center justify-center gap-1">
               <span className="text-[10px] text-muted-foreground uppercase">Tasks</span>
               <span className="text-lg font-semibold tabular-nums">{node.metrics.tasks}</span>
            </div>
            <div className="bg-muted/30 p-2 rounded flex flex-col items-center justify-center gap-1">
               <span className="text-[10px] text-muted-foreground uppercase">Uptime</span>
               <span className="text-sm font-semibold">{node.metrics.uptime}</span>
            </div>
        </div>
        <div className="space-y-2">
          <MetricThick label="CPU Usage" value={node.metrics.cpu} />
          <MetricThick label="Memory" value={node.metrics.mem} />
        </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 3: Technical / Dashboard ---
function CardTechnical({ node }: { node: AgentNode }) {
  return (
    <div className="border rounded-sm bg-card/50">
      {/* Header Bar */}
      <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/20">
        <div className="flex items-center gap-2">
           <span className={cn("w-1.5 h-1.5 rounded-sm", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
           <span className="text-xs font-bold uppercase tracking-wider">{node.name}</span>
        </div>
        <span className="text-[10px] font-mono opacity-60">{node.version}</span>
      </div>
      
      {/* Metrics Area */}
      <div className="p-3 grid grid-cols-3 gap-4">
        <MetricCircular label="CPU" value={node.metrics.cpu} icon={IconCpu} />
        <MetricCircular label="RAM" value={node.metrics.mem} icon={IconDatabase} />
        <div className="flex flex-col items-center justify-center">
            <span className="text-xl font-bold leading-none">{node.metrics.tasks}</span>
            <span className="text-[8px] uppercase text-muted-foreground mt-1">Tasks</span>
        </div>
      </div>
      
      {/* Footer Info */}
      <div className="px-3 py-1.5 bg-muted/10 border-t flex items-center justify-between text-[10px] text-muted-foreground font-mono">
         <span>{node.ip}</span>
         <span>UP: {node.metrics.uptime}</span>
      </div>
    </div>
  )
}

// --- Card Variant 4: Segmented (Industrial) ---
function CardIndustrial({ node }: { node: AgentNode }) {
  return (
    <div className="relative border p-4 bg-background group">
      {/* Corner Accents */}
      <div className="absolute top-0 left-0 w-2 h-2 border-l border-t border-foreground/30" />
      <div className="absolute top-0 right-0 w-2 h-2 border-r border-t border-foreground/30" />
      <div className="absolute bottom-0 left-0 w-2 h-2 border-l border-b border-foreground/30" />
      <div className="absolute bottom-0 right-0 w-2 h-2 border-r border-b border-foreground/30" />

      <div className="flex items-start justify-between mb-4">
        <div>
           <div className="flex items-center gap-2 mb-1">
             <IconServer className="w-4 h-4 text-primary" />
             <h3 className="text-sm font-bold uppercase tracking-widest">{node.name}</h3>
           </div>
           <div className="flex items-center gap-2 text-[10px] text-muted-foreground font-mono">
             <span>ID: {node.id.padStart(4, '0')}</span>
             <span>|</span>
             <span>{node.ip}</span>
           </div>
        </div>
        <StatusBadge status={node.status} />
      </div>

      <div className="space-y-3">
        <MetricSegmented label="CPU" value={node.metrics.cpu} />
        <MetricSegmented label="MEM" value={node.metrics.mem} />
        <MetricSegmented label="DSK" value={node.metrics.disk} />
      </div>
    </div>
  )
}


// --- Card Variant 5: Hero Status (Big Health Score) ---
function CardHeroStatus({ node }: { node: AgentNode }) {
  // Calculate a mock "Health Score" based on metrics
  const healthScore = Math.max(0, 100 - Math.max(node.metrics.cpu, node.metrics.mem) / 2)

  return (
    <div className="flex border rounded-xl overflow-hidden bg-card hover:shadow-md transition-[color,background-color,border-color,opacity,transform,box-shadow]">
      {/* Left: Status Hero */}
      <div className={cn(
        "w-24 flex flex-col items-center justify-center gap-2 p-2 border-r",
        node.status === "online" ? "bg-[var(--success)]/5" : "bg-muted/30"
      )}>
        <div className={cn(
          "w-12 h-12 rounded-full flex items-center justify-center border-4 text-xs font-bold",
          node.status === "online" 
            ? "border-[var(--success)] text-[var(--success)] bg-[var(--success)]/10" 
            : "border-muted-foreground text-muted-foreground bg-muted"
        )}>
          {node.status === "online" ? Math.round(healthScore) : "-"}
        </div>
        <span className={cn(
          "text-[10px] font-bold uppercase tracking-wider",
          node.status === "online" ? "text-[var(--success)]" : "text-muted-foreground"
        )}>
          {node.status}
        </span>
      </div>

      {/* Right: Details */}
      <div className="flex-1 p-4">
        <div className="flex justify-between items-start mb-4">
          <div>
            <h3 className="font-semibold text-sm">{node.name}</h3>
            <p className="text-xs text-muted-foreground">{node.ip} • {node.version}</p>
          </div>
          <Button variant="ghost" size="sm" className="h-6 w-6 p-0">
             <IconDotsVertical className="w-4 h-4" />
          </Button>
        </div>

        <div className="space-y-3">
          <div className="flex items-center gap-3 text-xs">
            <IconCpu className="w-3.5 h-3.5 text-muted-foreground" />
            <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
               <div className="h-full bg-foreground/70 rounded-full" style={{ width: `${node.metrics.cpu}%` }} />
            </div>
            <span className="w-8 text-right tabular-nums">{node.metrics.cpu}%</span>
          </div>
          <div className="flex items-center gap-3 text-xs">
            <IconDatabase className="w-3.5 h-3.5 text-muted-foreground" />
             <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
               <div className="h-full bg-foreground/70 rounded-full" style={{ width: `${node.metrics.mem}%` }} />
            </div>
            <span className="w-8 text-right tabular-nums">{node.metrics.mem}%</span>
          </div>
        </div>
      </div>
    </div>
  )
}

// --- Card Variant 6: Resource Monitor (Sparklines) ---
function CardResourceGraph({ node }: { node: AgentNode }) {
  // Mock sparkline data generator
  const generateSparkline = (base: number) => {
    return Array.from({ length: 12 }, () => {
      const noise = Math.random() * 20 - 10
      return Math.max(5, Math.min(100, base + noise))
    })
  }
  
  const cpuTrend = generateSparkline(node.metrics.cpu)
  const memTrend = generateSparkline(node.metrics.mem)

  const TrendLine = ({ data, color }: { data: number[], color: string }) => (
    <div className="flex items-end gap-[2px] h-8 w-full mt-2 opacity-80">
      {data.map((h, i) => (
        <div 
          key={i} 
          className={cn("flex-1 rounded-[1px]", color)} 
          style={{ height: `${h}%`, minHeight: '4px' }}
        />
      ))}
    </div>
  )

  return (
    <Card className="hover:border-primary/40 transition-colors">
      <CardHeader className="p-4 pb-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
             <StatusBadge status={node.status} />
             <span className="font-semibold text-sm">{node.name}</span>
          </div>
          <span className="text-[10px] text-muted-foreground">{node.metrics.uptime}</span>
        </div>
      </CardHeader>
      <CardContent className="p-4 pt-2">
         <div className="grid grid-cols-2 gap-4">
            <div className="p-2 rounded border bg-muted/10">
               <div className="flex justify-between items-end">
                 <span className="text-[10px] text-muted-foreground uppercase font-bold">CPU</span>
                 <span className="text-lg font-bold leading-none">{node.metrics.cpu}%</span>
               </div>
               <TrendLine data={cpuTrend} color={node.metrics.cpu > 80 ? "bg-[var(--warning)]" : "bg-primary"} />
            </div>
            <div className="p-2 rounded border bg-muted/10">
               <div className="flex justify-between items-end">
                 <span className="text-[10px] text-muted-foreground uppercase font-bold">MEM</span>
                 <span className="text-lg font-bold leading-none">{node.metrics.mem}%</span>
               </div>
               <TrendLine data={memTrend} color={node.metrics.mem > 80 ? "bg-[var(--warning)]" : "bg-primary"} />
            </div>
         </div>
         <div className="mt-3 text-[10px] text-muted-foreground flex justify-between items-center">
            <span>Tasks: {node.metrics.tasks} running</span>
            <span className="font-mono">{node.ip}</span>
         </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 7: Terminal/Console Style ---
function CardTerminal({ node }: { node: AgentNode }) {
  return (
    <div className="bg-[#0c0c0c] border border-zinc-800 rounded text-zinc-300 font-mono text-xs overflow-hidden shadow-lg">
       {/* Terminal Header */}
       <div className="flex items-center justify-between px-3 py-1.5 bg-zinc-900 border-b border-zinc-800">
          <div className="flex gap-1.5">
             <div className="w-2.5 h-2.5 rounded-full bg-red-500/80" />
             <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/80" />
             <div className="w-2.5 h-2.5 rounded-full bg-green-500/80" />
          </div>
          <div className="text-[10px] opacity-50">root@{node.name}</div>
       </div>
       
       {/* Terminal Body */}
       <div className="p-3 space-y-2">
          <div className="flex gap-2">
             <span className="text-green-500">➜</span>
             <span>status --check</span>
          </div>
          <div className="pl-4 opacity-80">
             [INFO] Node is {node.status.toUpperCase()}<br/>
             [INFO] Uptime: {node.metrics.uptime}<br/>
             [INFO] Version: {node.version}
          </div>
          
          <div className="flex gap-2 pt-1">
             <span className="text-green-500">➜</span>
             <span>resources</span>
          </div>
          <div className="pl-4 space-y-1">
             <div className="flex items-center gap-2">
                <span className="w-8">CPU</span>
                <div className="w-24 h-2 bg-zinc-800">
                   <div className="h-full bg-zinc-400" style={{ width: `${node.metrics.cpu}%` }} />
                </div>
                <span>{node.metrics.cpu}%</span>
             </div>
             <div className="flex items-center gap-2">
                <span className="w-8">MEM</span>
                <div className="w-24 h-2 bg-zinc-800">
                   <div className="h-full bg-zinc-400" style={{ width: `${node.metrics.mem}%` }} />
                </div>
                <span>{node.metrics.mem}%</span>
             </div>
          </div>
       </div>
    </div>
  )
}

// --- Card Variant 8: Minimalist Glass (Airy) ---
function CardGlass({ node }: { node: AgentNode }) {
  const load = Math.max(node.metrics.cpu, node.metrics.mem, node.metrics.disk)

  return (
    <div className="group relative overflow-hidden rounded-xl border bg-background/50 backdrop-blur-sm p-4 hover:bg-muted/20 transition-[color,background-color,border-color,opacity,transform,box-shadow]">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <div
              className={cn(
                "w-9 h-9 rounded-lg flex items-center justify-center transition-colors",
                node.status === "online" ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
              )}
            >
              <IconServer className="w-4 h-4" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="font-medium text-sm leading-tight truncate" title={node.name}>
                  {node.name}
                </h3>
                <Badge variant="outline" className="text-[10px] h-5 font-mono text-muted-foreground">
                  {node.version}
                </Badge>
              </div>
              <p className="text-[10px] text-muted-foreground font-mono truncate" title={node.hostname}>
                {node.hostname}
              </p>
            </div>
          </div>

          <div className="mt-2 text-[10px] text-muted-foreground font-mono truncate" title={node.ip}>
            {node.ip}
          </div>
        </div>

        <div className="flex flex-col items-end gap-1 shrink-0">
          <StatusBadge status={node.status} />
          <div className="text-[10px] text-muted-foreground">{node.metrics.tasks} tasks</div>
        </div>
      </div>

      <div className="mt-3 space-y-3">
        <div className="flex items-center justify-between text-[10px] text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <IconActivity className="w-3 h-3" />
            {node.lastHeartbeat}
          </span>
          <span className="uppercase tracking-wider">load {load}%</span>
        </div>

        <div className="space-y-2">
          <MetricSlim label="CPU" value={node.metrics.cpu} icon={IconCpu} />
          <MetricSlim label="MEM" value={node.metrics.mem} icon={IconDatabase} />
          <div className="space-y-1">
            <div className="flex justify-between items-center text-xs">
              <span className="text-muted-foreground flex items-center gap-1.5">
                <IconDatabase className="w-3.5 h-3.5" />
                <span>DISK</span>
              </span>
              <span className="font-mono font-medium">{node.metrics.disk}%</span>
            </div>
            <Progress value={node.metrics.disk} className="h-1" />
          </div>
        </div>
      </div>
    </div>
  )
}

// --- Card Variant 9: Data Grid (Information Density) ---
function CardDataGrid({ node }: { node: AgentNode }) {
  return (
    <Card className="hover:border-primary/50 transition-[color,background-color,border-color,opacity,transform,box-shadow]">
      <CardHeader className="p-3 pb-2 border-b bg-muted/20">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <IconServer className="w-4 h-4 text-muted-foreground" />
            <span className="font-semibold text-sm">{node.name}</span>
          </div>
          <StatusBadge status={node.status} />
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="grid grid-cols-2 text-xs divide-x divide-y border-b">
          <div className="p-2 space-y-0.5">
            <span className="text-[10px] text-muted-foreground uppercase">Hostname</span>
            <div className="font-mono truncate" title={node.hostname}>{node.hostname}</div>
          </div>
          <div className="p-2 space-y-0.5">
             <span className="text-[10px] text-muted-foreground uppercase">IP Address</span>
             <div className="font-mono">{node.ip}</div>
          </div>
          <div className="p-2 space-y-0.5">
             <span className="text-[10px] text-muted-foreground uppercase">Version</span>
             <div className="font-mono">{node.version}</div>
          </div>
          <div className="p-2 space-y-0.5">
             <span className="text-[10px] text-muted-foreground uppercase">Heartbeat</span>
             <div className="flex items-center gap-1.5">
               <IconActivity className="w-3 h-3 text-muted-foreground" />
               <span>{node.lastHeartbeat}</span>
             </div>
          </div>
        </div>
        
        <div className="p-3 space-y-3 bg-muted/5">
           <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">Active Tasks</span>
              <Badge variant="secondary" className="h-5 text-[10px]">{node.metrics.tasks}</Badge>
           </div>
           
           <div className="space-y-2">
              <MetricSlim label="CPU" value={node.metrics.cpu} icon={IconCpu} />
              <MetricSlim label="MEM" value={node.metrics.mem} icon={IconDatabase} />
              <div className="flex items-center justify-between text-xs pt-1">
                 <span className="text-muted-foreground flex items-center gap-1.5">
                    <IconDatabase className="w-3.5 h-3.5" />
                    DSK
                 </span>
                 <span className="font-mono">{node.metrics.disk}%</span>
              </div>
           </div>
        </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 10: Infrastructure Rack (Blade Server) ---
function CardInfraRack({ node }: { node: AgentNode }) {
  return (
    <div className="flex h-24 border rounded-md overflow-hidden bg-zinc-950 text-zinc-400 group hover:ring-1 hover:ring-zinc-700 transition-[color,background-color,border-color,opacity,transform,box-shadow]">
      {/* Handle/Status Strip */}
      <div className={cn(
        "w-1.5 h-full",
        node.status === "online" ? "bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.5)]" : "bg-zinc-700"
      )} />
      
      {/* Main Body */}
      <div className="flex-1 flex flex-col p-2 gap-1.5 bg-[url('/images/noise.png')]">
        <div className="flex items-center justify-between">
           <div className="flex items-center gap-2">
              <span className="font-bold text-zinc-100 text-sm tracking-tight">{node.name}</span>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400 font-mono">{node.ip}</span>
           </div>
           <div className="flex gap-1">
              <div className={cn("w-1.5 h-1.5 rounded-full", node.status === "online" ? "bg-emerald-500 animate-pulse" : "bg-red-900")} />
              <div className="w-1.5 h-1.5 rounded-full bg-zinc-800" />
           </div>
        </div>
        
        <div className="flex-1 grid grid-cols-2 gap-4 items-center">
           <div className="space-y-1 text-[10px] font-mono leading-tight">
              <div className="flex justify-between"><span>HOST:</span> <span className="text-zinc-300">{node.hostname}</span></div>
              <div className="flex justify-between"><span>VER :</span> <span className="text-zinc-300">{node.version}</span></div>
              <div className="flex justify-between"><span>HBT :</span> <span className="text-zinc-300">{node.lastHeartbeat}</span></div>
           </div>
           
           <div className="space-y-1.5">
              <div className="flex items-center gap-2 text-[9px] uppercase">
                 <span className="w-6">CPU</span>
                 <div className="flex-1 h-1 bg-zinc-800 overflow-hidden">
                    <div className="h-full bg-zinc-500" style={{ width: `${node.metrics.cpu}%` }} />
                 </div>
              </div>
              <div className="flex items-center gap-2 text-[9px] uppercase">
                 <span className="w-6">MEM</span>
                 <div className="flex-1 h-1 bg-zinc-800 overflow-hidden">
                    <div className="h-full bg-zinc-500" style={{ width: `${node.metrics.mem}%` }} />
                 </div>
              </div>
              <div className="flex items-center gap-2 text-[9px] uppercase">
                 <span className="w-6">DSK</span>
                 <div className="flex-1 h-1 bg-zinc-800 overflow-hidden">
                    <div className="h-full bg-zinc-500" style={{ width: `${node.metrics.disk}%` }} />
                 </div>
              </div>
           </div>
        </div>
      </div>
      
      {/* Right Actions / Task Count */}
      <div className="w-10 border-l border-zinc-800 flex flex-col items-center justify-center gap-1 bg-zinc-900/50">
         <span className="text-[10px] font-bold">{node.metrics.tasks}</span>
         <span className="text-[8px] uppercase opacity-50 -rotate-90 origin-center w-max">Tasks</span>
      </div>
    </div>
  )
}

// --- Card Variant 11: Comprehensive Analytics ---
function CardAnalytics({ node }: { node: AgentNode }) {
  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="p-4 pb-2">
         <div className="flex justify-between items-start">
            <div>
               <CardTitle className="text-base">{node.name}</CardTitle>
               <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                  <IconServer className="w-3 h-3" />
                  <span>{node.hostname}</span>
                  <span>•</span>
                  <span>{node.ip}</span>
               </div>
            </div>
            <StatusBadge status={node.status} />
         </div>
      </CardHeader>
      <CardContent className="p-4 pt-2">
         <div className="grid grid-cols-2 gap-4 mb-4 py-3 border-y border-dashed bg-muted/5">
             <div className="text-center border-r border-dashed">
                <div className="text-2xl font-bold tracking-tight">{node.metrics.tasks}</div>
                <div className="text-[10px] uppercase text-muted-foreground font-medium">Active Tasks</div>
             </div>
             <div className="text-center">
                <div className="text-base font-medium py-1">{node.lastHeartbeat}</div>
                <div className="text-[10px] uppercase text-muted-foreground font-medium">Last Contact</div>
             </div>
         </div>
         
         <div className="grid grid-cols-3 gap-3 text-center">
            <div className="space-y-1">
               <div className="relative mx-auto w-12 h-12 flex items-center justify-center rounded-full border-4 border-muted/30" style={{ borderColor: node.metrics.cpu > 80 ? 'var(--warning)' : undefined }}>
                  <span className="text-xs font-bold">{node.metrics.cpu}%</span>
               </div>
               <span className="text-[10px] uppercase text-muted-foreground">CPU</span>
            </div>
            <div className="space-y-1">
               <div className="relative mx-auto w-12 h-12 flex items-center justify-center rounded-full border-4 border-muted/30" style={{ borderColor: node.metrics.mem > 80 ? 'var(--warning)' : undefined }}>
                  <span className="text-xs font-bold">{node.metrics.mem}%</span>
               </div>
               <span className="text-[10px] uppercase text-muted-foreground">MEM</span>
            </div>
            <div className="space-y-1">
               <div className="relative mx-auto w-12 h-12 flex items-center justify-center rounded-full border-4 border-muted/30">
                  <span className="text-xs font-bold">{node.metrics.disk}%</span>
               </div>
               <span className="text-[10px] uppercase text-muted-foreground">DISK</span>
            </div>
         </div>
         
         <div className="mt-3 text-center">
            <Badge variant="outline" className="text-[10px] h-5 font-mono text-muted-foreground bg-muted/20">
               {node.version}
            </Badge>
         </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 12: Detailed Row (Compact Horizontal) ---
function CardCompactRow({ node }: { node: AgentNode }) {
  return (
    <div className="flex items-center gap-4 p-3 border rounded-lg bg-card hover:bg-muted/10 transition-colors">
       {/* Status Icon */}
       <div className={cn(
          "w-2 h-10 rounded-full shrink-0",
          node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground"
       )} />
       
       {/* Identity */}
       <div className="w-40 shrink-0">
          <div className="font-medium text-sm truncate" title={node.name}>{node.name}</div>
          <div className="text-[11px] text-muted-foreground truncate" title={node.hostname}>{node.hostname}</div>
       </div>
       
       {/* Network & Ver */}
       <div className="w-32 shrink-0 hidden sm:block">
          <div className="text-xs font-mono">{node.ip}</div>
          <div className="text-[10px] text-muted-foreground">{node.version}</div>
       </div>
       
       {/* Metrics Bar */}
       <div className="flex-1 grid grid-cols-3 gap-4 min-w-[200px]">
          <div className="space-y-1">
             <div className="flex justify-between text-[10px] uppercase text-muted-foreground">
                <span>CPU</span>
                <span>{node.metrics.cpu}%</span>
             </div>
             <Progress value={node.metrics.cpu} className="h-1" />
          </div>
          <div className="space-y-1">
             <div className="flex justify-between text-[10px] uppercase text-muted-foreground">
                <span>MEM</span>
                <span>{node.metrics.mem}%</span>
             </div>
             <Progress value={node.metrics.mem} className="h-1" />
          </div>
          <div className="space-y-1 hidden lg:block">
             <div className="flex justify-between text-[10px] uppercase text-muted-foreground">
                <span>DSK</span>
                <span>{node.metrics.disk}%</span>
             </div>
             <Progress value={node.metrics.disk} className="h-1" />
          </div>
       </div>
       
       {/* Stats */}
       <div className="w-24 shrink-0 flex flex-col items-end gap-0.5 text-right">
          <div className="text-xs font-medium">{node.metrics.tasks} Tasks</div>
          <div className="text-[10px] text-muted-foreground flex items-center gap-1">
             <IconClock className="w-3 h-3" />
             {node.lastHeartbeat}
          </div>
       </div>
       
       <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" aria-label="More actions">
          <IconDotsVertical className="w-4 h-4" />
       </Button>
    </div>
  )
}

// --- Curated Demo A: Glass + Meta Grid + Slim Metrics (All fields) ---
function CardCuratedGlassGrid({ node }: { node: AgentNode }) {
  const load = Math.max(node.metrics.cpu, node.metrics.mem, node.metrics.disk)

  return (
    <Card className="relative overflow-hidden hover:border-primary/40 transition-colors">
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-muted/30" />
      <CardHeader className="relative p-4 pb-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <div
                className={cn(
                  "w-9 h-9 rounded-lg flex items-center justify-center transition-colors",
                  node.status === "online" ? "bg-primary/10 text-primary" : "bg-muted text-muted-foreground"
                )}
              >
                <IconServer className="w-4 h-4" />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <CardTitle className="text-sm truncate" title={node.name}>
                    {node.name}
                  </CardTitle>
                  <Badge variant="outline" className="text-[10px] h-5 font-mono text-muted-foreground">
                    {node.version}
                  </Badge>
                </div>
                <div className="text-[10px] text-muted-foreground font-mono truncate" title={node.hostname}>
                  {node.hostname}
                </div>
              </div>
            </div>
            <div className="mt-2 text-[10px] text-muted-foreground font-mono truncate" title={node.ip}>
              {node.ip}
            </div>
          </div>

          <div className="flex flex-col items-end gap-1 shrink-0">
            <StatusBadge status={node.status} />
            <Badge variant="secondary" className="h-5 text-[10px] tabular-nums">
              {node.metrics.tasks} tasks
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="relative p-4 pt-0 space-y-3">
        <div className="grid grid-cols-2 gap-2 text-xs">
          <div className="rounded-lg border bg-muted/10 p-2">
            <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Heartbeat</div>
            <div className="mt-1 flex items-center gap-1.5">
              <IconActivity className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="font-medium">{node.lastHeartbeat}</span>
            </div>
          </div>
          <div className="rounded-lg border bg-muted/10 p-2">
            <div className="text-[10px] text-muted-foreground uppercase tracking-wider">Load</div>
            <div className="mt-1 font-mono font-semibold tabular-nums">{load}%</div>
          </div>
        </div>

        <div className="space-y-2">
          <MetricSlim label="CPU" value={node.metrics.cpu} icon={IconCpu} />
          <MetricSlim label="MEM" value={node.metrics.mem} icon={IconDatabase} />
          <MetricSlim label="DISK" value={node.metrics.disk} icon={IconDatabase} />
        </div>
      </CardContent>
    </Card>
  )
}

// --- Curated Demo B: Split Meta + Metrics Panel (All fields) ---
function CardCuratedSplitPanel({ node }: { node: AgentNode }) {
  return (
    <Card className="overflow-hidden hover:border-primary/40 transition-colors">
      <div className="grid grid-cols-1 md:grid-cols-[1fr_260px]">
        <div className="p-4">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <CardTitle className="text-base truncate" title={node.name}>
                  {node.name}
                </CardTitle>
                <StatusBadge status={node.status} />
              </div>
              <div className="mt-1 text-xs text-muted-foreground font-mono truncate" title={node.hostname}>
                {node.hostname}
              </div>
              <div className="text-xs text-muted-foreground font-mono">{node.ip}</div>

              <div className="mt-3 flex flex-wrap items-center gap-2">
                <Badge variant="outline" className="text-[10px] h-5 font-mono text-muted-foreground">
                  {node.version}
                </Badge>
                <Badge variant="secondary" className="text-[10px] h-5 tabular-nums">
                  {node.metrics.tasks} tasks
                </Badge>
                <span className="text-[10px] text-muted-foreground flex items-center gap-1.5">
                  <IconActivity className="w-3 h-3" />
                  {node.lastHeartbeat}
                </span>
              </div>
            </div>

            <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" aria-label="More actions">
              <IconDotsVertical className="w-4 h-4" />
            </Button>
          </div>
        </div>

        <div className="p-4 bg-muted/15 border-t md:border-t-0 md:border-l">
          <div className="space-y-3">
            <MetricThick label="CPU" value={node.metrics.cpu} />
            <MetricThick label="MEM" value={node.metrics.mem} />
            <MetricThick label="DISK" value={node.metrics.disk} />
          </div>
        </div>
      </div>
    </Card>
  )
}

// --- Curated Demo C: Meta Chips + Metric Tiles (All fields) ---
function CardCuratedMetaTiles({ node }: { node: AgentNode }) {
  return (
    <Card className="hover:border-primary/40 transition-colors">
      <CardHeader className="p-4 pb-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-base truncate" title={node.name}>
                {node.name}
              </CardTitle>
              <StatusBadge status={node.status} />
            </div>
            <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-muted-foreground">
              <span className="font-mono truncate" title={node.hostname}>{node.hostname}</span>
              <span className="opacity-50">•</span>
              <span className="font-mono" title={node.ip}>{node.ip}</span>
            </div>
          </div>
          <div className="flex flex-col items-end gap-1 shrink-0">
            <Badge variant="secondary" className="text-[10px] h-5 tabular-nums">
              {node.metrics.tasks} tasks
            </Badge>
            <Badge variant="outline" className="text-[10px] h-5 font-mono text-muted-foreground">
              {node.version}
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="p-4 pt-0 space-y-3">
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <IconActivity className="w-3.5 h-3.5" />
            Last heartbeat
          </span>
          <span className="font-medium text-foreground">{node.lastHeartbeat}</span>
        </div>

        <div className="grid grid-cols-3 gap-2">
          <div className="rounded-lg border bg-muted/10 p-2">
            <div className="text-[10px] uppercase text-muted-foreground">CPU</div>
            <div className="font-semibold tabular-nums">{node.metrics.cpu}%</div>
            <Progress value={node.metrics.cpu} className="h-1 mt-2" />
          </div>
          <div className="rounded-lg border bg-muted/10 p-2">
            <div className="text-[10px] uppercase text-muted-foreground">MEM</div>
            <div className="font-semibold tabular-nums">{node.metrics.mem}%</div>
            <Progress value={node.metrics.mem} className="h-1 mt-2" />
          </div>
          <div className="rounded-lg border bg-muted/10 p-2">
            <div className="text-[10px] uppercase text-muted-foreground">DSK</div>
            <div className="font-semibold tabular-nums">{node.metrics.disk}%</div>
            <Progress value={node.metrics.disk} className="h-1 mt-2" />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 13: Production Ready (Balanced) ---
function CardProduction({ node }: { node: AgentNode }) {
  return (
    <Card className="group overflow-hidden transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:shadow-md border-l-4" style={{ borderLeftColor: node.status === "online" ? "var(--success)" : node.status === "maintenance" ? "var(--warning)" : "var(--muted-foreground)" }}>
      <CardHeader className="p-4 pb-3">
        <div className="flex justify-between items-start">
          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-sm">{node.name}</h3>
              {node.status === "online" && (
                <span className="relative flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                </span>
              )}
            </div>
            <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground font-mono">
              <span className="truncate max-w-[100px]" title={node.hostname}>{node.hostname}</span>
              <span className="opacity-30">|</span>
              <span>{node.ip}</span>
            </div>
          </div>
          <Button variant="ghost" size="icon" className="h-8 w-8 -mr-2 -mt-2 text-muted-foreground hover:text-foreground" aria-label="More actions">
            <IconDotsVertical className="w-4 h-4" />
          </Button>
        </div>
      </CardHeader>
      
      <CardContent className="p-4 pt-0">
        {/* Metrics Grid */}
        <div className="grid grid-cols-3 gap-4 mb-4">
          <div className="space-y-1">
            <div className="flex justify-between text-[10px] text-muted-foreground uppercase tracking-wider">
              <span>CPU</span>
              <span className={cn("font-mono", node.metrics.cpu > 80 && "text-[var(--error)]")}>{node.metrics.cpu}%</span>
            </div>
            <Progress value={node.metrics.cpu} className={cn("h-1.5", node.metrics.cpu > 80 && "[&>div]:bg-[var(--error)]")} />
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[10px] text-muted-foreground uppercase tracking-wider">
              <span>Mem</span>
              <span className={cn("font-mono", node.metrics.mem > 80 && "text-[var(--warning)]")}>{node.metrics.mem}%</span>
            </div>
            <Progress value={node.metrics.mem} className={cn("h-1.5", node.metrics.mem > 80 && "[&>div]:bg-[var(--warning)]")} />
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[10px] text-muted-foreground uppercase tracking-wider">
              <span>Disk</span>
              <span className="font-mono">{node.metrics.disk}%</span>
            </div>
            <Progress value={node.metrics.disk} className="h-1.5" />
          </div>
        </div>

        {/* Footer Info */}
        <div className="flex items-center justify-between text-xs pt-3 border-t bg-muted/5 -mx-4 -mb-4 px-4 py-2.5">
          <div className="flex items-center gap-3 text-muted-foreground">
            <div className="flex items-center gap-1.5" title="Last Heartbeat">
              <IconActivity className="w-3.5 h-3.5" />
              <span>{node.lastHeartbeat}</span>
            </div>
            <div className="flex items-center gap-1.5" title="Agent Version">
              <span className="w-1 h-1 rounded-full bg-muted-foreground/50" />
              <span>{node.version}</span>
            </div>
          </div>
          <Badge variant="secondary" className="h-5 text-[10px] font-medium bg-background border shadow-sm">
            {node.metrics.tasks} Active Tasks
          </Badge>
        </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 14: Production List (Horizontal) ---
function CardProductionList({ node }: { node: AgentNode }) {
  return (
    <div className="group flex items-center gap-4 p-3 rounded-lg border bg-card hover:bg-muted/30 transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:shadow-sm">
      {/* Status Bar */}
      <div className={cn(
        "w-1 h-8 rounded-full",
        node.status === "online" ? "bg-[var(--success)]" : node.status === "maintenance" ? "bg-[var(--warning)]" : "bg-muted-foreground/30"
      )} />

      {/* Main Info */}
      <div className="w-[180px] shrink-0">
        <div className="flex items-center gap-2">
          <span className="font-semibold text-sm truncate" title={node.name}>{node.name}</span>
          <Badge variant="outline" className="text-[9px] h-4 px-1 py-0">{node.version}</Badge>
        </div>
        <div className="flex items-center gap-1.5 mt-0.5 text-xs text-muted-foreground font-mono">
          <span className="truncate" title={node.hostname}>{node.hostname}</span>
          <span className="opacity-30">/</span>
          <span>{node.ip}</span>
        </div>
      </div>

      {/* Metrics - Expanded */}
      <div className="flex-1 grid grid-cols-3 gap-6 px-4">
        <div className="space-y-1">
          <div className="flex justify-between text-[10px] text-muted-foreground">
            <span>CPU</span>
            <span>{node.metrics.cpu}%</span>
          </div>
          <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div className={cn("h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow]", getMetricColor(node.metrics.cpu))} style={{ width: `${node.metrics.cpu}%` }} />
          </div>
        </div>
        <div className="space-y-1">
          <div className="flex justify-between text-[10px] text-muted-foreground">
            <span>MEM</span>
            <span>{node.metrics.mem}%</span>
          </div>
          <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div className={cn("h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow]", getMetricColor(node.metrics.mem))} style={{ width: `${node.metrics.mem}%` }} />
          </div>
        </div>
        <div className="space-y-1 hidden xl:block">
          <div className="flex justify-between text-[10px] text-muted-foreground">
            <span>DSK</span>
            <span>{node.metrics.disk}%</span>
          </div>
          <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div className={cn("h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow]", getMetricColor(node.metrics.disk))} style={{ width: `${node.metrics.disk}%` }} />
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="flex items-center gap-6 text-xs text-muted-foreground shrink-0 border-l pl-6">
        <div className="flex flex-col items-end gap-0.5">
          <span className="font-medium text-foreground tabular-nums">{node.metrics.tasks}</span>
          <span className="text-[10px] uppercase">Tasks</span>
        </div>
        <div className="flex flex-col items-end gap-0.5 min-w-[60px]">
          <span className="font-medium text-foreground">{node.lastHeartbeat}</span>
          <span className="text-[10px] uppercase">Seen</span>
        </div>
      </div>

      {/* Actions */}
      <Button variant="ghost" size="icon" className="h-8 w-8 ml-2" aria-label="More actions">
        <IconDotsVertical className="w-4 h-4" />
      </Button>
    </div>
  )
}

// --- Helper: Metric with Limit ---
function MetricWithLimit({ label, value, limit }: { label: string, value: number, limit: number }) {
  const isWarning = value >= limit * 0.8
  const isCritical = value >= limit
  const color = isCritical ? "bg-[var(--error)]" : isWarning ? "bg-[var(--warning)]" : "bg-primary"
  
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-[10px] leading-tight">
        <span className="text-muted-foreground uppercase">{label}</span>
        <span className="font-mono">
          <span className={cn(isWarning && "text-[var(--warning)]", isCritical && "text-[var(--error)]")}>{value}%</span>
          <span className="text-muted-foreground/50 mx-0.5">/</span>
          <span className="text-muted-foreground">{limit}%</span>
        </span>
      </div>
      <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden relative">
         {/* Limit Marker */}
         <div className="absolute top-0 bottom-0 w-px bg-foreground/20 z-10" style={{ left: `${limit}%` }} />
         {/* Bar */}
         <div className={cn("h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow]", color)} style={{ width: `${Math.min(100, value)}%` }} />
      </div>
    </div>
  )
}

// --- Card Variant 15: High Density Grid (Explicit Limits) ---
function CardDensityGrid({ node }: { node: AgentNode }) {
  return (
    <Card className="hover:border-primary/50 transition-colors">
      <CardHeader className="p-3 pb-2 border-b bg-muted/10">
        <div className="flex items-start justify-between">
           <div className="flex items-center gap-2">
              <StatusBadge status={node.status} />
              <div>
                 <div className="font-semibold text-sm leading-none">{node.name}</div>
                 <div className="text-[10px] text-muted-foreground font-mono mt-0.5">{node.ip}</div>
              </div>
           </div>
           <Button variant="ghost" size="icon" className="h-6 w-6 -mr-1 -mt-1" aria-label="Open settings"><IconSettings className="w-3.5 h-3.5" /></Button>
        </div>
      </CardHeader>
      
      <CardContent className="p-3 space-y-4">
         {/* Meta Grid */}
         <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-[11px]">
            <div className="flex justify-between border-b border-dashed border-muted pb-1">
               <span className="text-muted-foreground">Host</span>
               <span className="font-mono truncate max-w-[80px]" title={node.hostname}>{node.hostname}</span>
            </div>
            <div className="flex justify-between border-b border-dashed border-muted pb-1">
               <span className="text-muted-foreground">Ver</span>
               <span className="font-mono">{node.version}</span>
            </div>
            <div className="flex justify-between pt-1">
               <span className="text-muted-foreground">Last Seen</span>
               <span>{node.lastHeartbeat}</span>
            </div>
            <div className="flex justify-between pt-1">
               <span className="text-muted-foreground">Uptime</span>
               <span>{node.metrics.uptime}</span>
            </div>
         </div>

         {/* Metrics with Limits */}
         <div className="space-y-2.5 bg-muted/5 p-2 rounded border border-dashed">
            <MetricWithLimit label="CPU Usage" value={node.metrics.cpu} limit={node.config.cpuThreshold} />
            <MetricWithLimit label="Memory" value={node.metrics.mem} limit={node.config.memThreshold} />
            <MetricWithLimit label="Disk Space" value={node.metrics.disk} limit={node.config.diskThreshold} />
         </div>

         {/* Tasks Footer */}
         <div className="flex items-center justify-between text-xs pt-1">
            <div className="flex items-center gap-2">
               <Badge variant="outline" className="h-5 px-1.5 text-[10px] font-normal">
                  Tasks: <strong className="ml-1">{node.metrics.tasks}</strong> / {node.config.maxTasks}
               </Badge>
            </div>
            <IconDotsVertical className="w-4 h-4 text-muted-foreground cursor-pointer" />
         </div>
      </CardContent>
    </Card>
  )
}

// --- Card Variant 16: Horizontal Density Row ---
function CardDensityRow({ node }: { node: AgentNode }) {
  return (
    <div className="flex items-center gap-4 p-2 pl-3 border rounded-md bg-card hover:bg-muted/10 transition-colors text-xs">
       {/* Status & Name */}
       <div className="w-48 shrink-0">
          <div className="flex items-center gap-2 mb-1">
             <div className={cn("w-2 h-2 rounded-full", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
             <span className="font-semibold text-sm truncate">{node.name}</span>
          </div>
          <div className="flex gap-2 text-[10px] text-muted-foreground font-mono">
             <span className="truncate max-w-[80px]" title={node.hostname}>{node.hostname}</span>
             <span>{node.ip}</span>
          </div>
       </div>
       
       {/* Configs (Static) */}
       <div className="w-24 shrink-0 hidden md:block text-[10px] text-muted-foreground space-y-0.5 border-l pl-4">
          <div>Ver: {node.version}</div>
          <div>Up: {node.metrics.uptime}</div>
       </div>

       {/* Metrics Bars */}
       <div className="flex-1 grid grid-cols-3 gap-4 border-l pl-4">
          <MetricWithLimit label="CPU" value={node.metrics.cpu} limit={node.config.cpuThreshold} />
          <MetricWithLimit label="MEM" value={node.metrics.mem} limit={node.config.memThreshold} />
          <div className="hidden lg:block">
             <MetricWithLimit label="DSK" value={node.metrics.disk} limit={node.config.diskThreshold} />
          </div>
       </div>

       {/* Tasks & Actions */}
       <div className="w-32 shrink-0 flex items-center justify-end gap-3 border-l pl-4">
          <div className="text-right">
             <div className="font-medium">{node.metrics.tasks} / {node.config.maxTasks}</div>
             <div className="text-[9px] text-muted-foreground uppercase">Tasks</div>
          </div>
          <Button variant="ghost" size="icon" className="h-7 w-7" aria-label="Open settings"><IconSettings className="w-3.5 h-3.5" /></Button>
       </div>
    </div>
  )
}

// --- Card Variant 17: Tabbed Tech Card ---
function CardTabbedTech({ node }: { node: AgentNode }) {
  return (
    <Card className="hover:shadow-md transition-shadow group">
       <div className="flex items-center justify-between p-3 border-b bg-muted/30">
          <div className="flex items-center gap-2">
             <div className={cn("w-1.5 h-8 rounded-full", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
             <div>
                <div className="font-bold text-sm leading-none">{node.name}</div>
                <div className="text-[10px] text-muted-foreground font-mono mt-1">{node.ip}</div>
             </div>
          </div>
          <div className="text-right">
             <Badge variant="secondary" className="h-5 text-[10px]">{node.version}</Badge>
             <div className="text-[10px] text-muted-foreground mt-0.5">{node.lastHeartbeat}</div>
          </div>
       </div>
       
       <div className="p-3 grid grid-cols-2 gap-4 text-xs">
          <div className="space-y-3">
             <MetricWithLimit label="CPU" value={node.metrics.cpu} limit={node.config.cpuThreshold} />
             <MetricWithLimit label="RAM" value={node.metrics.mem} limit={node.config.memThreshold} />
          </div>
          <div className="space-y-3">
             <MetricWithLimit label="DISK" value={node.metrics.disk} limit={node.config.diskThreshold} />
             <div className="space-y-1">
                <div className="flex justify-between text-[10px] uppercase text-muted-foreground">
                   <span>Tasks</span>
                   <span>{node.metrics.tasks} / {node.config.maxTasks}</span>
                </div>
                <Progress value={(node.metrics.tasks / node.config.maxTasks) * 100} className="h-1.5" />
             </div>
          </div>
       </div>
       
       <div className="px-3 py-2 border-t bg-muted/5 flex items-center justify-between text-[10px]">
          <span className="font-mono text-muted-foreground">{node.hostname}</span>
          <div className="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
             <button className="hover:underline">Logs</button>
             <button className="hover:underline">Config</button>
             <button className="hover:underline text-[var(--error)]">Restart</button>
          </div>
       </div>
    </Card>
  )
}

// --- Variant 15A: Density Grid (Clean Corporate) ---
function CardDensityCorporate({ node }: { node: AgentNode }) {
  return (
    <Card className="shadow-sm border-2 border-muted/40 hover:border-primary/20 transition-[color,background-color,border-color,opacity,transform,box-shadow]">
      <CardHeader className="p-3 pb-2 bg-muted/20 border-b border-muted/20">
        <div className="flex items-start justify-between">
           <div className="flex items-center gap-3">
              <div className={cn(
                "w-8 h-8 rounded-lg flex items-center justify-center border shadow-sm",
                node.status === "online" ? "bg-background border-emerald-200 text-emerald-600" : "bg-background border-muted text-muted-foreground"
              )}>
                 <IconServer className="w-4 h-4" />
              </div>
              <div>
                 <div className="font-bold text-sm text-foreground/90">{node.name}</div>
                 <div className="flex items-center gap-2 mt-0.5">
                    <Badge variant="secondary" className="text-[9px] h-4 px-1 rounded-sm font-mono text-muted-foreground bg-background border shadow-none hover:bg-background">
                       {node.ip}
                    </Badge>
                    <span className={cn("w-1.5 h-1.5 rounded-full", node.status === "online" ? "bg-emerald-500" : "bg-zinc-300")} />
                 </div>
              </div>
           </div>
           <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground/60" aria-label="Open settings"><IconSettings className="w-3.5 h-3.5" /></Button>
        </div>
      </CardHeader>
      
      <CardContent className="p-3 pt-4 space-y-4">
         <div className="grid grid-cols-2 gap-2 text-[11px]">
             <div className="bg-muted/30 p-2 rounded border border-muted/20 space-y-0.5">
                <span className="text-[9px] uppercase text-muted-foreground/70 font-bold tracking-wider">Host</span>
                <div className="font-medium truncate" title={node.hostname}>{node.hostname}</div>
             </div>
             <div className="bg-muted/30 p-2 rounded border border-muted/20 space-y-0.5">
                <span className="text-[9px] uppercase text-muted-foreground/70 font-bold tracking-wider">Uptime</span>
                <div className="font-medium truncate">{node.metrics.uptime}</div>
             </div>
         </div>

         <div className="space-y-3 px-1">
            <MetricWithLimit label="CPU Usage" value={node.metrics.cpu} limit={node.config.cpuThreshold} />
            <MetricWithLimit label="Memory" value={node.metrics.mem} limit={node.config.memThreshold} />
            <MetricWithLimit label="Disk Space" value={node.metrics.disk} limit={node.config.diskThreshold} />
         </div>

         <div className="flex items-center justify-between text-xs pt-2 border-t border-dashed">
            <span className="text-muted-foreground text-[10px]">
               Last seen: <span className="text-foreground font-medium">{node.lastHeartbeat}</span>
            </span>
            <div className="flex items-center gap-1.5">
               <span className="text-[10px] uppercase text-muted-foreground font-bold">Tasks</span>
               <span className="font-mono bg-primary/10 text-primary px-1.5 py-0.5 rounded text-[10px] font-bold">
                  {node.metrics.tasks}/{node.config.maxTasks}
               </span>
            </div>
         </div>
      </CardContent>
    </Card>
  )
}

// --- Variant 15B: Density Grid (Dark Console) ---
function CardDensityDark({ node }: { node: AgentNode }) {
  // Use explicit dark styles regardless of theme
  return (
    <div className="rounded border border-zinc-800 bg-zinc-950 text-zinc-400 font-mono text-xs overflow-hidden shadow-xl">
      <div className="p-3 border-b border-zinc-800 bg-zinc-900/50 flex justify-between items-center">
         <div className="flex gap-2 items-center">
            <span className={cn("text-lg leading-none", node.status === "online" ? "text-emerald-500" : "text-red-500")}>●</span>
            <span className="text-zinc-100 font-bold">{node.name}</span>
         </div>
         <div className="px-2 py-0.5 bg-zinc-900 border border-zinc-800 rounded text-[10px] text-zinc-500">
            {node.ip}
         </div>
      </div>
      
      <div className="p-4 space-y-4">
         <div className="grid grid-cols-2 gap-4 text-[10px] uppercase tracking-wide">
            <div>
               <span className="text-zinc-600 block">Hostname</span>
               <span className="text-zinc-300">{node.hostname}</span>
            </div>
            <div>
               <span className="text-zinc-600 block">Version</span>
               <span className="text-zinc-300">{node.version}</span>
            </div>
         </div>
         
         <div className="space-y-3 border-t border-zinc-900 pt-3">
             {[
               { l: "CPU", v: node.metrics.cpu, lim: node.config.cpuThreshold },
               { l: "MEM", v: node.metrics.mem, lim: node.config.memThreshold },
               { l: "DSK", v: node.metrics.disk, lim: node.config.diskThreshold }
             ].map((m, i) => (
                <div key={i} className="space-y-1">
                   <div className="flex justify-between text-[10px]">
                      <span>{m.l}</span>
                      <span><span className={m.v >= m.lim ? "text-red-500" : "text-zinc-300"}>{m.v}%</span> / {m.lim}%</span>
                   </div>
                   <div className="h-1 bg-zinc-900 w-full">
                      <div className={cn("h-full", m.v >= m.lim ? "bg-red-500" : "bg-emerald-500")} style={{ width: `${m.v}%` }} />
                   </div>
                </div>
             ))}
         </div>
         
         <div className="flex justify-between items-end pt-2 border-t border-zinc-900 text-[10px]">
            <div>
               <div className="text-zinc-600">UPTIME</div>
               <div className="text-zinc-300">{node.metrics.uptime}</div>
            </div>
            <div className="text-right">
               <div className="text-zinc-600">TASKS</div>
               <div className="text-zinc-100 font-bold text-sm">{node.metrics.tasks}<span className="text-zinc-600 text-[10px] font-normal">/{node.config.maxTasks}</span></div>
            </div>
         </div>
      </div>
    </div>
  )
}

// --- Variant 15B-2: Density Grid (Adaptive Monospace) ---
function CardDensityMonospace({ node }: { node: AgentNode }) {
  // Same layout as Dark Console, but uses theme variables for light/dark adaptability
  return (
    <div className="rounded border border-border bg-card text-muted-foreground font-mono text-xs overflow-hidden shadow-sm hover:shadow-md transition-shadow group">
      <div className="p-2.5 border-b border-border bg-muted/30 flex justify-between items-center">
         <div className="flex gap-2 items-center">
            <span className={cn("text-lg leading-none", node.status === "online" ? "text-[var(--success)]" : "text-[var(--error)]")}>●</span>
            <span className="text-foreground font-bold">{node.name}</span>
         </div>
         <div className="flex items-center gap-2">
            <div className="px-2 py-0.5 bg-background border border-border rounded text-[10px] text-muted-foreground">
               {node.ip}
            </div>
            <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-foreground" aria-label="More actions">
               <IconDotsVertical className="w-3.5 h-3.5" />
            </Button>
         </div>
      </div>
      
      <div className="p-3 space-y-4">
         <div className="grid grid-cols-2 gap-4 text-[10px] uppercase tracking-wide">
            <div>
               <span className="text-muted-foreground/60 block">Hostname</span>
               <span className="text-foreground truncate" title={node.hostname}>{node.hostname}</span>
            </div>
            <div className="text-right">
               <span className="text-muted-foreground/60 block">Version</span>
               <span className="text-foreground">{node.version}</span>
            </div>
         </div>
         
         <div className="space-y-3 border-t border-border pt-3">
             {[
               { l: "CPU", v: node.metrics.cpu, lim: node.config.cpuThreshold },
               { l: "MEM", v: node.metrics.mem, lim: node.config.memThreshold },
               { l: "DSK", v: node.metrics.disk, lim: node.config.diskThreshold }
             ].map((m, i) => (
                <div key={i} className="space-y-1">
                   <div className="flex justify-between text-[10px]">
                      <span>{m.l}</span>
                      <span><span className={m.v >= m.lim ? "text-[var(--error)]" : "text-foreground"}>{m.v}%</span> / {m.lim}%</span>
                   </div>
                   <div className="h-1 bg-muted w-full">
                      <div className={cn("h-full", m.v >= m.lim ? "bg-[var(--error)]" : "bg-[var(--success)]")} style={{ width: `${m.v}%` }} />
                   </div>
                </div>
             ))}
         </div>
         
         <div className="flex justify-between items-end pt-2 border-t border-border text-[10px]">
            <div className="flex gap-4">
               <div>
                  <div className="text-muted-foreground/60">UPTIME</div>
                  <div className="text-foreground">{node.metrics.uptime}</div>
               </div>
               <div>
                  <div className="text-muted-foreground/60">LAST SEEN</div>
                  <div className="text-foreground flex items-center gap-1.5">
                     <span className={cn("w-1.5 h-1.5 rounded-full animate-pulse", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
                     {node.lastHeartbeat}
                  </div>
               </div>
            </div>
            <div className="text-right">
               <div className="text-muted-foreground/60">TASKS</div>
               <div className="text-foreground font-bold text-sm">{node.metrics.tasks}<span className="text-muted-foreground text-[10px] font-normal">/{node.config.maxTasks}</span></div>
            </div>
         </div>
      </div>
    </div>
  )
}

// --- Variant 15B-3: Density Grid (Segmented Progress) ---
function CardDensityMonoSegmented({ node }: { node: AgentNode }) {
  return (
    <div className="rounded border border-border bg-card text-muted-foreground font-mono text-xs overflow-hidden shadow-sm hover:shadow-md transition-shadow group">
      <div className="p-2.5 border-b border-border bg-muted/30 flex justify-between items-center">
         <div className="flex gap-2 items-center">
            <span className={cn("text-lg leading-none", node.status === "online" ? "text-[var(--success)]" : "text-[var(--error)]")}>●</span>
            <span className="text-foreground font-bold">{node.name}</span>
         </div>
         <div className="flex items-center gap-2">
            <div className="px-2 py-0.5 bg-background border border-border rounded text-[10px] text-muted-foreground">{node.ip}</div>
            <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-foreground" aria-label="More actions"><IconDotsVertical className="w-3.5 h-3.5" /></Button>
         </div>
      </div>
      
      <div className="p-3 space-y-4">
         <div className="grid grid-cols-2 gap-4 text-[10px] uppercase tracking-wide">
            <div><span className="text-muted-foreground/60 block">Hostname</span><span className="text-foreground truncate" title={node.hostname}>{node.hostname}</span></div>
            <div className="text-right"><span className="text-muted-foreground/60 block">Version</span><span className="text-foreground">{node.version}</span></div>
         </div>
         
         <div className="space-y-3 border-t border-border pt-3">
             {[
               { l: "CPU", v: node.metrics.cpu, lim: node.config.cpuThreshold },
               { l: "MEM", v: node.metrics.mem, lim: node.config.memThreshold },
               { l: "DSK", v: node.metrics.disk, lim: node.config.diskThreshold }
             ].map((m, i) => (
                <div key={i} className="space-y-1">
                   <div className="flex justify-between text-[10px]">
                      <span>{m.l}</span>
                      <span><span className={m.v >= m.lim ? "text-[var(--error)]" : "text-foreground"}>{m.v}%</span> / {m.lim}%</span>
                   </div>
                   {/* Segmented Bar */}
                   <div className="flex gap-[2px] h-1.5 w-full">
                      {Array.from({ length: 25 }).map((_, idx) => {
                         const isActive = idx < (m.v / 4)
                         const isLimit = idx === Math.floor(m.lim / 4)
                         return (
                           <div 
                             key={idx} 
                             className={cn(
                               "flex-1 rounded-[1px]", 
                               isActive 
                                 ? (m.v >= m.lim ? "bg-[var(--error)]" : "bg-[var(--success)]") 
                                 : isLimit ? "bg-foreground/30" : "bg-muted"
                             )} 
                           />
                         )
                      })}
                   </div>
                </div>
             ))}
         </div>
         
         <div className="flex justify-between items-end pt-2 border-t border-border text-[10px]">
            <div className="flex gap-4">
               <div><div className="text-muted-foreground/60">UPTIME</div><div className="text-foreground">{node.metrics.uptime}</div></div>
               <div>
                  <div className="text-muted-foreground/60">LAST SEEN</div>
                  <div className="text-foreground flex items-center gap-1.5">
                     <span className={cn("w-1.5 h-1.5 rounded-full animate-pulse", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
                     {node.lastHeartbeat}
                  </div>
               </div>
            </div>
            <div className="text-right"><div className="text-muted-foreground/60">TASKS</div><div className="text-foreground font-bold text-sm">{node.metrics.tasks}<span className="text-muted-foreground text-[10px] font-normal">/{node.config.maxTasks}</span></div></div>
         </div>
      </div>
    </div>
  )
}

// --- Variant 15B-4: Density Grid (Striped Progress) ---
function CardDensityMonoStriped({ node }: { node: AgentNode }) {
  return (
    <div className="rounded border border-border bg-card text-muted-foreground font-mono text-xs overflow-hidden shadow-sm hover:shadow-md transition-shadow group">
      <div className="p-2.5 border-b border-border bg-muted/30 flex justify-between items-center">
         <div className="flex gap-2 items-center">
            <span className={cn("text-lg leading-none", node.status === "online" ? "text-[var(--success)]" : "text-[var(--error)]")}>●</span>
            <span className="text-foreground font-bold">{node.name}</span>
         </div>
         <div className="flex items-center gap-2">
            <div className="px-2 py-0.5 bg-background border border-border rounded text-[10px] text-muted-foreground">{node.ip}</div>
            <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-foreground" aria-label="More actions"><IconDotsVertical className="w-3.5 h-3.5" /></Button>
         </div>
      </div>
      
      <div className="p-3 space-y-4">
         <div className="grid grid-cols-2 gap-4 text-[10px] uppercase tracking-wide">
            <div><span className="text-muted-foreground/60 block">Hostname</span><span className="text-foreground truncate" title={node.hostname}>{node.hostname}</span></div>
            <div className="text-right"><span className="text-muted-foreground/60 block">Version</span><span className="text-foreground">{node.version}</span></div>
         </div>
         
         <div className="space-y-3 border-t border-border pt-3">
             {[
               { l: "CPU", v: node.metrics.cpu, lim: node.config.cpuThreshold },
               { l: "MEM", v: node.metrics.mem, lim: node.config.memThreshold },
               { l: "DSK", v: node.metrics.disk, lim: node.config.diskThreshold }
             ].map((m, i) => (
                <div key={i} className="space-y-1">
                   <div className="flex justify-between text-[10px]">
                      <span>{m.l}</span>
                      <span><span className={m.v >= m.lim ? "text-[var(--error)]" : "text-foreground"}>{m.v}%</span> / {m.lim}%</span>
                   </div>
                   {/* Striped Bar */}
                   <div className="h-2 w-full bg-muted rounded-sm overflow-hidden relative">
                      <div className="absolute top-0 bottom-0 w-[1px] bg-foreground/50 z-10" style={{ left: `${m.lim}%` }} />
                      <div 
                        className={cn("h-full transition-[color,background-color,border-color,opacity,transform,box-shadow] bg-[length:8px_8px] bg-[linear-gradient(45deg,rgba(255,255,255,0.2)_25%,transparent_25%,transparent_50%,rgba(255,255,255,0.2)_50%,rgba(255,255,255,0.2)_75%,transparent_75%,transparent)]", 
                           m.v >= m.lim ? "bg-[var(--error)]" : "bg-[var(--success)]"
                        )} 
                        style={{ width: `${m.v}%` }} 
                      />
                   </div>
                </div>
             ))}
         </div>
         
         <div className="flex justify-between items-end pt-2 border-t border-border text-[10px]">
            <div className="flex gap-4">
               <div><div className="text-muted-foreground/60">UPTIME</div><div className="text-foreground">{node.metrics.uptime}</div></div>
               <div>
                  <div className="text-muted-foreground/60">LAST SEEN</div>
                  <div className="text-foreground flex items-center gap-1.5">
                     <span className={cn("w-1.5 h-1.5 rounded-full animate-pulse", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
                     {node.lastHeartbeat}
                  </div>
               </div>
            </div>
            <div className="text-right"><div className="text-muted-foreground/60">TASKS</div><div className="text-foreground font-bold text-sm">{node.metrics.tasks}<span className="text-muted-foreground text-[10px] font-normal">/{node.config.maxTasks}</span></div></div>
         </div>
      </div>
    </div>
  )
}

// --- Variant 15B-5: Density Grid (Retro Gradient) ---
function CardDensityMonoGradient({ node }: { node: AgentNode }) {
  return (
    <div className="rounded border border-border bg-card text-muted-foreground font-mono text-xs overflow-hidden shadow-sm hover:shadow-md transition-shadow group">
      <div className="p-2.5 border-b border-border bg-muted/30 flex justify-between items-center">
         <div className="flex gap-2 items-center">
            <span className={cn("text-lg leading-none", node.status === "online" ? "text-[var(--success)]" : "text-[var(--error)]")}>●</span>
            <span className="text-foreground font-bold">{node.name}</span>
         </div>
         <div className="flex items-center gap-2">
            <div className="px-2 py-0.5 bg-background border border-border rounded text-[10px] text-muted-foreground">{node.ip}</div>
            <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-foreground" aria-label="More actions"><IconDotsVertical className="w-3.5 h-3.5" /></Button>
         </div>
      </div>
      
      <div className="p-3 space-y-4">
         <div className="grid grid-cols-2 gap-4 text-[10px] uppercase tracking-wide">
            <div><span className="text-muted-foreground/60 block">Hostname</span><span className="text-foreground truncate" title={node.hostname}>{node.hostname}</span></div>
            <div className="text-right"><span className="text-muted-foreground/60 block">Version</span><span className="text-foreground">{node.version}</span></div>
         </div>
         
         <div className="space-y-3 border-t border-border pt-3">
             {[
               { l: "CPU", v: node.metrics.cpu, lim: node.config.cpuThreshold },
               { l: "MEM", v: node.metrics.mem, lim: node.config.memThreshold },
               { l: "DSK", v: node.metrics.disk, lim: node.config.diskThreshold }
             ].map((m, i) => (
                <div key={i} className="space-y-1">
                   <div className="flex justify-between text-[10px]">
                      <span>{m.l}</span>
                      <span><span className={m.v >= m.lim ? "text-[var(--error)]" : "text-foreground"}>{m.v}%</span> / {m.lim}%</span>
                   </div>
                   {/* Gradient Bar */}
                   <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden relative">
                      <div className="absolute top-0 bottom-0 w-[1px] bg-foreground/50 z-10" style={{ left: `${m.lim}%` }} />
                      <div 
                        className="h-full transition-[color,background-color,border-color,opacity,transform,box-shadow]"
                        style={{ 
                           width: `${m.v}%`,
                           background: m.v >= m.lim ? "var(--error)" : `linear-gradient(90deg, var(--success) 0%, var(--success) 60%, var(--warning) 100%)`
                        }} 
                      />
                   </div>
                </div>
             ))}
         </div>
         
         <div className="flex justify-between items-end pt-2 border-t border-border text-[10px]">
            <div className="flex gap-4">
               <div><div className="text-muted-foreground/60">UPTIME</div><div className="text-foreground">{node.metrics.uptime}</div></div>
               <div>
                  <div className="text-muted-foreground/60">LAST SEEN</div>
                  <div className="text-foreground flex items-center gap-1.5">
                     <span className={cn("w-1.5 h-1.5 rounded-full animate-pulse", node.status === "online" ? "bg-[var(--success)]" : "bg-muted-foreground")} />
                     {node.lastHeartbeat}
                  </div>
               </div>
            </div>
            <div className="text-right"><div className="text-muted-foreground/60">TASKS</div><div className="text-foreground font-bold text-sm">{node.metrics.tasks}<span className="text-muted-foreground text-[10px] font-normal">/{node.config.maxTasks}</span></div></div>
         </div>
      </div>
    </div>
  )
}

// --- Variant 15C: Density Grid (Glassmorphism) ---
function CardDensityGlass({ node }: { node: AgentNode }) {
  return (
    <div className="group relative overflow-hidden rounded-xl border bg-background/40 backdrop-blur-md transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:bg-background/60 hover:shadow-lg">
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
      
      <div className="relative p-4 space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between">
           <div>
              <div className="flex items-center gap-2">
                 <span className={cn("w-2 h-2 rounded-full shadow-[0_0_10px_currentColor]", node.status === "online" ? "text-emerald-400 bg-emerald-400" : "text-zinc-400 bg-zinc-400")} />
                 <span className="font-semibold tracking-tight">{node.name}</span>
              </div>
              <div className="text-xs text-muted-foreground ml-4">{node.ip}</div>
           </div>
           <Badge variant="outline" className="font-mono text-[10px] bg-background/50 backdrop-blur">
              {node.version}
           </Badge>
        </div>

        {/* Info Pills */}
        <div className="flex flex-wrap gap-2 text-[10px]">
           <div className="px-2 py-1 rounded-md bg-muted/50 border border-muted truncate max-w-[120px]" title={node.hostname}>
              <span className="opacity-50 mr-1">host:</span>{node.hostname}
           </div>
           <div className="px-2 py-1 rounded-md bg-muted/50 border border-muted">
              <span className="opacity-50 mr-1">up:</span>{node.metrics.uptime}
           </div>
        </div>

        {/* Metrics */}
        <div className="space-y-2 pt-1">
           <MetricWithLimit label="CPU" value={node.metrics.cpu} limit={node.config.cpuThreshold} />
           <MetricWithLimit label="MEM" value={node.metrics.mem} limit={node.config.memThreshold} />
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between pt-2 border-t border-dashed border-primary/10">
           <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
              <IconActivity className="w-3 h-3" />
              {node.lastHeartbeat}
           </div>
           <div className="text-[10px] font-medium">
              {node.metrics.tasks} / {node.config.maxTasks} tasks
           </div>
        </div>
      </div>
    </div>
  )
}

// --- Main Exported Component ---
export function AgentNodeDesigns() {
  return (
    <div className="space-y-16">

      {/* Section 1: Progress Bar Variants */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">Progress Bar Variants</h2>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Slim Line</h3>
          <div className="max-w-xs space-y-3">
            <MetricSlim label="CPU" value={45} icon={IconCpu} />
            <MetricSlim label="MEM" value={88} icon={IconDatabase} />
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Thick Bar</h3>
          <div className="max-w-xs space-y-2">
            <MetricThick label="CPU Usage" value={45} />
            <MetricThick label="Memory" value={88} />
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Segmented</h3>
          <div className="max-w-xs space-y-2">
            <MetricSegmented label="CPU" value={45} />
            <MetricSegmented label="MEM" value={88} />
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Circular</h3>
          <div className="flex gap-4">
            <MetricCircular label="CPU" value={45} icon={IconCpu} />
            <MetricCircular label="MEM" value={88} icon={IconDatabase} />
          </div>
        </div>
      </section>

      {/* Section 2: Card Variants (Basic) */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">Card Variants (Basic)</h2>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 1: Compact List Item</h3>
          <div className="flex flex-col gap-2">
            {MOCK_NODES.map(node => <CardCompact key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 2: Modern Grid Card</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardModern key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 3: Technical / Dashboard</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardTechnical key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 4: Segmented (Industrial)</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardIndustrial key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 5: Hero Status</h3>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {MOCK_NODES.map(node => <CardHeroStatus key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 6: Resource Monitor</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardResourceGraph key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 7: Terminal / Console</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardTerminal key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 8: Minimalist Glass</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardGlass key={node.id} node={node} />)}
          </div>
        </div>
      </section>

      {/* Section 3: Detailed Card Variants (All Fields) */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">Detailed Cards (All Fields)</h2>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 9: Data Grid</h3>
          <p className="text-sm text-muted-foreground">Name, hostname, IP, last heartbeat, version, tasks, CPU/MEM/DSK.</p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardDataGrid key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 10: Infrastructure Rack</h3>
          <p className="text-sm text-muted-foreground">Blade server style with status strip.</p>
          <div className="flex flex-col gap-2">
            {MOCK_NODES.map(node => <CardInfraRack key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 11: Comprehensive Analytics</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardAnalytics key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 12: Compact Row</h3>
          <div className="flex flex-col gap-2">
            {MOCK_NODES.map(node => <CardCompactRow key={node.id} node={node} />)}
          </div>
        </div>
      </section>

      {/* Section 4: Curated Designs (All Fields) */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">Curated Mix Designs</h2>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Curated A: Glass Grid</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardCuratedGlassGrid key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Curated B: Split Panel</h3>
          <div className="flex flex-col gap-4">
            {MOCK_NODES.slice(0, 2).map(node => <CardCuratedSplitPanel key={node.id} node={node} />)}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Curated C: Meta Tiles</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.map(node => <CardCuratedMetaTiles key={node.id} node={node} />)}
          </div>
        </div>
      </section>

      {/* Section 5: Production Ready Candidates */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">Production Ready Candidates</h2>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 13: Production Balanced (Grid Card)</h3>
          <p className="text-sm text-muted-foreground">High contrast header + status indicator, clear metrics, footer meta.</p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.slice(0, 3).map(node => (
              <CardProduction key={node.id} node={node} />
            ))}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 14: Production List (Horizontal)</h3>
          <p className="text-sm text-muted-foreground">Optimized for high density lists. Expanded metrics view.</p>
          <div className="flex flex-col gap-2">
            {MOCK_NODES.map(node => (
              <CardProductionList key={node.id} node={node} />
            ))}
          </div>
        </div>
      </section>

      {/* Section 6: High Density & Thresholds (Response to Feedback) */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">High Density & Explicit Limits</h2>
        <p className="text-muted-foreground text-sm">
           Designs focusing on showing ALL available fields: Name, Host, IP, Heartbeat, Version, CPU/Limit, Mem/Limit, Disk/Limit, Tasks/Max, Uptime.
        </p>

        <div className="space-y-4">
          <h3 className="text-lg font-medium">Variant 15: Density Grid</h3>
          <p className="text-sm text-muted-foreground">Explicit &quot;Value / Limit&quot; text for all metrics.</p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {MOCK_NODES.slice(0, 3).map(node => (
              <CardDensityGrid key={node.id} node={node} />
            ))}
          </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Variant 16: Density Row</h3>
           <p className="text-sm text-muted-foreground">Horizontal layout with column alignment for easy scanning.</p>
           <div className="flex flex-col gap-2">
             {MOCK_NODES.map(node => (
               <CardDensityRow key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Variant 17: Tabbed Tech Card</h3>
           <p className="text-sm text-muted-foreground">Technical dashboard style with action footer.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardTabbedTech key={node.id} node={node} />
             ))}
           </div>
        </div>
      </section>

      {/* Section 7: Variant 15 Evolutions (Stylistic Variations) */}
      <section className="space-y-8">
        <h2 className="text-xl font-semibold border-b pb-2">Variant 15 Layout Evolutions</h2>
        <p className="text-sm text-muted-foreground">Exploring different visual styles for the same &quot;Density Grid&quot; information architecture.</p>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution A: Clean Corporate</h3>
           <p className="text-sm text-muted-foreground">Light, airy, rounded corners, professional software look.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityCorporate key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution B: Dark Console</h3>
           <p className="text-sm text-muted-foreground">Hacker/DevOps aesthetic. Monospace, high contrast, dark mode enforced.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityDark key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution B2: Adaptive Console (Light/Dark Aware)</h3>
           <p className="text-sm text-muted-foreground">The same &quot;Console&quot; layout, but adapted to project theme colors (works in Light Mode).</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityMonospace key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution B3: Segmented Progress</h3>
           <p className="text-sm text-muted-foreground">Digital / Industrial feel with segmented blocks.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityMonoSegmented key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution B4: Striped Warning</h3>
           <p className="text-sm text-muted-foreground">Attention-grabbing stripes, higher contrast.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityMonoStriped key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution B5: Gradient Flow</h3>
           <p className="text-sm text-muted-foreground">Smoother, modernized look while keeping the console layout.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityMonoGradient key={node.id} node={node} />
             ))}
           </div>
        </div>

        <div className="space-y-4">
           <h3 className="text-lg font-medium">Evolution C: Modern Glass</h3>
           <p className="text-sm text-muted-foreground">Translucent layers, gradients, blur effects. Modern SaaS look.</p>
           <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             {MOCK_NODES.slice(0, 3).map(node => (
               <CardDensityGlass key={node.id} node={node} />
             ))}
           </div>
        </div>
      </section>

    </div>
  )
}
