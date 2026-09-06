"use client"

import Link from "next/link"
import { useRecentVulnerabilities } from "@/hooks/use-vulnerabilities"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_CLASS,
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
import { getSeverityVariant } from "@/lib/severity-config"
import { getStatusToneBadgeClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export function RecentVulnerabilities() {
  const t = useTranslations("overview.recentVulns")
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

  const { data, isLoading } = useRecentVulnerabilities(5)

  const vulnerabilities = data?.vulnerabilities ?? []

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>{t("title")}</CardTitle>
          <CardDescription>{t("description")}</CardDescription>
        </div>
        <Link 
          href="/vulnerabilities/" 
          className={cn("flex gap-1 hover:text-foreground items-center transition-colors", textRole.bodySubtle)}
        >
          {t("viewAll")}
          <IconExternalLink className="h-3.5 w-3.5" />
        </Link>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : vulnerabilities.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground">
            {t("noData")}
          </div>
        ) : (
          <div className="border rounded-md">
            <Table data-row-rhythm="dense">
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
                      className={cn(TABLE_DENSE_ROW_CLASS, "hover:bg-muted/50")}
                    >
                      <TableCell className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                        <Link href={detailHref} className="block w-full">
                          <Badge
                            variant="outline"
                            className={cn(
                              "gap-1.5 cursor-default transition-[background-color,border-color,color]",
                              getStatusToneBadgeClass(isPending ? "info" : "muted")
                            )}
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
                      <TableCell className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                        <Link href={detailHref} className="block w-full">
                          <Badge variant={getSeverityVariant(vuln.severity as VulnerabilitySeverity)}>
                            {tSeverity(vuln.severity as VulnerabilitySeverity)}
                          </Badge>
                        </Link>
                      </TableCell>
                      <TableCell className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                        <Link href={detailHref} className="block w-full">
                          <Badge variant="outline">{vuln.source}</Badge>
                        </Link>
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, "font-medium max-w-[120px] truncate")}>
                        <Link href={detailHref} className="block truncate w-full">
                          {vuln.vulnType}
                        </Link>
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, "max-w-[200px] truncate", textRole.tableCellSecondary)}>
                        <Link href={detailHref} className="block truncate w-full">
                          {vuln.url}
                        </Link>
                      </TableCell>
                      <TableCell className={cn(TABLE_DENSE_CELL_RHYTHM_CLASS, "whitespace-nowrap", textRole.tableCellSecondary)}>
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
