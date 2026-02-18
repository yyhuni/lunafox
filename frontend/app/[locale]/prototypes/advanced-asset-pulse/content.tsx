"use client"

import React, { useState, useMemo } from "react"
import {
  LineChart, AreaChart, BarChart, ComposedChart, RadarChart, RadialBarChart, ScatterChart, Treemap,
  Line, Area, Bar, XAxis, YAxis, ZAxis, CartesianGrid, Brush, ReferenceLine, Scatter,
  Radar, RadialBar, PolarGrid, PolarAngleAxis, PolarRadiusAxis,
  Tooltip, Legend, Cell, ResponsiveContainer
} from "recharts"
import { 
  IconActivity, IconTrendingUp, IconServer, IconWorld, 
  Monitor as IconDeviceDesktop,
  IconRadar, IconScan, IconClock, Filter as IconFilter,
  IconChevronLeft, IconChevronRight
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { motion, AnimatePresence } from "framer-motion"
import { useStatisticsHistory } from "@/hooks/use-dashboard"
import type { StatisticsHistoryItem } from "@/types/dashboard.types"

// --- Shared Utilities ---

// Fill missing dates
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

// HOC for Real Data
function withRealData<P extends object>(Component: React.ComponentType<P & { data: StatisticsHistoryItem[] }>) {
  return function WithRealData(props: P) {
    const { data: rawData, isLoading } = useStatisticsHistory(14)
    const data = useMemo(() => {
        if (!rawData || rawData.length === 0) return []
        return fillMissingDates(rawData, 14)
    }, [rawData])

    if (isLoading) {
        return (
          <div className="w-full h-[300px] border border-border bg-card flex items-center justify-center">
             <div className="flex flex-col items-center gap-2 text-muted-foreground/50">
                <IconActivity className="w-6 h-6 animate-pulse" />
                <span className="text-[10px] font-mono uppercase tracking-widest">Loading Telemetry...</span>
             </div>
          </div>
        )
    }
    
    if (data.length === 0) {
       // Mock Data for Preview
       const fallbackData = []
       const now = new Date()
       for(let i=0; i<14; i++) {
           const d = new Date(now)
           d.setDate(d.getDate() - (13-i))
           fallbackData.push({
               date: d.toISOString().split('T')[0],
               totalSubdomains: Math.floor(Math.random() * 50) + 100 + (i * 5),
               totalIps: Math.floor(Math.random() * 30) + 50 + (i * 2),
               totalEndpoints: Math.floor(Math.random() * 80) + 200 + (i * 8),
               totalWebsites: Math.floor(Math.random() * 20) + 40 + i,
               totalVulns: i % 3 === 0 ? Math.floor(Math.random() * 5) + 1 : 0,
               totalTargets: 5, totalAssets: 0
           })
       }
       return <Component {...props} data={fallbackData} />
    }

    return <Component {...props} data={data} />
  }
}

// --- Variant 1: Forensic Timeline (Mixed Chart) ---
const ForensicTimeline = withRealData(({ data }) => {
  const [activeSeries, setActiveSeries] = useState<'all' | 'endpoints' | 'subdomains' | 'ips' | 'websites'>('all')
  
  // Custom Legend/Toggle Component
  const SeriesToggle = ({ id, label, color, current, onClick }: { id: string, label: string, color: string, current: string, onClick: () => void }) => (
    <button 
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 px-2 py-1 text-[10px] uppercase font-bold border transition-[color,background-color,border-color,opacity,transform,box-shadow]",
        current === id 
          ? "bg-secondary text-foreground border-border shadow-sm" 
          : "bg-transparent text-muted-foreground border-transparent hover:bg-secondary/20 hover:text-foreground"
      )}
    >
      <div className="w-2 h-2 rounded-full" style={{ backgroundColor: color }}></div>
      {label}
    </button>
  )

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
      {/* Header */}
      <div className="flex flex-col gap-2 px-4 py-3 border-b border-border bg-secondary/30">
        <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
            <IconScan className="w-4 h-4 text-primary" />
            <span className="text-xs font-bold tracking-widest text-foreground uppercase">Forensic Timeline</span>
            </div>
            <div className="text-[10px] font-mono text-muted-foreground">
                {(activeSeries === 'all' ? 'MULTI-VARIATE ANALYSIS' : activeSeries.toUpperCase() + ' TREND')}
            </div>
        </div>
        <div className="flex gap-1 flex-wrap">
           <SeriesToggle id="all" label="All" color="var(--foreground)" current={activeSeries} onClick={() => setActiveSeries('all')} />
           <SeriesToggle id="subdomains" label="Subdomains" color="var(--chart-1)" current={activeSeries} onClick={() => setActiveSeries('subdomains')} />
           <SeriesToggle id="ips" label="IPs" color="var(--chart-2)" current={activeSeries} onClick={() => setActiveSeries('ips')} />
           <SeriesToggle id="endpoints" label="URLs" color="var(--chart-3)" current={activeSeries} onClick={() => setActiveSeries('endpoints')} />
           <SeriesToggle id="websites" label="Sites" color="var(--chart-4)" current={activeSeries} onClick={() => setActiveSeries('websites')} />
        </div>
      </div>

      {/* Chart */}
      <div className="flex-1 p-4 relative">
         <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
               <defs>
                  <linearGradient id="vulnGradient" x1="0" y1="0" x2="0" y2="1">
                     <stop offset="0%" stopColor="var(--destructive)" stopOpacity={0.3}/>
                     <stop offset="100%" stopColor="var(--destructive)" stopOpacity={0.05}/>
                  </linearGradient>
               </defs>
               <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
               <XAxis 
                 dataKey="date" 
                 tickLine={false} 
                 axisLine={false} 
                 tick={{fontSize: 10, fill: 'var(--muted-foreground)', fontFamily: 'var(--font-mono)'}} 
                 tickFormatter={d => d.slice(5)} 
                 dy={10}
               />
               <YAxis 
                 yAxisId="left"
                 tickLine={false} 
                 axisLine={false} 
                 tick={{fontSize: 10, fill: 'var(--muted-foreground)', fontFamily: 'var(--font-mono)'}} 
               />
               <YAxis 
                 yAxisId="right"
                 orientation="right"
                 tickLine={false} 
                 axisLine={false} 
                 hide
               />
               <Tooltip 
                 contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                 itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                 labelStyle={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--muted-foreground)', marginBottom: 8 }}
               />
               
               {/* Background Bars: Vulnerabilities */}
               <Bar 
                 yAxisId="right"
                 dataKey="totalVulns" 
                 fill="url(#vulnGradient)" 
                 barSize={30}
                 radius={[2, 2, 0, 0]}
                 name="New Vulns"
               />

               {/* Lines */}
               {(activeSeries === 'all' || activeSeries === 'subdomains') && (
                 <Line yAxisId="left" type="monotone" dataKey="totalSubdomains" name="Subdomains" stroke="var(--chart-1)" strokeWidth={2} dot={false} activeDot={{r: 4}} />
               )}
               {(activeSeries === 'all' || activeSeries === 'ips') && (
                 <Line yAxisId="left" type="monotone" dataKey="totalIps" name="IPs" stroke="var(--chart-2)" strokeWidth={2} dot={false} activeDot={{r: 4}} />
               )}
               {(activeSeries === 'all' || activeSeries === 'endpoints') && (
                 <Line yAxisId="left" type="monotone" dataKey="totalEndpoints" name="URLs" stroke="var(--chart-3)" strokeWidth={2} dot={false} activeDot={{r: 4}} />
               )}
               {(activeSeries === 'all' || activeSeries === 'websites') && (
                 <Line yAxisId="left" type="monotone" dataKey="totalWebsites" name="Sites" stroke="var(--chart-4)" strokeWidth={2} dot={false} activeDot={{r: 4}} />
               )}
               
               {/* Event Markers */}
               {data.map((entry, index) => (
                  entry.totalVulns > 0 ? (
                    <ReferenceLine 
                      yAxisId="right"
                      key={index} 
                      x={entry.date} 
                      stroke="var(--destructive)" 
                      strokeDasharray="3 3" 
                      strokeOpacity={0.5}
                    />
                  ) : null
               ))}
            </ComposedChart>
         </ResponsiveContainer>
      </div>
    </div>
  )
})

// --- Variant 2: Interactive Scrubber (Zoom & Stacked Area) ---
const InteractiveScrubber = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconFilter className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Asset Composition</span>
          </div>
          <div className="flex items-center gap-4 text-[10px] font-mono uppercase text-muted-foreground">
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-4)]"></div>Sites</div>
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-3)]"></div>URLs</div>
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-2)]"></div>IPs</div>
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-1)]"></div>Subdomains</div>
          </div>
       </div>
       
       <div className="flex-1 px-4 pt-4 pb-0">
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={data} margin={{ top: 10, right: 0, left: -20, bottom: 0 }}>
                <defs>
                   <linearGradient id="fillChart1" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0.05}/>
                   </linearGradient>
                   <linearGradient id="fillChart2" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--chart-2)" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="var(--chart-2)" stopOpacity={0.05}/>
                   </linearGradient>
                   <linearGradient id="fillChart3" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--chart-3)" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="var(--chart-3)" stopOpacity={0.05}/>
                   </linearGradient>
                   <linearGradient id="fillChart4" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--chart-4)" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="var(--chart-4)" stopOpacity={0.05}/>
                   </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                <XAxis 
                   dataKey="date" 
                   tickLine={false} 
                   axisLine={false} 
                   tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} 
                   tickFormatter={d => d.slice(5)} 
                />
                <YAxis 
                   tickLine={false} 
                   axisLine={false} 
                   tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} 
                />
                <Tooltip 
                   cursor={{ stroke: 'var(--foreground)', strokeWidth: 1, strokeDasharray: '5 5' }}
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                />
                <Area 
                   type="monotone" 
                   dataKey="totalSubdomains" 
                   stackId="1" 
                   stroke="var(--chart-1)" 
                   fill="url(#fillChart1)" 
                   name="Subdomains"
                />
                <Area 
                   type="monotone" 
                   dataKey="totalIps" 
                   stackId="1" 
                   stroke="var(--chart-2)" 
                   fill="url(#fillChart2)" 
                   name="IPs"
                />
                <Area 
                   type="monotone" 
                   dataKey="totalEndpoints" 
                   stackId="1" 
                   stroke="var(--chart-3)" 
                   fill="url(#fillChart3)" 
                   name="URLs"
                />
                <Area 
                   type="monotone" 
                   dataKey="totalWebsites" 
                   stackId="1" 
                   stroke="var(--chart-4)" 
                   fill="url(#fillChart4)" 
                   name="Sites"
                />
                <Brush 
                   dataKey="date" 
                   height={30} 
                   stroke="var(--muted-foreground)" 
                   fill="var(--card)"
                   tickFormatter={() => ''}
                   travellerWidth={10}
                />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 3: System Breath (Pulse & Glow) ---
const SystemBreath = withRealData(({ data }) => {
  // Calculate system status based on mock vuln data
  const hasRecentVulns = data.slice(-3).some(d => d.totalVulns > 0)
  const statusColor = hasRecentVulns ? "var(--warning)" : "var(--primary)" // Amber if vulns, Blue if safe
  
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[320px] relative">
       {/* Ambient Glow Background */}
       <div 
         className="absolute inset-0 opacity-10 pointer-events-none transition-colors duration-1000"
         style={{ background: `radial-gradient(circle at 90% 10%, ${statusColor}, transparent 70%)` }}
       />
       
       <div className="flex items-center justify-between px-4 py-3 border-b border-border z-10 bg-card/50 backdrop-blur-sm">
          <div className="flex items-center gap-2">
             <IconActivity className="w-4 h-4" style={{ color: statusColor }} />
             <span className="text-xs font-bold tracking-widest uppercase" style={{ color: statusColor }}>
                System Pulse // {hasRecentVulns ? 'CAUTION' : 'STABLE'}
             </span>
          </div>
          <div className="flex items-center gap-2">
             <span className="relative flex h-2 w-2">
               <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ backgroundColor: statusColor }}></span>
               <span className="relative inline-flex rounded-full h-2 w-2" style={{ backgroundColor: statusColor }}></span>
             </span>
             <span className="text-[10px] font-mono text-muted-foreground">LIVE FEED</span>
          </div>
       </div>

       <div className="flex-1 p-4 z-10">
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis dataKey="date" hide />
                <YAxis hide domain={['auto', 'auto']} />
                <Tooltip 
                   cursor={{ stroke: statusColor, strokeWidth: 1 }}
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: statusColor, borderRadius: 0 }}
                />
                <Line 
                   type="monotone" 
                   dataKey="totalEndpoints" 
                   stroke={statusColor} 
                   strokeWidth={3} 
                   dot={false}
                   strokeLinecap="round"
                   style={{ filter: `drop-shadow(0 0 8px ${statusColor})` }}
                />
                <Line 
                   type="monotone" 
                   dataKey="totalIps" 
                   stroke={statusColor} 
                   strokeWidth={1} 
                   strokeDasharray="4 4"
                   strokeOpacity={0.5}
                   dot={false}
                />
             </LineChart>
          </ResponsiveContainer>
          
          {/* Decorative overlay lines */}
          <div className="absolute inset-0 pointer-events-none border-[0.5px] border-transparent border-t-white/5 bg-[linear-gradient(transparent_50%,rgba(0,0,0,0.1)_50%)] bg-[size:100%_4px] opacity-20"></div>
       </div>
       
       {/* Bottom Status Bar */}
       <div className="border-t border-border p-2 px-4 flex justify-between items-center text-[10px] font-mono text-muted-foreground bg-secondary/20">
          <span>LATENCY: 24ms</span>
          <span>UPTIME: 99.9%</span>
          <span>WORKERS: 4/4</span>
       </div>
    </div>
  )
})

// --- Variant 4: Asset Radar Scan (Composition) ---
const AssetRadar = withRealData(({ data }) => {
  // Aggregate data for the radar chart (using latest snapshot or average)
  const latestData = data[data.length - 1] || { totalSubdomains: 0, totalIps: 0, totalEndpoints: 0, totalWebsites: 0 }
  
  // Normalize data for visualization balance (logarithmic scale simulation or simple ratio)
  // In a real app, you might want to normalize relative to a baseline or max capacity
  const radarData = [
    { subject: 'Subdomains', A: latestData.totalSubdomains, fullMark: 150 },
    { subject: 'IPs', A: latestData.totalIps, fullMark: 150 },
    { subject: 'URLs', A: latestData.totalEndpoints, fullMark: 150 },
    { subject: 'Sites', A: latestData.totalWebsites, fullMark: 150 },
  ]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px] relative">
      <div className="absolute top-0 left-0 w-full h-1 bg-primary/20 overflow-hidden">
         <div className="h-full w-1/3 bg-primary animate-border-flow"></div>
      </div>
      
      <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
         <div className="flex items-center gap-2">
             <IconRadar className="w-4 h-4 text-primary animate-spin-slow" style={{ animationDuration: '4s' }} />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Asset Radar // Signature</span>
         </div>
         <span className="text-[10px] font-mono text-muted-foreground">SNAPSHOT: {new Date().toLocaleDateString()}</span>
      </div>

      <div className="flex-1 relative flex items-center justify-center p-4">
          <ResponsiveContainer width="100%" height="100%">
            <RadarChart cx="50%" cy="50%" outerRadius="80%" data={radarData}>
              <PolarGrid stroke="var(--border)" strokeDasharray="3 3" />
              <PolarAngleAxis dataKey="subject" tick={{ fill: 'var(--muted-foreground)', fontSize: 10, fontWeight: 'bold' }} />
              <PolarRadiusAxis angle={30} domain={[0, 'auto']} tick={false} axisLine={false} />
              <Radar
                name="Asset Count"
                dataKey="A"
                stroke="var(--primary)"
                strokeWidth={2}
                fill="var(--primary)"
                fillOpacity={0.2}
              />
              <Tooltip 
                 contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                 itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
              />
            </RadarChart>
          </ResponsiveContainer>

          {/* Scanning Effect Overlay */}
          <div className="absolute inset-0 pointer-events-none rounded-full overflow-hidden flex items-center justify-center opacity-20">
             <div className="w-[80%] h-[80%] rounded-full border border-primary/30"></div>
             <div className="w-[60%] h-[60%] rounded-full border border-primary/20 absolute"></div>
             <div className="w-full h-1/2 absolute top-0 bg-gradient-to-b from-primary/10 to-transparent animate-scan-down"></div>
          </div>
      </div>
    </div>
  )
})

// --- Variant 5: Metric Seismograph (Ridgeline) ---
const MetricSeismograph = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[400px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconActivity className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Metric Seismograph</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">SYNCED TIME SERIES</span>
       </div>
       
       <div className="flex-1 flex flex-col divide-y divide-border/50">
          {[
            { key: 'totalSubdomains', name: 'Subdomains', color: 'var(--chart-1)' },
            { key: 'totalIps', name: 'IPs', color: 'var(--chart-2)' },
            { key: 'totalEndpoints', name: 'URLs', color: 'var(--chart-3)' },
            { key: 'totalWebsites', name: 'Sites', color: 'var(--chart-4)' },
          ].map((metric, i) => (
             <div key={metric.key} className="flex-1 flex items-center px-4 relative group">
                <div className="w-24 text-[10px] font-mono font-bold uppercase text-muted-foreground flex flex-col justify-center">
                   <span style={{ color: metric.color }}>{metric.name}</span>
                   <span className="opacity-50">{data[data.length-1]?.[metric.key as keyof typeof data[0]]}</span>
                </div>
                <div className="flex-1 h-full">
                   <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={data} syncId="seismograph">
                         <defs>
                            <linearGradient id={`grad-${i}`} x1="0" y1="0" x2="0" y2="1">
                               <stop offset="0%" stopColor={metric.color} stopOpacity={0.2}/>
                               <stop offset="100%" stopColor={metric.color} stopOpacity={0}/>
                            </linearGradient>
                         </defs>
                         <Tooltip 
                            contentStyle={{ display: 'none' }} // Hide default tooltip, use cursor only or custom
                            cursor={{ stroke: 'var(--foreground)', strokeWidth: 1 }}
                         />
                         <Area 
                            type="monotone" 
                            dataKey={metric.key} 
                            stroke={metric.color} 
                            fill={`url(#grad-${i})`} 
                            strokeWidth={1.5}
                            isAnimationActive={false}
                         />
                      </AreaChart>
                   </ResponsiveContainer>
                </div>
             </div>
          ))}
       </div>
    </div>
  )
})

