"use client"

import React, { useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis, ResponsiveContainer, Area, AreaChart, Bar, BarChart, Tooltip } from "recharts"
import { 
  IconActivity, IconServer, IconWorld, 
  Monitor as IconDeviceDesktop, Signal as IconWifi
} from "@/components/icons"
import { cn } from "@/lib/utils"
import { motion } from "framer-motion"
import { sanitizeBarShapeProps } from "./bar-shape"

// --- Mock Data ---
const generateData = (days: number) => {
  const data = []
  const baseDate = new Date()
  baseDate.setDate(baseDate.getDate() - days)
  
  for (let i = 0; i < days; i++) {
    const date = new Date(baseDate)
    date.setDate(date.getDate() + i)
    data.push({
      date: date.toISOString().split('T')[0],
      subdomains: Math.floor(Math.random() * 50) + 120,
      ips: Math.floor(Math.random() * 30) + 80,
      endpoints: Math.floor(Math.random() * 80) + 200,
      websites: Math.floor(Math.random() * 20) + 40,
    })
  }
  return data
}

const MOCK_DATA = generateData(14)

// --- Shared Types ---
type SeriesKey = 'subdomains' | 'ips' | 'endpoints' | 'websites'
const ALL_SERIES: SeriesKey[] = ['subdomains', 'ips', 'endpoints', 'websites']

const SERIES_CONFIG = {
  subdomains: { label: 'SUBDOMAINS', color: '#3b82f6', icon: IconWorld },
  ips: { label: 'IP ADDRESSES', color: '#f97316', icon: IconWifi },
  endpoints: { label: 'ENDPOINTS', color: '#eab308', icon: IconServer },
  websites: { label: 'WEBSITES', color: '#22c55e', icon: IconDeviceDesktop },
}

