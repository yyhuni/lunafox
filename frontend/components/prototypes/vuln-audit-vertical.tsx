"use client"

import React from "react"
import {
  IconSearch,
  IconChevronDown,
  Circle,
  CheckCircle2
} from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { cn } from "@/lib/utils"
import { VulnerabilityAuditDetail } from "@/components/prototypes/vulnerability-audit-detail"
import type { Vulnerability } from "@/types/vulnerability.types"
import { SEVERITY_STYLES } from "@/lib/severity-config"
import { useVerticalResize } from "@/hooks/use-vertical-resize"
import { useUrlState } from "@/hooks/use-url-state"

const MOCK_VULNS: Vulnerability[] = Array.from({ length: 18 }, (_, i) => {
  const severity = (["critical", "high", "medium", "low"] as const)[i % 4]
  // Simulate long content for edge case testing
  const isLong = i === 1 || i === 5
  // Simulate Dalfox finding
  const isDalfox = i % 3 === 1
  const source = isDalfox ? "dalfox" : i % 3 === 0 ? "nuclei" : "custom"

  return {
    id: i + 1,
    vulnType: isLong
      ? "Blind SQL Injection via User-Agent Header in /api/v1/auth/login endpoint resulting in Database Extraction (Time-Based)"
      : isDalfox
        ? "XSS (Reflected) in search parameter"
        : i % 3 === 0
          ? "SQL Injection in /api/login"
          : "Sensitive Data Exposure",
    severity,
    source,
    url: isLong
      ? "https://api.target-system.com/v1/endpoints/search?q=test&category=all&sort=desc&filter[active]=true&filter[role]=admin&session_id=abcdef1234567890&tracking_code=utm_source_google_campaign_spring_sale_2024&very_long_param_to_test_truncation=true"
      : `https://api.target-system.com/v1/endpoints/${i}`,
    isReviewed: i < 5,
    createdAt: new Date(Date.now() - i * 1000 * 60 * 60).toISOString(),
    updatedAt: new Date().toISOString(),
    description: isLong
      ? "The application appears to be vulnerable to SQL Injection. The input parameter 'id' is not properly sanitized before being used in a SQL query. This allows an attacker to manipulate the query structure, potentially accessing unauthorized data, modifying database contents, or executing administrative operations. The vulnerability was confirmed using a time-based blind injection technique where the server response was delayed by 5 seconds."
      : "The application appears to be vulnerable to SQL Injection.",
    cvssScore: severity === "critical" ? 9.8 : severity === "high" ? 7.5 : 5.0,
    rawOutput: {
      "curl-command": isLong
        ? `curl -X POST "https://api.target-system.com/v1/endpoints/${i}?token=very_long_token_string_that_goes_on_and_on" -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36" -H "Content-Type: application/json" -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" -d '{"id": "1 OR SLEEP(5)", "nested": {"deeply": {"nested": "value"}}}'`
        : `curl -X POST https://api.target-system.com/v1/endpoints/${i} -d 'id=1 OR 1=1'`,
      // Dalfox fields
      payload: isDalfox ? "\"><script>alert(1)</script>" : undefined,
      evidence: isDalfox ? "GET /?q=\"><script>alert(1)</script> HTTP/1.1" : undefined,
      param: isDalfox ? "q" : undefined,
      method: "GET",

      // Nuclei fields
      request: !isDalfox
        ? (isLong
            ? `POST /v1/endpoints/${i}?token=very_long_token_string_that_goes_on_and_on HTTP/1.1
Host: api.target-system.com
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
Content-Type: application/json
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
Accept: */*
X-Forwarded-For: 127.0.0.1
Connection: keep-alive

{
  "id": "1' OR SLEEP(5)--",
  "comment": "This is a very long payload to test how the code block handles wrapping and scrolling. It should handle it gracefully without breaking the layout.",
  "payload": "UNION SELECT 1,2,3,4,5,6,7,8,9,user(),database(),version(),@@hostname,@@datadir,group_concat(table_name) FROM information_schema.tables WHERE table_schema=database()--"
}`
            : `POST /v1/endpoints/${i} HTTP/1.1\nHost: api.target-system.com\nContent-Type: application/json\n\n{\n  "id": "1' OR '1'='1"\n}`)
        : undefined,

      response: !isDalfox
        ? (isLong
            ? `HTTP/1.1 500 Internal Server Error
Date: Mon, 27 Jul 2024 12:28:53 GMT
Content-Type: application/json; charset=utf-8
Content-Length: 1204
Server: nginx/1.18.0 (Ubuntu)
X-Powered-By: Express

{
  "error": "Database Error",
  "message": "Syntax error or access violation: 1064 You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '' OR SLEEP(5)--' at line 1",
  "stack": "Error: ER_PARSE_ERROR: You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '' OR SLEEP(5)--' at line 1\\n    at Query.Sequence._packetToError (/app/node_modules/mysql/lib/protocol/sequences/Sequence.js:47:14)\\n    at Query.ErrorPacket (/app/node_modules/mysql/lib/protocol/sequences/Query.js:79:18)\\n    at Protocol._parsePacket (/app/node_modules/mysql/lib/protocol/Protocol.js:291:23)\\n    at Parser.write (/app/node_modules/mysql/lib/protocol/Parser.js:80:12)\\n    at Protocol.write (/app/node_modules/mysql/lib/protocol/Protocol.js:38:16)\\n    at Socket.<anonymous> (/app/node_modules/mysql/lib/Connection.js:88:28)\\n    at Socket.<anonymous> (/app/node_modules/mysql/lib/Connection.js:526:10)\\n    at Socket.emit (events.js:315:20)\\n    at addChunk (_stream_readable.js:309:12)\\n    at readableAddChunk (_stream_readable.js:284:9)"
}`
            : `HTTP/1.1 200 OK\nContent-Type: application/json\n\n{\n  "id": 1,\n  "role": "admin",\n  "data": "sensitive"\n}`)
        : undefined,

      info: {
        description: isDalfox
          ? "Reflected Cross-Site Scripting (XSS) occurs when an application receives data in an HTTP request and includes that data within the immediate response in an unsafe way."
          : "SQL Injection detected via boolean-based blind technique.",
        classification: {
          "cwe-id": isDalfox ? ["CWE-79"] : ["CWE-89"],
          "cve-id": "CVE-2024-XXXX"
        },
        reference: [
          "https://owasp.org/www-community/attacks/SQL_Injection",
          "https://cwe.mitre.org/data/definitions/89.html",
          "https://portswigger.net/web-security/sql-injection",
          "https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html"
        ]
      }
    }
  }
})