// --- Variant 6: Digital Mosaic (Heatmap) ---
const DigitalMosaic = withRealData(({ data }) => {
  // Normalize values for opacity (0.1 to 1.0)
  const getOpacity = (val: number, max: number) => {
     return Math.max(0.1, Math.min(1, val / (max || 1)))
  }

  // Find max values for normalization
  const maxVals = useMemo(() => ({
     sub: Math.max(...data.map(d => d.totalSubdomains)),
     ip: Math.max(...data.map(d => d.totalIps)),
     url: Math.max(...data.map(d => d.totalEndpoints)),
     site: Math.max(...data.map(d => d.totalWebsites)),
  }), [data])

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconServer className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Digital Mosaic</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">ACTIVITY DENSITY</span>
       </div>
       
       <div className="p-6 overflow-x-auto">
          <div className="grid gap-1 min-w-[600px]" style={{ gridTemplateColumns: `auto repeat(${data.length}, 1fr)` }}>
             {/* Header Row (Dates) */}
             <div className="h-6"></div>
             {data.map((d, i) => (
                <div key={i} className="flex items-end justify-center pb-1">
                   <span className="text-[9px] font-mono text-muted-foreground rotate-[-45deg] origin-bottom-left whitespace-nowrap translate-x-2">
                      {d.date.slice(5)}
                   </span>
                </div>
             ))}

             {/* Rows */}
             {[
               { key: 'totalSubdomains', label: 'SUB', color: 'var(--chart-1)', max: maxVals.sub },
               { key: 'totalIps', label: 'IP', color: 'var(--chart-2)', max: maxVals.ip },
               { key: 'totalEndpoints', label: 'URL', color: 'var(--chart-3)', max: maxVals.url },
               { key: 'totalWebsites', label: 'WEB', color: 'var(--chart-4)', max: maxVals.site },
             ].map(row => (
                <React.Fragment key={row.key}>
                   <div className="flex items-center justify-end pr-2 h-10">
                      <span className="text-[10px] font-bold font-mono text-muted-foreground">{row.label}</span>
                   </div>
                   {data.map((d, i) => {
                      const val = d[row.key as keyof typeof d] as number
                      const opacity = getOpacity(val, row.max)
                      return (
                        <div key={i} className="h-10 relative group bg-secondary/20 border border-transparent hover:border-foreground/50 transition-colors">
                           <div 
                              className="absolute inset-[1px]" 
                              style={{ backgroundColor: row.color, opacity: opacity }}
                           ></div>
                           {/* Tooltip */}
                           <div className="opacity-0 group-hover:opacity-100 absolute bottom-full left-1/2 -translate-x-1/2 mb-1 bg-popover text-popover-foreground text-[10px] px-2 py-1 rounded border border-border pointer-events-none whitespace-nowrap z-10 shadow-lg">
                              {val}
                           </div>
                        </div>
                      )
                   })}
                </React.Fragment>
             ))}
          </div>
       </div>
    </div>
  )
})

// --- Variant 7: Orbital Gauge (Radial Bar) ---
const OrbitalGauge = withRealData(({ data }) => {
  const latestData = data[data.length - 1] || {}
  
  // Transform for RadialBarChart
  const gaugeData = [
    { name: 'Sites', count: latestData.totalWebsites || 0, fill: 'var(--chart-4)' },
    { name: 'URLs', count: latestData.totalEndpoints || 0, fill: 'var(--chart-3)' },
    { name: 'IPs', count: latestData.totalIps || 0, fill: 'var(--chart-2)' },
    { name: 'Subdomains', count: latestData.totalSubdomains || 0, fill: 'var(--chart-1)' },
  ]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconClock className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Orbital Gauge</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">ASSET VOLUME</span>
       </div>
       
       <div className="flex-1 p-4 relative">
          <ResponsiveContainer width="100%" height="100%">
             <RadialBarChart 
                cx="50%" 
                cy="50%" 
                innerRadius="20%" 
                outerRadius="90%" 
                barSize={15} 
                data={gaugeData}
                startAngle={90}
                endAngle={-270}
             >
                <RadialBar
                   label={{ position: 'insideStart', fill: '#fff', fontSize: 10, fontWeight: 'bold' }}
                   background={{ fill: 'var(--muted)', opacity: 0.2 }}
                   dataKey="count"
                   cornerRadius={2}
                />
                <Legend 
                   iconSize={10} 
                   layout="vertical" 
                   verticalAlign="middle" 
                   align="right"
                   wrapperStyle={{ fontFamily: 'var(--font-mono)', fontSize: '11px', textTransform: 'uppercase' }}
                />
                <Tooltip 
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                   itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                />
             </RadialBarChart>
          </ResponsiveContainer>
          
          {/* Center Info */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
             <div className="flex flex-col items-center">
                <span className="text-2xl font-bold font-mono tracking-tighter">
                   {(latestData.totalAssets || 0).toLocaleString()}
                </span>
                <span className="text-[9px] text-muted-foreground uppercase tracking-widest">Total Assets</span>
             </div>
          </div>
       </div>
    </div>
  )
})

// --- Variant 8: DNA Barcode (Stacked Bar 100%) ---
const DnaBarcode = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[280px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconScan className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">DNA Barcode</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">COMPOSITION RATIO</span>
       </div>
       
       <div className="flex-1 p-4 px-6 flex flex-col justify-center">
          <ResponsiveContainer width="100%" height={160}>
             <BarChart data={data} layout="horizontal" barCategoryGap={1}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.3} />
                <XAxis 
                   dataKey="date" 
                   tickLine={false} 
                   axisLine={false} 
                   tick={{fontSize: 9, fill: 'var(--muted-foreground)', fontFamily: 'var(--font-mono)'}} 
                   tickFormatter={d => d.slice(5)}
                   dy={10}
                />
                <Tooltip 
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                   itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                   formatter={(value, name) => [value, name]}
                />
                <Bar dataKey="totalSubdomains" stackId="a" fill="var(--chart-1)" radius={[0,0,0,0]} />
                <Bar dataKey="totalIps" stackId="a" fill="var(--chart-2)" radius={[0,0,0,0]} />
                <Bar dataKey="totalEndpoints" stackId="a" fill="var(--chart-3)" radius={[0,0,0,0]} />
                <Bar dataKey="totalWebsites" stackId="a" fill="var(--chart-4)" radius={[0,0,0,0]} />
             </BarChart>
          </ResponsiveContainer>
          
          <div className="flex justify-between mt-4 text-[10px] font-mono text-muted-foreground uppercase px-2">
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-1)]"></div> Subdomains</div>
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-2)]"></div> IPs</div>
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-3)]"></div> URLs</div>
             <div className="flex items-center gap-1"><div className="w-2 h-2 bg-[var(--chart-4)]"></div> Sites</div>
          </div>
       </div>
    </div>
  )
})

// --- Variant 9: Flux Stream (Silhouette) ---
const FluxStream = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconActivity className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Flux Stream</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">ORGANIC GROWTH</span>
       </div>
       
       <div className="flex-1 p-4 relative">
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={data} stackOffset="silhouette" margin={{top: 10, right: 0, left: -20, bottom: 0}}>
                <defs>
                   <linearGradient id="flux1" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={0.8}/>
                      <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0}/>
                   </linearGradient>
                   <linearGradient id="flux2" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--chart-2)" stopOpacity={0.8}/>
                      <stop offset="100%" stopColor="var(--chart-2)" stopOpacity={0}/>
                   </linearGradient>
                   <linearGradient id="flux3" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--chart-3)" stopOpacity={0.8}/>
                      <stop offset="100%" stopColor="var(--chart-3)" stopOpacity={0}/>
                   </linearGradient>
                   <linearGradient id="flux4" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--chart-4)" stopOpacity={0.8}/>
                      <stop offset="100%" stopColor="var(--chart-4)" stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.2} />
                <XAxis 
                   dataKey="date" 
                   tickLine={false} 
                   axisLine={false} 
                   tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} 
                   tickFormatter={d => d.slice(5)} 
                />
                <Tooltip 
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                   itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                />
                <Area type="monotone" dataKey="totalSubdomains" stackId="1" stroke="var(--chart-1)" fill="url(#flux1)" />
                <Area type="monotone" dataKey="totalIps" stackId="1" stroke="var(--chart-2)" fill="url(#flux2)" />
                <Area type="monotone" dataKey="totalEndpoints" stackId="1" stroke="var(--chart-3)" fill="url(#flux3)" />
                <Area type="monotone" dataKey="totalWebsites" stackId="1" stroke="var(--chart-4)" fill="url(#flux4)" />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 10: Binary Punchcard (Scatter) ---
const BinaryPunchcard = withRealData(({ data }) => {
  type ScatterPoint = { x: string; y: number; z: number; type: string; fill: string }
  // Transform data for scatter plot: flatten to array of { date, type, value, index }
  const scatterData = useMemo(() => {
     const result: ScatterPoint[] = []
     data.forEach((d) => {
        result.push({ x: d.date, y: 1, z: d.totalSubdomains, type: 'Subdomains', fill: 'var(--chart-1)' })
        result.push({ x: d.date, y: 2, z: d.totalIps, type: 'IPs', fill: 'var(--chart-2)' })
        result.push({ x: d.date, y: 3, z: d.totalEndpoints, type: 'URLs', fill: 'var(--chart-3)' })
        result.push({ x: d.date, y: 4, z: d.totalWebsites, type: 'Sites', fill: 'var(--chart-4)' })
     })
     return result
  }, [data])

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconServer className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Binary Punchcard</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">EVENT DENSITY</span>
       </div>
       
       <div className="flex-1 p-4 relative">
          <ResponsiveContainer width="100%" height="100%">
             <ScatterChart margin={{top: 20, right: 20, bottom: 20, left: 20}}>
                <CartesianGrid strokeDasharray="3 3" horizontal={true} vertical={true} stroke="var(--border)" strokeOpacity={0.3} />
                <XAxis 
                   dataKey="x" 
                   type="category" 
                   tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} 
                   tickFormatter={d => d.slice(5)} 
                   interval={0}
                   axisLine={false}
                   tickLine={false}
                />
                <YAxis 
                   dataKey="y" 
                   type="number" 
                   domain={[0, 5]} 
                   tickCount={6} 
                   tickFormatter={(val) => {
                      if (val === 1) return 'Sub'
                      if (val === 2) return 'IP'
                      if (val === 3) return 'URL'
                      if (val === 4) return 'Site'
                      return ''
                   }}
                   tick={{fontSize: 10, fill: 'var(--muted-foreground)', fontWeight: 'bold'}}
                   axisLine={false}
                   tickLine={false}
                   width={40}
                />
                <ZAxis type="number" dataKey="z" range={[50, 400]} />
                <Tooltip 
                   cursor={{ strokeDasharray: '3 3' }} 
                   content={({ active, payload }) => {
                      if (active && payload && payload.length) {
                         const data = payload[0].payload
                         return (
                            <div className="bg-popover text-popover-foreground border border-border p-2 text-xs shadow-lg">
                               <div className="font-bold">{data.type}</div>
                               <div>Date: {data.x}</div>
                               <div>Count: {data.z}</div>
                            </div>
                         )
                      }
                      return null
                   }}
                />
                <Scatter data={scatterData} shape="circle">
                   {scatterData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                   ))}
                </Scatter>
             </ScatterChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 11: Circuit Treemap ---
const CircuitTreemap = withRealData(({ data }) => {
  const latestData = data[data.length - 1] || {}
  
  const treeData = [
     { name: 'Subdomains', size: latestData.totalSubdomains || 10, fill: 'var(--chart-1)' },
     { name: 'IPs', size: latestData.totalIps || 10, fill: 'var(--chart-2)' },
     { name: 'URLs', size: latestData.totalEndpoints || 10, fill: 'var(--chart-3)' },
     { name: 'Sites', size: latestData.totalWebsites || 10, fill: 'var(--chart-4)' },
  ]

  // Custom Content for Treemap Node
  type TreemapNodeProps = {
    x?: number
    y?: number
    width?: number
    height?: number
    index?: number
    payload?: { fill?: string; name?: string; size?: number }
    colors?: string[]
    name?: string
  }
  const CustomizedContent = ({ x = 0, y = 0, width = 0, height = 0, index = 0, payload, colors, name }: TreemapNodeProps) => {
    
    // Safety check for payload and depth
    // In this flat structure, depth 1 are the data items. Depth 0 is root.
    // We try to safely access fill, falling back to color by index or default.
    const fill = payload?.fill || (colors && colors[index]) || 'var(--primary)';
    
    return (
      <g>
        <rect
          x={x}
          y={y}
          width={width}
          height={height}
          style={{
            fill: fill,
            stroke: 'var(--card)',
            strokeWidth: 2,
            opacity: 0.8,
          }}
        />
        {width > 50 && height > 30 && (
           <text x={x + width / 2} y={y + height / 2} textAnchor="middle" fill="#fff" fontSize={12} dy={-6} style={{ fontWeight: 'bold', textShadow: '0 1px 2px rgba(0,0,0,0.5)' }}>
             {name || (payload?.name)}
           </text>
        )}
        {width > 50 && height > 30 && (
           <text x={x + width / 2} y={y + height / 2} textAnchor="middle" fill="rgba(255,255,255,0.8)" fontSize={10} dy={10} fontFamily="var(--font-mono)">
             {payload?.size}
           </text>
        )}
      </g>
    );
  };

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconFilter className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Circuit Treemap</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">SIZE HIERARCHY</span>
       </div>
       
       <div className="flex-1 p-4 relative">
          <ResponsiveContainer width="100%" height="100%">
             <Treemap
                data={treeData}
                dataKey="size"
                aspectRatio={4 / 3}
                stroke="#fff"
                content={<CustomizedContent />}
             >
                <Tooltip 
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                   itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                />
             </Treemap>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 12: Terminal Monitor (ASCII/Text) ---
const TerminalMonitor = withRealData(({ data }) => {
  const latestData = data[data.length - 1] || {}
  
  // Calculate max for bar scaling
  const maxVal = Math.max(latestData.totalSubdomains || 0, latestData.totalIps || 0, latestData.totalEndpoints || 0, latestData.totalWebsites || 0, 1)
  
  const metrics = [
     { label: 'SUBDOMAINS', val: latestData.totalSubdomains || 0, color: 'var(--chart-1)' },
     { label: 'IP ADDRESSES', val: latestData.totalIps || 0, color: 'var(--chart-2)' },
     { label: 'URL ENDPOINTS', val: latestData.totalEndpoints || 0, color: 'var(--chart-3)' },
     { label: 'WEB SITES', val: latestData.totalWebsites || 0, color: 'var(--chart-4)' },
  ]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[320px] font-mono text-xs">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-black text-green-500">
          <div className="flex items-center gap-2">
             <span className="animate-blink">_</span>
             <span className="font-bold tracking-widest uppercase">TERMINAL.MONITOR.EXE</span>
          </div>
          <span className="opacity-70">PID: {Math.floor(Math.random() * 9000) + 1000}</span>
       </div>
       
       <div className="flex-1 p-6 bg-black text-white/80 overflow-y-auto space-y-6">
          <div className="border-b border-white/20 pb-2 mb-4 flex justify-between">
             <span>SYSTEM_STATUS: <span className="text-green-500">ONLINE</span></span>
             <span>UPTIME: 14d 2h 12m</span>
          </div>

          <div className="space-y-4">
             {metrics.map((m, i) => {
                const widthPercent = (m.val / maxVal) * 100
                const barChars = Math.floor(widthPercent / 2.5) // 40 chars max approx
                const barString = "█".repeat(barChars)
                
                return (
                   <div key={i} className="space-y-1">
                      <div className="flex justify-between text-[10px] text-white/50">
                         <span>{m.label}</span>
                         <span>[{m.val}]</span>
                      </div>
                      <div className="flex items-center gap-2 text-sm">
                         <span style={{ color: m.color }}>{barString}</span>
                         <span className="text-white/30 text-[10px]">{widthPercent.toFixed(1)}%</span>
                      </div>
                   </div>
                )
             })}
          </div>

          <div className="pt-4 text-white/40 text-[10px]">
             &gt; scanning assets...<br/>
             &gt; updating indexes...<br/>
             &gt; _
          </div>
       </div>
    </div>
  )
})

// --- Variant 13: Neon Equalizer (Mirrored Bar) ---
const NeonEqualizer = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-2">
             <IconActivity className="w-4 h-4 text-muted-foreground" />
             <span className="text-xs font-bold tracking-widest text-foreground uppercase">Neon Equalizer</span>
          </div>
          <span className="text-[10px] font-mono text-muted-foreground">SPECTRUM ANALYSIS</span>
       </div>
       
       <div className="flex-1 p-4 relative flex items-center bg-black/5">
          <ResponsiveContainer width="100%" height="100%">
             <BarChart data={data} barCategoryGap={2}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.1} />
                <Tooltip 
                   cursor={{ fill: 'var(--primary)', opacity: 0.1 }}
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                   itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                />
                {/* Mirrored effect visually using Stacked bars but effectively distinct */}
                <Bar dataKey="totalSubdomains" stackId="a" fill="var(--chart-1)" radius={[2, 2, 0, 0]} />
                <Bar dataKey="totalIps" stackId="a" fill="var(--chart-2)" radius={[2, 2, 0, 0]} />
                <Bar dataKey="totalEndpoints" stackId="a" fill="var(--chart-3)" radius={[2, 2, 0, 0]} />
                <Bar dataKey="totalWebsites" stackId="a" fill="var(--chart-4)" radius={[2, 2, 0, 0]} />
             </BarChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 14: Incremental Pulse Grid (The "Growth" View) ---
