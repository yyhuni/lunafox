"use client"

import { useState, useMemo } from "react"
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts"
import { useStatisticsHistory } from "@/hooks/use-overview"
import type { StatisticsHistoryItem } from "@/types/overview.types"
import { useLocale, useTranslations } from "next-intl"
import { formatAssetTrendAxisTick } from "@/components/overview/overview-axis-format"
import { OverviewNumberTicker } from "@/components/overview/overview-number-ticker"
import { IconTrendingUp, IconActivity } from "@/components/icons"
import { cn } from "@/lib/utils"
import { Skeleton } from "@/components/ui/skeleton"
import { motion } from "framer-motion"
import {
   Card,
   CardContent,
   CardDescription,
   CardHeader,
   CardTitle,
} from "@/components/ui/card"
import { getTrendToneTextClass } from "@/lib/status-config"

const ASSET_DELTA_TEXT_CLASS = getTrendToneTextClass("positive")

/**
 * Fill missing date data, ensure always returning complete days
 * Based on the earliest record date, fill backwards, missing dates filled with 0
 */
function fillMissingDates(data: StatisticsHistoryItem[] | undefined, days: number): StatisticsHistoryItem[] {
   if (!data || data.length === 0) return []

   const dataMap = new Map(data.map(item => [item.date, item]))
   const earliestDate = new Date(data[0].date)
   const result: StatisticsHistoryItem[] = []
   const startDate = new Date(earliestDate)
   startDate.setDate(startDate.getDate() - (days - data.length))

   for (let i = 0; i < days; i++) {
      const currentDate = new Date(startDate)
      currentDate.setDate(startDate.getDate() + i)
      const dateStr = currentDate.toISOString().split('T')[0]

      const existing = dataMap.get(dateStr)
      if (existing) {
         result.push(existing)
      } else {
         result.push({
            date: dateStr,
            totalTargets: 0,
            totalSubdomains: 0,
            totalIps: 0,
            totalEndpoints: 0,
            totalWebsites: 0,
            totalVulns: 0,
            totalAssets: 0,
         })
      }
   }
   return result
}

