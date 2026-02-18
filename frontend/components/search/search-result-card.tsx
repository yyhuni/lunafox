"use client"

import { useState, useRef, useEffect } from "react"
import { ChevronDown, ChevronUp, Eye } from "@/components/icons"
import { useTranslations } from "next-intl"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import type { SearchResult, Vulnerability, WebsiteSearchResult } from "@/types/search.types"

// Type guard: Check if it is WebsiteSearchResult
function isWebsiteResult(result: SearchResult): result is WebsiteSearchResult {
  return 'vulnerabilities' in result
}

interface SearchResultCardProps {
  result: SearchResult
  onViewVulnerability?: (vuln: Vulnerability) => void
}

import { SEVERITY_STYLES } from "@/lib/severity-config"

// Vulnerability severity color configuration
const severityColors: Record<string, string> = {
  critical: SEVERITY_STYLES.critical.className,
  high: SEVERITY_STYLES.high.className,
  medium: SEVERITY_STYLES.medium.className,
  low: SEVERITY_STYLES.low.className,
  info: SEVERITY_STYLES.info.className,
}

// Status code Badge variant
function getStatusVariant(status: number | null): "default" | "secondary" | "destructive" | "outline" {
  if (!status) return "outline"
  if (status >= 200 && status < 300) return "default"
  if (status >= 300 && status < 400) return "secondary"
  if (status >= 400) return "destructive"
  return "outline"
}