const IncrementalPulseGrid = withRealData(({ data }) => {
  // Process data to calculate Deltas
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        if (i === 0) return { ...curr, newSub: 0, newIp: 0, newUrl: 0, newSite: 0 }
        const prev = data[i-1]
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     }).slice(1) // Remove first day as it has no delta context
  }, [data])

  const latest = data[data.length - 1] || {}
  const latestDelta = processedData[processedData.length - 1] || { newSub: 0, newIp: 0, newUrl: 0, newSite: 0 }

  const MetricCard = ({ label, total, delta, dataKey, color }: { label: string, total: number, delta: number, dataKey: string, color: string }) => (
     <div className="border border-border bg-card p-4 flex flex-col justify-between h-[160px] relative overflow-hidden group hover:border-foreground/20 transition-colors">
        <div className="flex justify-between items-start z-10">
           <div className="flex flex-col">
              <span className="text-[10px] font-mono font-bold uppercase text-muted-foreground tracking-wider">{label}</span>
              <span className="text-2xl font-bold tracking-tight mt-1">{total.toLocaleString()}</span>
           </div>
           <div className={cn("text-xs font-mono font-bold px-1.5 py-0.5 border flex items-center gap-1", delta > 0 ? "bg-primary/10 text-primary border-primary/20" : "bg-muted text-muted-foreground border-border")}>
              {delta > 0 ? '+' : ''}{delta}
              {delta > 0 && <IconTrendingUp className="w-3 h-3" />}
           </div>
        </div>

        {/* Micro Chart */}
        <div className="h-[60px] mt-auto -mx-4 -mb-4 w-[calc(100%+32px)]">
           <ResponsiveContainer width="100%" height="100%">
              <BarChart data={processedData}>
                 <Bar dataKey={dataKey} fill={color} radius={[1, 1, 0, 0]} barSize={8} />
                 <Tooltip 
                    cursor={{ fill: 'var(--muted)', opacity: 0.1 }}
                    content={({ active, payload }) => {
                       if (active && payload && payload.length) {
                          return (
                             <div className="bg-popover text-popover-foreground text-[10px] px-2 py-1 border border-border shadow-sm font-mono">
                                +{payload[0].value}
                             </div>
                          )
                       }
                       return null
                    }}
                 />
              </BarChart>
           </ResponsiveContainer>
        </div>
     </div>
  )

  return (
    <div className="w-full">
       <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricCard 
             label="Subdomains" 
             total={latest.totalSubdomains || 0} 
             delta={latestDelta.newSub} 
             dataKey="newSub" 
             color="var(--chart-1)" 
          />
          <MetricCard 
             label="IP Addresses" 
             total={latest.totalIps || 0} 
             delta={latestDelta.newIp} 
             dataKey="newIp" 
             color="var(--chart-2)" 
          />
          <MetricCard 
             label="URL Endpoints" 
             total={latest.totalEndpoints || 0} 
             delta={latestDelta.newUrl} 
             dataKey="newUrl" 
             color="var(--chart-3)" 
          />
          <MetricCard 
             label="Web Sites" 
             total={latest.totalWebsites || 0} 
             delta={latestDelta.newSite} 
             dataKey="newSite" 
             color="var(--chart-4)" 
          />
       </div>
    </div>
  )
})

// --- Variant 15: Master Control Panel (Total + Trend) ---
const MasterControlPanel = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  // Calculate Deltas on the fly
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])

  const latest = processedData[processedData.length - 1] || {}
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: 'var(--chart-4)' },
  }

  const activeConfig = config[activeTab]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col md:flex-row h-[400px]">
       {/* Sidebar Selector */}
       <div className="w-full md:w-[240px] border-b md:border-b-0 md:border-r border-border flex flex-col bg-secondary/10">
          <div className="p-3 border-b border-border">
             <span className="text-[10px] font-mono font-bold uppercase text-muted-foreground tracking-widest">ASSET SELECTOR</span>
          </div>
          <div className="flex-1 overflow-y-auto">
             {(Object.keys(config) as Array<keyof typeof config>).map(key => {
                const conf = config[key]
                const total = latest[conf.totalKey as keyof typeof latest] as number
                const delta = latest[conf.deltaKey as keyof typeof latest] as number
                
                return (
                   <button
                      key={key}
                      onClick={() => setActiveTab(key)}
                      className={cn(
                         "w-full text-left p-4 border-b border-border transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:bg-secondary/20 relative",
                         activeTab === key ? "bg-card" : "text-muted-foreground"
                      )}
                   >
                      {activeTab === key && (
                         <div className="absolute left-0 top-0 bottom-0 w-[3px]" style={{ backgroundColor: conf.color }}></div>
                      )}
                      <div className="text-xs font-bold uppercase mb-1">{conf.label}</div>
                      <div className="flex justify-between items-end">
                         <span className={cn("text-lg font-mono tracking-tight", activeTab === key ? "text-foreground" : "")}>
                            {total?.toLocaleString()}
                         </span>
                         {delta > 0 && (
                            <span className="text-[10px] text-green-500 font-mono">+{delta}</span>
                         )}
                      </div>
                   </button>
                )
             })}
          </div>
       </div>

       {/* Main Chart Area */}
       <div className="flex-1 flex flex-col min-w-0">
          <div className="p-4 border-b border-border flex justify-between items-center bg-card">
             <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full" style={{ backgroundColor: activeConfig.color }}></div>
                <span className="text-sm font-bold tracking-tight">{activeConfig.label} Trend Analysis</span>
             </div>
             <div className="flex gap-4 text-[10px] font-mono text-muted-foreground uppercase">
                <div className="flex items-center gap-1">
                   <div className="w-3 h-1" style={{ backgroundColor: activeConfig.color }}></div>
                   <span>Total Volume</span>
                </div>
                <div className="flex items-center gap-1">
                   <div className="w-2 h-2 rounded-[1px] opacity-30" style={{ backgroundColor: activeConfig.color }}></div>
                   <span>Daily New</span>
                </div>
             </div>
          </div>
          
          <div className="flex-1 p-4 relative">
             <ResponsiveContainer width="100%" height="100%">
                <ComposedChart data={processedData}>
                   <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                   <XAxis 
                      dataKey="date" 
                      tickLine={false} 
                      axisLine={false} 
                      tick={{fontSize: 10, fill: 'var(--muted-foreground)', fontFamily: 'var(--font-mono)'}} 
                      tickFormatter={d => d.slice(5)} 
                      dy={10}
                   />
                   <YAxis 
                      yAxisId="left"
                      tickLine={false} 
                      axisLine={false} 
                      tick={{fontSize: 10, fill: 'var(--muted-foreground)', fontFamily: 'var(--font-mono)'}} 
                   />
                   <YAxis 
                      yAxisId="right"
                      orientation="right"
                      tickLine={false} 
                      axisLine={false} 
                      tick={{fontSize: 10, fill: 'var(--muted-foreground)', fontFamily: 'var(--font-mono)'}} 
                   />
                   <Tooltip 
                      contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }}
                      itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}
                      labelStyle={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--muted-foreground)', marginBottom: 8 }}
                   />
                   
                   {/* Daily New (Bar) */}
                   <Bar 
                      yAxisId="right"
                      dataKey={activeConfig.deltaKey} 
                      fill={activeConfig.color} 
                      opacity={0.3}
                      barSize={20}
                      radius={[2, 2, 0, 0]}
                      name="Daily New"
                   />

                   {/* Total Volume (Line) */}
                   <Line 
                      yAxisId="left"
                      type="monotone" 
                      dataKey={activeConfig.totalKey} 
                      stroke={activeConfig.color} 
                      strokeWidth={2} 
                      dot={false}
                      activeDot={{r: 5, fill: activeConfig.color}}
                      name="Total Volume"
                   />
                </ComposedChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 16: Command Deck (Top Tabs) ---
const CommandDeck = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')

  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[420px]">
       {/* Top Deck */}
       <div className="flex border-b border-border bg-secondary/10 overflow-x-auto">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number
             const delta = latest[conf.deltaKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 min-w-[140px] p-4 border-r border-border transition-[color,background-color,border-color,opacity,transform,box-shadow] relative group text-left",
                      isActive ? "bg-card shadow-[inset_0_2px_0_0_currentColor]" : "hover:bg-card/50 text-muted-foreground"
                   )}
                   style={{ color: isActive ? conf.color : undefined }}
                >
                   <div className="text-[10px] font-bold uppercase tracking-widest mb-1 opacity-70">{conf.label}</div>
                   <div className="flex items-baseline gap-2">
                      <span className={cn("text-xl font-mono tracking-tight", isActive ? "text-foreground" : "")}>{total?.toLocaleString()}</span>
                      {delta > 0 && <span className="text-[10px] font-mono text-green-500">+{delta}</span>}
                   </div>
                   {isActive && (
                      <div className="absolute bottom-0 left-0 right-0 h-[2px]" style={{ backgroundColor: conf.color }}></div>
                   )}
                </button>
             )
          })}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6 relative">
          <ResponsiveContainer width="100%" height="100%">
             <ComposedChart data={processedData} margin={{top: 10, right: 10, left: 0, bottom: 0}}>
                <defs>
                   <linearGradient id={`deckGrad-${activeTab}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.2}/>
                      <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis yAxisId="left" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }} />
                
                <Area yAxisId="left" type="monotone" dataKey={activeConfig.totalKey} stroke={activeConfig.color} fill={`url(#deckGrad-${activeTab})`} strokeWidth={2} name="Total Volume" />
                <Bar yAxisId="right" dataKey={activeConfig.deltaKey} fill={activeConfig.color} opacity={0.5} barSize={20} radius={[2, 2, 0, 0]} name="Daily New" />
             </ComposedChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 17: Swimlane Monitor (All-in-One) ---
const SwimlaneMonitor = withRealData(({ data }) => {
   // Process Deltas
   const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const items = [
     { key: 'sub', label: 'SUBDOMAINS', total: latest.totalSubdomains, delta: latest.newSub, color: 'var(--chart-1)', totalKey: 'totalSubdomains', deltaKey: 'newSub' },
     { key: 'ip', label: 'IP ADDRESSES', total: latest.totalIps, delta: latest.newIp, color: 'var(--chart-2)', totalKey: 'totalIps', deltaKey: 'newIp' },
     { key: 'url', label: 'URL ENDPOINTS', total: latest.totalEndpoints, delta: latest.newUrl, color: 'var(--chart-3)', totalKey: 'totalEndpoints', deltaKey: 'newUrl' },
     { key: 'site', label: 'WEBSITES', total: latest.totalWebsites, delta: latest.newSite, color: 'var(--chart-4)', totalKey: 'totalWebsites', deltaKey: 'newSite' },
  ]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col">
       {items.map((item, index) => (
          <div key={item.key} className={cn("flex flex-col md:flex-row h-[100px] border-b border-border last:border-0", index % 2 === 0 ? "bg-card" : "bg-card/50")}>
             {/* Left Stats */}
             <div className="w-full md:w-[180px] p-4 flex flex-col justify-center border-b md:border-b-0 md:border-r border-border bg-secondary/5">
                <div className="flex items-center gap-2 mb-1">
                   <div className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: item.color }}></div>
                   <span className="text-[10px] font-bold uppercase text-muted-foreground">{item.label}</span>
                </div>
                <div className="flex items-baseline gap-2">
                   <span className="text-xl font-mono tracking-tight font-medium">{item.total?.toLocaleString()}</span>
                   {item.delta > 0 && <span className="text-[10px] font-mono text-green-500">+{item.delta}</span>}
                </div>
             </div>
             
             {/* Right Chart */}
             <div className="flex-1 p-2 relative">
                <ResponsiveContainer width="100%" height="100%">
                   <ComposedChart data={processedData}>
                      <Tooltip contentStyle={{ display: 'none' }} cursor={{ stroke: 'var(--foreground)', strokeWidth: 1 }} />
                      <Bar dataKey={item.deltaKey} fill={item.color} opacity={0.2} barSize={10} radius={[2, 2, 0, 0]} />
                      <Line type="monotone" dataKey={item.totalKey} stroke={item.color} strokeWidth={2} dot={false} />
                   </ComposedChart>
                </ResponsiveContainer>
                
                {/* Overlay Delta for latest (Optional Sparkline feel) */}
                <div className="absolute right-4 top-1/2 -translate-y-1/2 pointer-events-none opacity-20">
                   <IconActivity className="w-8 h-8" style={{ color: item.color }} />
                </div>
             </div>
          </div>
       ))}
    </div>
  )
})

// --- Variant 18: Rich List Console (Sidebar Sparklines) ---
const RichListConsole = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('ip')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col md:flex-row h-[450px]">
       {/* Sidebar */}
       <div className="w-full md:w-[260px] border-r border-border bg-secondary/5 flex flex-col">
          <div className="p-3 border-b border-border bg-muted/20">
             <span className="text-[10px] font-mono font-bold uppercase text-muted-foreground">ASSET_INDEX</span>
          </div>
          <div className="flex-1 overflow-y-auto">
             {(Object.keys(config) as Array<keyof typeof config>).map(key => {
                const conf = config[key]
                const isActive = activeTab === key
                const total = latest[conf.totalKey as keyof typeof latest] as number
                
                return (
                   <button
                      key={key}
                      onClick={() => setActiveTab(key)}
                      className={cn(
                         "w-full text-left p-3 border-b border-border transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:bg-card/80 group h-[72px] flex items-center justify-between",
                         isActive ? "bg-card border-l-4 border-l-[var(--primary)]" : "border-l-4 border-l-transparent text-muted-foreground"
                      )}
                   >
                      <div className="flex flex-col">
                         <span className="text-[10px] font-bold uppercase">{conf.label}</span>
                         <span className={cn("text-sm font-mono", isActive ? "text-foreground" : "")}>{total?.toLocaleString()}</span>
                      </div>
                      
                      {/* Mini Sparkline in Sidebar */}
                      <div className="w-[80px] h-[30px]">
                         <ResponsiveContainer width="100%" height="100%">
                            <LineChart data={processedData}>
                               <Line type="monotone" dataKey={conf.totalKey} stroke={conf.color} strokeWidth={1.5} dot={false} />
                            </LineChart>
                         </ResponsiveContainer>
                      </div>
                   </button>
                )
             })}
          </div>
       </div>

       {/* Main Chart */}
       <div className="flex-1 flex flex-col relative bg-gradient-to-br from-card to-background">
          <div className="absolute top-4 right-4 z-10 flex gap-2">
             <div className="px-2 py-1 bg-secondary text-[10px] font-mono rounded text-muted-foreground border border-border">DAILY_DELTA_MODE</div>
          </div>
          <div className="flex-1 p-6">
             <ResponsiveContainer width="100%" height="100%">
                <ComposedChart data={processedData}>
                   <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                   <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} dy={10} />
                   <YAxis yAxisId="left" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                   <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                   <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }} />
                   
                   <ReferenceLine yAxisId="right" y={0} stroke="var(--border)" />
                   
                   {/* Highlighted Delta Bars */}
                   <Bar yAxisId="right" dataKey={activeConfig.deltaKey} fill={activeConfig.color} barSize={30} radius={[2, 2, 0, 0]} name="New Assets" />
                   
                   {/* Contextual Line */}
                   <Line yAxisId="left" type="step" dataKey={activeConfig.totalKey} stroke={activeConfig.color} strokeWidth={1} strokeDasharray="4 4" dot={false} name="Total Trend" />
                </ComposedChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 19: Split Comparator (Dual Select) ---
const SplitComparator = withRealData(({ data }) => {
   const [leftKey, setLeftKey] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
   const [rightKey, setRightKey] = useState<'sub' | 'ip' | 'url' | 'site'>('ip')

   const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
   } as const
   type MetricKey = keyof typeof config
   
   const Selector = ({ value, onChange }: { value: MetricKey; onChange: (v: MetricKey) => void }) => (
      <div className="flex gap-1">
         {(Object.keys(config) as Array<keyof typeof config>).map(key => (
            <button
               key={key}
               onClick={() => onChange(key)}
               className={cn(
                  "w-6 h-6 rounded flex items-center justify-center text-[10px] font-bold border transition-[color,background-color,border-color,opacity,transform,box-shadow]",
                  value === key ? "border-foreground text-foreground" : "border-transparent text-muted-foreground hover:bg-secondary"
               )}
               style={{ backgroundColor: value === key ? config[key].color : undefined, color: value === key ? '#fff' : undefined }}
               title={config[key].label}
            >
               {key[0].toUpperCase()}
            </button>
         ))}
      </div>
   )

   return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[380px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <div className="flex items-center gap-4">
             <div className="flex items-center gap-2">
                <IconScan className="w-4 h-4 text-muted-foreground" />
                <span className="text-xs font-bold tracking-widest text-foreground uppercase">Correlation</span>
             </div>
             
             {/* Controls */}
             <div className="flex items-center gap-2 bg-card border border-border rounded-full px-2 py-1 shadow-sm">
                <Selector value={leftKey} onChange={setLeftKey} />
                <span className="text-muted-foreground text-[10px] font-mono">VS</span>
                <Selector value={rightKey} onChange={setRightKey} />
             </div>
          </div>
          
          <div className="flex gap-4 text-[10px] font-mono">
             <span style={{ color: config[leftKey].color }}>● {config[leftKey].label}</span>
             <span style={{ color: config[rightKey].color }}>● {config[rightKey].label}</span>
          </div>
       </div>

       <div className="flex-1 p-4 relative">
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={data} margin={{ top: 10, right: 0, left: 0, bottom: 0 }}>
                <defs>
                   <linearGradient id="splitLeft" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={config[leftKey].color} stopOpacity={0.3}/>
                      <stop offset="100%" stopColor={config[leftKey].color} stopOpacity={0}/>
                   </linearGradient>
                   <linearGradient id="splitRight" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={config[rightKey].color} stopOpacity={0.3}/>
                      <stop offset="100%" stopColor={config[rightKey].color} stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis yAxisId="left" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }} />
                
                <Area yAxisId="left" type="monotone" dataKey={config[leftKey].totalKey} stroke={config[leftKey].color} fill="url(#splitLeft)" strokeWidth={2} name={config[leftKey].label} />
                <Area yAxisId="right" type="monotone" dataKey={config[rightKey].totalKey} stroke={config[rightKey].color} fill="url(#splitRight)" strokeWidth={2} name={config[rightKey].label} />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
   )
})