export function AssetTrendChart() {
   const { data: rawData, isLoading } = useStatisticsHistory(14)
   const locale = useLocale()
   const t = useTranslations("overview.assetTrend")
   const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')

   // Prepare data with deltas
   const processedData = useMemo(() => {
      if (!rawData || rawData.length === 0) return []
      const filled = fillMissingDates(rawData, 14)

      return filled.map((curr, i) => {
         const prev = filled[i - 1] || curr
         return {
            ...curr,
            newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
            newIp: Math.max(0, curr.totalIps - prev.totalIps),
            newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
            newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
         }
      })
   }, [rawData])

   const latest = useMemo(() =>
      processedData.length > 0 ? processedData[processedData.length - 1] : null
      , [processedData])

   // Chart color configuration
   const config = {
      sub: {
         label: t('subdomains'),
         totalKey: 'totalSubdomains',
         deltaKey: 'newSub',
         color: "var(--foreground)",
         bg: "bg-foreground",
         text: "text-foreground"
      },
      ip: {
         label: t('ips'),
         totalKey: 'totalIps',
         deltaKey: 'newIp',
         color: "var(--foreground)",
         bg: "bg-foreground",
         text: "text-foreground"
      },
      url: {
         label: t('endpoints'),
         totalKey: 'totalEndpoints',
         deltaKey: 'newUrl',
         color: "var(--foreground)",
         bg: "bg-foreground",
         text: "text-foreground"
      },
      site: {
         label: t('websites'),
         totalKey: 'totalWebsites',
         deltaKey: 'newSite',
         color: "var(--foreground)",
         bg: "bg-foreground",
         text: "text-foreground"
      },
   } as const

   const activeConfig = config[activeTab]
   const activeAxisMaxValue = Math.max(
      0,
      ...processedData.map((item) => Number(item[activeConfig.totalKey as keyof typeof item] ?? 0))
   )

   if (isLoading) {
      return (
         <Card className="overview-chart-card flex w-full flex-col">
            <div className="overview-chart-kicker chart-kicker">
               <IconActivity className="size-4" />
               <span>{t("kicker")}</span>
            </div>
            <Skeleton className="h-[300px] m-4" />
         </Card>
      )
   }

   if (!latest) {
      return (
         <Card className="flex h-[300px] items-center justify-center w-full">
            <div className="flex flex-col gap-2 items-center text-muted-foreground">
               <IconActivity className="h-8 opacity-20 w-8" />
               <span className="opacity-50 text-sm">{t("noData")}</span>
            </div>
         </Card>
      )
   }

   return (
      <Card className="overview-chart-card flex h-[300px] flex-col overflow-hidden pb-0">
         <div className="overview-chart-kicker chart-kicker">
            <IconActivity className="size-4" />
            <span>{t("kicker")} / {activeConfig.label.toUpperCase()}</span>
         </div>

         <CardHeader className="overview-chart-header pb-2">
            <div className="flex items-center justify-between">
               <div className="space-y-1">
                  <CardTitle className="overview-chart-title">{t("title")}</CardTitle>
                  <CardDescription>{t("description")}</CardDescription>
               </div>
               {/* Mini Sparkline Logic */}
               {latest && (
                  <div className="bg-muted/30 flex gap-2 items-center px-2 py-1 rounded text-sm">
                     <span className="font-mono text-muted-foreground text-xs uppercase">{t('current')}:</span>
                     <span className={cn("font-bold font-mono", activeConfig.text)}>
                        <OverviewNumberTicker value={latest[activeConfig.totalKey as keyof typeof latest] as number} locale={locale} />
                     </span>
                     <div className="bg-border h-4 mx-1 w-[1px]"></div>
                     <span className={cn("flex font-mono gap-1 items-center text-xs", ASSET_DELTA_TEXT_CLASS)}>
                        <IconTrendingUp className="h-3 w-3" />
                        +{latest[activeConfig.deltaKey as keyof typeof latest]?.toLocaleString(locale)}
                     </span>
                  </div>
               )}
            </div>
         </CardHeader>

         {/* Main Content Area */}
         <CardContent className="flex flex-1 flex-col min-h-0 p-0 pb-2">
            <div className="flex-1 h-full w-full">
               <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={processedData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                     <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                     <XAxis
                        dataKey="date"
                        tickLine={false}
                        axisLine={false}
                        tick={{ fontSize: 10 }}
                        tickMargin={8}
                        tickFormatter={(d) => {
                           const date = new Date(d);
                           return `${date.getMonth() + 1}/${date.getDate()}`;
                        }}
                     />
                     <YAxis
                        tickLine={false}
                        axisLine={false}
                        tick={{ fontSize: 10 }}
                        tickMargin={8}
                        tickFormatter={(value) => formatAssetTrendAxisTick(Number(value), activeAxisMaxValue, locale)}
                     />
                     <Tooltip
                        contentStyle={{
                           backgroundColor: 'var(--card)',
                           borderColor: 'var(--border)',
                           color: 'var(--foreground)',
                           borderRadius: 'var(--radius)',
                           boxShadow: 'var(--chart-tooltip-shadow)'
                        }}
                        itemStyle={{ fontFamily: 'monospace', fontWeight: 'bold', color: activeConfig.color }}
                        cursor={{ stroke: activeConfig.color, strokeWidth: 1, strokeDasharray: '4 4' }}
                        formatter={(value: number) => [value.toLocaleString(locale), activeConfig.label]}
                        labelFormatter={(label) => new Date(label).toLocaleDateString(locale)}
                     />
                     <Line
                        type="monotone"
                        dataKey={activeConfig.totalKey}
                        stroke="var(--foreground)"
                        strokeWidth={2}
                        dot={{ r: 3, fill: "var(--card)", stroke: "var(--foreground)", strokeWidth: 2 }}
                        activeDot={{ r: 6, fill: activeConfig.color, stroke: "var(--card)", strokeWidth: 2 }}
                     />
                  </LineChart>
               </ResponsiveContainer>
            </div>
         </CardContent>

         {/* Tabs Navigation (Footer) */}
         <div className="border-border border-t divide-border divide-x flex mt-auto">
            {(Object.keys(config) as Array<keyof typeof config>).map(key => {
               const conf = config[key]
               const isActive = activeTab === key

               return (
                  <button type="button"
                     key={key}
                     onClick={() => setActiveTab(key)}
                     className={cn(
                        "flex-1 pt-0.5 pb-1 px-2 flex flex-col items-center justify-end gap-0.5 transition-colors relative group hover:bg-muted/50 h-10",
                        isActive ? "bg-muted/30" : "bg-card"
                     )}
                  >
                     {isActive && (
                        <motion.div
                           layoutId="activeTabIndicator"
                           className={cn("absolute top-0 left-0 w-full h-0.5", conf.bg)}
                           transition={{ duration: 0.2, ease: "easeInOut" }}
                        />
                     )}
                     <div
                        className={cn("w-2 h-2 rounded-full", isActive ? conf.bg : "bg-muted-foreground")}
                     />
                     <span className={cn(
                        "text-[10px] font-bold uppercase tracking-wider",
                        isActive ? "text-foreground" : "text-muted-foreground"
                     )}>
                        {conf.label}
                     </span>
                  </button>
               )
            })}
         </div>
      </Card>
   )
}