// --- Component ---

export function VulnAuditVertical() {
  // 1. URL State Management
  const [selectedIdStr, setSelectedId] = useUrlState("id", String(MOCK_VULNS[0].id))
  const [filter, setFilter] = useUrlState<"all" | "pending" | "reviewed">("filter", "all")
  
  const selectedId = parseInt(selectedIdStr, 10) || MOCK_VULNS[0].id

  // 2. Vertical Resizing
  const { height, startResizing, containerRef, isResizing } = useVerticalResize(45)

  // Filter items
  const filteredItems = React.useMemo(() => {
    return MOCK_VULNS.filter(item => {
      if (filter === "all") return true
      if (filter === "pending") return !item.isReviewed
      if (filter === "reviewed") return item.isReviewed
      return true
    })
  }, [filter])

  // Pagination Logic (Client-side simulation)
  const [pageStr, setPage] = useUrlState<string>("page", "1")
  const page = parseInt(pageStr, 10) || 1
  const pageSize = 10
  const totalPages = Math.ceil(filteredItems.length / pageSize)
  const paginatedItems = filteredItems.slice((page - 1) * pageSize, page * pageSize)
  
  const selectedItem = MOCK_VULNS.find(i => i.id === selectedId) || MOCK_VULNS[0]

  const severityConfig = {
    critical: { className: SEVERITY_STYLES.critical.className },
    high: { className: SEVERITY_STYLES.high.className },
    medium: { className: SEVERITY_STYLES.medium.className },
    low: { className: SEVERITY_STYLES.low.className },
    info: { className: SEVERITY_STYLES.info.className },
  }

  return (
    <div ref={containerRef} className="flex flex-col h-full min-h-0 bg-background border rounded-lg overflow-hidden select-none">
      {/* Top: List View (Table) */}
      <div 
        className="flex flex-col border-b resize-y overflow-hidden relative min-h-[200px]"
        style={{ height: `${height}%` }}
      >
        <div className="h-12 border-b bg-card flex items-center justify-between px-4 shrink-0">
           <div className="flex items-center gap-4">
              <div className="flex items-center bg-muted/50 p-1 rounded-lg border">
                 <button 
                    onClick={() => { setFilter("all"); setPage("1") }}
                    className={cn("px-3 py-1 text-xs font-medium rounded-md transition-[color,background-color,border-color,opacity,transform,box-shadow]", filter === "all" ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground")}
                 >
                    All <span className="opacity-50 ml-1">{MOCK_VULNS.length}</span>
                 </button>
                 <button 
                    onClick={() => { setFilter("pending"); setPage("1") }}
                    className={cn("px-3 py-1 text-xs font-medium rounded-md transition-[color,background-color,border-color,opacity,transform,box-shadow]", filter === "pending" ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground")}
                 >
                    Pending <span className="opacity-50 ml-1">{MOCK_VULNS.filter(i => !i.isReviewed).length}</span>
                 </button>
                 <button 
                    onClick={() => { setFilter("reviewed"); setPage("1") }}
                    className={cn("px-3 py-1 text-xs font-medium rounded-md transition-[color,background-color,border-color,opacity,transform,box-shadow]", filter === "reviewed" ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground")}
                 >
                    Reviewed <span className="opacity-50 ml-1">{MOCK_VULNS.filter(i => i.isReviewed).length}</span>
                 </button>
              </div>
           </div>
           
           <div className="flex items-center gap-2">
              <div className="flex items-center gap-1 text-xs text-muted-foreground mr-2">
                 <span>Page {page} of {totalPages || 1}</span>
                 <div className="flex gap-1 ml-2">
                    <Button variant="outline" size="icon" className="h-6 w-6" disabled={page <= 1} onClick={() => setPage(String(page - 1))} aria-label="Previous page">
                       <IconChevronDown className="h-3 w-3 rotate-90" />
                    </Button>
                    <Button variant="outline" size="icon" className="h-6 w-6" disabled={page >= totalPages} onClick={() => setPage(String(page + 1))} aria-label="Next page">
                       <IconChevronDown className="h-3 w-3 -rotate-90" />
                    </Button>
                 </div>
              </div>
              <div className="relative w-48">
                 <IconSearch className="absolute left-2 top-2 size-3 text-muted-foreground" />
                 <Input type="search" name="vulnerabilitySearch" autoComplete="off" placeholder="Search…" className="h-7 pl-7 text-xs bg-background" />
              </div>
           </div>
        </div>
        
        <div className="flex-1 overflow-auto bg-background">
           <div className="flex flex-col divide-y divide-border/40">
              {/* Table Header */}
              <div className="flex items-center px-4 py-2 bg-muted/30 text-[10px] uppercase text-muted-foreground font-semibold tracking-wider sticky top-0 z-10 border-b min-w-max">
                 <div className="w-8 flex justify-center shrink-0"><Checkbox className="h-3 w-3" /></div>
                 <div className="w-8 shrink-0 text-center">St</div>
                 <div className="w-20 shrink-0">Severity</div>
                 <div className="flex-[2] min-w-[200px] px-2">Vulnerability</div>
                 <div className="w-24 shrink-0 px-2">Source</div>
                 <div className="flex-[3] min-w-[250px] px-2">Target URL</div>
                 <div className="w-32 shrink-0 text-right px-2">Found At</div>
              </div>

              {paginatedItems.map(item => (
                 <div 
                    key={item.id} 
                    className={cn(
                       "group flex items-center py-2 px-4 hover:bg-muted/40 transition-colors relative border-b border-border/40 last:border-0 min-w-max",
                       selectedId === item.id ? "bg-muted/60" : ""
                    )}
                 >
                    {/* Active Indicator Bar */}
                    {selectedId === item.id && (
                       <div className="absolute left-0 top-0 bottom-0 w-[3px] bg-primary"></div>
                    )}

                    {/* Checkbox */}
                    <div className="w-8 flex justify-center shrink-0">
                       <Checkbox
                          className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100 data-[state=checked]:opacity-100 transition-opacity"
                          onClick={(e) => e.stopPropagation()}
                          onKeyDown={(e) => e.stopPropagation()}
                       />
                    </div>
                    
                    {/* Status */}
                    <div className="w-8 shrink-0 flex justify-center">
                       {item.isReviewed ? (
                          <div className="text-green-600" title="Reviewed">
                             <CheckCircle2 className="h-3.5 w-3.5" />
                          </div>
                       ) : (
                          <div className="text-blue-500" title="Pending">
                             <Circle className="h-3.5 w-3.5" />
                          </div>
                       )}
                    </div>

                    {/* Severity */}
                    <div className="w-20 shrink-0">
                       <Badge className={cn("h-5 px-1.5 text-[10px] font-bold uppercase rounded-sm border-0 w-full justify-center", severityConfig[item.severity].className)}>
                          {item.severity.slice(0, 3)}
                       </Badge>
                    </div>

                    {/* Vuln Type (Title) */}
                    <div className="flex-[2] min-w-[200px] px-2">
                      <button
                        type="button"
                        onClick={() => setSelectedId(String(item.id))}
                        className="w-full truncate text-left font-medium text-sm text-foreground hover:underline focus-visible:underline focus-visible:outline-none"
                        title={item.vulnType}
                        aria-label={`Open details for ${item.vulnType}`}
                      >
                        {item.vulnType}
                      </button>
                    </div>

                    {/* Source */}
                    <div className="w-24 shrink-0 px-2">
                       <Badge variant="outline" className="h-5 px-1.5 text-[10px] font-normal text-muted-foreground w-full justify-center">
                          {item.source}
                       </Badge>
                    </div>

                    {/* URL */}
                    <div className="flex-[3] min-w-[250px] px-2 text-xs text-muted-foreground truncate font-mono opacity-70" title={item.url}>
                       {item.url}
                    </div>

                    {/* Created At */}
                    <div className="w-32 shrink-0 text-right px-2 text-xs text-muted-foreground tabular-nums">
                       {new Date(item.createdAt).toLocaleDateString()} <span className="opacity-50">{new Date(item.createdAt).toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'})}</span>
                    </div>
                 </div>
              ))}
              
              {paginatedItems.length === 0 && (
                 <div className="p-8 text-center text-muted-foreground text-sm">
                    No vulnerabilities found in this view.
                 </div>
              )}
           </div>
        </div>
        
        {/* Resize Handle */}
        <button
           type="button"
           className={cn(
              "absolute bottom-0 left-0 right-0 h-1.5 cursor-row-resize z-50 transition-colors flex items-center justify-center group/handle hover:bg-primary/10",
              isResizing ? "bg-primary/20" : "bg-transparent"
           )}
           onMouseDown={startResizing}
           aria-label="Resize vulnerability list and details panels"
        >
           <div className={cn(
              "w-12 h-1 rounded-full transition-colors",
              isResizing ? "bg-primary" : "bg-border group-hover/handle:bg-primary/50"
           )}></div>
        </button>
      </div>

      {/* Bottom: Detail View (New Audit Layout) */}
      <div className="flex-1 flex flex-col min-h-0 bg-background relative overflow-hidden">
         <VulnerabilityAuditDetail 
            vulnerability={selectedItem} 
            className="h-full"
         />
      </div>
    </div>
  )
}