// --- Variant 20: Focus Lens (Minimalist) ---
const FocusLens = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('site')
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]
  const latestVal = data[data.length - 1]?.[activeConfig.totalKey as keyof typeof data[0]] as number || 0

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[320px] relative">
       {/* Floating Controls */}
       <div className="absolute top-4 left-4 z-10 flex flex-col gap-1">
           <div className="text-[10px] font-mono font-bold text-muted-foreground uppercase">Metric Focus</div>
           <div className="flex bg-background border border-border rounded-md overflow-hidden shadow-sm">
              {(Object.keys(config) as Array<keyof typeof config>).map(key => (
                 <button
                    key={key}
                    onClick={() => setActiveTab(key)}
                    className={cn(
                       "px-3 py-1.5 text-xs font-bold transition-colors hover:bg-secondary/50",
                       activeTab === key ? "bg-secondary text-foreground" : "text-muted-foreground"
                    )}
                 >
                    {config[key].label}
                 </button>
              ))}
           </div>
       </div>

       {/* Big Number Overlay */}
       <div className="absolute top-4 right-6 z-10 text-right pointer-events-none">
          <div className="text-4xl font-black tracking-tighter" style={{ color: activeConfig.color }}>
             {latestVal.toLocaleString()}
          </div>
          <div className="text-[10px] font-mono text-muted-foreground uppercase">Current Count</div>
       </div>

       <div className="flex-1 pt-12 pb-0 px-0">
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={data} margin={{ top: 20, right: 0, left: 0, bottom: 0 }}>
                <defs>
                   <linearGradient id={`focusGrad-${activeTab}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.5}/>
                      <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0 }} />
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.3} />
                
                <Area 
                   type="stepAfter" 
                   dataKey={activeConfig.totalKey} 
                   stroke={activeConfig.color} 
                   fill={`url(#focusGrad-${activeTab})`} 
                   strokeWidth={3} 
                   name={activeConfig.label}
                   animationDuration={1500}
                />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 21: Crosshair Analytics (2x2 Grid) ---
const CrosshairAnalytics = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden h-[400px] flex flex-wrap">
       {/* Q1: Subdomain Line */}
       <div className="w-1/2 h-1/2 border-r border-b border-border relative p-2">
          <div className="absolute top-2 left-2 text-[10px] font-bold text-muted-foreground uppercase">Q1 // Subdomain Trend</div>
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <Line type="monotone" dataKey="totalSubdomains" stroke="var(--chart-1)" strokeWidth={2} dot={false} />
                <Tooltip contentStyle={{ display: 'none' }} />
             </LineChart>
          </ResponsiveContainer>
       </div>
       {/* Q2: IP Area */}
       <div className="w-1/2 h-1/2 border-b border-border relative p-2">
          <div className="absolute top-2 left-2 text-[10px] font-bold text-muted-foreground uppercase">Q2 // IP Volumetrics</div>
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={data}>
                <Area type="step" dataKey="totalIps" stroke="var(--chart-2)" fill="var(--chart-2)" fillOpacity={0.2} />
                <Tooltip contentStyle={{ display: 'none' }} />
             </AreaChart>
          </ResponsiveContainer>
       </div>
       {/* Q3: URL Bar */}
       <div className="w-1/2 h-1/2 border-r border-border relative p-2">
          <div className="absolute top-2 left-2 text-[10px] font-bold text-muted-foreground uppercase">Q3 // URL Capacity</div>
          <ResponsiveContainer width="100%" height="100%">
             <BarChart data={data}>
                <Bar dataKey="totalEndpoints" fill="var(--chart-3)" />
                <Tooltip contentStyle={{ display: 'none' }} />
             </BarChart>
          </ResponsiveContainer>
       </div>
       {/* Q4: Site Scatter (Simulated with Line dots) */}
       <div className="w-1/2 h-1/2 relative p-2">
          <div className="absolute top-2 left-2 text-[10px] font-bold text-muted-foreground uppercase">Q4 // Site Distribution</div>
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <Line type="monotone" dataKey="totalWebsites" stroke="transparent" dot={{ fill: 'var(--chart-4)', r: 3 }} />
                <Tooltip contentStyle={{ display: 'none' }} />
             </LineChart>
          </ResponsiveContainer>
       </div>
       
       {/* Crosshair Overlay */}
       <div className="absolute inset-0 pointer-events-none flex items-center justify-center">
          <div className="w-full h-[1px] bg-border absolute"></div>
          <div className="h-full w-[1px] bg-border absolute"></div>
          <div className="w-2 h-2 rounded-full bg-foreground absolute"></div>
       </div>
    </div>
  )
})

// --- Variant 22: Timeline Stack (Vertical Log) ---
const TimelineStack = withRealData(({ data }) => {
  // Take last 5 days
  const recentData = data.slice(-5).reverse()
  
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[400px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <span className="text-xs font-bold tracking-widest text-foreground uppercase">Timeline Stack</span>
          <span className="text-[10px] font-mono text-muted-foreground">RECENT SNAPSHOTS</span>
       </div>
       <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {recentData.map((d, i) => (
             <div key={i} className="flex gap-4 items-center">
                {/* Time Node */}
                <div className="flex flex-col items-center min-w-[60px]">
                   <span className="text-xs font-bold font-mono text-foreground">{d.date.slice(5)}</span>
                   <div className="h-full w-[1px] bg-border mt-1 min-h-[20px]"></div>
                </div>
                
                {/* Snapshot Bar */}
                <div className="flex-1 bg-secondary/10 border border-border p-2 rounded relative overflow-hidden">
                   <div className="flex justify-between text-[10px] text-muted-foreground mb-1 z-10 relative">
                      <span>ASSETS_LOG_ENTRY_{i+1}</span>
                      <span>TOTAL: {(d.totalSubdomains + d.totalIps + d.totalEndpoints + d.totalWebsites).toLocaleString()}</span>
                   </div>
                   <div className="h-4 flex rounded-sm overflow-hidden z-10 relative">
                      <div style={{ width: `${(d.totalSubdomains / (d.totalSubdomains + d.totalIps + d.totalEndpoints + d.totalWebsites)) * 100}%`, backgroundColor: 'var(--chart-1)' }} title="Subdomains"></div>
                      <div style={{ width: `${(d.totalIps / (d.totalSubdomains + d.totalIps + d.totalEndpoints + d.totalWebsites)) * 100}%`, backgroundColor: 'var(--chart-2)' }} title="IPs"></div>
                      <div style={{ width: `${(d.totalEndpoints / (d.totalSubdomains + d.totalIps + d.totalEndpoints + d.totalWebsites)) * 100}%`, backgroundColor: 'var(--chart-3)' }} title="URLs"></div>
                      <div style={{ width: `${(d.totalWebsites / (d.totalSubdomains + d.totalIps + d.totalEndpoints + d.totalWebsites)) * 100}%`, backgroundColor: 'var(--chart-4)' }} title="Sites"></div>
                   </div>
                </div>
             </div>
          ))}
       </div>
    </div>
  )
})

// --- Variant 23: Iso-Metric Blocks (Stylized) ---
const IsoMetricBlocks = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[360px]">
       <div className="px-4 py-3 border-b border-border flex justify-between items-center bg-secondary/10">
          <span className="text-xs font-bold tracking-widest text-foreground uppercase">Iso-Metric Blocks</span>
          <span className="text-[10px] font-mono text-muted-foreground">VOLUME STACK</span>
       </div>
       <div className="flex-1 p-6 relative">
          <ResponsiveContainer width="100%" height="100%">
             <BarChart data={data} barCategoryGap={10}>
                <Tooltip cursor={{ fill: 'transparent' }} contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }} />
                <defs>
                   <linearGradient id="isoBar" x1="0" y1="0" x2="1" y2="0">
                      <stop offset="0%" stopColor="var(--chart-1)" stopOpacity={1}/>
                      <stop offset="50%" stopColor="var(--chart-1)" stopOpacity={0.8}/>
                      <stop offset="100%" stopColor="var(--chart-1)" stopOpacity={0.6}/>
                   </linearGradient>
                </defs>
                <Bar dataKey="totalSubdomains" fill="url(#isoBar)" stroke="var(--background)" strokeWidth={1} stackId="a" />
                <Bar dataKey="totalIps" fill="var(--chart-2)" opacity={0.8} stackId="a" />
                <Bar dataKey="totalEndpoints" fill="var(--chart-3)" opacity={0.8} stackId="a" />
             </BarChart>
          </ResponsiveContainer>
          {/* Faux 3D effect overlay - simplified */}
          <div className="absolute bottom-0 left-0 w-full h-[20px] bg-gradient-to-t from-background to-transparent pointer-events-none"></div>
       </div>
    </div>
  )
})

// --- Variant 24: Signal Noise (Oscilloscope) ---
const SignalNoise = withRealData(({ data }) => {
  return (
    <div className="w-full border border-border bg-black text-green-500 overflow-hidden flex flex-col h-[320px] font-mono relative">
       {/* CRT Grid Background */}
       <div className="absolute inset-0 opacity-20 pointer-events-none" 
            style={{ 
               backgroundImage: 'linear-gradient(var(--border) 1px, transparent 1px), linear-gradient(90deg, var(--border) 1px, transparent 1px)', 
               backgroundSize: '20px 20px' 
            }}
       ></div>
       
       <div className="px-4 py-3 border-b border-white/10 flex justify-between items-center z-10 bg-black/50">
          <span className="text-xs font-bold tracking-widest uppercase text-green-400">SIGNAL_NOISE.OSC</span>
          <span className="text-[10px] animate-pulse">LIVE INPUT</span>
       </div>
       
       <div className="flex-1 p-0 relative z-10">
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <Tooltip contentStyle={{ backgroundColor: '#000', border: '1px solid #333', color: '#0f0' }} itemStyle={{ color: '#0f0' }} />
                <Line 
                   type="basis" 
                   dataKey="totalEndpoints" 
                   stroke="#22c55e" 
                   strokeWidth={2} 
                   dot={false}
                   style={{ filter: 'drop-shadow(0 0 4px #22c55e)' }} 
                />
                <Line 
                   type="basis" 
                   dataKey="totalSubdomains" 
                   stroke="#22c55e" 
                   strokeWidth={1} 
                   strokeDasharray="3 3"
                   dot={false}
                   opacity={0.5}
                />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 25: Asset Ledger (Table + Sparklines) ---
const AssetLedger = withRealData(({ data }) => {
  const categories = [
     { key: 'totalSubdomains', label: 'SUBDOMAINS', color: 'var(--chart-1)' },
     { key: 'totalIps', label: 'IP ADDRESSES', color: 'var(--chart-2)' },
     { key: 'totalEndpoints', label: 'URL ENDPOINTS', color: 'var(--chart-3)' },
     { key: 'totalWebsites', label: 'WEB SITES', color: 'var(--chart-4)' },
  ]
  
  const latest = data[data.length - 1] || {}

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col">
       <div className="px-6 py-4 border-b border-border bg-secondary/5">
          <h3 className="text-sm font-bold uppercase tracking-widest">Asset Ledger</h3>
       </div>
       <div className="divide-y divide-border">
          {categories.map(cat => (
             <div key={cat.key} className="flex items-center p-4 hover:bg-secondary/10 transition-colors">
                <div className="w-[180px] flex flex-col">
                   <span className="text-[10px] font-bold text-muted-foreground uppercase">{cat.label}</span>
                   <span className="text-lg font-mono font-medium">{latest[cat.key as keyof typeof latest]?.toLocaleString()}</span>
                </div>
                <div className="flex-1 h-[40px]">
                   <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={data}>
                         <Line type="monotone" dataKey={cat.key} stroke={cat.color} strokeWidth={2} dot={false} />
                      </LineChart>
                   </ResponsiveContainer>
                </div>
                <div className="w-[100px] text-right pl-4 border-l border-border ml-4">
                   <span className="text-[10px] text-muted-foreground block">LAST 14 DAYS</span>
                   <span className="text-xs font-mono font-bold" style={{ color: cat.color }}>TRENDING</span>
                </div>
             </div>
          ))}
       </div>
    </div>
  )
})

// --- Variant 26: Tactical Command (Military/HUD) ---
const TacticalCommand = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#fbbf24' }, // Amber
     ip: { label: 'IP ADDRESSES', totalKey: 'totalIps', deltaKey: 'newIp', color: '#3b82f6' }, // Blue
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#ef4444' }, // Red
     site: { label: 'WEB SITES', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#10b981' }, // Green
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full border-2 border-primary/50 bg-black text-primary font-mono overflow-hidden flex flex-col h-[420px] relative">
       {/* Corner decorations */}
       <div className="absolute top-0 left-0 w-4 h-4 border-t-2 border-l-2 border-primary"></div>
       <div className="absolute top-0 right-0 w-4 h-4 border-t-2 border-r-2 border-primary"></div>
       <div className="absolute bottom-0 left-0 w-4 h-4 border-b-2 border-l-2 border-primary"></div>
       <div className="absolute bottom-0 right-0 w-4 h-4 border-b-2 border-r-2 border-primary"></div>

       {/* Top Deck */}
       <div className="flex border-b border-primary/30">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 p-3 border-r border-primary/30 transition-[color,background-color,border-color,opacity,transform,box-shadow] relative text-center uppercase text-xs tracking-widest",
                      isActive ? "bg-primary/20 text-primary-foreground font-bold" : "hover:bg-primary/10 text-primary/70"
                   )}
                >
                   <div>{conf.label}</div>
                   <div className="text-lg mt-1">{total?.toLocaleString()}</div>
                   {isActive && (
                      <div className="absolute top-0 right-0 p-1">
                         <div className="w-2 h-2 bg-primary animate-pulse"></div>
                      </div>
                   )}
                </button>
             )
          })}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6 relative">
          <div className="absolute top-2 right-2 text-[10px] border border-primary/50 px-2 py-1 bg-black z-10">
             STATUS: MONITORING // MODE: {activeConfig.label}
          </div>
          <ResponsiveContainer width="100%" height="100%">
             <ComposedChart data={processedData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(var(--primary-rgb), 0.2)" />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'currentColor'}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis yAxisId="left" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'currentColor'}} />
                <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'currentColor'}} />
                <Tooltip 
                   contentStyle={{ backgroundColor: '#000', border: '1px solid var(--primary)', color: 'var(--primary)' }} 
                   itemStyle={{ color: 'var(--primary)' }}
                   cursor={{ stroke: 'var(--primary)', strokeWidth: 1, strokeDasharray: '5 5' }}
                />
                
                <Area yAxisId="left" type="step" dataKey={activeConfig.totalKey} stroke="currentColor" fill="currentColor" fillOpacity={0.1} strokeWidth={2} />
                <Bar yAxisId="right" dataKey={activeConfig.deltaKey} fill="currentColor" opacity={0.5} barSize={10} />
             </ComposedChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 27: Corporate Dashboard (Clean/Soft) ---
const CorporateDashboard = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('ip')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full bg-card rounded-xl shadow-sm border border-border/50 overflow-hidden flex flex-col h-[420px]">
       {/* Top Deck */}
       <div className="flex p-2 bg-muted/30 gap-2">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number
             const delta = latest[conf.deltaKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 p-3 rounded-lg transition-[color,background-color,border-color,opacity,transform,box-shadow] text-left border",
                      isActive ? "bg-background border-border shadow-sm" : "bg-transparent border-transparent hover:bg-background/50"
                   )}
                >
                   <div className="text-xs font-semibold text-muted-foreground">{conf.label}</div>
                   <div className="flex justify-between items-end mt-2">
                      <span className="text-xl font-bold text-foreground">{total?.toLocaleString()}</span>
                      {delta > 0 && <span className="text-xs font-medium text-emerald-600 bg-emerald-100 dark:bg-emerald-900/30 px-1.5 py-0.5 rounded-full">+{delta}</span>}
                   </div>
                </button>
             )
          })}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6">
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={processedData}>
                <defs>
                   <linearGradient id={`corpGrad-${activeTab}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.3}/>
                      <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.4} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 11, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis tickLine={false} axisLine={false} tick={{fontSize: 11, fill: 'var(--muted-foreground)'}} />
                <Tooltip 
                   contentStyle={{ backgroundColor: 'var(--popover)', borderColor: 'var(--border)', borderRadius: '8px', boxShadow: 'var(--shadow-md)' }} 
                   cursor={{ stroke: activeConfig.color, strokeWidth: 1, strokeDasharray: '4 4' }}
                />
                
                <Area type="monotone" dataKey={activeConfig.totalKey} stroke={activeConfig.color} fill={`url(#corpGrad-${activeTab})`} strokeWidth={3} activeDot={{r: 6, strokeWidth: 0}} />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 28: Cyber Nexus (Neon/Tech) ---
const CyberNexus = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', color: '#ec4899' }, // Pink
     ip: { label: 'IP_ADDR', totalKey: 'totalIps', color: '#8b5cf6' }, // Violet
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', color: '#06b6d4' }, // Cyan
     site: { label: 'SITES', totalKey: 'totalWebsites', color: '#f59e0b' }, // Amber
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full border border-cyan-500/30 bg-slate-950 text-cyan-50 overflow-hidden flex flex-col h-[420px] relative shadow-[0_0_15px_rgba(6,182,212,0.1)]">
       {/* Background Grid */}
       <div className="absolute inset-0 opacity-10" style={{ backgroundImage: 'radial-gradient(#06b6d4 1px, transparent 1px)', backgroundSize: '20px 20px' }}></div>

       {/* Top Deck */}
       <div className="flex z-10 bg-slate-900/80 backdrop-blur-sm border-b border-cyan-500/20">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 py-4 transition-[color,background-color,border-color,opacity,transform,box-shadow] relative text-center group",
                      isActive ? "text-white" : "text-cyan-500/50 hover:text-cyan-400"
                   )}
                >
                   <div className={cn("text-[10px] font-bold tracking-widest mb-1", isActive ? "animate-pulse" : "")}>{conf.label}</div>
                   <div className="text-xl font-mono" style={{ textShadow: isActive ? `0 0 10px ${conf.color}` : 'none' }}>{total?.toLocaleString()}</div>
                   {isActive && (
                      <div className="absolute bottom-0 left-0 right-0 h-[2px] shadow-[0_0_10px_currentColor]" style={{ backgroundColor: conf.color }}></div>
                   )}
                </button>
             )
          })}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6 relative z-10">
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={processedData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#06b6d4" strokeOpacity={0.1} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#06b6d4', opacity: 0.7}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#06b6d4', opacity: 0.7}} />
                <Tooltip 
                   contentStyle={{ backgroundColor: '#0f172a', borderColor: activeConfig.color, borderRadius: 0, boxShadow: `0 0 10px ${activeConfig.color}` }} 
                   itemStyle={{ color: activeConfig.color }}
                />
                <Line 
                   type="linear" 
                   dataKey={activeConfig.totalKey} 
                   stroke={activeConfig.color} 
                   strokeWidth={2} 
                   dot={{ r: 4, strokeWidth: 2, fill: '#0f172a', stroke: activeConfig.color }}
                   activeDot={{ r: 6, fill: activeConfig.color }}
                   style={{ filter: `drop-shadow(0 0 5px ${activeConfig.color})` }}
                />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 29: Minimalist Zen (Whitespace) ---
const MinimalistZen = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('site')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub' },
     ip: { label: 'IPs', totalKey: 'totalIps', deltaKey: 'newIp' },
     url: { label: 'URLs', totalKey: 'totalEndpoints', deltaKey: 'newUrl' },
     site: { label: 'Sites', totalKey: 'totalWebsites', deltaKey: 'newSite' },
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex flex-col h-[420px]">
       {/* Top Deck - Text Only */}
       <div className="flex p-6 pb-0 gap-8">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex flex-col items-start transition-[color,background-color,border-color,opacity,transform,box-shadow] group",
                      isActive ? "opacity-100" : "opacity-40 hover:opacity-70"
                   )}
                >
                   <span className="text-3xl font-light tracking-tighter text-foreground">{total?.toLocaleString()}</span>
                   <span className="text-xs font-medium uppercase tracking-wide text-foreground">{conf.label}</span>
                   {isActive && <div className="mt-2 w-full h-[1px] bg-black dark:bg-white"></div>}
                </button>
             )
          })}
       </div>

       {/* Main Chart - Line Art */}
       <div className="flex-1 p-8">
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={processedData}>
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#888'}} tickFormatter={d => d.slice(5)} dy={10} />
                <Tooltip contentStyle={{ border: 'none', boxShadow: 'none', backgroundColor: 'transparent', fontWeight: 'bold' }} />
                <Line 
                   type="monotone" 
                   dataKey={activeConfig.totalKey} 
                   stroke="currentColor" 
                   strokeWidth={1.5} 
                   dot={false}
                   strokeOpacity={0.8}
                />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 30: Data Lab (Grid Paper) ---
