"use client"

import React, { useState } from "react"
import {
  IconSearch,
  IconFilter,
  IconLayoutGrid,
  IconList,
  IconServer,
  IconActivity,
  IconClock,
  IconDotsVertical,
  IconSettings,
  IconCheck,
  IconPlus,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

// --- Mock Data ---
const MOCK_WORKERS = Array.from({ length: 12 }).map((_, i) => ({
  id: `worker-${i}`,
  name: `worker-prod-${String(i + 1).padStart(2, '0')}`,
  hostname: `aws-use1-node-${String(i + 1).padStart(2, '0')}`,
  ip: `10.0.1.${10 + i}`,
  status: i === 2 ? "offline" : i === 5 ? "maintenance" : "online",
  version: "v1.2.0",
  lastHeartbeat: i === 2 ? "2h ago" : "2s ago",
  metrics: {
    cpu: i === 2 ? 0 : Math.floor(Math.random() * 60) + 20,
    mem: i === 2 ? 0 : Math.floor(Math.random() * 70) + 20,
    disk: Math.floor(Math.random() * 40) + 10,
    tasks: i === 2 ? 0 : Math.floor(Math.random() * 15),
    uptime: i === 2 ? "-" : `${Math.floor(Math.random() * 20)}d 4h`,
  },
  config: {
    cpuThreshold: 90,
    memThreshold: 90,
    diskThreshold: 85,
    maxTasks: 20
  }
}))

type WorkerNode = typeof MOCK_WORKERS[0]

// --- Helper Components ---

function StatusDot({ status }: { status: string }) {
  const color = status === "online" ? "bg-[var(--success)]" : status === "offline" ? "bg-[var(--error)]" : "bg-[var(--warning)]"
  return (
    <span className={cn("flex h-2 w-2 relative")}>
      {status === "online" && <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>}
      <span className={cn("relative inline-flex rounded-full h-2 w-2", color)}></span>
    </span>
  )
}

function MetricBar({ label, value, limit }: { label: string, value: number, limit: number }) {
  const isWarning = value >= limit * 0.8
  const isCritical = value >= limit
  // Adaptive colors:
  // Normal: bg-primary (Light: dark grey/black, Dark: white) or specific brand color
  // We use semantic vars for adaptability
  
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-[10px] uppercase tracking-wider text-muted-foreground font-medium">
        <span>{label}</span>
        <span>
           <span className={cn(isCritical ? "text-[var(--error)]" : isWarning ? "text-[var(--warning)]" : "text-foreground")}>{value}%</span>
           <span className="opacity-50"> / {limit}%</span>
        </span>
      </div>
      <div className="h-1.5 w-full bg-muted/50 rounded-full overflow-hidden">
        <div 
          className={cn("h-full rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow]", 
            isCritical ? "bg-[var(--error)]" : isWarning ? "bg-[var(--warning)]" : "bg-[var(--success)]"
          )} 
          style={{ width: `${Math.min(100, value)}%` }} 
        />
      </div>
    </div>
  )
}

// --- Card Component (Adaptive Console) ---
function WorkerCard({ node }: { node: WorkerNode }) {
  return (
    <div className="group flex flex-col rounded-lg border border-border bg-card text-card-foreground shadow-sm hover:shadow-md transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-200">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-border/50 bg-muted/20">
        <div className="flex items-center gap-3">
          <StatusDot status={node.status} />
          <div>
            <div className="font-bold text-sm leading-none flex items-center gap-2">
               {node.name}
               {node.status === "maintenance" && <Badge variant="outline" className="text-[9px] h-4 py-0 px-1 border-yellow-500/50 text-yellow-600">MAINT</Badge>}
            </div>
            <div className="text-[10px] text-muted-foreground font-mono mt-1 opacity-80">{node.ip}</div>
          </div>
        </div>
        
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-foreground" aria-label="More actions">
              <IconDotsVertical className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem><IconSettings className="w-4 h-4 mr-2" /> Configure</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-[var(--error)]"><IconServer className="w-4 h-4 mr-2" /> Delete</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Body */}
      <div className="p-4 space-y-4 flex-1">
        {/* Meta Grid */}
        <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-[11px]">
           <div className="space-y-0.5">
              <span className="text-[9px] uppercase text-muted-foreground font-bold tracking-wider">Hostname</span>
              <div className="font-mono text-xs truncate" title={node.hostname}>{node.hostname}</div>
           </div>
           <div className="space-y-0.5 text-right">
              <span className="text-[9px] uppercase text-muted-foreground font-bold tracking-wider">Version</span>
              <div className="font-mono text-xs">{node.version}</div>
           </div>
        </div>

        {/* Metrics */}
        <div className="space-y-3 pt-2">
           <MetricBar label="CPU" value={node.metrics.cpu} limit={node.config.cpuThreshold} />
           <MetricBar label="MEM" value={node.metrics.mem} limit={node.config.memThreshold} />
           <MetricBar label="DSK" value={node.metrics.disk} limit={node.config.diskThreshold} />
        </div>
      </div>

      {/* Footer */}
      <div className="p-3 border-t border-border/50 bg-muted/5 flex items-center justify-between text-[10px]">
         <div className="flex items-center gap-3 text-muted-foreground">
            <div className="flex items-center gap-1.5" title="Last Heartbeat">
               <IconActivity className="w-3 h-3" />
               <span className={cn(node.status === "offline" && "text-[var(--error)]")}>{node.lastHeartbeat}</span>
            </div>
            <div className="flex items-center gap-1.5" title="Uptime">
               <IconClock className="w-3 h-3" />
               <span>{node.metrics.uptime}</span>
            </div>
         </div>
         
         <div className="flex items-center gap-1.5">
            <span className="text-[9px] uppercase font-bold text-muted-foreground">Tasks</span>
            <Badge variant="secondary" className="h-5 text-[10px] font-mono px-1.5 bg-muted border-border text-foreground">
               {node.metrics.tasks} <span className="opacity-40 mx-0.5">/</span> {node.config.maxTasks}
            </Badge>
         </div>
      </div>
    </div>
  )
}

