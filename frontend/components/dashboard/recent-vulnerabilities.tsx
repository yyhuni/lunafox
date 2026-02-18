"use client"

import { useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import Link from "next/link"
import { VulnerabilityService } from "@/services/vulnerability.service"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { IconExternalLink } from "@/components/icons"
import { Circle, CheckCircle2 } from "@/components/icons"
import type { Vulnerability, VulnerabilitySeverity } from "@/types/vulnerability.types"
import { useTranslations } from "next-intl"
import { useLocale } from "next-intl"
import { SEVERITY_STYLES } from "@/lib/severity-config"

export function RecentVulnerabilities() {
  const t = useTranslations("dashboard.recentVulns")
  const tSeverity = useTranslations("severity")
  const tColumns = useTranslations("columns")
  const tTooltips = useTranslations("tooltips")
  const locale = useLocale()
  
  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleString(locale === 'zh' ? 'zh-CN' : 'en-US', {
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  }

  const severityConfig = useMemo(() => ({
    critical: { label: tSeverity("critical"), className: SEVERITY_STYLES.critical.className },
    high: { label: tSeverity("high"), className: SEVERITY_STYLES.high.className },
    medium: { label: tSeverity("medium"), className: SEVERITY_STYLES.medium.className },
    low: { label: tSeverity("low"), className: SEVERITY_STYLES.low.className },
    info: { label: tSeverity("info"), className: SEVERITY_STYLES.info.className },
  }), [tSeverity])

  const { data, isLoading } = useQuery({
    queryKey: ["dashboard", "recent-vulnerabilities"],
    queryFn: () => VulnerabilityService.getAllVulnerabilities({ page: 1, pageSize: 5 }),
  })

  const vulnerabilities = data?.results ?? []

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>{t("title")}</CardTitle>
          <CardDescription>{t("description")}</CardDescription>
        </div>
        <Link 
          href="/vulnerabilities/" 
          className="text-sm text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1"
        >
          {t("viewAll")}
          <IconExternalLink className="h-3.5 w-3.5" />
        </Link>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : vulnerabilities.length === 0 ? (
          <div className="text-center text-muted-foreground py-8">
            {t("noData")}
          </div>
        ) : (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{tColumns("common.status")}</TableHead>
                  <TableHead>{tColumns("vulnerability.severity")}</TableHead>
                  <TableHead>{tColumns("vulnerability.source")}</TableHead>
                  <TableHead>{tColumns("common.type")}</TableHead>
                  <TableHead>{tColumns("common.url")}</TableHead>
                  <TableHead>{tColumns("common.createdAt")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {vulnerabilities.map((vuln: Vulnerability) => {
                  const isReviewed = vuln.isReviewed
                  const isPending = !isReviewed
                  const detailHref = `/vulnerabilities/?id=${vuln.id}`

                  return (
                    <TableRow
                      key={vuln.id}
                      className="hover:bg-muted/50"
                    >
                      <TableCell>
                        <Link href={detailHref} className="block w-full">
                          <Badge
                            variant="outline"
                            className={`transition-[background-color,border-color,color] gap-1.5 cursor-default ${isPending
                              ? "bg-blue-500/10 text-blue-600 border-blue-500/30 dark:text-blue-400 dark:border-blue-400/30"
                              : "bg-muted/50 text-muted-foreground border-muted-foreground/20"
                            }`}
                          >
                            {isPending ? (
                              <Circle className="h-3 w-3" />
                            ) : (
                              <CheckCircle2 className="h-3 w-3" />
                            )}
                            {isPending ? tTooltips("pending") : tTooltips("reviewed")}
                          </Badge>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Link href={detailHref} className="block w-full">
                          <Badge className={severityConfig[vuln.severity as VulnerabilitySeverity]?.className}>
                            {severityConfig[vuln.severity as VulnerabilitySeverity]?.label ?? vuln.severity}
                          </Badge>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <Link href={detailHref} className="block w-full">
                          <Badge variant="outline">{vuln.source}</Badge>
                        </Link>
                      </TableCell>
                      <TableCell className="font-medium max-w-[120px] truncate">
                        <Link href={detailHref} className="block w-full truncate">
                          {vuln.vulnType}
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs max-w-[200px] truncate">
                        <Link href={detailHref} className="block w-full truncate">
                          {vuln.url}
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-xs whitespace-nowrap">
                        <Link href={detailHref} className="block w-full">
                          {formatTime(vuln.createdAt)}
                        </Link>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
