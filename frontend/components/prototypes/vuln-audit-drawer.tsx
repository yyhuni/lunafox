"use client"

import React, { useState } from "react"
import { motion, AnimatePresence } from "framer-motion"
import {
  IconCheck,
  IconX,
  IconSearch,
  IconCode,
  IconBrowser,
  IconActivity,
  IconEye,
  IconChevronRight
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"

// --- Shared Mock Data & Components (Same as above) ---
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

export function VulnAuditDrawer() {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [items] = useState(MOCK_VULNS)
  
  const selectedItem = items.find(i => i.id === selectedId)

  return (
    <div className="flex h-[calc(100vh-4rem)] bg-background border rounded-lg overflow-hidden shadow-sm relative">
      {/* Main Table (Full Width) */}
      <div className={cn("flex-1 flex flex-col transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300", selectedId ? "mr-[500px]" : "")}>
        <div className="h-14 border-b border-border bg-card flex items-center justify-between px-4 shrink-0">
           <div className="flex items-center gap-3">
              <h2 className="font-bold">Vulnerability Audit</h2>
              <div className="flex gap-2">
                 <Badge variant="secondary">Total: {items.length}</Badge>
                 <Badge variant="outline" className="text-yellow-500 border-yellow-500/30 bg-yellow-500/5">Pending: {items.filter(i => i.status === 'pending').length}</Badge>
              </div>
           </div>
           <div className="w-64 relative">
              <IconSearch className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
              <Input type="search" name="vulnerabilitySearch" autoComplete="off" placeholder="Search vulnerabilities…" className="pl-9" />
           </div>
        </div>
        
        <div className="flex-1 overflow-auto bg-muted/5 p-4">
           <div className="border rounded-md bg-background shadow-sm overflow-hidden">
              <table className="w-full text-left border-collapse">
                 <thead className="bg-muted/30 text-xs uppercase text-muted-foreground font-semibold border-b border-border">
                    <tr>
                       <th className="px-4 py-3 w-16">Sev</th>
                       <th className="px-4 py-3">Vulnerability</th>
                       <th className="px-4 py-3 w-32">Source</th>
                       <th className="px-4 py-3 w-24">Method</th>
                       <th className="px-4 py-3">Target</th>
                       <th className="px-4 py-3 w-32 text-right">Status</th>
                       <th className="px-4 py-3 w-10">
                          <span className="sr-only">Open details</span>
                       </th>
                    </tr>
                 </thead>
                 <tbody className="text-sm">
                    {items.map(item => (
                       <tr 
                          key={item.id} 
                          className={cn(
                             "border-b border-border/50 hover:bg-muted/50 transition-colors group",
                             selectedId === item.id ? "bg-muted/50" : ""
                          )}
                       >
                          <td className="px-4 py-3"><SeverityBadge severity={item.severity} /></td>
                          <td className="px-4 py-3 font-medium">{item.title}</td>
                          <td className="px-4 py-3"><SourceBadge source={item.source} /></td>
                          <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{item.method}</td>
                          <td className="px-4 py-3 text-muted-foreground truncate max-w-[200px]" title={item.target}>{item.path}</td>
                          <td className="px-4 py-3 text-right">
                             {item.status === "reviewed" ? (
                                <Badge variant="outline" className={cn(
                                   "ml-auto",
                                   item.reviewResult === "confirmed" ? "text-green-500 border-green-500/30" : "text-red-500 border-red-500/30"
                                )}>
                                   {item.reviewResult === "confirmed" ? "Fixed" : "False Pos"}
                                </Badge>
                             ) : (
                                <span className="text-muted-foreground text-xs italic">Pending</span>
                             )}
                          </td>
                          <td className="px-4 py-3 text-center text-muted-foreground">
                             <button
                               type="button"
                               onClick={() => setSelectedId(item.id)}
                               aria-label={`Open details for ${item.title}`}
                               className="inline-flex items-center justify-center rounded p-1 hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                             >
                               <IconChevronRight className="size-4 opacity-0 group-hover:opacity-100 transition-opacity" />
                             </button>
                          </td>
                       </tr>
                    ))}
                 </tbody>
              </table>
           </div>
        </div>
      </div>

      {/* Drawer (Slide-over) */}
      <AnimatePresence>
         {selectedId && selectedItem && (
            <motion.div 
               initial={{ x: "100%" }}
               animate={{ x: 0 }}
               exit={{ x: "100%" }}
               transition={{ type: "spring", damping: 25, stiffness: 200 }}
               className="absolute top-0 right-0 bottom-0 w-[500px] bg-background border-l border-border shadow-xl z-20 flex flex-col"
            >
               {/* Drawer Header */}
               <div className="h-14 border-b border-border flex items-center justify-between px-4 shrink-0 bg-muted/10">
                  <div className="font-bold flex items-center gap-2">
                     <span className="text-muted-foreground font-mono">{selectedItem.id}</span>
                     <span>Details</span>
                  </div>
                  <Button variant="ghost" size="icon" onClick={() => setSelectedId(null)} aria-label="Close details panel">
                     <IconX className="size-4" />
                  </Button>
               </div>

               <ScrollArea className="flex-1">
                  <div className="p-6 space-y-6">
                     <div>
                        <div className="flex gap-2 mb-2">
                           <SeverityBadge severity={selectedItem.severity} />
                           <SourceBadge source={selectedItem.source} />
                        </div>
                        <h3 className="text-xl font-bold leading-tight">{selectedItem.title}</h3>
                        <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground bg-muted/30 p-2 rounded border border-border/50">
                           <IconBrowser className="size-3.5" />
                           <span className="font-mono truncate">{selectedItem.target}</span>
                        </div>
                     </div>

                     <div className="space-y-4">
                        <div className="border rounded-md overflow-hidden">
                           <div className="bg-muted/30 px-3 py-2 text-xs font-bold border-b text-muted-foreground uppercase">Request Payload</div>
                           <div className="bg-[#1e1e1e] text-[#d4d4d4] p-3 font-mono text-xs overflow-x-auto">
                              <span className="text-[#569cd6]">{selectedItem.method}</span> <span className="text-[#ce9178]">{selectedItem.path}</span> HTTP/1.1<br/>
                              <br/>
                              {`{ "id": "1' OR '1'='1" }`}
                           </div>
                        </div>
                        
                        <div className="border rounded-md overflow-hidden">
                           <div className="bg-muted/30 px-3 py-2 text-xs font-bold border-b text-muted-foreground uppercase">Description</div>
                           <div className="p-3 text-sm text-muted-foreground leading-relaxed">
                              The application appears to be vulnerable to SQL Injection. The input parameter <code>id</code> is not properly sanitized before being used in a SQL query.
                           </div>
                        </div>
                     </div>
                  </div>
               </ScrollArea>

               {/* Drawer Footer Actions */}
               <div className="p-4 border-t border-border bg-muted/10 grid grid-cols-2 gap-3">
                  <Button variant="outline" className="border-red-200 text-red-600 hover:bg-red-50 hover:text-red-700 dark:hover:bg-red-950/30">
                     <IconX className="size-4 mr-2" />
                     False Positive
                  </Button>
                  <Button className="bg-green-600 hover:bg-green-700 text-white">
                     <IconCheck className="size-4 mr-2" />
                     Confirm
                  </Button>
               </div>
            </motion.div>
         )}
      </AnimatePresence>
    </div>
  )
}
