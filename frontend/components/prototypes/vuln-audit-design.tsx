"use client"

import React, { useState, useEffect } from "react"
import { motion } from "framer-motion"
import {
  IconCheck,
  IconX,
  IconSearch,
  IconFilter,
  IconCode,
  IconActivity,
  IconDotsVertical,
  IconEye,
  IconWorld,
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

// --- Mock Data ---
const SEVERITIES = {
  critical: { color: "bg-red-500", text: "text-red-500", border: "border-red-500", bg: "bg-red-500/10" },
  high: { color: "bg-orange-500", text: "text-orange-500", border: "border-orange-500", bg: "bg-orange-500/10" },
  medium: { color: "bg-yellow-500", text: "text-yellow-500", border: "border-yellow-500", bg: "bg-yellow-500/10" },
  low: { color: "bg-green-500", text: "text-green-500", border: "border-green-500", bg: "bg-green-500/10" },
  info: { color: "bg-blue-500", text: "text-blue-500", border: "border-blue-500", bg: "bg-blue-500/10" },
}

const MOCK_VULNS = Array.from({ length: 20 }).map((_, i) => {
  const severities = Object.keys(SEVERITIES)
  const severity = severities[i % severities.length] as keyof typeof SEVERITIES
  const source = i % 3 === 0 ? "nuclei" : i % 3 === 1 ? "passive" : "custom"
  return {
    id: `vuln-${i + 1}`,
    title: i % 3 === 0 ? "SQL Injection in /api/login" : i % 3 === 1 ? "Cross-Site Scripting (Reflected)" : "Sensitive Data Exposure",
    severity,
    source,
    target: `https://api.target-system.com/v1/endpoints/${i}`,
    status: i < 5 ? "reviewed" : "pending",
    reviewResult: i < 2 ? "confirmed" : i < 5 ? "false_positive" : null,
    time: "2 mins ago",
    method: "POST",
    path: "/api/auth/v1/login"
  }
})

// --- Components ---

function SeverityBadge({ severity, className }: { severity: string, className?: string }) {
  const style = SEVERITIES[severity as keyof typeof SEVERITIES]
  return (
    <Badge variant="outline" className={cn("uppercase text-[10px] h-5 px-1.5 border-0 font-bold", style.bg, style.text, className)}>
      {severity.slice(0, 3)}
    </Badge>
  )
}

function SourceBadge({ source }: { source: string }) {
  const icon = source === "nuclei" ? <IconActivity className="size-3" /> 
             : source === "passive" ? <IconEye className="size-3" />
             : <IconCode className="size-3" />
  
  return (
    <div className="flex items-center gap-1 text-[10px] font-mono opacity-70 bg-muted/50 px-1.5 py-0.5 rounded border border-border/50 uppercase">
       {icon}
       <span>{source}</span>
    </div>
  )
}

function AuditProgressBar({ total, reviewed }: { total: number, reviewed: number }) {
  const progress = (reviewed / total) * 100
  return (
    <div className="flex items-center gap-4 text-xs font-mono">
      <div className="flex flex-col gap-1 min-w-[120px]">
        <div className="flex justify-between opacity-70">
          <span>AUDIT PROGRESS</span>
          <span>{Math.round(progress)}%</span>
        </div>
        <div className="h-1.5 w-full bg-muted overflow-hidden rounded-full">
          <motion.div 
            initial={{ width: 0 }}
            animate={{ width: `${progress}%` }}
            className="h-full bg-primary"
          />
        </div>
      </div>
      <div className="flex gap-3 border-l border-border pl-4">
        <div>
          <span className="text-muted-foreground mr-2">PENDING</span>
          <span className="font-bold">{total - reviewed}</span>
        </div>
        <div>
          <span className="text-muted-foreground mr-2">REVIEWED</span>
          <span className="font-bold text-primary">{reviewed}</span>
        </div>
      </div>
    </div>
  )
}

export function VulnAuditDesign() {
  const [selectedId, setSelectedId] = useState<string>(MOCK_VULNS[5].id) // Start with a pending one
  const [items, setItems] = useState(MOCK_VULNS)
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  
  const selectedItem = items.find(i => i.id === selectedId) || items[0]
  const reviewedCount = items.filter(i => i.status === "reviewed").length

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        const idx = items.findIndex(i => i.id === selectedId)
        if (idx < items.length - 1) setSelectedId(items[idx + 1].id)
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        const idx = items.findIndex(i => i.id === selectedId)
        if (idx > 0) setSelectedId(items[idx - 1].id)
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [selectedId, items])

  const handleAction = (id: string, result: "confirmed" | "false_positive") => {
    setItems(prev => prev.map(item => {
      if (item.id === id) {
        return { ...item, status: "reviewed", reviewResult: result }
      }
      return item
    }))
    
    // Auto advance to next pending
    const currentIndex = items.findIndex(i => i.id === id)
    const nextPending = items.slice(currentIndex + 1).find(i => i.status === "pending")
    if (nextPending) {
      setTimeout(() => setSelectedId(nextPending.id), 200) // Small delay for animation
    }
  }

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)] bg-background border rounded-lg overflow-hidden shadow-sm">
      {/* 1. Audit Header */}
      <div className="h-14 border-b border-border bg-card flex items-center justify-between px-4 shrink-0 z-10">
        <div className="flex items-center gap-3">
           <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)} aria-label="Toggle sidebar">
              <IconDotsVertical className="size-4" />
           </Button>
           <div className="flex flex-col">
              <span className="text-sm font-bold flex items-center gap-2">
                 Session #2024-X9 
                 <Badge variant="outline" className="text-[10px] py-0 h-4">MANUAL AUDIT</Badge>
              </span>
              <span className="text-[10px] text-muted-foreground font-mono">Target: production-api-cluster-01</span>
           </div>
        </div>

        <AuditProgressBar total={items.length} reviewed={reviewedCount} />

        <div className="flex items-center gap-2">
           <Button variant="outline" size="sm" className="h-8 text-xs font-mono">
              <IconFilter className="size-3 mr-2" />
              FILTER
           </Button>
           <Button size="sm" className="h-8 text-xs font-mono">
              FINISH REVIEW
           </Button>
        </div>
      </div>

      {/* 2. Main Workspace */}
      <div className="flex-1 flex overflow-hidden">
        
        {/* Left: Sidebar List */}
        <motion.div 
           initial={false}
           animate={{ width: isSidebarCollapsed ? 0 : 320, opacity: isSidebarCollapsed ? 0 : 1 }}
           className="border-r border-border bg-muted/10 flex flex-col shrink-0"
        >
           {/* Search */}
           <div className="p-3 border-b border-border">
              <div className="relative">
                 <IconSearch className="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" />
                 <Input type="search" name="vulnerabilitySearch" autoComplete="off" placeholder="Search vulnerabilities…" className="h-9 pl-8 text-xs bg-background" />
              </div>
           </div>

           {/* List */}
           <ScrollArea className="flex-1">
              <div className="flex flex-col p-2 gap-1">
                 {items.map((item) => (
                    <button
                       key={item.id}
                       onClick={() => setSelectedId(item.id)}
                       className={cn(
                          "flex flex-col gap-1.5 p-3 rounded-md text-left transition-[color,background-color,border-color,opacity,transform,box-shadow] border border-transparent",
                          selectedId === item.id 
                             ? "bg-background shadow-sm border-border ring-1 ring-primary/20" 
                             : "hover:bg-muted/50 hover:border-border/50 text-muted-foreground",
                          item.status === "reviewed" && "opacity-60 grayscale-[0.5]"
                       )}
                    >
                       <div className="flex items-center justify-between w-full">
                          <div className="flex items-center gap-2">
                             <SeverityBadge severity={item.severity} />
                             <SourceBadge source={item.source} />
                          </div>
                          {item.status === "reviewed" && (
                             item.reviewResult === "confirmed" 
                                ? <IconCheck className="size-3 text-green-500" />
                                : <IconX className="size-3 text-red-500" />
                          )}
                       </div>
                       <div className={cn("text-xs font-medium line-clamp-2", selectedId === item.id ? "text-foreground" : "")}>
                          {item.title}
                       </div>
                       <div className="flex items-center justify-between text-[10px] font-mono opacity-70">
                          <span className="truncate max-w-[140px]">{item.path}</span>
                          <span>{item.time}</span>
                       </div>
                    </button>
                 ))}
              </div>
           </ScrollArea>
        </motion.div>

        {/* Right: Detail View */}
        <div className="flex-1 flex flex-col min-w-0 bg-background relative">
           {/* Detail Header */}
           <div className="p-6 pb-2">
              <div className="flex items-start gap-4">
                 <div className={cn(
                    "w-2 h-14 rounded-full shrink-0", 
                    SEVERITIES[selectedItem.severity as keyof typeof SEVERITIES].color
                 )}></div>
                 <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                       <span className="font-mono text-xs text-muted-foreground">ID: {selectedItem.id}</span>
                       <span className="text-muted-foreground/30">•</span>
                       <span className="font-mono text-xs text-muted-foreground uppercase">{selectedItem.method}</span>
                       <span className="text-muted-foreground/30">•</span>
                       <SourceBadge source={selectedItem.source} />
                    </div>
                    <h1 className="text-2xl font-bold tracking-tight mb-2">{selectedItem.title}</h1>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground bg-muted/30 p-1.5 px-3 rounded-md border border-border/50 w-fit">
                       <IconWorld className="size-3.5" />
                       <span className="font-mono truncate max-w-md">{selectedItem.target}</span>
                       <Button variant="ghost" size="icon" className="h-4 w-4 ml-2 hover:text-foreground" aria-label="View request code">
                          <IconCode className="size-3" />
                       </Button>
                    </div>
                 </div>
              </div>
           </div>

           {/* Detail Tabs */}
           <Tabs defaultValue="request" className="flex-1 flex flex-col min-h-0">
              <div className="px-6 border-b border-border">
                 <TabsList className="bg-transparent h-10 gap-6 p-0">
                    <TabsTrigger value="request" className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none h-full px-0">Request</TabsTrigger>
                    <TabsTrigger value="response" className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none h-full px-0">Response</TabsTrigger>
                    <TabsTrigger value="advisory" className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none h-full px-0">Advisory</TabsTrigger>
                    <TabsTrigger value="evidence" className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-primary rounded-none h-full px-0">Evidence Payload</TabsTrigger>
                 </TabsList>
              </div>
              
              <ScrollArea className="flex-1">
                 <div className="p-6">
                    <TabsContent value="request" className="mt-0">
                       <div className="font-mono text-xs bg-[#1e1e1e] text-[#d4d4d4] p-4 rounded-md overflow-x-auto border border-border">
                          <div className="text-[#569cd6]">POST</div> <span className="text-[#ce9178]">{selectedItem.path}</span> HTTP/1.1<br/>
                          <div className="text-[#9cdcfe]">Host:</div> api.target-system.com<br/>
                          <div className="text-[#9cdcfe]">Content-Type:</div> application/json<br/>
                          <div className="text-[#9cdcfe]">Authorization:</div> Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...<br/>
                          <br/>
                          {`{`}
                          <div className="pl-4">
                             <span className="text-[#9cdcfe]">{`"username"`}</span>: <span className="text-[#ce9178]">{`"admin"`}</span>,<br/>
                             <span className="text-[#9cdcfe]">{`"password"`}</span>: <span className="text-[#ce9178]">{`"' OR '1'='1"`}</span>
                          </div>
                          {`}`}
                       </div>
                    </TabsContent>
                    <TabsContent value="response" className="mt-0">
                       <div className="font-mono text-xs bg-[#1e1e1e] text-[#d4d4d4] p-4 rounded-md overflow-x-auto border border-border">
                          HTTP/1.1 <span className="text-[#b5cea8]">200</span> OK<br/>
                          <div className="text-[#9cdcfe]">Date:</div> Mon, 27 Jul 2024 12:28:53 GMT<br/>
                          <div className="text-[#9cdcfe]">Content-Type:</div> application/json<br/>
                          <div className="text-[#9cdcfe]">Server:</div> nginx<br/>
                          <br/>
                          {`{`}
                          <div className="pl-4">
                             <span className="text-[#9cdcfe]">{`"success"`}</span>: <span className="text-[#569cd6]">true</span>,<br/>
                             <span className="text-[#9cdcfe]">{`"token"`}</span>: <span className="text-[#ce9178]">{`"admin_access_granted_8829"`}</span>,<br/>
                             <span className="text-[#9cdcfe]">{`"user"`}</span>: {`{`}<br/>
                                <div className="pl-4">
                                   <span className="text-[#9cdcfe]">{`"id"`}</span>: <span className="text-[#b5cea8]">1</span>,<br/>
                                   <span className="text-[#9cdcfe]">{`"role"`}</span>: <span className="text-[#ce9178]">{`"superuser"`}</span>
                                </div>
                             {`}`}
                          </div>
                          {`}`}
                       </div>
                    </TabsContent>
                 </div>
              </ScrollArea>
           </Tabs>

           {/* 3. Floating Action Bar */}
           <div className="absolute bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-2 p-1.5 bg-foreground/90 backdrop-blur-md rounded-full shadow-2xl border border-white/10 text-background animate-in slide-in-from-bottom-4 duration-500">
              <Button 
                 variant="ghost" 
                 size="sm" 
                 onClick={() => handleAction(selectedId, "false_positive")}
                 className="rounded-full px-4 h-9 hover:bg-white/10 hover:text-red-300 text-background/80"
              >
                 <IconX className="size-4 mr-2" />
                 False Positive
                 <kbd className="ml-2 bg-white/10 px-1 rounded text-[10px] font-mono">I</kbd>
              </Button>
              <div className="w-px h-4 bg-white/20"></div>
              <Button 
                 variant="ghost" 
                 size="sm"
                 className="rounded-full px-4 h-9 hover:bg-white/10 hover:text-white text-background/80"
              >
                 Skip
                 <kbd className="ml-2 bg-white/10 px-1 rounded text-[10px] font-mono">→</kbd>
              </Button>
              <div className="w-px h-4 bg-white/20"></div>
              <Button 
                 variant="ghost" 
                 size="sm"
                 onClick={() => handleAction(selectedId, "confirmed")}
                 className="rounded-full px-4 h-9 bg-green-500 hover:bg-green-400 text-white font-bold shadow-[0_0_10px_rgba(34,197,94,0.4)]"
              >
                 <IconCheck className="size-4 mr-2" />
                 Confirm Issue
                 <kbd className="ml-2 bg-black/20 px-1 rounded text-[10px] font-mono">C</kbd>
              </Button>
           </div>
        </div>
      </div>
    </div>
  )
}