// --- Page Layout Component ---

export function WorkersPageDesign() {
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid")
  const [search, setSearch] = useState("")

  const stats = [
    { label: "Total Nodes", value: 12, icon: IconServer, color: "text-foreground" },
    { label: "Online", value: 10, icon: IconCheck, color: "text-[var(--success)]" },
    { label: "Offline", value: 1, icon: IconActivity, color: "text-[var(--error)]" },
    { label: "Maintenance", value: 1, icon: IconSettings, color: "text-[var(--warning)]" },
  ]

  return (
    <div className="flex flex-col h-full bg-background/50">
      
      {/* 1. Header Section */}
      <div className="flex flex-col gap-6 p-6 pb-0">
         <div className="flex items-start justify-between">
            <div>
               <h1 className="text-2xl font-semibold tracking-tight">Workers & Agents</h1>
               <p className="text-muted-foreground mt-1">Manage scan execution nodes and monitor their health.</p>
            </div>
            <div className="flex gap-2">
               <Button variant="outline"><IconSettings className="w-4 h-4 mr-2" /> Global Config</Button>
               <Button><IconPlus className="w-4 h-4 mr-2" /> Deploy New Agent</Button>
            </div>
         </div>

         {/* Stats Bar */}
         <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {stats.map((stat, i) => (
               <div key={i} className="flex items-center gap-3 p-3 rounded-lg border bg-card shadow-sm">
                  <div className={cn("p-2 rounded-md bg-muted/50", stat.color)}>
                     <stat.icon className="w-5 h-5" />
                  </div>
                  <div>
                     <div className="text-2xl font-bold leading-none">{stat.value}</div>
                     <div className="text-xs text-muted-foreground mt-1 font-medium">{stat.label}</div>
                  </div>
               </div>
            ))}
         </div>
      </div>

      {/* 2. Toolbar */}
      <div className="p-6 sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b md:border-none md:pb-4">
         <div className="flex flex-col sm:flex-row gap-4 items-center justify-between">
            <div className="relative w-full sm:w-96">
               <IconSearch className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
               <Input 
                  type="search"
                  name="workerSearch"
                  autoComplete="off"
                  placeholder="Search by name, IP, or hostname…" 
                  className="pl-9 bg-background" 
                  value={search}
                  onChange={e => setSearch(e.target.value)}
               />
            </div>
            
            <div className="flex items-center gap-2 w-full sm:w-auto">
               <Select defaultValue="all">
                  <SelectTrigger className="w-[140px]">
                     <IconFilter className="w-3.5 h-3.5 mr-2 text-muted-foreground" />
                     <SelectValue placeholder="Status" />
                  </SelectTrigger>
                  <SelectContent>
                     <SelectItem value="all">All Status</SelectItem>
                     <SelectItem value="online">Online</SelectItem>
                     <SelectItem value="offline">Offline</SelectItem>
                  </SelectContent>
               </Select>
               
               <div className="flex items-center border rounded-md bg-background p-0.5 ml-auto sm:ml-0">
                  <Button 
                     variant={viewMode === "grid" ? "secondary" : "ghost"} 
                     size="sm" 
                     className="h-8 w-8 p-0" 
                     onClick={() => setViewMode("grid")}
                  >
                     <IconLayoutGrid className="w-4 h-4" />
                  </Button>
                  <Button 
                     variant={viewMode === "list" ? "secondary" : "ghost"} 
                     size="sm" 
                     className="h-8 w-8 p-0" 
                     onClick={() => setViewMode("list")}
                  >
                     <IconList className="w-4 h-4" />
                  </Button>
               </div>
            </div>
         </div>
      </div>

      {/* 3. Content Grid */}
      <div className="flex-1 p-6 pt-0 overflow-auto">
         {viewMode === "grid" ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-10">
               {MOCK_WORKERS.map(node => (
                  <WorkerCard key={node.id} node={node} />
               ))}
            </div>
         ) : (
            <div className="rounded-md border bg-card">
               <div className="p-8 text-center text-muted-foreground">
                  List view implementation placeholder...
               </div>
            </div>
         )}
      </div>
    </div>
  )
}