export function SearchResultCard({ result, onViewVulnerability }: SearchResultCardProps) {
  const t = useTranslations('search.card')
  const [vulnOpen, setVulnOpen] = useState(false)
  const [techExpanded, setTechExpanded] = useState(false)
  const [isOverflowing, setIsOverflowing] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const formatHeaders = (headers: Record<string, string>) => {
    return Object.entries(headers)
      .map(([key, value]) => `${key}: ${value}`)
      .join("\n")
  }

  // Number of formatted bytes
  const formatBytes = (bytes: number | null) => {
    if (bytes === null || bytes === undefined) return null
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  // Check if content overflows
  const maxHeight = 26 * 4
  
  useEffect(() => {
    const el = containerRef.current
    if (!el || techExpanded) return

    const checkOverflow = () => {
      setIsOverflowing(el.scrollHeight > maxHeight)
    }

    checkOverflow()

    const resizeObserver = new ResizeObserver(checkOverflow)
    resizeObserver.observe(el)

    return () => resizeObserver.disconnect()
  }, [result.technologies, techExpanded, maxHeight])

  const handleViewVulnerability = (vuln: Vulnerability) => {
    if (onViewVulnerability) {
      onViewVulnerability(vuln)
    }
  }

  return (
    <Card className="overflow-hidden py-0 gap-0">
      <CardContent className="p-0">
        {/* Top URL + Badge row */}
        <div className="px-4 py-2 bg-muted/30 border-b space-y-2">
          <h3 className="font-mono text-sm break-all">
            {result.url || result.host}
          </h3>
          {/* Badge row */}
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={getStatusVariant(result.statusCode)} className="font-mono text-xs">
              {result.statusCode ?? '-'}
            </Badge>
            {result.webserver && (
              <Badge variant="outline" className="font-mono text-xs">
                {result.webserver}
              </Badge>
            )}
            {result.contentType && (
              <Badge variant="outline" className="font-mono text-xs">
                {result.contentType.split(';')[0]}
              </Badge>
            )}
            {formatBytes(result.contentLength) && (
              <Badge variant="outline" className="font-mono text-xs">
                {formatBytes(result.contentLength)}
              </Badge>
            )}

          </div>
        </div>

        {/* Center left and right columns */}
        <div className="flex flex-col md:flex-row">
          {/* Information area on the left */}
          <div className="w-full md:w-[320px] md:shrink-0 px-4 py-3 border-b md:border-b-0 md:border-r flex flex-col">
            <div className="space-y-1.5 text-sm">
              <div className="flex items-baseline">
                <span className="text-muted-foreground w-12 shrink-0">Title</span>
                <span className="truncate" title={result.title}>{result.title || '-'}</span>
              </div>
              <div className="flex items-baseline">
                <span className="text-muted-foreground w-12 shrink-0">Host</span>
                <span className="font-mono truncate" title={result.host}>{result.host || '-'}</span>
              </div>
            </div>

            {/* Technologies */}
            {result.technologies && result.technologies.length > 0 && (
              <div className="mt-3 flex flex-col gap-1">
                <div
                  ref={containerRef}
                  className="flex flex-wrap items-start gap-1 overflow-hidden transition-[max-height] duration-200"
                  style={{ maxHeight: techExpanded ? "none" : `${maxHeight}px` }}
                >
                  {result.technologies.map((tech, index) => (
                    <Badge
                      key={`${tech}-${index}`}
                      variant="secondary"
                      className="text-xs"
                    >
                      {tech}
                    </Badge>
                  ))}
                </div>
                {(isOverflowing || techExpanded) && (
                  <button type="button"
                    onClick={() => setTechExpanded(!techExpanded)}
                    className="inline-flex items-center gap-0.5 text-xs text-muted-foreground hover:text-foreground transition-colors self-start"
                  >
                    {techExpanded ? (
                      <>
                        <ChevronUp className="h-3 w-3" />
                        <span>{t('collapse')}</span>
                      </>
                    ) : (
                      <>
                        <ChevronDown className="h-3 w-3" />
                        <span>{t('expand')}</span>
                      </>
                    )}
                  </button>
                )}
              </div>
            )}
          </div>

          {/* Right Tab area */}
          <div className="w-full md:flex-1 flex flex-col">
            <Tabs defaultValue="header" className="w-full h-full flex flex-col gap-0">
              <TabsList className="h-[28px] gap-4 rounded-none border-b bg-transparent px-4 pt-1">
                <TabsTrigger 
                  value="header" 
                  className="h-full rounded-none border-b-2 border-transparent border-x-0 border-t-0 bg-transparent px-1 text-sm shadow-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 data-[state=active]:border-b-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                >
                  Header
                </TabsTrigger>
                <TabsTrigger 
                  value="body" 
                  className="h-full rounded-none border-b-2 border-transparent border-x-0 border-t-0 bg-transparent px-1 text-sm shadow-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 data-[state=active]:border-b-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                >
                  Body
                </TabsTrigger>
                {result.location && (
                  <TabsTrigger 
                    value="location" 
                    className="h-full rounded-none border-b-2 border-transparent border-x-0 border-t-0 bg-transparent px-1 text-sm shadow-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 data-[state=active]:border-b-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                  >
                    Location
                  </TabsTrigger>
                )}
              </TabsList>
              <TabsContent value="header" className="flex-1 overflow-auto bg-muted/30 px-4 py-2 max-h-[200px]">
                <pre className="text-xs font-mono whitespace-pre-wrap">
                  {result.responseHeaders ? formatHeaders(result.responseHeaders) : '-'}
                </pre>
              </TabsContent>
              <TabsContent value="body" className="flex-1 overflow-auto bg-muted/30 px-4 py-2 max-h-[200px]">
                <pre className="text-xs font-mono whitespace-pre-wrap">
                  {result.responseBody || '-'}
                </pre>
              </TabsContent>
              {result.location && (
                <TabsContent value="location" className="flex-1 overflow-auto bg-muted/30 px-4 py-2 max-h-[200px]">
                  <pre className="text-xs font-mono whitespace-pre-wrap">
                    {result.location}
                  </pre>
                </TabsContent>
              )}
            </Tabs>
          </div>
        </div>

        {/* Bottom vulnerability section - shown only for Website results */}
        {isWebsiteResult(result) && result.vulnerabilities && result.vulnerabilities.length > 0 && (
          <div className="border-t">
            <Collapsible open={vulnOpen} onOpenChange={setVulnOpen}>
              <CollapsibleTrigger className="flex items-center gap-1 px-4 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors w-full">
                {vulnOpen ? (
                  <ChevronDown className="size-4" />
                ) : (
                  <ChevronUp className="size-4 rotate-90" />
                )}
                <span>{t('vulnerabilities', { count: result.vulnerabilities.length })}</span>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="px-4 pb-4">
                  <Table className="table-fixed">
                    <TableHeader>
                      <TableRow>
                        <TableHead className="text-xs w-[50%]">{t('vulnName')}</TableHead>
                        <TableHead className="text-xs w-[20%]">{t('vulnType')}</TableHead>
                        <TableHead className="text-xs w-[20%]">{t('severity')}</TableHead>
                        <TableHead className="text-xs w-[10%]"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {result.vulnerabilities.map((vuln, index) => (
                        <TableRow key={`${vuln.name}-${index}`}>
                          <TableCell className="text-xs font-medium">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="truncate block max-w-full cursor-default">
                                  {vuln.name}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="max-w-[400px]">
                                {vuln.name}
                              </TooltipContent>
                            </Tooltip>
                          </TableCell>
                          <TableCell className="text-xs">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="truncate block max-w-full cursor-default">
                                  {vuln.vulnType}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="top">
                                {vuln.vulnType}
                              </TooltipContent>
                            </Tooltip>
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant="outline"
                              className={`text-xs ${severityColors[vuln.severity] || severityColors.info}`}
                            >
                              {vuln.severity}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 px-2"
                              onClick={() => handleViewVulnerability(vuln)}
                            >
                              <Eye className="h-3.5 w-3.5" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </CollapsibleContent>
            </Collapsible>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