const DataLab = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#333' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: '#333' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#333' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#333' },
  }
  const activeConfig = config[activeTab]
  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full border border-stone-300 bg-[#fdfaf6] dark:bg-stone-900 text-stone-700 dark:text-stone-300 flex flex-col h-[420px] relative">
       {/* Graph Paper Grid */}
       <div className="absolute inset-0 opacity-[0.03] pointer-events-none" 
            style={{ 
               backgroundImage: 'linear-gradient(#000 1px, transparent 1px), linear-gradient(90deg, #000 1px, transparent 1px)', 
               backgroundSize: '20px 20px' 
            }}
       ></div>

       {/* Top Deck - Tabs */}
       <div className="flex border-b border-stone-300 relative z-10">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 p-3 border-r border-stone-300 text-left font-serif transition-colors",
                      isActive ? "bg-stone-200/50 dark:bg-stone-800/50" : "hover:bg-stone-100 dark:hover:bg-stone-800"
                   )}
                >
                   <div className="text-xs italic opacity-70">{conf.label}</div>
                   <div className="text-lg font-bold">{total?.toLocaleString()}</div>
                </button>
             )
          })}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6 relative z-10">
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={processedData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#ccc" />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'serif'}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'serif'}} />
                <Tooltip 
                   contentStyle={{ backgroundColor: '#fff', border: '1px solid #999', fontFamily: 'serif' }} 
                />
                
                {/* Hand-drawn style line attempt */}
                <Line 
                   type="monotone" 
                   dataKey={activeConfig.totalKey} 
                   stroke="currentColor" 
                   strokeWidth={2} 
                   dot={{ r: 3, fill: '#fff', stroke: 'currentColor' }}
                />
                {/* Annotations mockup */}
                {processedData.length > 5 && (
                   <ReferenceLine x={processedData[processedData.length-3].date} stroke="red" strokeDasharray="3 3" label={{ position: 'top', value: 'Anomaly', fill: 'red', fontSize: 10 }} />
                )}
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 31: Orbital Selector (Sci-Fi) ---
const OrbitalSelector = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'SUB', totalKey: 'totalSubdomains', angle: 0, color: 'var(--chart-1)' },
     ip: { label: 'IP', totalKey: 'totalIps', angle: 90, color: 'var(--chart-2)' },
     url: { label: 'URL', totalKey: 'totalEndpoints', angle: 180, color: 'var(--chart-3)' },
     site: { label: 'WEB', totalKey: 'totalWebsites', angle: 270, color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex h-[400px] relative">
       {/* Rotating Menu */}
       <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[200px] h-[200px] pointer-events-none z-0 opacity-10">
          <div className="w-full h-full rounded-full border-2 border-dashed border-foreground animate-spin-slow"></div>
       </div>

       {/* Selector Buttons */}
       <div className="w-[100px] flex flex-col justify-center gap-4 p-2 z-10 border-r border-border bg-secondary/10">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => (
             <button
                key={key}
                onClick={() => setActiveTab(key)}
                className={cn(
                   "w-full aspect-square rounded-full border-2 flex items-center justify-center transition-[color,background-color,border-color,opacity,transform,box-shadow]",
                   activeTab === key ? "border-primary bg-primary text-primary-foreground scale-110" : "border-border text-muted-foreground hover:border-primary/50"
                )}
             >
                <span className="text-[10px] font-bold">{config[key].label}</span>
             </button>
          ))}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6 relative z-10 bg-gradient-to-r from-card to-background">
          <div className="text-sm font-bold uppercase tracking-widest mb-4" style={{ color: activeConfig.color }}>
             Sector: {activeConfig.label}
          </div>
          <ResponsiveContainer width="100%" height="90%">
             <AreaChart data={processedData}>
                <defs>
                   <linearGradient id={`orbGrad-${activeTab}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.4}/>
                      <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} />
                <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
                <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }} />
                <Area type="natural" dataKey={activeConfig.totalKey} stroke={activeConfig.color} fill={`url(#orbGrad-${activeTab})`} strokeWidth={3} />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 32: Stacked Accordion (Mobile-Friendly) ---
const StackedAccordion = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  }

  const latest = processedData[processedData.length - 1] || data[data.length - 1] || {}

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[450px]">
       {(Object.keys(config) as Array<keyof typeof config>).map(key => {
          const conf = config[key]
          const isActive = activeTab === key
          
          return (
             <div key={key} className="flex flex-col flex-1 transition-[color,background-color,border-color,opacity,transform,box-shadow] overflow-hidden border-b border-border last:border-0" style={{ flexGrow: isActive ? 5 : 1 }}>
                <button 
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex items-center justify-between px-4 py-3 hover:bg-secondary/10 transition-colors",
                      isActive ? "bg-secondary/20" : ""
                   )}
                >
                   <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full" style={{ backgroundColor: conf.color }}></div>
                      <span className="text-xs font-bold uppercase">{conf.label}</span>
                   </div>
                   <span className="text-xs font-mono">{latest[conf.totalKey as keyof typeof latest]?.toLocaleString()}</span>
                </button>
                
                {isActive && (
                   <div className="flex-1 p-4 bg-card relative">
                      <ResponsiveContainer width="100%" height="100%">
                         <AreaChart data={processedData}>
                            <defs>
                               <linearGradient id={`accGrad-${key}`} x1="0" y1="0" x2="0" y2="1">
                                  <stop offset="0%" stopColor={conf.color} stopOpacity={0.2}/>
                                  <stop offset="100%" stopColor={conf.color} stopOpacity={0}/>
                               </linearGradient>
                            </defs>
                            <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }} />
                            <Area type="monotone" dataKey={conf.totalKey} stroke={conf.color} fill={`url(#accGrad-${key})`} strokeWidth={2} />
                         </AreaChart>
                      </ResponsiveContainer>
                   </div>
                )}
             </div>
          )
       })}
    </div>
  )
})

// --- Variant 33: Hover Preview (Instant Swap) ---
const HoverPreview = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[400px]">
       <div className="flex-1 relative p-6">
          <AnimatePresence mode="wait">
             <motion.div 
                key={activeTab}
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 1.05 }}
                transition={{ duration: 0.2 }}
                className="absolute inset-0 p-6"
             >
                <div className="text-sm font-bold uppercase mb-2 tracking-widest text-muted-foreground">{activeConfig.label} Trend</div>
                <ResponsiveContainer width="100%" height="90%">
                   <LineChart data={processedData}>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                      <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} />
                      <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }} />
                      <Line type="monotone" dataKey={activeConfig.totalKey} stroke={activeConfig.color} strokeWidth={3} dot={false} activeDot={{r: 6}} />
                   </LineChart>
                </ResponsiveContainer>
             </motion.div>
          </AnimatePresence>
       </div>

       <div className="h-[80px] flex border-t border-border">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => (
             <button
                key={key}
                type="button"
                onMouseEnter={() => setActiveTab(key)}
                onFocus={() => setActiveTab(key)}
                onClick={() => setActiveTab(key)}
                className={cn(
                   "flex-1 flex flex-col items-center justify-center cursor-pointer transition-colors border-r border-border last:border-0 hover:bg-secondary/20",
                   activeTab === key ? "bg-secondary/30" : "bg-card"
                )}
             >
                <span className="text-[10px] font-bold uppercase text-muted-foreground">{config[key].label}</span>
                <span className="text-sm font-mono font-bold" style={{ color: activeTab === key ? config[key].color : undefined }}>
                   {latest[config[key].totalKey as keyof typeof latest]?.toLocaleString()}
                </span>
             </button>
          ))}
       </div>
    </div>
  )
})

// --- Variant 34: Gallery Slider (Carousel) ---
const GallerySlider = withRealData(({ data }) => {
  const [index, setIndex] = useState(0)
  const keys: Array<'sub' | 'ip' | 'url' | 'site'> = ['sub', 'ip', 'url', 'site']
  
  const next = () => setIndex((prev) => (prev + 1) % keys.length)
  const prev = () => setIndex((prev) => (prev - 1 + keys.length) % keys.length)
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  }
  
  const activeKey = keys[index]
  const activeConfig = config[activeKey]

  return (
    <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[400px] relative group">
       {/* Navigation Arrows */}
       <button onClick={prev} className="absolute left-2 top-1/2 -translate-y-1/2 z-20 w-8 h-8 rounded-full bg-secondary/50 flex items-center justify-center hover:bg-secondary transition-colors opacity-0 group-hover:opacity-100" aria-label="Previous item">
          <IconChevronLeft className="w-4 h-4" />
       </button>
       <button onClick={next} className="absolute right-2 top-1/2 -translate-y-1/2 z-20 w-8 h-8 rounded-full bg-secondary/50 flex items-center justify-center hover:bg-secondary transition-colors opacity-0 group-hover:opacity-100" aria-label="Next item">
          <IconChevronRight className="w-4 h-4" />
       </button>

       {/* Pagination Dots */}
       <div className="absolute bottom-4 left-1/2 -translate-x-1/2 z-20 flex gap-2">
          {keys.map((k, i) => (
             <button 
                key={k} 
                onClick={() => setIndex(i)}
                aria-label={`Go to ${config[k].label}`}
                className={cn("w-2 h-2 rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow]", i === index ? "bg-foreground scale-125" : "bg-muted-foreground/30 hover:bg-muted-foreground/50")} 
             />
          ))}
       </div>

       <div className="flex-1 p-8">
          <div className="text-center mb-4">
             <h3 className="text-xl font-bold tracking-tight">{activeConfig.label}</h3>
             <p className="text-sm text-muted-foreground">Historical Performance Gallery</p>
          </div>
          <ResponsiveContainer width="100%" height="80%">
             <BarChart data={processedData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} />
                <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }} />
                <Bar dataKey={activeConfig.totalKey} fill={activeConfig.color} radius={[4, 4, 0, 0]} />
             </BarChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 35: Drill-Down Matrix (Zoom In) ---
const DrillDownMatrix = withRealData(({ data }) => {
  const [zoomedKey, setZoomedKey] = useState<'sub' | 'ip' | 'url' | 'site' | null>(null)
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  } as const

  type MetricKey = keyof typeof config
  type MiniChartConfig = (typeof config)[MetricKey] & { key: MetricKey }

  // Mini Chart Component
  const MiniChart = ({ conf }: { conf: MiniChartConfig }) => (
     <button
        type="button"
        aria-label={`Zoom into ${conf.label}`}
        onClick={() => setZoomedKey(conf.key)}
        className="w-full border border-border bg-card p-4 flex flex-col cursor-pointer hover:border-primary/50 transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:shadow-md relative group h-full text-left"
     >
        <div className="text-xs font-bold uppercase text-muted-foreground mb-2 group-hover:text-primary transition-colors">{conf.label}</div>
        <div className="flex-1 min-h-[80px]">
           <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={processedData}>
                 <Area type="monotone" dataKey={conf.totalKey} stroke={conf.color} fill={conf.color} fillOpacity={0.1} strokeWidth={2} dot={false} />
              </AreaChart>
           </ResponsiveContainer>
        </div>
        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
           <div className="bg-primary text-primary-foreground text-[10px] px-1.5 py-0.5 rounded">ZOOM +</div>
        </div>
     </button>
  )

  if (zoomedKey) {
     const activeConfig = config[zoomedKey]
     return (
        <div className="w-full border border-border bg-card overflow-hidden flex flex-col h-[400px]">
           <div className="p-4 border-b border-border flex justify-between items-center bg-secondary/10">
              <div className="flex items-center gap-2">
                 <button type="button" onClick={() => setZoomedKey(null)} className="text-xs font-bold hover:underline flex items-center gap-1">
                    <IconChevronLeft className="w-3 h-3" /> BACK
                 </button>
                 <span className="text-border">|</span>
                 <span className="text-sm font-bold">{activeConfig.label} Analysis</span>
              </div>
           </div>
           <div className="flex-1 p-6">
              <ResponsiveContainer width="100%" height="100%">
                 <AreaChart data={processedData}>
                    <defs>
                       <linearGradient id="zoomGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.3}/>
                          <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                       </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10}} tickFormatter={d => d.slice(5)} />
                    <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10}} />
                    <Tooltip contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }} />
                    <Area type="monotone" dataKey={activeConfig.totalKey} stroke={activeConfig.color} fill="url(#zoomGrad)" strokeWidth={3} />
                 </AreaChart>
              </ResponsiveContainer>
           </div>
        </div>
     )
  }

  return (
    <div className="w-full h-[400px] grid grid-cols-2 gap-4">
       <MiniChart conf={{ ...config.sub, key: 'sub' }} />
       <MiniChart conf={{ ...config.ip, key: 'ip' }} />
       <MiniChart conf={{ ...config.url, key: 'url' }} />
       <MiniChart conf={{ ...config.site, key: 'site' }} />
    </div>
  )
})

// --- Variant 36: Bauhaus Command (The Perfect Match) ---
const BauhausCommand = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#3b82f6', accent: 'bg-blue-500 text-white' },
     ip: { label: 'IP ADDRESSES', totalKey: 'totalIps', deltaKey: 'newIp', color: '#f97316', accent: 'bg-orange-500 text-white' },
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#eab308', accent: 'bg-yellow-500 text-black' },
     site: { label: 'WEB SITES', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#22c55e', accent: 'bg-green-500 text-white' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full border-2 border-border border-t-4 border-t-primary bg-card flex flex-col h-[450px]">
       {/* Kicker */}
       <div className="flex items-center gap-2 p-2 border-b border-border bg-secondary/10">
          <IconScan className="w-4 h-4 text-primary" />
          <span className="text-[10px] font-mono tracking-widest font-bold uppercase text-muted-foreground">Command Deck // {activeConfig.label}</span>
       </div>

       {/* Top Tabs */}
       <div className="flex border-b border-border divide-x divide-border">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number
             const delta = latest[conf.deltaKey as keyof typeof latest] as number

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 p-4 text-left transition-[color,background-color,border-color,opacity,transform,box-shadow] relative group hover:bg-secondary/20",
                      isActive ? "bg-secondary/10" : "bg-card"
                   )}
                >
                   <div className="flex justify-between items-start mb-2">
                      <span className={cn("text-[10px] font-bold font-mono uppercase tracking-wider", isActive ? "text-foreground" : "text-muted-foreground")}>
                         {conf.label}
                      </span>
                      {isActive && <div className="w-2 h-2 bg-primary"></div>}
                   </div>
                   <div className="flex items-baseline gap-2">
                      <span className="text-xl font-bold tracking-tight">{total?.toLocaleString()}</span>
                      {delta > 0 && (
                         <span className={cn("text-[10px] font-mono px-1", conf.accent)}>+{delta}</span>
                      )}
                   </div>
                   {isActive && (
                      <div className="absolute bottom-0 left-0 right-0 h-1" style={{ backgroundColor: conf.color }}></div>
                   )}
                </button>
             )
          })}
       </div>

       {/* Main Chart */}
       <div className="flex-1 p-6 relative bg-card">
          <ResponsiveContainer width="100%" height="100%">
             <ComposedChart data={processedData} margin={{top: 10, right: 0, left: -20, bottom: 0}}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.6} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} dy={10} />
                <YAxis yAxisId="left" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} />
                <YAxis yAxisId="right" orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} />
                <Tooltip 
                   contentStyle={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)', borderRadius: 0, borderWidth: '2px' }}
                   itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12, fontWeight: 'bold' }}
                   labelStyle={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--muted-foreground)', marginBottom: 8, textTransform: 'uppercase' }}
                />
                
                <Area 
                   yAxisId="left" 
                   type="step" 
                   dataKey={activeConfig.totalKey} 
                   stroke={activeConfig.color} 
                   fill={activeConfig.color} 
                   fillOpacity={0.1} 
                   strokeWidth={2} 
                   activeDot={{r: 0}}
                />
                <Bar 
                   yAxisId="right" 
                   dataKey={activeConfig.deltaKey} 
                   fill={activeConfig.color} 
                   opacity={0.6} 
                   barSize={12} 
                />
             </ComposedChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 37: Industrial Panel (Physical) ---
