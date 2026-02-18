"use client"

import { useState, useMemo, useEffect } from "react"
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts"
import { useStatisticsHistory } from "@/hooks/use-dashboard"
import type { StatisticsHistoryItem } from "@/types/dashboard.types"
import { useTranslations } from "next-intl"
import { IconTrendingUp, IconActivity } from "@/components/icons"
import { cn } from "@/lib/utils"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { motion, useSpring, useTransform } from "framer-motion"

function NumberTicker({ value }: { value: number }) {
  // Initialize from 0 to create a "count up" effect on mount
  const spring = useSpring(0, { mass: 0.8, stiffness: 75, damping: 15 });
  const display = useTransform(spring, (current) => Math.round(current).toLocaleString());

  useEffect(() => {
    spring.set(value);
  }, [value, spring]);

  return <motion.span>{display}</motion.span>;
}

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
  const t = useTranslations("dashboard.assetTrend")
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')

  // Prepare data with deltas
  const processedData = useMemo(() => {
    if (!rawData || rawData.length === 0) return []
    const filled = fillMissingDates(rawData, 14)
    
    return filled.map((curr, i) => {
       const prev = filled[i-1] || curr
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

  // Bauhaus Colors configuration
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

  if (isLoading) {
    return (
      <Card className="w-full flex flex-col">
         <div className="bauhaus-kicker hidden [[data-theme=bauhaus]_&]:flex">
            <IconActivity className="size-4" />
            <span>ASSET PULSE</span>
         </div>
         <Skeleton className="h-[300px] m-4" />
      </Card>
    )
  }

  if (!latest) {
    return (
      <Card className="h-[300px] w-full flex items-center justify-center">
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <IconActivity className="w-8 h-8 opacity-20" />
          <span className="text-sm opacity-50">{t("noData")}</span>
        </div>
      </Card>
    )
  }

  return (
    <Card className="flex flex-col h-[300px] overflow-hidden pb-0 [[data-theme=bauhaus]_&]:pt-0 [[data-theme=bauhaus]_&]:gap-0">
       {/* Bauhaus Kicker (Consistent with other cards) */}
       <div className="bauhaus-kicker hidden [[data-theme=bauhaus]_&]:flex">
          <IconActivity className="size-4" />
          <span>ASSET PULSE // {activeConfig.label.toUpperCase()}</span>
       </div>

       <CardHeader className="pb-2 [[data-theme=bauhaus]_&]:pt-4">
          <div className="flex items-center justify-between">
             <div className="space-y-1">
                <CardTitle className="[[data-theme=bauhaus]_&]:hidden">{t("title")}</CardTitle>
                <CardDescription>{t("description")}</CardDescription>
             </div>
             {/* Mini Sparkline Logic */}
             {latest && (
                <div className="flex items-center gap-2 text-sm bg-muted/30 px-2 py-1 rounded">
                   <span className="text-muted-foreground font-mono text-xs uppercase">{t('current')}:</span>
                   <span className={cn("font-bold font-mono", activeConfig.text)}>
                      <NumberTicker value={latest[activeConfig.totalKey as keyof typeof latest] as number} />
                   </span>
                   <div className="h-4 w-[1px] bg-border mx-1"></div>
                   <span className="flex items-center gap-1 font-mono text-xs text-muted-foreground">
                      <IconTrendingUp className="w-3 h-3" />
                      +{latest[activeConfig.deltaKey as keyof typeof latest]?.toLocaleString()}
                   </span>
                </div>
             )}
          </div>
       </CardHeader>

       {/* Main Content Area */}
       <CardContent className="flex-1 flex flex-col p-0 pb-2 min-h-0">
          <div className="flex-1 w-full h-full">
             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={processedData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                   <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                   <XAxis 
                      dataKey="date" 
                      tickLine={false} 
                      axisLine={false} 
                      tick={{fontSize: 10}} 
                      tickMargin={8}
                      tickFormatter={(d) => {
                         const date = new Date(d);
                         return `${date.getMonth() + 1}/${date.getDate()}`;
                      }} 
                   />
                   <YAxis 
                      tickLine={false} 
                      axisLine={false} 
                      tick={{fontSize: 10}} 
                      tickMargin={8}
                      tickFormatter={(value) => {
                         if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
                         return value;
                      }}
                   />
                   <Tooltip 
                      contentStyle={{ 
                        backgroundColor: 'var(--card)', 
                        borderColor: 'var(--border)', 
                        color: 'var(--foreground)', 
                        borderRadius: 'var(--radius)',
                        boxShadow: '0 4px 12px rgba(0,0,0,0.1)'
                      }}
                      itemStyle={{ fontFamily: 'monospace', fontWeight: 'bold', color: activeConfig.color }}
                      cursor={{ stroke: activeConfig.color, strokeWidth: 1, strokeDasharray: '4 4' }}
                      formatter={(value: number) => [value.toLocaleString(), activeConfig.label]}
                      labelFormatter={(label) => new Date(label).toLocaleDateString()}
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
       <div className="mt-auto border-t border-border divide-x divide-border flex">
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
                        transition={{ type: "spring", stiffness: 500, damping: 30 }}
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