// --- Variant 1: Bauhaus Tactical (The Default Refined) ---
function VariantBauhausTactical() {
  const [activeSeries, setActiveSeries] = useState<Set<SeriesKey>>(new Set(ALL_SERIES))
  
  const toggleSeries = (key: SeriesKey) => {
    const next = new Set(activeSeries)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    setActiveSeries(next)
  }

  return (
    <div className="w-full border border-border bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border bg-secondary/30 px-4 py-3">
        <div className="flex items-center gap-2 text-muted-foreground">
          <IconActivity className="h-4 w-4" />
          <span className="text-[11px] font-mono tracking-[0.2em] font-bold">ASSET PULSE // TACTICAL</span>
        </div>
        <div className="flex items-center gap-2">
           <div className="h-1.5 w-1.5 rounded-full bg-success animate-pulse" />
           <span className="text-[10px] font-mono text-success tracking-widest">LIVE</span>
        </div>
      </div>

      {/* Chart Area */}
      <div className="relative h-[240px] w-full p-4 bg-[linear-gradient(var(--border)_1px,transparent_1px),linear-gradient(90deg,var(--border)_1px,transparent_1px)] bg-[size:40px_40px]">
         {/* Scanline Effect */}
        <motion.div 
          className="absolute inset-0 z-0 pointer-events-none border-r border-primary/20 bg-gradient-to-r from-transparent to-primary/5"
          animate={{ x: ['0%', '100%'] }}
          transition={{ duration: 3, repeat: Infinity, ease: "linear" }}
        />
        
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={MOCK_DATA}>
            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
            <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} tickFormatter={d => d.slice(5)} />
            <YAxis tickLine={false} axisLine={false} tick={{fontSize: 10, fill: 'var(--muted-foreground)'}} />
            <Tooltip 
              contentStyle={{ borderRadius: 0, border: '1px solid var(--border)', background: 'var(--card)' }}
              itemStyle={{ fontSize: 12, fontFamily: 'var(--font-mono)' }}
              labelStyle={{ fontSize: 10, marginBottom: 8, color: 'var(--muted-foreground)' }}
            />
            {ALL_SERIES.map(key => activeSeries.has(key) && (
              <Line 
                key={key} 
                type="step" 
                dataKey={key} 
                stroke={SERIES_CONFIG[key].color} 
                strokeWidth={2} 
                dot={false}
                activeDot={{ r: 4, strokeWidth: 0, fill: SERIES_CONFIG[key].color }} 
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Controls */}
      <div className="grid grid-cols-4 border-t border-border divide-x divide-border">
        {ALL_SERIES.map(key => {
          const isActive = activeSeries.has(key)
          const config = SERIES_CONFIG[key]
          const Icon = config.icon
          
          return (
            <button
              key={key}
              onClick={() => toggleSeries(key)}
              className={cn(
                "group relative flex flex-col gap-2 p-3 text-left transition-colors hover:bg-secondary/50",
                isActive && "bg-secondary/30"
              )}
            >
              {isActive && (
                <div 
                  className="absolute left-0 top-0 bottom-0 w-[3px]" 
                  style={{ backgroundColor: config.color }} 
                />
              )}
              <div className="flex items-center justify-between w-full">
                <span className={cn(
                  "text-[10px] tracking-widest font-bold",
                  isActive ? "text-foreground" : "text-muted-foreground"
                )}>
                  {config.label}
                </span>
                <Icon className={cn("h-3 w-3", isActive ? "text-foreground" : "text-muted-foreground")} />
              </div>
              <span className={cn(
                "text-lg font-mono font-medium",
                isActive ? "text-foreground" : "text-muted-foreground"
              )}>
                {MOCK_DATA[MOCK_DATA.length-1][key]}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

// --- Variant 2: Cyber Terminal ---
function VariantCyberTerminal() {
  return (
    <div className="w-full bg-black border border-primary/30 p-1 relative overflow-hidden group">
      {/* Corner Accents */}
      <div className="absolute top-0 left-0 w-4 h-4 border-t-2 border-l-2 border-primary z-10" />
      <div className="absolute top-0 right-0 w-4 h-4 border-t-2 border-r-2 border-primary z-10" />
      <div className="absolute bottom-0 left-0 w-4 h-4 border-b-2 border-l-2 border-primary z-10" />
      <div className="absolute bottom-0 right-0 w-4 h-4 border-b-2 border-r-2 border-primary z-10" />

      <div className="bg-primary/5 border border-primary/20 h-full p-4 relative">
        <div className="flex justify-between items-end mb-4 border-b border-primary/30 pb-2">
          <div>
            <div className="text-primary text-xs font-mono mb-1 animate-pulse">SYSTEM_MONITOR_V2.0</div>
            <div className="text-primary/70 text-[10px] font-mono">TARGET: LUNAFOX_MAIN_NODE</div>
          </div>
          <div className="text-right">
             <div className="text-primary/50 text-[10px] font-mono">REFRESH_RATE: 60Hz</div>
             <div className="text-primary font-bold font-mono">ONLINE</div>
          </div>
        </div>

        <div className="h-[200px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={MOCK_DATA}>
              <defs>
                <linearGradient id="cyberGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--primary)" stopOpacity={0.3}/>
                  <stop offset="95%" stopColor="var(--primary)" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid stroke="var(--primary)" strokeOpacity={0.1} vertical={false} />
              <XAxis dataKey="date" hide />
              <Tooltip 
                contentStyle={{ backgroundColor: '#000', borderColor: 'var(--primary)', color: 'var(--primary)' }}
                itemStyle={{ color: 'var(--primary)' }}
              />
              <Area 
                type="monotone" 
                dataKey="endpoints" 
                stroke="var(--primary)" 
                fillOpacity={1} 
                fill="url(#cyberGradient)" 
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
        
        {/* Glitch text decoration */}
        <div className="absolute bottom-2 right-4 text-[10px] text-primary/40 font-mono">
           0x3F2A...BC91
        </div>
      </div>
    </div>
  )
}

// --- Variant 3: Minimalist Spark ---
function VariantMinimalistSpark() {
  const [hovered, setHovered] = useState<number | null>(null)

  return (
    <div className="w-full bg-card p-6 rounded-xl border border-border/50 shadow-sm">
      <div className="flex items-center justify-between mb-8">
        <div>
           <h3 className="text-sm font-medium text-muted-foreground tracking-wide">ASSET GROWTH</h3>
           <div className="text-3xl font-light mt-1 text-foreground">
             {MOCK_DATA.reduce((acc, curr) => acc + curr.endpoints, 0).toLocaleString()}
             <span className="text-sm text-muted-foreground ml-2 font-normal">total endpoints</span>
           </div>
        </div>
        <div className="flex gap-2">
          <span className="h-2 w-2 rounded-full bg-foreground"></span>
          <span className="h-2 w-2 rounded-full bg-muted"></span>
        </div>
      </div>

      <div className="h-[120px] w-full">
         <ResponsiveContainer width="100%" height="100%">
            <BarChart data={MOCK_DATA} barGap={2} onMouseMove={(state) => {
              if (state.activeTooltipIndex !== undefined) setHovered(state.activeTooltipIndex)
            }} onMouseLeave={() => setHovered(null)}>
              <Bar 
                dataKey="endpoints" 
                fill="var(--foreground)" 
                radius={[2, 2, 0, 0]}
                fillOpacity={0.2}
                shape={(props: unknown) => {
                  const barProps = sanitizeBarShapeProps(props)
                  const isHovered = barProps.index === hovered
                  return (
                    <rect 
                       x={barProps.x}
                       y={barProps.y}
                       width={barProps.width}
                       height={barProps.height}
                       rx={2}
                       ry={2}
                       fill={isHovered ? "var(--foreground)" : "var(--muted)"} 
                       className="transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300"
                    />
                  )
                }}
              />
            </BarChart>
         </ResponsiveContainer>
      </div>
    </div>
  )
}

// --- Variant 4: Radar Scope ---
function VariantRadarScope() {
  return (
    <div className="w-full aspect-[2/1] bg-black rounded-lg overflow-hidden relative border border-emerald-900/50 shadow-[0_0_30px_rgba(16,185,129,0.1)]">
      {/* Grid */}
      <div className="absolute inset-0 bg-[radial-gradient(circle,rgba(16,185,129,0.2)_1px,transparent_1px)] bg-[size:20px_20px] opacity-20"></div>
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="w-[80%] h-[80%] border border-emerald-500/20 rounded-full"></div>
        <div className="w-[60%] h-[60%] border border-emerald-500/20 rounded-full absolute"></div>
        <div className="w-[40%] h-[40%] border border-emerald-500/20 rounded-full absolute"></div>
        <div className="w-[20%] h-[20%] border border-emerald-500/20 rounded-full absolute"></div>
      </div>
      
      {/* Radar Line */}
      <motion.div 
        className="absolute top-1/2 left-1/2 w-[50%] h-[2px] bg-emerald-500/50 origin-left"
        style={{ boxShadow: '0 0 10px 2px rgba(16,185,129,0.5)' }}
        animate={{ rotate: 360 }}
        transition={{ duration: 4, repeat: Infinity, ease: "linear" }}
      />

      <div className="absolute inset-0 p-6 flex flex-col justify-between pointer-events-none">
        <div className="flex justify-between text-emerald-500 font-mono text-xs">
           <span>R-SCAN: ACTIVE</span>
           <span>SEC-LVL: 4</span>
        </div>
        <div className="text-emerald-500/50 font-mono text-[10px]">
           DETECTED: 1,240 ASSETS<br/>
           LAST PING: 4ms
        </div>
      </div>
      
      {/* Chart Overlay */}
      <div className="absolute bottom-0 left-0 right-0 h-[100px] opacity-60">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={MOCK_DATA}>
            <Line type="monotone" dataKey="ips" stroke="#10b981" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}

// --- Variant 5: Industrial Panel ---
function VariantIndustrialPanel() {
  return (
    <div className="w-full bg-[#e5e5e5] p-2 rounded-sm border-t border-l border-white border-b border-r border-gray-400 shadow-md">
       <div className="bg-[#d4d4d4] border border-gray-500 p-3">
          <div className="flex justify-between items-center mb-3">
             <div className="text-xs font-bold text-gray-700 uppercase bg-gray-300 px-2 py-1 border border-gray-400 shadow-inner">Panel A-12</div>
             <div className="h-3 w-3 rounded-full bg-red-500 border border-red-700 shadow-[0_0_5px_rgba(239,68,68,0.8)] animate-pulse"></div>
          </div>
          
          <div className="bg-black border-4 border-gray-600 rounded-lg p-2 h-[180px] relative shadow-inner">
             <div className="absolute inset-0 bg-[linear-gradient(rgba(0,255,0,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(0,255,0,0.05)_1px,transparent_1px)] bg-[size:10px_10px]"></div>
             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={MOCK_DATA}>
                   <CartesianGrid stroke="#333" strokeDasharray="2 2" />
                   <Line type="stepAfter" dataKey="endpoints" stroke="#00ff00" strokeWidth={2} dot={false} />
                   <Tooltip contentStyle={{ backgroundColor: '#000', border: '1px solid #00ff00', color: '#00ff00', fontFamily: 'monospace' }} />
                </LineChart>
             </ResponsiveContainer>
          </div>

          <div className="flex gap-4 mt-3">
             {['PWR', 'NET', 'IO'].map(label => (
                <div key={label} className="flex flex-col items-center gap-1">
                   <div className="w-8 h-12 bg-gradient-to-b from-gray-200 to-gray-400 rounded-sm border border-gray-500 shadow-md flex items-center justify-center active:translate-y-0.5 cursor-pointer">
                      <div className="w-4 h-8 bg-gray-300 border border-gray-400 rounded-sm"></div>
                   </div>
                   <span className="text-[10px] font-bold text-gray-600">{label}</span>
                </div>
             ))}
          </div>
       </div>
    </div>
  )
}

// --- Variant 6: Data Stream ---
function VariantDataStream() {
  return (
    <div className="w-full flex h-[260px] border border-border bg-card">
      <div className="w-1/3 border-r border-border p-4 flex flex-col justify-between bg-muted/20">
         <div>
            <div className="text-xs text-muted-foreground mb-1 uppercase tracking-wider">Total Assets</div>
            <div className="text-3xl font-bold text-foreground font-mono">2,491</div>
         </div>
         <div className="space-y-4">
            {ALL_SERIES.map(key => (
               <div key={key} className="flex justify-between items-center text-sm">
                  <span className="text-muted-foreground uppercase text-[10px] tracking-wide">{key}</span>
                  <span className="font-mono">{MOCK_DATA[MOCK_DATA.length-1][key]}</span>
               </div>
            ))}
         </div>
         <div className="text-[10px] text-muted-foreground mt-4">
            Updated: 20s ago
         </div>
      </div>
      <div className="w-2/3 p-4 relative">
         <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={MOCK_DATA}>
               <defs>
                  <linearGradient id="streamGradient" x1="0" y1="0" x2="1" y2="0">
                     <stop offset="0%" stopColor="var(--primary)" stopOpacity={0.4}/>
                     <stop offset="100%" stopColor="var(--secondary)" stopOpacity={0.1}/>
                  </linearGradient>
               </defs>
               <CartesianGrid vertical={false} strokeDasharray="3 3" stroke="var(--border)" />
               <Area type="natural" dataKey="endpoints" stroke="var(--primary)" fill="url(#streamGradient)" strokeWidth={3} />
            </AreaChart>
         </ResponsiveContainer>
      </div>
    </div>
  )
}

// --- Variant 7: Split Cards ---
function VariantSplitCards() {
  return (
    <div className="grid grid-cols-2 gap-3 w-full">
      {ALL_SERIES.map(key => {
        const config = SERIES_CONFIG[key]
        return (
           <div key={key} className="bg-card border border-border p-3 flex flex-col justify-between h-[120px]">
              <div className="flex justify-between items-start">
                 <span className="text-[10px] font-bold text-muted-foreground uppercase">{config.label}</span>
                 <config.icon className="h-4 w-4 text-muted-foreground" />
              </div>
              <div className="h-[50px] w-full">
                 <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={MOCK_DATA}>
                       <Line type="monotone" dataKey={key} stroke={config.color} strokeWidth={2} dot={false} />
                    </LineChart>
                 </ResponsiveContainer>
              </div>
              <div className="text-xl font-mono font-medium">{MOCK_DATA[MOCK_DATA.length-1][key]}</div>
           </div>
        )
      })}
    </div>
  )
}

// --- Variant 8: Holographic ---
function VariantHolographic() {
  return (
    <div className="w-full h-[280px] rounded-2xl bg-gradient-to-br from-indigo-900/90 to-purple-900/90 p-1 relative overflow-hidden backdrop-blur-xl shadow-2xl">
       <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20"></div>
       <div className="h-full w-full bg-white/5 rounded-xl border border-white/10 p-6 flex flex-col relative z-10">
          <div className="flex justify-between items-center mb-6">
             <h3 className="text-white/80 font-light tracking-widest text-sm">HOLOGRAM_VIEW</h3>
             <div className="px-3 py-1 bg-white/10 rounded-full text-xs text-white/90 backdrop-blur-md">Live</div>
          </div>
          
          <div className="flex-1 relative">
             <div className="absolute inset-0 bg-gradient-to-t from-indigo-500/20 to-transparent blur-2xl"></div>
             <ResponsiveContainer width="100%" height="100%">
                <LineChart data={MOCK_DATA}>
                   <Line type="monotone" dataKey="endpoints" stroke="#a78bfa" strokeWidth={4} dot={false} style={{ filter: 'drop-shadow(0 0 10px rgba(167, 139, 250, 0.7))' }} />
                   <Line type="monotone" dataKey="subdomains" stroke="#60a5fa" strokeWidth={2} dot={false} strokeDasharray="5 5" style={{ filter: 'drop-shadow(0 0 8px rgba(96, 165, 250, 0.5))' }} />
                </LineChart>
             </ResponsiveContainer>
          </div>
       </div>
    </div>
  )
}

// --- Variant 9: High Contrast (E-Ink) ---
function VariantHighContrast() {
  return (
    <div className="w-full bg-white border-4 border-black p-4 font-mono text-black">
      <div className="border-b-4 border-black pb-2 mb-4 flex justify-between items-end">
         <h1 className="text-2xl font-black uppercase italic">ASSET_LOG</h1>
         <div className="text-xs font-bold">VOL. 24</div>
      </div>
      
      <div className="h-[180px] w-full border-2 border-black border-dashed p-2">
         <ResponsiveContainer width="100%" height="100%">
            <BarChart data={MOCK_DATA}>
               <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fill: 'black', fontSize: 10}} tickFormatter={d => d.slice(8)} />
               <Bar dataKey="endpoints" fill="black" />
            </BarChart>
         </ResponsiveContainer>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-4">
         <div className="bg-black text-white p-2 text-center font-bold text-xl">
            {MOCK_DATA[MOCK_DATA.length-1].endpoints}
         </div>
         <div className="border-2 border-black p-2 text-center font-bold text-xl">
            {MOCK_DATA[MOCK_DATA.length-1].subdomains}
         </div>
      </div>
    </div>
  )
}

// --- Variant 10: Vertical Stack ---
function VariantVerticalStack() {
  return (
    <div className="w-full flex gap-4 bg-background">
      <div className="w-[120px] flex flex-col gap-2">
         {ALL_SERIES.map(key => (
            <div key={key} className="bg-secondary p-3 rounded-none border border-border border-l-4 border-l-transparent hover:border-l-primary transition-[color,background-color,border-color,opacity,transform,box-shadow] cursor-pointer group">
               <div className="text-[10px] text-muted-foreground uppercase mb-1">{key.slice(0,3)}</div>
               <div className="text-lg font-bold group-hover:text-primary transition-colors">{MOCK_DATA[MOCK_DATA.length-1][key]}</div>
            </div>
         ))}
      </div>
      <div className="flex-1 border border-border bg-card p-4 relative">
         <div className="absolute top-2 right-2 flex gap-1">
            {[1,2,3].map(i => <div key={i} className="w-1 h-1 bg-muted-foreground/30 rounded-full"></div>)}
         </div>
         <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={MOCK_DATA}>
               <defs>
                  <pattern id="pattern-lines" x="0" y="0" width="10" height="10" patternUnits="userSpaceOnUse">
                     <path d="M0 10L10 0" stroke="currentColor" strokeWidth="0.5" className="text-muted-foreground/20"/>
                  </pattern>
               </defs>
               <CartesianGrid vertical={false} stroke="var(--border)" />
               <Area type="step" dataKey="endpoints" stroke="var(--primary)" fill="url(#pattern-lines)" strokeWidth={2} />
            </AreaChart>
         </ResponsiveContainer>
      </div>
    </div>
  )
}

// --- Main Page ---
export default function AssetPulseDesignsPage() {
  return (
    <div className="p-8 max-w-7xl mx-auto space-y-12 pb-32">
      <div className="space-y-2">
         <h1 className="text-3xl font-bold tracking-tight">Asset Pulse Design Studies</h1>
         <p className="text-muted-foreground">Exploration of 10 different visualization styles for the dashboard asset monitor.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-12">
        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">01 // Bauhaus Tactical (Recommended)</h2>
           <p className="text-sm text-muted-foreground">Refined version of current design. Strict grid, modular controls, tactical aesthetic.</p>
           <VariantBauhausTactical />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">02 // Cyber Terminal</h2>
           <p className="text-sm text-muted-foreground">High-immersion sci-fi interface. Glitch effects, heavy borders.</p>
           <VariantCyberTerminal />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">03 // Minimalist Spark</h2>
           <p className="text-sm text-muted-foreground">Clean, airy, focus on data shape. Subtle interactions.</p>
           <VariantMinimalistSpark />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">04 // Radar Scope</h2>
           <p className="text-sm text-muted-foreground">Circular/Scanning metaphor. Monochrome green phosphorus style.</p>
           <VariantRadarScope />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">05 // Industrial Panel</h2>
           <p className="text-sm text-muted-foreground">Skeuomorphic touches. Physical buttons, matte finishes.</p>
           <VariantIndustrialPanel />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">06 // Data Stream</h2>
           <p className="text-sm text-muted-foreground">Layout emphasizing the flow of data. Metrics on side.</p>
           <VariantDataStream />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">07 // Split Cards</h2>
           <p className="text-sm text-muted-foreground">Deconstructed view. Good for comparing distinct metrics.</p>
           <VariantSplitCards />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">08 // Holographic</h2>
           <p className="text-sm text-muted-foreground">Modern glassmorphism. Depth, glow, gradients.</p>
           <VariantHolographic />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">09 // High Contrast (E-Ink)</h2>
           <p className="text-sm text-muted-foreground">Brutalist / E-Reader aesthetic. Pure black and white.</p>
           <VariantHighContrast />
        </div>

        <div className="space-y-4">
           <h2 className="text-lg font-semibold text-muted-foreground">10 // Vertical Stack</h2>
           <p className="text-sm text-muted-foreground">Side navigation metaphor applied to charts.</p>
           <VariantVerticalStack />
        </div>
      </div>
    </div>
  )
}