const IndustrialPanel = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('ip')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUB-DOM', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#64748b' },
     ip: { label: 'IP-ADDR', totalKey: 'totalIps', deltaKey: 'newIp', color: '#64748b' },
     url: { label: 'URL-END', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#64748b' },
     site: { label: 'WEB-SITE', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#64748b' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-slate-200 dark:bg-slate-800 p-4 rounded-xl shadow-inner border border-slate-300 dark:border-slate-700 flex flex-col h-[460px]">
       {/* Panel Screws */}
       <div className="flex justify-between mb-4 px-1">
          <div className="w-2 h-2 rounded-full bg-slate-400 dark:bg-slate-600 shadow-inner"></div>
          <div className="w-2 h-2 rounded-full bg-slate-400 dark:bg-slate-600 shadow-inner"></div>
       </div>

       {/* Button Row */}
       <div className="flex gap-2 mb-4 bg-slate-300 dark:bg-slate-900 p-2 rounded-lg shadow-inner">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 py-3 px-2 rounded font-bold text-xs uppercase tracking-tight transition-[color,background-color,border-color,opacity,transform,box-shadow] active:scale-95 shadow-md border-b-2",
                      isActive 
                         ? "bg-blue-600 text-white border-blue-800 shadow-blue-900/20" 
                         : "bg-slate-100 dark:bg-slate-700 text-slate-500 border-slate-300 dark:border-slate-900 hover:bg-white"
                   )}
                >
                   {conf.label}
                </button>
             )
          })}
       </div>

       {/* Chart Screen (Recessed) */}
       <div className="flex-1 bg-black rounded-lg border-4 border-slate-300 dark:border-slate-700 shadow-[inset_0_0_20px_rgba(0,0,0,0.5)] p-4 relative overflow-hidden">
          {/* Screen Glare */}
          <div className="absolute top-0 right-0 w-[200px] h-[200px] bg-gradient-to-bl from-white/5 to-transparent pointer-events-none rounded-tr-lg"></div>
          
          <div className="text-blue-400 font-mono text-xs mb-2 flex justify-between">
             <span>MONITORING: {activeConfig.label}</span>
             <span>VAL: {latest[activeConfig.totalKey as keyof typeof latest]?.toLocaleString()}</span>
          </div>
          
          <ResponsiveContainer width="100%" height="85%">
             <LineChart data={processedData}>
                <CartesianGrid strokeDasharray="2 2" stroke="#1e293b" />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#475569'}} tickFormatter={d => d.slice(5)} />
                <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#475569'}} />
                <Tooltip 
                   contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #1e293b', color: '#60a5fa' }}
                   itemStyle={{ color: '#60a5fa' }}
                   cursor={{ stroke: '#60a5fa', strokeWidth: 1 }}
                />
                <Line type="monotone" dataKey={activeConfig.totalKey} stroke="#3b82f6" strokeWidth={2} dot={false} />
                <Line type="monotone" dataKey={activeConfig.deltaKey} stroke="#22c55e" strokeWidth={1} strokeDasharray="3 3" dot={false} />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 38: Swiss Grid (Redesigned: Bauhaus Typography) ---
const SwissGrid = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#3b82f6', bg: 'bg-blue-500', text: 'text-blue-500' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: '#f97316', bg: 'bg-orange-500', text: 'text-orange-500' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#eab308', bg: 'bg-yellow-500', text: 'text-yellow-500' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#22c55e', bg: 'bg-green-500', text: 'text-green-500' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-card text-card-foreground border-2 border-primary flex flex-col h-[450px]">
       {/* Header */}
       <div className="p-4 border-b-2 border-primary flex justify-between items-baseline bg-secondary/10">
          <h2 className="text-xl font-bold tracking-tighter uppercase font-mono flex items-center gap-2">
             <div className={`w-3 h-3 ${activeConfig.bg}`}></div>
             Asset_Grid_System
          </h2>
          <span className="font-mono text-xs text-muted-foreground">REF: {activeTab.toUpperCase()}_01</span>
       </div>

       <div className="flex flex-1 overflow-hidden">
          {/* Sidebar Tabs */}
          <div className="w-1/3 border-r-2 border-primary flex flex-col bg-secondary/5">
             {(Object.keys(config) as Array<keyof typeof config>).map(key => {
                const conf = config[key]
                const isActive = activeTab === key
                
                return (
                   <button
                      key={key}
                      onClick={() => setActiveTab(key)}
                      className={cn(
                         "flex-1 p-4 text-left border-b border-border last:border-b-0 transition-[color,background-color,border-color,opacity,transform,box-shadow] flex flex-col justify-center relative overflow-hidden group",
                         isActive ? "bg-card" : "hover:bg-secondary/20"
                      )}
                   >
                      {isActive && (
                         <div className={`absolute left-0 top-0 bottom-0 w-1 ${conf.bg}`}></div>
                      )}
                      <span className={cn(
                         "text-[10px] font-bold uppercase mb-1 font-mono tracking-wider",
                         isActive ? "text-foreground" : "text-muted-foreground group-hover:text-foreground"
                      )}>
                         {conf.label}
                      </span>
                      <span className={cn(
                         "text-xl font-mono font-bold",
                         isActive ? conf.text : "text-foreground"
                      )}>
                         {latest[conf.totalKey as keyof typeof latest]?.toLocaleString()}
                      </span>
                   </button>
                )
             })}
          </div>

          {/* Main Chart */}
          <div className="flex-1 p-6 flex flex-col bg-card relative">
             <div className="absolute top-0 right-0 p-2 opacity-10">
                <IconActivity className="w-24 h-24" />
             </div>
             
             <div className="mb-6 flex justify-between items-end relative z-10">
                <div>
                   <h3 className="text-2xl font-black uppercase tracking-tight">{activeConfig.label}</h3>
                   <div className="h-1 w-12 mt-1" style={{ backgroundColor: activeConfig.color }}></div>
                </div>
                <div className="text-xs font-mono text-muted-foreground flex items-center gap-2">
                   <span className="inline-block w-2 h-2 rounded-full animate-pulse" style={{ backgroundColor: activeConfig.color }}></span>
                   LIVE MONITORING
                </div>
             </div>
             
             <div className="flex-1 border border-border p-2 bg-secondary/5 relative z-10">
                <ResponsiveContainer width="100%" height="100%">
                   <ComposedChart data={processedData}>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                      <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} dy={10} />
                      <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} />
                      <Tooltip 
                         contentStyle={{ backgroundColor: 'var(--card)', border: '2px solid var(--primary)', borderRadius: 0, boxShadow: '4px 4px 0 var(--border)' }}
                         cursor={{ stroke: activeConfig.color, strokeWidth: 1 }}
                         itemStyle={{ fontFamily: 'var(--font-mono)', fontSize: 12, fontWeight: 'bold' }}
                         labelStyle={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--muted-foreground)', marginBottom: 8 }}
                      />
                      <Bar dataKey={activeConfig.deltaKey} fill={activeConfig.color} barSize={20} opacity={0.3} radius={[2, 2, 0, 0]} />
                      <Line type="step" dataKey={activeConfig.totalKey} stroke={activeConfig.color} strokeWidth={3} dot={{r: 4, fill: 'var(--card)', stroke: activeConfig.color, strokeWidth: 2}} activeDot={{r: 6}} />
                   </ComposedChart>
                </ResponsiveContainer>
             </div>
          </div>
       </div>
    </div>
  )
})

// --- Variant 39: Blueprint Schematics (Redesigned: System Schematic) ---
const BlueprintSchematics = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('site')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#3b82f6' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', deltaKey: 'newIp', color: '#f97316' },
     url: { label: 'URL Endpoints', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#eab308' },
     site: { label: 'Web Sites', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#22c55e' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-card text-card-foreground font-mono overflow-hidden flex flex-col h-[420px] border border-border relative">
       {/* Technical Grid Background */}
       <div className="absolute inset-0 opacity-[0.03] pointer-events-none" 
            style={{ 
               backgroundImage: `linear-gradient(var(--foreground) 1px, transparent 1px), linear-gradient(90deg, var(--foreground) 1px, transparent 1px)`, 
               backgroundSize: '20px 20px' 
            }}
       ></div>
       <div className="absolute inset-0 border-4 border-primary/10 pointer-events-none m-1"></div>

       {/* Top Tabs - Technical Style */}
       <div className="flex border-b border-border relative z-10 bg-secondary/10">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 py-3 text-xs border-r border-border transition-[color,background-color,border-color,opacity,transform,box-shadow] flex items-center justify-center gap-2",
                      isActive ? "bg-card font-bold border-b-2 border-b-primary" : "hover:bg-card/50 text-muted-foreground border-b-2 border-b-transparent"
                   )}
                >
                   <div className={cn("w-2 h-2 rounded-sm", isActive ? "" : "opacity-50")} style={{ backgroundColor: conf.color }}></div>
                   {conf.label}
                </button>
             )
          })}
       </div>

       {/* Main Content */}
       <div className="flex-1 p-6 relative z-10 flex gap-6">
          {/* Left Stats Panel - Floating Card Look */}
          <div className="w-[140px] flex flex-col gap-4 py-2">
             <div className="border border-border bg-card p-3 shadow-sm relative overflow-hidden group">
                <div className="absolute top-0 left-0 w-1 h-full transition-[color,background-color,border-color,opacity,transform,box-shadow] group-hover:w-full opacity-10" style={{ backgroundColor: activeConfig.color }}></div>
                <div className="text-[10px] text-muted-foreground mb-1 uppercase tracking-wider">Total Count</div>
                <div className="text-xl font-bold" style={{ color: activeConfig.color }}>
                   {latest[activeConfig.totalKey as keyof typeof latest]?.toLocaleString()}
                </div>
             </div>
             
             <div className="border border-border border-dashed bg-secondary/5 p-3 relative">
                <div className="absolute -left-[1px] top-1/2 w-2 h-[1px] bg-border"></div>
                <div className="text-[10px] text-muted-foreground mb-1 uppercase tracking-wider">24h Delta</div>
                <div className="text-lg font-bold flex items-center gap-1">
                   <IconTrendingUp className="w-3 h-3" />
                   +{latest[activeConfig.deltaKey as keyof typeof latest]?.toLocaleString()}
                </div>
             </div>

             <div className="mt-auto text-[10px] text-muted-foreground leading-tight opacity-70">
                <p>SYS.ID: {Math.random().toString(36).substr(2, 6).toUpperCase()}</p>
                <p>LATENCY: 12ms</p>
                <p>STATUS: OK</p>
             </div>
          </div>

          {/* Chart Area - Schematic Look */}
          <div className="flex-1 border border-border bg-card/50 p-2 relative">
             {/* Corner Markers */}
             <div className="absolute top-0 left-0 w-2 h-2 border-t-2 border-l-2 border-primary"></div>
             <div className="absolute top-0 right-0 w-2 h-2 border-t-2 border-r-2 border-primary"></div>
             <div className="absolute bottom-0 left-0 w-2 h-2 border-b-2 border-l-2 border-primary"></div>
             <div className="absolute bottom-0 right-0 w-2 h-2 border-b-2 border-r-2 border-primary"></div>

             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={processedData}>
                   <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" opacity={0.4} />
                   <XAxis 
                      dataKey="date" 
                      tickLine={true} 
                      axisLine={{ stroke: 'var(--border)' }} 
                      tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} 
                      tickFormatter={d => d.slice(5)} 
                   />
                   <YAxis 
                      tickLine={true} 
                      axisLine={{ stroke: 'var(--border)' }} 
                      tick={{fontSize: 10, fontFamily: 'var(--font-mono)', fill: 'var(--muted-foreground)'}} 
                   />
                   <Tooltip 
                      contentStyle={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                      itemStyle={{ fontFamily: 'var(--font-mono)', color: activeConfig.color }}
                      cursor={{ stroke: 'var(--muted-foreground)', strokeWidth: 1, strokeDasharray: '4 4' }}
                   />
                   <Line 
                      type="monotone" 
                      dataKey={activeConfig.totalKey} 
                      stroke={activeConfig.color} 
                      strokeWidth={2} 
                      dot={{ r: 3, fill: 'var(--card)', stroke: activeConfig.color, strokeWidth: 2 }} 
                      activeDot={{ r: 5, strokeWidth: 0, fill: activeConfig.color }}
                   />
                   {/* Measurement Lines (Mock) */}
                   {processedData.length > 0 && (
                      <ReferenceLine 
                         y={latest[activeConfig.totalKey as keyof typeof latest]} 
                         stroke={activeConfig.color} 
                         strokeDasharray="5 5" 
                         opacity={0.5}
                         label={{ position: 'right', value: 'CURRENT', fill: activeConfig.color, fontSize: 10, fontFamily: 'var(--font-mono)' }} 
                      />
                   )}
                </LineChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 40: Constructivist Blocks (Poster Art) ---
const ConstructivistBlocks = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', color: '#ef4444' }, // Red
     ip: { label: 'IPs', totalKey: 'totalIps', color: '#171717' }, // Black
     url: { label: 'URLs', totalKey: 'totalEndpoints', color: '#eab308' }, // Yellow
     site: { label: 'SITES', totalKey: 'totalWebsites', color: '#fff' }, // White
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-[#f5f5f4] flex flex-col h-[450px] border-4 border-black p-4 gap-4">
       {/* Top Deck - Big Blocks */}
       <div className="grid grid-cols-4 gap-4 h-[100px]">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             const total = latest[conf.totalKey as keyof typeof latest] as number
             const isWhite = conf.color === '#fff'

             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "h-full flex flex-col justify-center items-center transition-transform hover:-translate-y-1 border-2 border-black shadow-[4px_4px_0_0_rgba(0,0,0,1)]",
                      isActive ? "translate-y-1 shadow-none" : ""
                   )}
                   style={{ backgroundColor: conf.color, color: isWhite ? 'black' : 'white' }}
                >
                   <span className="text-2xl font-black">{total?.toLocaleString()}</span>
                   <span className="text-xs font-bold uppercase tracking-widest">{conf.label}</span>
                </button>
             )
          })}
       </div>

       {/* Main Chart Area */}
       <div className="flex-1 border-2 border-black bg-white p-4 shadow-[4px_4px_0_0_rgba(0,0,0,1)] relative overflow-hidden">
          {/* Decorative Diagonal */}
          <div className="absolute top-0 right-0 w-[100px] h-[100px] bg-black -rotate-45 translate-x-1/2 -translate-y-1/2 z-0"></div>
          
          <ResponsiveContainer width="100%" height="100%">
             <AreaChart data={processedData}>
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 12, fontWeight: 'bold', fill: 'black'}} tickFormatter={d => d.slice(5)} />
                <Tooltip 
                   contentStyle={{ backgroundColor: 'black', border: 'none', color: 'white' }}
                   itemStyle={{ color: 'white' }}
                   cursor={{ stroke: 'black', strokeWidth: 2 }}
                />
                <Area 
                   type="linear" 
                   dataKey={activeConfig.totalKey} 
                   stroke="black" 
                   strokeWidth={4} 
                   fill={activeConfig.color === '#171717' ? '#ccc' : activeConfig.color} 
                   fillOpacity={1}
                />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// --- Variant 41: Bauhaus Schematic (Engineering) ---
const BauhausSchematicEngineering = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUB-DOM', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: '#3b82f6', accent: 'border-blue-500' },
     ip: { label: 'IP-ADDR', totalKey: 'totalIps', deltaKey: 'newIp', color: '#f97316', accent: 'border-orange-500' },
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: '#eab308', accent: 'border-yellow-500' },
     site: { label: 'WEBSITES', totalKey: 'totalWebsites', deltaKey: 'newSite', color: '#22c55e', accent: 'border-green-500' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-card text-card-foreground border-2 border-primary flex flex-col h-[450px] font-mono relative">
       {/* Background Grid */}
       <div className="absolute inset-0 opacity-[0.05] pointer-events-none" 
            style={{ 
               backgroundImage: `radial-gradient(circle, var(--foreground) 1px, transparent 1px)`, 
               backgroundSize: '20px 20px' 
            }}
       ></div>

       {/* Header */}
       <div className="flex border-b-2 border-primary bg-secondary/10">
          <div className="p-3 border-r-2 border-primary flex items-center justify-center bg-primary text-primary-foreground">
             <IconServer className="w-5 h-5" />
          </div>
          <div className="flex-1 flex items-center px-4 justify-between">
             <span className="font-bold tracking-widest uppercase">Schematic.View // V.41</span>
             <div className="flex gap-2 text-[10px] text-muted-foreground">
                <span>GRID: ON</span>
                <span>SYNC: OK</span>
             </div>
          </div>
       </div>

       {/* Top Controls Row */}
       <div className="grid grid-cols-4 border-b-2 border-primary bg-secondary/5">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex flex-col p-3 border-r border-primary last:border-r-0 transition-[color,background-color,border-color,opacity,transform,box-shadow] text-left relative overflow-hidden group hover:bg-secondary/10",
                      isActive ? `bg-card ${conf.accent}` : "bg-transparent"
                   )}
                >
                   <div className="flex justify-between items-start mb-1">
                      <div className="text-[10px] font-bold text-muted-foreground group-hover:text-foreground">{conf.label}</div>
                      {isActive && (
                         <div className="w-2 h-2 rounded-full" style={{ backgroundColor: conf.color }}></div>
                      )}
                   </div>
                   <div className="text-lg font-bold tracking-tighter" style={{ color: isActive ? conf.color : 'inherit' }}>
                      {latest[conf.totalKey as keyof typeof latest]?.toLocaleString()}
                   </div>
                   {isActive && (
                      <div className="absolute bottom-0 left-0 w-full h-1" style={{ backgroundColor: conf.color }}></div>
                   )}
                </button>
             )
          })}
       </div>

       {/* Main Layout */}
       <div className="flex flex-1 overflow-hidden">
          {/* Chart Area */}
          <div className="flex-1 p-6 relative flex flex-col">
             {/* Chart Header */}
             <div className="flex justify-between items-end mb-4 border-b border-border border-dashed pb-2">
                <div className="flex items-center gap-2">
                   <div className="w-2 h-2 rounded-full animate-pulse" style={{ backgroundColor: activeConfig.color }}></div>
                   <h3 className="text-xl font-bold uppercase">{activeConfig.label} Analysis</h3>
                </div>
                <div className="text-xs font-mono">
                   DELTA: <span style={{ color: activeConfig.color }}>+{latest[activeConfig.deltaKey as keyof typeof latest]}</span>
                </div>
             </div>

             <div className="flex-1 relative">
                {/* Crosshairs */}
                <div className="absolute top-1/2 left-0 w-full h-[1px] bg-border border-t border-dashed opacity-50"></div>
                <div className="absolute left-1/2 top-0 h-full w-[1px] bg-border border-l border-dashed opacity-50"></div>

                <ResponsiveContainer width="100%" height="100%">
                   <AreaChart data={processedData}>
                      <defs>
                         <pattern id="diagonalHatch" width="4" height="4" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
                            <rect width="2" height="4" transform="translate(0,0)" fill={activeConfig.color} fillOpacity={0.1} />
                         </pattern>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" opacity={0.5} />
                      <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} tickFormatter={d => d.slice(5)} />
                      <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} />
                      <Tooltip 
                         contentStyle={{ backgroundColor: 'var(--card)', border: `2px solid ${activeConfig.color}`, borderRadius: 0 }}
                         itemStyle={{ fontFamily: 'var(--font-mono)' }}
                         cursor={{ stroke: activeConfig.color, strokeWidth: 1 }}
                      />
                      <Area 
                         type="step" 
                         dataKey={activeConfig.totalKey} 
                         stroke={activeConfig.color} 
                         strokeWidth={2} 
                         fill="url(#diagonalHatch)" 
                      />
                   </AreaChart>
                </ResponsiveContainer>
             </div>
          </div>
       </div>
    </div>
  )
})

