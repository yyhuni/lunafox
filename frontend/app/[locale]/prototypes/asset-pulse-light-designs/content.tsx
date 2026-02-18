"use client"

import React, { useMemo } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis, ResponsiveContainer, Area, AreaChart, Bar, BarChart, ReferenceLine, ComposedChart } from "recharts"
import { 
  IconActivity, IconWorld, 
  IconAlertTriangle,
  Zap as IconSun, IconCircleDot as IconDroplet
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { useStatisticsHistory } from "@/hooks/use-dashboard"
import type { StatisticsHistoryItem } from "@/types/dashboard.types"

// --- Helper: Fill Missing Dates ---
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

// --- Helper Types ---

// --- Helper Layout ---
function DesignCard({ title, description, children, className }: { title: string, description: string, children: React.ReactNode, className?: string }) {
  return (
    <div className={cn("flex flex-col gap-3 group", className)}>
      <div className="flex flex-col gap-1">
        <h3 className="text-sm font-bold text-slate-800 uppercase tracking-widest">{title}</h3>
        <p className="text-xs text-slate-500">{description}</p>
      </div>
      {/* Force Light Mode Container */}
      <div className="bg-white text-slate-900 rounded-lg overflow-hidden border border-slate-200 shadow-sm transition-shadow hover:shadow-md relative" data-theme="light">
        {children}
      </div>
    </div>
  )
}

// --- Data Provider Wrapper ---
// This HOC fetches the real data and passes it to the chart components
function withRealData<P extends object>(Component: React.ComponentType<P & { data: StatisticsHistoryItem[] }>) {
  return function WithRealData(props: P) {
    const { data: rawData, isLoading } = useStatisticsHistory(14)
    // Fill missing dates to ensure we have data to display, even if empty
    const data = useMemo(() => {
        if (!rawData || rawData.length === 0) return []
        return fillMissingDates(rawData, 14)
    }, [rawData])

    if (isLoading) {
        return <div className="w-full h-[240px] flex items-center justify-center text-slate-400 text-xs animate-pulse">LOADING DATA...</div>
    }
    
    if (data.length === 0) {
       // Fallback mock data if real API returns nothing (for development preview)
       const fallbackData = []
       const now = new Date()
       for(let i=0; i<14; i++) {
           const d = new Date(now)
           d.setDate(d.getDate() - (13-i))
           fallbackData.push({
               date: d.toISOString().split('T')[0],
               totalSubdomains: Math.floor(Math.random() * 50) + 100,
               totalIps: Math.floor(Math.random() * 30) + 50,
               totalEndpoints: Math.floor(Math.random() * 80) + 200,
               totalWebsites: Math.floor(Math.random() * 20) + 40,
               totalVulns: 0, totalTargets: 0, totalAssets: 0
           })
       }
       return <Component {...props} data={fallbackData} />
    }

    return <Component {...props} data={data} />
  }
}


// ==========================================
// 1. Clinical Clean (sterile laboratory)
// ==========================================
const VariantClinical = withRealData(({ data }) => {
  return (
    <div className="p-6 bg-white w-full h-[240px] relative">
      <div className="flex justify-between items-center mb-4">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-teal-400 shadow-[0_0_8px_rgba(45,212,191,0.5)]"></div>
          <span className="text-xs font-medium text-slate-400 tracking-wide uppercase">System Vitals</span>
        </div>
        <span className="text-2xl font-light text-slate-700 font-mono tracking-tighter">98.4%</span>
      </div>
      <ResponsiveContainer width="100%" height={160}>
        <AreaChart data={data}>
          <defs>
            <linearGradient id="clinicalGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#2dd4bf" stopOpacity={0.1}/>
              <stop offset="95%" stopColor="#2dd4bf" stopOpacity={0}/>
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
          <XAxis dataKey="date" hide />
          <Area type="monotone" dataKey="totalEndpoints" stroke="#2dd4bf" strokeWidth={2} fillOpacity={1} fill="url(#clinicalGradient)" />
          <Area type="monotone" dataKey="totalSubdomains" stroke="#94a3b8" strokeWidth={1} strokeDasharray="4 4" fill="transparent" />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
})

// ==========================================
// 2. Swiss Grid (Swiss Grid)
// ==========================================
const VariantSwiss = withRealData(({ data }) => {
  return (
    <div className="bg-[#f0f0f0] w-full h-[240px] relative font-sans">
      <div className="absolute top-0 left-0 p-3 bg-red-600 text-white font-bold text-xs uppercase">
        Figure 02
      </div>
      <div className="pt-12 px-6 pb-6 h-full flex flex-col">
        <h3 className="text-3xl font-black text-black tracking-tight leading-none mb-4">ASSET<br/>DISTRIBUTION</h3>
        <div className="flex-1 border-t-4 border-black pt-4">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data.slice(-7)} barGap={0}>
              <Bar dataKey="totalEndpoints" fill="#000000" />
              <Bar dataKey="totalIps" fill="#ff3e00" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
})

// ==========================================
// 3. Architect Blueprint
// ==========================================
const VariantBlueprint = withRealData(({ data }) => {
  return (
    <div className="bg-[#f4f7fa] w-full h-[240px] relative overflow-hidden p-4 border border-blue-200">
      {/* Grid Background */}
      <div className="absolute inset-0 bg-[linear-gradient(#e0e7ff_1px,transparent_1px),linear-gradient(90deg,#e0e7ff_1px,transparent_1px)] bg-[size:20px_20px]"></div>
      
      <div className="relative z-10 h-full flex flex-col">
        <div className="flex justify-between border-b-2 border-blue-900/10 pb-2 mb-2">
          <span className="font-mono text-[10px] text-blue-900/60">DWG. NO. A-702</span>
          <span className="font-mono text-[10px] text-blue-900/60">SCALE: 1:100</span>
        </div>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data}>
            <CartesianGrid strokeDasharray="5 5" stroke="#cbd5e1" />
            <Line type="step" dataKey="totalEndpoints" stroke="#1e3a8a" strokeWidth={2} dot={{r: 3, fill: '#fff', stroke: '#1e3a8a'}} />
            <ReferenceLine y={250} label={{ position: 'right', value: 'MAX_CAP', fontSize: 8, fill: '#1e3a8a' }} stroke="#1e3a8a" strokeDasharray="3 3" />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
})

// ==========================================
// 4. E-Ink Paper
// ==========================================
const VariantEInk = withRealData(({ data }) => {
  return (
    <div className="bg-[#e4e4e4] w-full h-[240px] relative p-4 font-mono">
      {/* Noise Texture */}
      <div className="absolute inset-0 opacity-10 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] mix-blend-multiply"></div>
      
      <div className="relative z-10 border-2 border-black h-full p-2 bg-[#f0f0f0] shadow-[4px_4px_0px_0px_rgba(0,0,0,1)]">
        <div className="border-b border-black mb-2 pb-1 flex justify-between items-baseline">
           <span className="font-bold text-sm">READER_MODE</span>
           <span className="text-[10px]">PAGE 1/1</span>
        </div>
        <ResponsiveContainer width="100%" height={160}>
          <LineChart data={data}>
            <XAxis dataKey="date" tickLine={false} axisLine={true} stroke="#000" tick={{fontSize: 9, fill: '#000'}} tickFormatter={d => d.slice(8)} />
            <Line type="linear" dataKey="totalEndpoints" stroke="#000" strokeWidth={1.5} dot={false} />
            <Line type="linear" dataKey="totalSubdomains" stroke="#000" strokeWidth={1.5} strokeDasharray="2 2" dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
})

// ==========================================
// 5. Financial Times (financial statements)
// ==========================================
const VariantFinancial = withRealData(({ data }) => {
  return (
    <div className="bg-[#fff1e5] w-full h-[240px] p-5 font-serif relative">
      <div className="border-t-4 border-black w-8 mb-4"></div>
      <h3 className="text-xl font-bold text-slate-900 mb-1">Market Overview</h3>
      <p className="text-xs text-slate-600 mb-4 italic">Daily asset fluctuation index</p>
      
      <ResponsiveContainer width="100%" height={120}>
        <AreaChart data={data}>
          <CartesianGrid vertical={false} stroke="#e2d6cc" />
          <XAxis dataKey="date" axisLine={false} tickLine={false} tick={{fontSize: 10, fill: '#555'}} tickFormatter={d => d.slice(8)} />
          <Area type="monotone" dataKey="totalEndpoints" stroke="#991b1b" fill="#fca5a5" fillOpacity={0.2} strokeWidth={2} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
})

// ==========================================
// 6. Soft Neumorphism
// ==========================================
const VariantNeumorphism = withRealData(({ data }) => {
  return (
    <div className="bg-[#efeff2] w-full h-[240px] p-6 flex flex-col justify-center items-center">
      <div className="w-full h-full rounded-2xl bg-[#efeff2] shadow-[10px_10px_20px_#cdcdd0,-10px_-10px_20px_#ffffff] p-4 flex flex-col">
         <div className="flex justify-between items-center mb-4 px-2">
            <span className="text-slate-500 font-medium text-xs">Activity</span>
            <div className="w-8 h-8 rounded-full bg-[#efeff2] shadow-[5px_5px_10px_#cdcdd0,-5px_-5px_10px_#ffffff] flex items-center justify-center text-slate-400">
               <IconActivity className="w-4 h-4" />
            </div>
         </div>
         <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data}>
               <Line type="natural" dataKey="totalEndpoints" stroke="#6366f1" strokeWidth={3} dot={false} />
            </LineChart>
         </ResponsiveContainer>
      </div>
    </div>
  )
})

// ==========================================
// 7. Frosted Glass Light (light-colored frosted glass)
// ==========================================
const VariantFrosted = withRealData(({ data }) => {
  return (
    <div className="bg-gradient-to-br from-blue-50 to-purple-50 w-full h-[240px] relative p-4 flex items-center justify-center overflow-hidden">
      {/* Orbs */}
      <div className="absolute top-[-20%] right-[-10%] w-40 h-40 bg-blue-300 rounded-full blur-3xl opacity-60"></div>
      <div className="absolute bottom-[-10%] left-[-10%] w-40 h-40 bg-purple-300 rounded-full blur-3xl opacity-60"></div>
      
      <div className="w-full h-full bg-white/40 backdrop-blur-xl border border-white/50 rounded-xl p-4 shadow-xl relative z-10 flex flex-col">
         <div className="text-slate-600 font-semibold text-sm mb-2">Glass Metrics</div>
         <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data.slice(-10)} barSize={8}>
               <Bar dataKey="totalEndpoints" fill="rgba(255,255,255,0.8)" radius={[4,4,4,4]} />
               <Bar dataKey="totalIps" fill="rgba(99, 102, 241, 0.6)" radius={[4,4,4,4]} />
            </BarChart>
         </ResponsiveContainer>
      </div>
    </div>
  )
})

// ==========================================
// 8. Technical Manual
// ==========================================
const VariantManual = withRealData(({ data }) => {
  return (
    <div className="bg-white w-full h-[240px] p-4 border-2 border-slate-800 relative">
       <div className="absolute top-2 right-2 bg-yellow-400 text-black text-[10px] font-bold px-1 border border-black">
          FIG 1.4
       </div>
       <div className="flex flex-col h-full">
          <div className="border-b border-slate-800 pb-2 mb-2">
             <h4 className="font-bold text-slate-900 text-sm uppercase">Endpoint Variance</h4>
             <p className="text-[10px] text-slate-600 font-mono">See reference pg. 42</p>
          </div>
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <CartesianGrid stroke="#e2e8f0" />
                <Line type="monotone" dataKey="totalEndpoints" stroke="#0f172a" strokeWidth={1} dot={{r: 2, fill: '#000'}} />
                <Line type="monotone" dataKey="totalWebsites" stroke="#64748b" strokeWidth={1} strokeDasharray="3 3" dot={false} />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// ==========================================
// 9. Ceramic Glaze
// ==========================================
const VariantCeramic = withRealData(({ data }) => {
  return (
    <div className="bg-[#fdfbf7] w-full h-[240px] p-6">
       <div className="w-full h-full bg-white rounded-[2rem] shadow-[inset_0_2px_15px_rgba(0,0,0,0.05),0_10px_20px_rgba(0,0,0,0.02)] p-5 border border-white flex flex-col items-center">
          <span className="text-slate-400 font-serif italic text-sm mb-2">organic growth</span>
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <Line type="basis" dataKey="totalEndpoints" stroke="#d4b996" strokeWidth={4} strokeLinecap="round" dot={false} />
                <Line type="basis" dataKey="totalIps" stroke="#a5b4fc" strokeWidth={4} strokeLinecap="round" dot={false} />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// ==========================================
// 10. Retro OS Light (retro system)
// ==========================================
const VariantRetroOS = withRealData(({ data }) => {
  return (
    <div className="bg-[#008080] w-full h-[240px] p-4 flex items-center justify-center">
       <div className="bg-[#c0c0c0] border-t-2 border-l-2 border-white border-b-2 border-r-2 border-black w-full h-full flex flex-col p-1 shadow-md">
          <div className="bg-[#000080] text-white px-2 py-0.5 text-xs font-bold flex justify-between items-center bg-gradient-to-r from-[#000080] to-[#1084d0]">
             <span>Performance.exe</span>
             <div className="bg-[#c0c0c0] text-black w-3 h-3 flex items-center justify-center text-[8px] border border-white border-b-black border-r-black">x</div>
          </div>
          <div className="flex-1 p-2 bg-white border border-gray-500 m-1">
             <ResponsiveContainer width="100%" height="100%">
                <BarChart data={data.slice(-8)}>
                   <CartesianGrid strokeDasharray="2 2" />
                   <Bar dataKey="totalEndpoints" fill="#000080" />
                   <Bar dataKey="totalSubdomains" fill="#008000" />
                </BarChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// ==========================================
// 11. Grid Paper
// ==========================================
const VariantGridPaper = withRealData(({ data }) => {
  return (
    <div className="bg-white w-full h-[240px] relative p-6 border border-slate-200">
       <div className="absolute inset-0 bg-[linear-gradient(#e5e5f7_1px,transparent_1px),linear-gradient(90deg,#e5e5f7_1px,transparent_1px)] bg-[size:20px_20px]"></div>
       <div className="absolute left-6 top-0 w-[1px] h-full bg-red-300 opacity-50"></div>
       
       <div className="relative z-10 h-full">
          <h3 className="font-handwriting text-slate-500 text-lg mb-2 rotate-[-2deg] origin-left">Weekly Stats</h3>
          <ResponsiveContainer width="100%" height="80%">
             <LineChart data={data}>
                <Line type="monotone" dataKey="totalEndpoints" stroke="#3b82f6" strokeWidth={2} dot={false} style={{opacity: 0.8}} />
                <Line type="monotone" dataKey="totalIps" stroke="#ef4444" strokeWidth={2} dot={false} style={{opacity: 0.8}} />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// ==========================================
// 12. Corporate Clean
// ==========================================
const VariantCorporate = withRealData(({ data }) => {
  return (
    <div className="bg-white w-full h-[240px] p-6">
       <div className="flex justify-between items-end mb-6">
          <div>
             <div className="text-sm font-semibold text-slate-900">Total Requests</div>
             <div className="text-3xl font-bold text-slate-900 tracking-tight">
                {data.length > 0 ? (data[data.length-1].totalEndpoints / 1000).toFixed(1) + 'K' : '0'}
             </div>
          </div>
          <div className="bg-green-100 text-green-700 px-2 py-1 rounded-full text-xs font-semibold">
             +12.5%
          </div>
       </div>
       <ResponsiveContainer width="100%" height={100}>
          <BarChart data={data}>
             <Bar dataKey="totalEndpoints" fill="#6366f1" radius={[2,2,0,0]} />
          </BarChart>
       </ResponsiveContainer>
    </div>
  )
})

// ==========================================
// 13. Solar Flare
// ==========================================
const VariantSolar = withRealData(({ data }) => {
  return (
    <div className="bg-[#fffbf0] w-full h-[240px] p-6 relative overflow-hidden">
       <div className="absolute top-[-50%] left-[-20%] w-[100%] h-[100%] bg-orange-200/20 blur-3xl rounded-full"></div>
       <div className="relative z-10 h-full flex flex-col justify-end">
          <div className="mb-auto text-orange-900/80 font-medium uppercase tracking-wider text-xs flex items-center gap-2">
             <IconSun className="w-4 h-4" /> Solar Metrics
          </div>
          <ResponsiveContainer width="100%" height="70%">
             <AreaChart data={data}>
                <defs>
                   <linearGradient id="solarGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.4}/>
                      <stop offset="95%" stopColor="#f59e0b" stopOpacity={0}/>
                   </linearGradient>
                </defs>
                <Area type="monotone" dataKey="totalEndpoints" stroke="#d97706" strokeWidth={2} fill="url(#solarGradient)" />
             </AreaChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// ==========================================
// 14. Ice Sheet
// ==========================================
const VariantIce = withRealData(({ data }) => {
  return (
    <div className="bg-[#f0f9ff] w-full h-[240px] p-0 relative border-t-4 border-sky-500">
       <div className="p-4 flex justify-between items-center bg-sky-50">
          <span className="font-bold text-sky-900 text-sm">Cold Storage</span>
          <IconDroplet className="w-4 h-4 text-sky-500" />
       </div>
       <div className="h-[180px] w-full p-4">
          <ResponsiveContainer width="100%" height="100%">
             <LineChart data={data}>
                <CartesianGrid stroke="#e0f2fe" vertical={false} />
                <Line type="linear" dataKey="totalEndpoints" stroke="#0ea5e9" strokeWidth={3} dot={{r: 4, fill: '#fff', strokeWidth: 2}} />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// ==========================================
// 15. Code Editor Light
// ==========================================
const VariantCodeEditor = withRealData(({ data }) => {
  return (
    <div className="bg-[#fdfdfd] w-full h-[240px] font-mono text-xs border border-slate-200 flex flex-col">
       <div className="bg-[#f3f3f3] px-4 py-2 text-slate-500 text-[10px] flex gap-4 border-b border-slate-200">
          <span>metrics.json</span>
          <span className="text-slate-300">|</span>
          <span>UTF-8</span>
       </div>
       <div className="flex-1 p-4 relative overflow-hidden">
          <div className="absolute left-0 top-4 bottom-0 w-8 text-right pr-2 text-slate-300 border-r border-slate-100 select-none leading-5">
             {Array.from({length: 8}).map((_, i) => <div key={i}>{i+1}</div>)}
          </div>
          <div className="ml-8 h-full">
             <div className="text-purple-600 mb-2">import <span className="text-blue-600">Stats</span> from <span className="text-orange-600">&apos;./core&apos;</span>;</div>
             <div className="h-[100px] w-full mt-4">
                <ResponsiveContainer width="100%" height="100%">
                   <LineChart data={data}>
                      <Line type="step" dataKey="totalEndpoints" stroke="#059669" strokeWidth={2} dot={false} />
                      <Line type="step" dataKey="totalIps" stroke="#d97706" strokeWidth={2} dot={false} />
                   </LineChart>
                </ResponsiveContainer>
             </div>
          </div>
       </div>
    </div>
  )
})

// ==========================================
// 16. Scientific Journal
// ==========================================
const VariantJournal = withRealData(({ data }) => {
  return (
    <div className="bg-white w-full h-[240px] p-5 font-serif border border-slate-200">
       <div className="text-center mb-4 border-b-2 border-black pb-2">
          <h4 className="font-bold text-lg uppercase">Table III</h4>
          <p className="text-[10px] italic">Longitudinal analysis of subdomain propagation</p>
       </div>
       <div className="flex gap-4 h-[120px]">
          <div className="w-full">
             <ResponsiveContainer width="100%" height="100%">
                <ComposedChart data={data}>
                   <CartesianGrid strokeDasharray="1 1" stroke="#000" strokeOpacity={0.1} />
                   <XAxis dataKey="date" tick={{fontSize: 8, fontFamily: 'serif'}} tickFormatter={d => d.slice(8)} />
                   <YAxis tick={{fontSize: 8, fontFamily: 'serif'}} />
                   <Bar dataKey="totalEndpoints" fill="#ccc" barSize={10} />
                   <Line type="monotone" dataKey="totalSubdomains" stroke="#000" strokeWidth={1.5} dot={{r: 2, fill: '#000'}} />
                </ComposedChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// ==========================================
// 17. Metro UI (tile style)
// ==========================================
const VariantMetro = withRealData(({ data }) => {
  const latest = data.length > 0 ? data[data.length-1] : { totalSubdomains: 0, totalVulns: 0 }
  return (
    <div className="w-full h-[240px] grid grid-cols-2 grid-rows-2 gap-1 bg-white">
       <div className="bg-[#00a4ef] p-3 text-white flex flex-col justify-between">
          <IconWorld className="w-6 h-6" />
          <div>
             <div className="text-2xl font-bold">{latest.totalSubdomains}</div>
             <div className="text-xs opacity-80">Domains</div>
          </div>
       </div>
       <div className="bg-[#ffb900] p-3 text-white flex flex-col justify-between">
          <IconAlertTriangle className="w-6 h-6" />
          <div>
             <div className="text-2xl font-bold">{latest.totalVulns}</div>
             <div className="text-xs opacity-80">Alerts</div>
          </div>
       </div>
       <div className="col-span-2 bg-[#f25022] p-3 relative">
          <div className="absolute top-3 left-3 text-white text-xs font-bold uppercase">Activity Trend</div>
          <div className="absolute bottom-0 left-0 w-full h-[60%] px-2">
             <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data}>
                   <Area type="monotone" dataKey="totalEndpoints" stroke="white" fill="rgba(255,255,255,0.3)" strokeWidth={2} />
                </AreaChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// ==========================================
// 18. Japanese Minimal (Japanese minimalist)
// ==========================================
const VariantJapanese = withRealData(({ data }) => {
  return (
    <div className="bg-[#fcfaf5] w-full h-[240px] p-6 relative">
       <div className="absolute right-4 top-4 bottom-4 w-8 border-l border-stone-300 flex flex-col items-center py-2 text-stone-400 text-xs font-serif writing-vertical-rl tracking-widest">
          データ分析
       </div>
       <div className="mr-12 h-full flex flex-col justify-end">
          <div className="mb-auto">
             <div className="w-8 h-8 rounded-full bg-red-600 mb-2"></div>
             <h3 className="text-stone-800 font-bold">Zen Metrics</h3>
          </div>
          <div className="h-[100px] border-b border-stone-800 pb-1">
             <ResponsiveContainer width="100%" height="100%">
                <BarChart data={data.slice(-8)}>
                   <Bar dataKey="totalEndpoints" fill="#292524" radius={[2,2,0,0]} barSize={4} />
                   <Bar dataKey="totalSubdomains" fill="#a8a29e" radius={[2,2,0,0]} barSize={4} />
                </BarChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})

// ==========================================
// 19. Isometric Light (Isometric – Pseudo 3D)
// ==========================================
const VariantIsometric = withRealData(({ data }) => {
  return (
    <div className="bg-slate-100 w-full h-[240px] flex items-center justify-center overflow-hidden perspective-1000">
       <div className="transform rotate-x-12 rotate-z-[-4deg] skew-y-12 bg-white shadow-2xl p-4 w-[80%] h-[60%] border border-slate-200 rounded-lg">
          <div className="flex justify-between items-center mb-4 border-b border-slate-100 pb-2">
             <div className="flex gap-1">
                <div className="w-2 h-2 rounded-full bg-red-400"></div>
                <div className="w-2 h-2 rounded-full bg-yellow-400"></div>
                <div className="w-2 h-2 rounded-full bg-green-400"></div>
             </div>
          </div>
          <ResponsiveContainer width="100%" height="80%">
             <LineChart data={data}>
                <Line type="monotone" dataKey="totalEndpoints" stroke="#3b82f6" strokeWidth={3} dot={false} />
                <Area type="monotone" dataKey="totalEndpoints" fill="#dbeafe" stroke="none" />
             </LineChart>
          </ResponsiveContainer>
       </div>
    </div>
  )
})

// ==========================================
// 20. Wireframe (wireframe)
// ==========================================
const VariantWireframe = withRealData(({ data }) => {
  return (
    <div className="bg-white w-full h-[240px] p-4 relative">
       {/* Crosshairs */}
       <div className="absolute top-4 left-4 w-4 h-4 border-t border-l border-blue-500"></div>
       <div className="absolute top-4 right-4 w-4 h-4 border-t border-r border-blue-500"></div>
       <div className="absolute bottom-4 left-4 w-4 h-4 border-b border-l border-blue-500"></div>
       <div className="absolute bottom-4 right-4 w-4 h-4 border-b border-r border-blue-500"></div>
       
       <div className="h-full w-full border border-dashed border-blue-200 p-2 flex items-center justify-center">
          <div className="w-full h-full relative">
             <div className="absolute inset-0 flex items-center justify-center text-blue-100 text-[100px] font-bold opacity-20 pointer-events-none">X</div>
             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={data}>
                   <CartesianGrid stroke="#e2e8f0" />
                   <Line type="linear" dataKey="totalEndpoints" stroke="#3b82f6" strokeWidth={1} dot={{r: 3, stroke: '#3b82f6', fill: 'white'}} />
                </LineChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
})


export default function AssetPulseLightDesignsPage() {
  return (
    <div className="p-8 max-w-7xl mx-auto space-y-12 pb-32 bg-slate-50 min-h-screen text-slate-900" data-theme="light">
      <div className="space-y-2">
         <h1 className="text-3xl font-bold tracking-tight text-slate-900">Asset Pulse: Light Mode Collection (Live Data)</h1>
         <p className="text-slate-500">20 distinct high-fidelity visualization styles connected to real asset history data.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8">
        <DesignCard title="01 // Clinical Clean" description="Sterile, teal accents, precise data.">
           <VariantClinical />
        </DesignCard>

        <DesignCard title="02 // Swiss Grid" description="Helvetica, high contrast, international style.">
           <VariantSwiss />
        </DesignCard>

        <DesignCard title="03 // Architect Blueprint" description="Construction aesthetic, grid lines, scale.">
           <VariantBlueprint />
        </DesignCard>
        
        <DesignCard title="04 // E-Ink Paper" description="High legibility, noise texture, monochrome.">
           <VariantEInk />
        </DesignCard>

        <DesignCard title="05 // Financial Times" description="Serif fonts, salmon background, classic.">
           <VariantFinancial />
        </DesignCard>

        <DesignCard title="06 // Soft Neumorphism" description="Subtle shadows, depth, soft tactile feel.">
           <VariantNeumorphism />
        </DesignCard>

        <DesignCard title="07 // Frosted Glass" description="Translucent layers, colorful blur.">
           <VariantFrosted />
        </DesignCard>

        <DesignCard title="08 // Technical Manual" description="Instructional design, heavy strokes.">
           <VariantManual />
        </DesignCard>

        <DesignCard title="09 // Ceramic Glaze" description="Organic curves, warm whites, soft finish.">
           <VariantCeramic />
        </DesignCard>

        <DesignCard title="10 // Retro OS" description="Windows 95/System 7 nostalgia.">
           <VariantRetroOS />
        </DesignCard>

        <DesignCard title="11 // Grid Paper" description="Hand-drawn feel on graph paper.">
           <VariantGridPaper />
        </DesignCard>

        <DesignCard title="12 // Corporate Clean" description="Standard SaaS metric card.">
           <VariantCorporate />
        </DesignCard>

        <DesignCard title="13 // Solar Flare" description="Warm gradients, energetic.">
           <VariantSolar />
        </DesignCard>

        <DesignCard title="14 // Ice Sheet" description="Cold storage, sharp, blue tones.">
           <VariantIce />
        </DesignCard>

        <DesignCard title="15 // Code Editor" description="IDE inspired syntax highlighting.">
           <VariantCodeEditor />
        </DesignCard>

        <DesignCard title="16 // Scientific Journal" description="Academic rigor, Times New Roman.">
           <VariantJournal />
        </DesignCard>

        <DesignCard title="17 // Metro UI" description="Flat blocks, vibrant colors.">
           <VariantMetro />
        </DesignCard>

        <DesignCard title="18 // Japanese Minimal" description="Zen garden aesthetic, vertical text.">
           <VariantJapanese />
        </DesignCard>

        <DesignCard title="19 // Isometric" description="3D perspective, floating cards.">
           <VariantIsometric />
        </DesignCard>

        <DesignCard title="20 // Wireframe" description="Blue pencil structure, prototyping look.">
           <VariantWireframe />
        </DesignCard>
      </div>
    </div>
  )
}