// --- Variant 42: Bauhaus Schematic (Architect) ---
const BauhausSchematicArchitect = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('ip')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: '#3b82f6' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: '#f97316' },
     url: { label: 'Endpoints', totalKey: 'totalEndpoints', color: '#eab308' },
     site: { label: 'Websites', totalKey: 'totalWebsites', color: '#22c55e' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-card text-card-foreground border-4 border-foreground flex flex-col h-[450px] font-mono shadow-[8px_8px_0_0_rgba(0,0,0,0.2)] dark:shadow-[8px_8px_0_0_rgba(255,255,255,0.1)]">
       {/* Top Bar */}
       <div className="h-12 border-b-4 border-foreground flex items-center px-4 justify-between bg-foreground text-background">
          <span className="font-bold text-lg uppercase tracking-wider">Plan_View_42</span>
          <div className="flex gap-4 text-xs font-bold">
             {['A', 'B', 'C', 'D'].map(l => <span key={l} className="opacity-50">{l}-SEC</span>)}
          </div>
       </div>

       {/* Content */}
       <div className="flex flex-1 flex-col">
          {/* Main Display */}
          <div className="flex-1 p-6 relative">
             <div className="absolute inset-0 border-[16px] border-card pointer-events-none z-10"></div>
             
             <div className="h-full border-2 border-dashed border-foreground/30 p-4 relative">
                {/* Labels */}
                <div className="absolute top-0 left-0 -mt-3 ml-4 bg-card px-2 text-xs font-bold text-foreground">FIG 1.1</div>
                <div className="absolute bottom-0 right-0 -mb-3 mr-4 bg-card px-2 text-xs font-bold text-foreground">SCALE 1:100</div>

                <ResponsiveContainer width="100%" height="100%">
                   <BarChart data={processedData} barCategoryGap={2}>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="currentColor" opacity={0.2} />
                      <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'currentColor'}} tickFormatter={d => d.slice(5)} />
                      <Tooltip 
                         contentStyle={{ backgroundColor: 'var(--card)', border: '2px solid currentColor', borderRadius: 0, boxShadow: '4px 4px 0 currentColor' }}
                         cursor={{ fill: 'currentColor', opacity: 0.1 }}
                      />
                      <Bar 
                         dataKey={activeConfig.totalKey} 
                         fill={activeConfig.color} 
                         stroke="currentColor" 
                         strokeWidth={2}
                      />
                   </BarChart>
                </ResponsiveContainer>
             </div>
          </div>

          {/* Bottom Tabs (Moved from sidebar) */}
          <div className="h-16 border-t-4 border-foreground flex divide-x-2 divide-foreground">
             {(Object.keys(config) as Array<keyof typeof config>).map((key, idx) => {
                const conf = config[key]
                const isActive = activeTab === key
                return (
                   <button
                      key={key}
                      onClick={() => setActiveTab(key)}
                      className={cn(
                         "flex-1 flex items-center justify-center gap-3 transition-colors",
                         isActive ? "bg-secondary" : "hover:bg-secondary/20"
                      )}
                   >
                      <div className={cn(
                         "w-6 h-6 border-2 border-foreground rounded-full flex items-center justify-center font-bold text-xs",
                         isActive ? "bg-card" : "bg-transparent opacity-50"
                      )}>
                         {idx + 1}
                      </div>
                      <span className="font-bold uppercase text-xs">{conf.label}</span>
                   </button>
                )
             })}
          </div>
       </div>
    </div>
  )
})

// --- Variant 43: Bauhaus Schematic (Circuit) ---
const BauhausSchematicCircuit = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', color: '#3b82f6' },
     ip: { label: 'IPS', totalKey: 'totalIps', color: '#f97316' },
     url: { label: 'URLS', totalKey: 'totalEndpoints', color: '#eab308' },
     site: { label: 'SITES', totalKey: 'totalWebsites', color: '#22c55e' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-[#111] text-white border border-gray-800 flex flex-col h-[450px] font-mono rounded-xl overflow-hidden shadow-2xl">
       {/* Circuit Header */}
       <div className="flex bg-[#1a1a1a] border-b border-gray-800 p-1 gap-1">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 py-3 px-4 rounded-lg flex items-center justify-between transition-[color,background-color,border-color,opacity,transform,box-shadow]",
                      isActive ? "bg-white/10 shadow-inner" : "hover:bg-white/5"
                   )}
                >
                   <span className={cn("text-xs font-bold", isActive ? "text-white" : "text-gray-500")}>{conf.label}</span>
                   <div className={cn("w-2 h-2 rounded-full shadow-[0_0_10px_currentColor]", isActive ? "opacity-100 scale-125" : "opacity-30")} style={{ color: conf.color, backgroundColor: conf.color }}></div>
                </button>
             )
          })}
       </div>

       {/* Display Screen */}
       <div className="flex-1 p-6 relative">
          {/* Background Circuit Lines */}
          <svg className="absolute inset-0 w-full h-full opacity-10 pointer-events-none" xmlns="http://www.w3.org/2000/svg">
             <path d="M50 0 V450 M150 0 V450 M250 0 V450 M350 0 V450" stroke="white" strokeWidth="1" strokeDasharray="4 4" />
             <path d="M0 50 H1000 M0 150 H1000 M0 250 H1000 M0 350 H1000" stroke="white" strokeWidth="1" strokeDasharray="4 4" />
             <circle cx="50" cy="50" r="4" fill="white" />
             <circle cx="350" cy="350" r="4" fill="white" />
          </svg>

          {/* Main Chart */}
          <div className="h-full border border-gray-700 bg-black/50 rounded-lg p-4 backdrop-blur-sm relative overflow-hidden">
             <div className="absolute top-0 right-0 p-2 text-xs text-gray-500">Node: {activeConfig.label}</div>
             <div className="absolute bottom-2 left-4 text-4xl font-black tracking-tighter opacity-20" style={{ color: activeConfig.color }}>
                {latest[activeConfig.totalKey as keyof typeof latest]?.toLocaleString()}
             </div>

             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={processedData}>
                   <CartesianGrid strokeDasharray="3 3" stroke="#333" vertical={false} />
                   <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#666'}} tickFormatter={d => d.slice(5)} />
                   <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fill: '#666'}} />
                   <Tooltip 
                      contentStyle={{ backgroundColor: '#000', border: '1px solid #333', color: '#fff' }}
                      cursor={{ stroke: activeConfig.color, strokeWidth: 1 }}
                   />
                   <Line 
                      type="stepAfter" 
                      dataKey={activeConfig.totalKey} 
                      stroke={activeConfig.color} 
                      strokeWidth={2} 
                      dot={{ r: 4, fill: '#000', stroke: activeConfig.color, strokeWidth: 2 }} 
                   />
                </LineChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 44: Retro-Modular (Braun Style) ---
const RetroModular = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: '#ea580c' }, // Orange
     ip: { label: 'IP Addr', totalKey: 'totalIps', color: '#16a34a' }, // Green
     url: { label: 'Endpoints', totalKey: 'totalEndpoints', color: '#0891b2' }, // Cyan/Blue
     site: { label: 'Websites', totalKey: 'totalWebsites', color: '#4b5563' }, // Gray
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-[#f3f4f6] dark:bg-[#1f2937] text-foreground rounded-xl shadow-lg border border-border flex flex-col h-[460px] overflow-hidden">
       {/* Top Control Bar (Physical Buttons) */}
       <div className="bg-[#e5e7eb] dark:bg-[#111827] p-4 flex gap-4 items-center border-b border-border shadow-sm z-10">
          <div className="w-3 h-3 rounded-full bg-red-500 shadow-sm"></div>
          <div className="h-8 w-[1px] bg-border mx-2"></div>
          
          <div className="flex-1 flex gap-2">
             {(Object.keys(config) as Array<keyof typeof config>).map(key => {
                const conf = config[key]
                const isActive = activeTab === key
                return (
                   <button
                      key={key}
                      onClick={() => setActiveTab(key)}
                      className={cn(
                         "flex-1 h-12 rounded-md flex items-center justify-between px-4 transition-[color,background-color,border-color,opacity,transform,box-shadow] shadow-[0_2px_0_0_rgba(0,0,0,0.1)] active:shadow-none active:translate-y-[2px]",
                         isActive 
                           ? "bg-white dark:bg-[#374151] ring-1 ring-border" 
                           : "bg-[#d1d5db] dark:bg-[#1f2937] text-muted-foreground hover:bg-[#dbeafe] dark:hover:bg-[#374151]"
                      )}
                   >
                      <span className="text-xs font-bold uppercase tracking-wide">{conf.label}</span>
                      {isActive && <div className="w-2 h-2 rounded-full" style={{ backgroundColor: conf.color }}></div>}
                   </button>
                )
             })}
          </div>
       </div>

       {/* Main Display Area (Matte Screen) */}
       <div className="flex-1 p-8 bg-[#f9fafb] dark:bg-[#030712] relative">
          <div className="flex justify-between items-baseline mb-6">
             <div className="text-4xl font-light tracking-tighter" style={{ color: activeConfig.color }}>
                {latest[activeConfig.totalKey as keyof typeof latest]?.toLocaleString()}
             </div>
             <div className="text-xs text-muted-foreground font-mono uppercase">
                {"// "}{activeConfig.label}{" _SEQ_01"}
             </div>
          </div>

          <div className="h-[240px] w-full">
             <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={processedData}>
                   <defs>
                      <linearGradient id="retroGrad" x1="0" y1="0" x2="0" y2="1">
                         <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.2}/>
                         <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                      </linearGradient>
                   </defs>
                   <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                   <Tooltip 
                      contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
                   />
                   <Area 
                      type="monotone" 
                      dataKey={activeConfig.totalKey} 
                      stroke={activeConfig.color} 
                      strokeWidth={3} 
                      fill="url(#retroGrad)" 
                   />
                </AreaChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 45: Flight Monitor (Airport Style - Light) ---
const FlightMonitor = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', deltaKey: 'newSub' },
     ip: { label: 'IP ADDRESSES', totalKey: 'totalIps', deltaKey: 'newIp' },
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', deltaKey: 'newUrl' },
     site: { label: 'WEBSITES', totalKey: 'totalWebsites', deltaKey: 'newSite' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-white text-black flex flex-col h-[450px] font-sans border-2 border-black">
       {/* Top Signage Board */}
       <div className="flex border-b-2 border-black">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 h-14 flex items-center px-6 justify-between transition-colors border-r-2 border-black last:border-r-0 relative group",
                      isActive ? "bg-[#eab308] text-black" : "hover:bg-neutral-100 bg-white"
                   )}
                >
                   <span className="font-bold tracking-tight text-sm">{conf.label}</span>
                   {isActive && <div className="absolute bottom-0 left-0 w-full h-1 bg-black/20"></div>}
                   {!isActive && <span className="text-xs text-black/60 group-hover:text-black font-mono">{latest[conf.totalKey as keyof typeof latest]?.toLocaleString()}</span>}
                </button>
             )
          })}
       </div>

       {/* Main Content Area */}
       <div className="flex-1 p-8 flex flex-col relative overflow-hidden">
          {/* Background Grid Lines */}
          <div className="absolute inset-0 pointer-events-none opacity-10"
               style={{ backgroundImage: 'linear-gradient(#000 1px, transparent 1px)', backgroundSize: '100% 40px' }}>
          </div>

          <div className="flex justify-between items-end mb-8 relative z-10">
             <div>
                <div className="text-black/60 text-xs font-bold mb-1 uppercase tracking-widest">Current Status</div>
                <div className="text-5xl font-black tracking-tighter">
                   {latest[activeConfig.totalKey as keyof typeof latest]?.toLocaleString()}
                </div>
             </div>
             <div className="text-right">
                <div className="text-black/60 text-xs font-bold mb-1 uppercase tracking-widest">24h Change</div>
                <div className="text-2xl font-bold text-black flex items-center justify-end gap-2">
                   <IconTrendingUp className="w-5 h-5 text-[#eab308]" />
                   +{latest[activeConfig.deltaKey as keyof typeof latest]?.toLocaleString()}
                </div>
             </div>
          </div>

          <div className="flex-1 relative z-10">
             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={processedData}>
                   <Tooltip 
                      contentStyle={{ backgroundColor: '#fff', border: '2px solid #000', color: '#000', boxShadow: '4px 4px 0 rgba(0,0,0,0.2)' }}
                      cursor={{ stroke: '#000', strokeWidth: 1, strokeDasharray: '4 4' }}
                   />
                   <Line 
                      type="stepAfter" 
                      dataKey={activeConfig.totalKey} 
                      stroke="#000" 
                      strokeWidth={3} 
                      dot={false}
                      activeDot={{ r: 6, fill: '#eab308', stroke: '#000', strokeWidth: 2 }}
                   />
                </LineChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 46: Signal Processor (Audio/Lab Gear) ---
const SignalProcessor = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const config = {
     sub: { label: 'CH-1: SUB', totalKey: 'totalSubdomains', color: '#10b981' },
     ip: { label: 'CH-2: IP', totalKey: 'totalIps', color: '#3b82f6' },
     url: { label: 'CH-3: URL', totalKey: 'totalEndpoints', color: '#f43f5e' },
     site: { label: 'CH-4: WEB', totalKey: 'totalWebsites', color: '#8b5cf6' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-[#262626] text-white rounded-lg shadow-2xl flex flex-col h-[460px] border border-[#404040]">
       {/* Screen Area */}
       <div className="flex-1 m-4 mb-0 bg-black rounded border border-[#404040] shadow-inner relative overflow-hidden flex flex-col">
           {/* Scanlines */}
           <div className="absolute inset-0 pointer-events-none z-20 opacity-10"
                style={{ background: 'linear-gradient(rgba(18, 16, 16, 0) 50%, rgba(0, 0, 0, 0.25) 50%), linear-gradient(90deg, rgba(255, 0, 0, 0.06), rgba(0, 255, 0, 0.02), rgba(0, 0, 255, 0.06))', backgroundSize: '100% 2px, 3px 100%' }}>
           </div>
           
           <div className="flex-1 p-4 relative z-10">
              <ResponsiveContainer width="100%" height="100%">
                 <AreaChart data={processedData}>
                    <defs>
                       <linearGradient id={`grad-${activeTab}`} x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.5}/>
                          <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                       </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#333" />
                    <Tooltip 
                       contentStyle={{ backgroundColor: 'rgba(0,0,0,0.8)', border: `1px solid ${activeConfig.color}`, color: activeConfig.color, fontFamily: 'monospace' }}
                       cursor={{ stroke: activeConfig.color }}
                    />
                    <Area 
                       type="monotone" 
                       dataKey={activeConfig.totalKey} 
                       stroke={activeConfig.color} 
                       strokeWidth={2} 
                       fill={`url(#grad-${activeTab})`}
                       isAnimationActive={true}
                    />
                 </AreaChart>
              </ResponsiveContainer>
           </div>
           
           {/* On-Screen Display */}
           <div className="absolute top-4 left-4 font-mono text-xs opacity-70 z-10">
              {activeConfig.label} &gt; GAIN: +{Math.floor(Math.random() * 20)}dB
           </div>
       </div>

       {/* Bottom Control Strip */}
       <div className="h-20 bg-[#333] border-t border-[#404040] flex items-center justify-around px-4 rounded-b-lg">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className="flex flex-col items-center gap-1 group"
                >
                   {/* Toggle Switch Graphic */}
                   <div className={cn(
                      "w-10 h-6 rounded-full relative transition-colors duration-200 border border-black",
                      isActive ? "bg-[#555]" : "bg-[#222]"
                   )}>
                      <div className={cn(
                         "absolute top-1 w-4 h-4 rounded-full shadow-md transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-200",
                         isActive 
                           ? "left-[22px] bg-white shadow-[0_0_8px_rgba(255,255,255,0.8)]" 
                           : "left-1 bg-[#444]"
                      )} style={{ backgroundColor: isActive ? conf.color : undefined }}></div>
                   </div>
                   <span className={cn(
                      "text-[10px] font-bold font-mono tracking-wider transition-colors",
                      isActive ? "text-white" : "text-[#777]"
                   )}>
                      {conf.label.split(':')[1]}
                   </span>
                </button>
             )
          })}
       </div>
    </div>
  )
})

// --- Variant 47: Bauhaus Tabular (Folder Style) ---
const BauhausTabular = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('sub')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP ADDRESSES', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'WEBSITES', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full flex flex-col h-[450px]">
       {/* Top Tabs - Folder/Card Style */}
       <div className="flex pl-2 gap-1 overflow-x-auto items-end">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 min-w-[120px] py-3 px-4 rounded-t-lg border-t-2 border-l-2 border-r-2 transition-[color,background-color,border-color,opacity,transform,box-shadow] relative top-[2px] z-10",
                      isActive 
                        ? "bg-card border-primary text-foreground pb-4" 
                        : "bg-secondary/30 border-transparent text-muted-foreground hover:bg-secondary/50 pb-3"
                   )}
                >
                   <div className="flex flex-col items-start gap-1">
                      <span className="text-[10px] font-bold font-mono uppercase tracking-wider">{conf.label}</span>
                      <span className={cn("text-lg font-bold font-mono leading-none", isActive ? "text-primary" : "")}>
                         {latest[conf.totalKey as keyof typeof latest]?.toLocaleString()}
                      </span>
                   </div>
                   {isActive && (
                      <div className="absolute top-0 left-0 w-full h-1 bg-primary rounded-t-sm"></div>
                   )}
                </button>
             )
          })}
       </div>

       {/* Main Content Box */}
       <div className="flex-1 bg-card border-2 border-primary rounded-b-lg rounded-tr-lg p-6 relative z-0 shadow-sm">
          <div className="flex justify-between items-center mb-6 border-b border-border pb-2">
             <div className="flex items-center gap-2">
                 <IconActivity className="w-5 h-5 text-primary" />
                 <h3 className="text-lg font-bold font-mono uppercase tracking-tight">{activeConfig.label} TRENDS</h3>
             </div>
             <div className="flex gap-4">
                 <div className="flex items-center gap-2 text-xs font-mono">
                    <span className="w-3 h-1 bg-primary"></span>
                    <span>TOTAL</span>
                 </div>
                 <div className="flex items-center gap-2 text-xs font-mono opacity-50">
                    <span className="w-3 h-1 bg-muted-foreground"></span>
                    <span>AVG</span>
                 </div>
             </div>
          </div>

          <div className="h-[280px] w-full">
             <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={processedData}>
                   <defs>
                      <linearGradient id="folderGrad" x1="0" y1="0" x2="0" y2="1">
                         <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.2}/>
                         <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                      </linearGradient>
                   </defs>
                   <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                   <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} tickFormatter={d => d.slice(5)} dy={10} />
                   <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} />
                   <Tooltip 
                      contentStyle={{ backgroundColor: 'var(--card)', border: '2px solid var(--border)', borderRadius: '4px' }}
                      itemStyle={{ fontFamily: 'var(--font-mono)', fontWeight: 'bold' }}
                   />
                   <Area 
                      type="step" 
                      dataKey={activeConfig.totalKey} 
                      stroke={activeConfig.color} 
                      strokeWidth={3} 
                      fill="url(#folderGrad)" 
                      activeDot={{r: 6, strokeWidth: 0, fill: activeConfig.color}}
                   />
                </AreaChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// --- Variant 48: Bauhaus Split-Deck (Bottom Controls) ---
const BauhausSplitDeck = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('ip')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'SUBDOMAINS', totalKey: 'totalSubdomains', deltaKey: 'newSub', color: 'var(--chart-1)' },
     ip: { label: 'IP ADDRESSES', totalKey: 'totalIps', deltaKey: 'newIp', color: 'var(--chart-2)' },
     url: { label: 'ENDPOINTS', totalKey: 'totalEndpoints', deltaKey: 'newUrl', color: 'var(--chart-3)' },
     site: { label: 'WEBSITES', totalKey: 'totalWebsites', deltaKey: 'newSite', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-card border-2 border-border flex flex-col h-[480px]">
       {/* Main Chart Area (Top) */}
       <div className="flex-1 p-6 relative border-b-2 border-border">
          <div className="absolute top-0 left-0 bg-primary text-primary-foreground px-3 py-1 text-xs font-bold font-mono rounded-br-lg z-10">
             MONITORING: {activeConfig.label}
          </div>
          
          <ResponsiveContainer width="100%" height="100%">
             <BarChart data={processedData} margin={{top: 20}}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.5} />
                <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} tickFormatter={d => d.slice(5)} />
                <YAxis orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} />
                <Tooltip 
                   cursor={{fill: 'var(--secondary)', opacity: 0.5}}
                   contentStyle={{ backgroundColor: 'var(--card)', border: '2px solid var(--border)' }}
                />
                <Bar 
                   dataKey={activeConfig.totalKey} 
                   fill={activeConfig.color} 
                   radius={[4, 4, 0, 0]} 
                   barSize={40}
                />
             </BarChart>
          </ResponsiveContainer>
       </div>

       {/* Bottom Control Deck */}
       <div className="h-32 flex divide-x-2 divide-border bg-secondary/10">
          {(Object.keys(config) as Array<keyof typeof config>).map(key => {
             const conf = config[key]
             const isActive = activeTab === key
             return (
                <button
                   key={key}
                   onClick={() => setActiveTab(key)}
                   className={cn(
                      "flex-1 flex flex-col items-center justify-center gap-2 transition-[color,background-color,border-color,opacity,transform,box-shadow] relative group",
                      isActive ? "bg-card" : "hover:bg-secondary/20"
                   )}
                >
                   {isActive && <div className="absolute top-0 left-0 w-full h-1 bg-primary"></div>}
                   <div className={cn(
                      "w-8 h-8 rounded flex items-center justify-center border-2 transition-colors",
                      isActive ? "border-primary bg-primary/10 text-primary" : "border-muted-foreground/30 text-muted-foreground"
                   )}>
                      {key === 'sub' && <IconScan className="w-4 h-4" />}
                      {key === 'ip' && <IconServer className="w-4 h-4" />}
                      {key === 'url' && <IconWorld className="w-4 h-4" />}
                      {key === 'site' && <IconDeviceDesktop className="w-4 h-4" />}
                   </div>
                   <div className="text-center">
                      <div className="text-[10px] font-bold uppercase text-muted-foreground mb-1">{conf.label}</div>
                      <div className={cn("text-lg font-bold font-mono leading-none", isActive ? "text-foreground" : "text-muted-foreground")}>
                         {latest[conf.totalKey as keyof typeof latest]?.toLocaleString()}
                      </div>
                   </div>
                </button>
             )
          })}
       </div>
    </div>
  )
})

// --- Variant 49: Bauhaus Inline (Compact Header) ---
const BauhausInline = withRealData(({ data }) => {
  const [activeTab, setActiveTab] = useState<'sub' | 'ip' | 'url' | 'site'>('url')
  
  const processedData = useMemo(() => {
     if (data.length < 2) return []
     return data.map((curr, i) => {
        const prev = data[i-1] || curr
        return {
           ...curr,
           newSub: Math.max(0, curr.totalSubdomains - prev.totalSubdomains),
           newIp: Math.max(0, curr.totalIps - prev.totalIps),
           newUrl: Math.max(0, curr.totalEndpoints - prev.totalEndpoints),
           newSite: Math.max(0, curr.totalWebsites - prev.totalWebsites),
        }
     })
  }, [data])
  
  const latest = processedData[processedData.length - 1] || {}
  const config = {
     sub: { label: 'Subdomains', totalKey: 'totalSubdomains', color: 'var(--chart-1)' },
     ip: { label: 'IP Addresses', totalKey: 'totalIps', color: 'var(--chart-2)' },
     url: { label: 'Endpoints', totalKey: 'totalEndpoints', color: 'var(--chart-3)' },
     site: { label: 'Websites', totalKey: 'totalWebsites', color: 'var(--chart-4)' },
  }
  const activeConfig = config[activeTab]

  return (
    <div className="w-full bg-card border border-border rounded-lg shadow-sm flex flex-col h-[420px] overflow-hidden">
       {/* Inline Header & Tabs */}
       <div className="h-14 border-b border-border flex items-center justify-between px-4 bg-secondary/5">
          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded bg-primary/10 flex items-center justify-center border border-primary/20">
                <IconTrendingUp className="w-4 h-4 text-primary" />
             </div>
             <div>
                <h3 className="text-sm font-bold leading-none">ASSET ANALYTICS</h3>
                <span className="text-[10px] text-muted-foreground font-mono">LIVE DATA FEED</span>
             </div>
          </div>
          
          <div className="flex bg-secondary/20 p-1 rounded-md">
             {(Object.keys(config) as Array<keyof typeof config>).map(key => {
                const conf = config[key]
                const isActive = activeTab === key
                return (
                   <button
                      key={key}
                      onClick={() => setActiveTab(key)}
                      className={cn(
                         "px-3 py-1.5 rounded text-xs font-bold transition-[color,background-color,border-color,opacity,transform,box-shadow]",
                         isActive 
                           ? "bg-card text-foreground shadow-sm ring-1 ring-border" 
                           : "text-muted-foreground hover:text-foreground hover:bg-card/50"
                      )}
                   >
                      {conf.label}
                   </button>
                )
             })}
          </div>
       </div>

       {/* Content Area */}
       <div className="flex-1 p-6 relative">
           <div className="absolute top-6 left-6 z-10">
              <div className="text-3xl font-bold tracking-tight text-foreground">
                 {latest[activeConfig.totalKey as keyof typeof latest]?.toLocaleString()}
              </div>
              <div className="flex items-center gap-2 mt-1">
                 <span className="text-xs font-mono text-muted-foreground uppercase">{activeConfig.label} TOTAL</span>
                 <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-green-500/10 text-green-500">+12%</span>
              </div>
           </div>

           <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={processedData} margin={{top: 50}}>
                 <defs>
                    <linearGradient id="inlineGrad" x1="0" y1="0" x2="0" y2="1">
                       <stop offset="0%" stopColor={activeConfig.color} stopOpacity={0.1}/>
                       <stop offset="100%" stopColor={activeConfig.color} stopOpacity={0}/>
                    </linearGradient>
                 </defs>
                 <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" strokeOpacity={0.4} />
                 <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} tickFormatter={d => d.slice(5)} />
                 <YAxis orientation="right" tickLine={false} axisLine={false} tick={{fontSize: 10, fontFamily: 'var(--font-mono)'}} />
                 <Tooltip 
                    contentStyle={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)', borderRadius: '6px' }}
                    itemStyle={{ fontFamily: 'var(--font-mono)' }}
                 />
                 <Area 
                    type="monotone" 
                    dataKey={activeConfig.totalKey} 
                    stroke={activeConfig.color} 
                    strokeWidth={3} 
                    fill="url(#inlineGrad)" 
                 />
              </AreaChart>
           </ResponsiveContainer>
       </div>
    </div>
  )
})

void DnaBarcode
void NeonEqualizer
void MasterControlPanel
void FocusLens
void AssetLedger
void DataLab

// --- Main Page ---
export default function AdvancedAssetPulsePage() {
  return (
    <div className="p-8 max-w-5xl mx-auto space-y-12 pb-32">
      <div className="space-y-2">
         <h1 className="text-3xl font-bold tracking-tight">Advanced Asset Pulse Concepts</h1>
         <p className="text-muted-foreground">Functional prototypes focusing on data narrative, correlation, and system state.</p>
      </div>

      <div className="space-y-16">
        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">01 //</span> Forensic Timeline
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Correlates asset growth with security events. Bar chart (background) shows new vulnerabilities found, while line chart (foreground) tracks asset inventory. Red lines mark critical alerts.
              </p>
           </div>
           <ForensicTimeline />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">02 //</span> Interactive Scrubber
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Features a &quot;Brush&quot; tool at the bottom, allowing users to zoom into specific timeframes. In a real app, this would filter the asset table below.
              </p>
           </div>
           <InteractiveScrubber />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">03 //</span> System Breath
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Uses color, glow, and animation to convey the &quot;health&quot; of the system at a glance. Ambient background light shifts based on alert level.
              </p>
           </div>
           <SystemBreath />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">04 //</span> Asset Radar Scan
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Compares the relative volume of the four asset types in a snapshot. Shows the &quot;fingerprint&quot; of the attack surface.
              </p>
           </div>
           <AssetRadar />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">05 //</span> Metric Seismograph
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Isolated trend lines for each asset type, synchronized by time. Allows precise comparison of growth patterns without overlap.
              </p>
           </div>
           <MetricSeismograph />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">06 //</span> Digital Mosaic
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Heatmap visualization showing activity density. Darker cells indicate higher values. Useful for spotting growth spurts.
              </p>
           </div>
           <DigitalMosaic />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">07 //</span> Orbital Gauge
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Radial progress bars showing asset volume relative to the whole or a target. Visualizes the &quot;mass&quot; of the inventory.
              </p>
           </div>
           <OrbitalGauge />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">09 //</span> Flux Stream
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Silhouette area chart centered on the axis. Visualizes the organic flow and volume of asset discovery over time.
              </p>
           </div>
           <FluxStream />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">10 //</span> Binary Punchcard
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Scatter plot grid where density equals volume. Gives a retro &quot;data punch card&quot; feel to asset inventory logging.
              </p>
           </div>
           <BinaryPunchcard />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">11 //</span> Circuit Treemap
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Nested rectangles showing the hierarchical size of each asset category in the latest snapshot.
              </p>
           </div>
           <CircuitTreemap />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">12 //</span> Terminal Monitor
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Pure text-based visualization using ASCII-style bars. simulates a CLI tool output for a raw, hacker-centric aesthetic.
              </p>
           </div>
           <TerminalMonitor />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">14 //</span> Incremental Pulse Grid (Recommended)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Splits assets into 4 cards. Focuses on &quot;Daily New&quot; growth with micro-charts, while keeping the Total Count visible as the primary metric. Best for &quot;at a glance&quot; monitoring.
              </p>
           </div>
           <IncrementalPulseGrid />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">16 //</span> Command Deck (Top Tabs)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Top metrics act as tabs. Clicking one updates the main chart below. Classic, stable, and easy to understand.
              </p>
           </div>
           <CommandDeck />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">17 //</span> Swimlane Monitor
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Four parallel &quot;lanes,&quot; each with its own stats and chart. Allows comparing all assets simultaneously without switching views.
              </p>
           </div>
           <SwimlaneMonitor />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">18 //</span> Rich List Console
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Sidebar enhancement. List items include mini &quot;Sparkline&quot; charts, so you can see trends before you even click.
              </p>
           </div>
           <RichListConsole />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">19 //</span> Split Comparator
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Dual-selector interface. Choose any two metrics to compare directly on a shared timeline (Left Axis vs Right Axis).
              </p>
           </div>
           <SplitComparator />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">21 //</span> Crosshair Analytics
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 2x2 Dashboard grid for simultaneous monitoring. Each quadrant uses a different visualization type optimized for that specific metric.
              </p>
           </div>
           <CrosshairAnalytics />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">22 //</span> Timeline Stack
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Vertical chronological list. Each entry is a historical snapshot, visualizing the composition ratio at that specific point in time.
              </p>
           </div>
           <TimelineStack />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">23 //</span> Iso-Metric Blocks
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Stylized 3D-effect bar chart. Uses stacked bars and gradients to create depth, emphasizing the &quot;mass&quot; of accumulated assets.
              </p>
           </div>
           <IsoMetricBlocks />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">24 //</span> Signal Noise
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Oscilloscope-style visualization. High contrast, neon green on black, emphasizing the &quot;live signal&quot; nature of asset detection.
              </p>
           </div>
           <SignalNoise />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">26 //</span> Tactical Command
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 High-contrast military HUD aesthetic. Uses monospaced fonts, sharp borders, and amber/green terminal colors for a field-ops feel.
              </p>
           </div>
           <TacticalCommand />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">27 //</span> Corporate Dashboard
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Clean, modern SaaS style. Soft shadows, rounded corners, and a professional palette. Fits seamlessly into standard enterprise tools.
              </p>
           </div>
           <CorporateDashboard />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">28 //</span> Cyber Nexus
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Futuristic neon aesthetic with glowing lines and grid backgrounds. High-tech, dark mode optimized for security centers.
              </p>
           </div>
           <CyberNexus />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">29 //</span> Minimalist Zen
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Maximum whitespace, simple typography. Removes all chart junk (grids, axes) to focus purely on the data shape and primary metrics.
              </p>
           </div>
           <MinimalistZen />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">31 //</span> Orbital Selector
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Circular menu selection. Rotate the focus to update the background chart. Offers a gamified, sci-fi interface experience.
              </p>
           </div>
           <OrbitalSelector />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">32 //</span> Stacked Accordion
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Vertical list where clicking a row expands its chart. Efficient use of space, mobile-friendly, and very clear navigation.
              </p>
           </div>
           <StackedAccordion />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">33 //</span> Hover Preview
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Instant visual feedback. Simply hover over the bottom tabs to swap the main chart view. Fastest way to explore multiple metrics.
              </p>
           </div>
           <HoverPreview />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">34 //</span> Gallery Slider
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Horizontal carousel navigation with arrows. Treats each asset chart as a &quot;slide&quot; in a presentation gallery.
              </p>
           </div>
           <GallerySlider />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">35 //</span> Drill-Down Matrix
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Overview grid that allows zooming into details. Click any mini-chart to expand it to full size.
              </p>
           </div>
           <DrillDownMatrix />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">36 //</span> Bauhaus Command (The Perfect Match)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Designed to perfectly match your current Dashboard aesthetic. High-contrast primary colors, thick borders, mono typography, and geometric layout.
              </p>
           </div>
           <BauhausCommand />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">37 //</span> Industrial Panel
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Physical interface metaphor with recessed screens and beveled buttons. Feels like heavy machinery control.
              </p>
           </div>
           <IndustrialPanel />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">38 //</span> Bauhaus Grid (Typographic)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Strict grid alignment layout inspired by Swiss design but adapted to the Bauhaus Dashboard palette. High information density with clear typographic hierarchy.
              </p>
           </div>
           <SwissGrid />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">39 //</span> System Schematic
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Engineering monitor style. Dashed guidelines, measurement markers, and a technical grid overlay using the dashboard&apos;s semantic colors.
              </p>
           </div>
           <BlueprintSchematics />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">40 //</span> Constructivist Blocks
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Poster art aesthetic. Bold solid color blocks, thick black strokes, and heavy typography. Very graphical and impactful.
              </p>
           </div>
           <ConstructivistBlocks />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">41 //</span> Bauhaus Schematic (Engineering)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 A structural engineering take on the schematic view. Features measurement grids, crosshairs, and a side control panel.
              </p>
           </div>
           <BauhausSchematicEngineering />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">42 //</span> Bauhaus Schematic (Architect)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Clean, high-contrast architectural plan view. Bold strokes, clear figure labeling, and a focus on structural layout.
              </p>
           </div>
           <BauhausSchematicArchitect />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">43 //</span> Bauhaus Schematic (Circuit)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Dark mode schematic representing electronic signal flow. Uses glowing nodes and backdrop circuit traces for a high-tech feel.
              </p>
           </div>
           <BauhausSchematicCircuit />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">44 //</span> Retro-Modular (Braun Style)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Inspired by classic 1960s industrial design (Dieter Rams). Physical buttons, matte off-white finish, and clean typography.
              </p>
           </div>
           <RetroModular />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">45 //</span> Flight Monitor (Light Airport)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 High-legibility flight information display system (FIDS). White background with bold black text and yellow highlights.
              </p>
           </div>
           <FlightMonitor />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">46 //</span> Signal Processor (Lab)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Audio/Laboratory equipment aesthetic. Toggle switches at the bottom, scanlines on the screen, and LED indicators.
              </p>
           </div>
           <SignalProcessor />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">47 //</span> Bauhaus Tabular (Folder Style)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Uses a physical &quot;folder tab&quot; metaphor matching the dashboard&apos;s stroke weight and colors. The active tab seamlessly connects to the content area.
              </p>
           </div>
           <BauhausTabular />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">48 //</span> Bauhaus Split-Deck (Bottom Controls)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 A variation of the Command Deck with controls at the bottom. Large, easy-to-hit targets with clear active state indicators.
              </p>
           </div>
           <BauhausSplitDeck />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">49 //</span> Bauhaus Inline (Compact Header)
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Space-saving layout where tabs live inside the header bar. Clean, modern, and fits perfectly into smaller dashboard widgets.
              </p>
           </div>
           <BauhausInline />
        </section>

        <section className="space-y-4">
           <div className="flex flex-col gap-1">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                 <span className="text-primary font-mono">40 //</span> Constructivist Blocks
              </h2>
              <p className="text-sm text-muted-foreground max-w-2xl">
                 Poster art aesthetic. Bold solid color blocks, thick black strokes, and heavy typography. Very graphical and impactful.
              </p>
           </div>
           <ConstructivistBlocks />
        </section>
      </div>
    </div>
  )
}
