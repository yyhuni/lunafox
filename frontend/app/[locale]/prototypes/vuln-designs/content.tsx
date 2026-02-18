"use client"

import React, { useMemo } from "react"
import { Pie, PieChart, Cell, RadialBarChart, RadialBar, ResponsiveContainer, Tooltip as RechartsTooltip } from "recharts"
import { Card, CardContent } from "@/components/ui/card"
import { IconShield, IconAlertTriangle, IconActivity, IconServer, IconRadar, IconDashboard, IconCode } from "@/components/icons"
import { cn } from "@/lib/utils"

// Mock data
const MOCK_DATA = [
  { severity: "critical", count: 12, fill: "#ef4444", label: "CRITICAL" }, // Red
  { severity: "high", count: 45, fill: "#f97316", label: "HIGH" },     // Orange
  { severity: "medium", count: 89, fill: "#eab308", label: "MEDIUM" },   // Yellow
  { severity: "low", count: 120, fill: "#22c55e", label: "LOW" },      // Green
  { severity: "info", count: 200, fill: "#3b82f6", label: "INFO" },     // Blue
]

const TOTAL_COUNT = MOCK_DATA.reduce((acc, cur) => acc + cur.count, 0)

// --- Demo 1: Radial Hazard (Concentric Circle Warning Device) ---
function RadialHazard() {
  // Transform data for RadialBar: needs to be sorted and have background fill
  const radialData = [...MOCK_DATA].reverse().map(d => ({
    name: d.label,
    uv: d.count,
    fill: d.fill
  }))

  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconShield className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">RADIAL.HAZARD // V.01</span>
       </div>
       
       <CardContent className="flex-1 p-0 relative flex items-center justify-center bg-card">
          <ResponsiveContainer width="100%" height="100%">
            <RadialBarChart 
              innerRadius="30%" 
              outerRadius="100%" 
              data={radialData} 
              startAngle={180} 
              endAngle={0}
              cy="70%"
            >
              <RadialBar
                label={{ position: 'insideStart', fill: '#fff', fontSize: 10, fontWeight: 'bold' }}
                background
                dataKey="uv"
                cornerRadius={2}
              />
              <text x="50%" y="65%" textAnchor="middle" dominantBaseline="middle" className="fill-foreground text-3xl font-black tracking-tighter">
                {TOTAL_COUNT}
              </text>
              <text x="50%" y="75%" textAnchor="middle" dominantBaseline="middle" className="fill-muted-foreground text-xs font-mono uppercase">
                Total Vulns
              </text>
            </RadialBarChart>
          </ResponsiveContainer>
          
          {/* Legend Overlay */}
          <div className="absolute top-4 right-4 flex flex-col gap-1 text-[10px] font-mono">
             {MOCK_DATA.slice(0,3).map(d => (
                <div key={d.label} className="flex items-center gap-1.5">
                   <div className="w-2 h-2 rounded-full" style={{background: d.fill}}></div>
                   <span className="opacity-70">{d.label}</span>
                   <span className="font-bold ml-auto">{d.count}</span>
                </div>
             ))}
          </div>
       </CardContent>
    </Card>
  )
}

// --- Demo 2: Threat DNA (Threat Gene Profile - Stacked Bar) ---
function ThreatDNA() {
  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconAlertTriangle className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">THREAT.DNA // SPECTRUM</span>
       </div>

       <CardContent className="flex-1 p-6 flex flex-col justify-center gap-8">
          
          {/* Main Number */}
          <div className="flex justify-between items-end">
             <div>
                <div className="text-4xl font-black tracking-tighter">{TOTAL_COUNT}</div>
                <div className="text-xs text-muted-foreground font-mono uppercase tracking-widest">System Vulnerabilities</div>
             </div>
             <div className="text-right">
                <div className="text-sm font-bold text-red-500 flex items-center justify-end gap-1">
                   <span className="w-2 h-2 bg-red-500 animate-pulse rounded-full"></span>
                   {MOCK_DATA[0].count} CRITICAL
                </div>
                <div className="text-[10px] text-muted-foreground">Requires immediate action</div>
             </div>
          </div>

          {/* The DNA Bar */}
          <div className="h-16 w-full flex rounded-md overflow-hidden border border-border shadow-inner relative">
             {/* Grid overlay */}
             <div className="absolute inset-0 z-10 opacity-20 pointer-events-none" 
                  style={{backgroundImage: 'linear-gradient(90deg, transparent 95%, #000 95%)', backgroundSize: '20px 100%'}}>
             </div>
             
             {MOCK_DATA.map((d) => {
                const percent = (d.count / TOTAL_COUNT) * 100
                return (
                   <div 
                      key={d.label}
                      style={{ width: `${percent}%`, backgroundColor: d.fill }}
                      className="h-full relative group transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:brightness-110"
                   >
                      <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                         <span className="text-[10px] font-bold text-white bg-black/50 px-1 rounded backdrop-blur-sm">
                            {Math.round(percent)}%
                         </span>
                      </div>
                   </div>
                )
             })}
          </div>

          {/* Legend Grid */}
          <div className="grid grid-cols-5 gap-2 pt-2 border-t border-border border-dashed">
             {MOCK_DATA.map(d => (
                <div key={d.label} className="flex flex-col items-center">
                   <div className="w-full h-1 mb-1 rounded-full" style={{background: d.fill}}></div>
                   <span className="text-[10px] font-bold text-muted-foreground">{d.label.substring(0,3)}</span>
                   <span className="text-xs font-mono font-bold">{d.count}</span>
                </div>
             ))}
          </div>

       </CardContent>
    </Card>
  )
}

// --- Demo 3: Impact Matrix (lattice/waffle) ---
function ImpactMatrix() {
  // Generate 100 dots representing percentages
  const dots = useMemo(() => {
     const totalDots = 100
     const result = []
     
     // Sort by severity priority
     const sortedData = [...MOCK_DATA] 
     
     sortedData.forEach(cat => {
        const count = Math.round((cat.count / TOTAL_COUNT) * totalDots)
        for(let i=0; i<count; i++) {
           if(result.length < totalDots) {
              result.push({ color: cat.fill, severity: cat.label })
           }
        }
     })
     
     // Fill remaining if rounding errors
     while(result.length < totalDots) {
        result.push({ color: 'var(--muted)', severity: 'NONE' })
     }
     
     return result
  }, [])

  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconActivity className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">IMPACT.MATRIX // GRID</span>
       </div>

       <div className="flex flex-1 p-4 gap-6">
          {/* Left: The Matrix */}
          <div className="aspect-square h-full flex items-center justify-center">
             <div className="grid grid-cols-10 gap-1.5 w-full max-w-[220px]">
                {dots.map((dot, i) => (
                   <div 
                      key={i} 
                      className="aspect-square rounded-[1px] transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:scale-125 hover:z-10"
                      style={{ backgroundColor: dot.color }}
                      title={dot.severity}
                   ></div>
                ))}
             </div>
          </div>

          {/* Right: Stats List */}
          <div className="flex-1 flex flex-col justify-center gap-3">
             <div className="mb-2">
                <div className="text-xs text-muted-foreground font-mono uppercase">Total Detected</div>
                <div className="text-3xl font-black">{TOTAL_COUNT}</div>
             </div>
             
            <div className="space-y-2">
                {MOCK_DATA.map(d => (
                   <div key={d.label} className="flex items-center justify-between text-xs border-b border-border/50 pb-1 last:border-0">
                      <div className="flex items-center gap-2">
                         <div className="w-2 h-2 rounded-sm" style={{ backgroundColor: d.fill }}></div>
                         <span className="font-bold opacity-70">{d.label}</span>
                      </div>
                      <span className="font-mono">{d.count}</span>
                   </div>
                ))}
             </div>
          </div>
       </div>
    </Card>
  )
}

// --- Demo 4: Structural Pillar ---
function StructuralPillar() {
  const maxCount = Math.max(...MOCK_DATA.map(d => d.count))
  
  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconServer className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">STRUCT.PILLAR // LEVEL</span>
       </div>
       
       <div className="flex flex-1 p-6 gap-6 items-end justify-center">
          {MOCK_DATA.map((d) => {
             const heightPercent = (d.count / maxCount) * 100
             return (
                <div key={d.label} className="flex flex-col items-center gap-2 group h-full justify-end w-12">
                   <div className="text-[10px] font-mono font-bold opacity-0 group-hover:opacity-100 transition-opacity mb-1">{d.count}</div>
                   <div className="w-full relative bg-muted/20 border border-border/50 rounded-sm overflow-hidden" style={{height: '100%'}}>
                      <div 
                         className="absolute bottom-0 w-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 ease-out flex flex-col-reverse gap-[1px]" 
                         style={{height: `${heightPercent}%`}}
                      >
                         {/* Segmented look */}
                         {Array.from({length: 10}).map((_, i) => (
                            <div key={i} className="flex-1 w-full opacity-80" style={{backgroundColor: d.fill, opacity: (i+1)/12 + 0.2}}></div>
                         ))}
                      </div>
                      
                      {/* Measurement ticks */}
                      <div className="absolute inset-0 flex flex-col justify-between py-1 pointer-events-none">
                         {[...Array(5)].map((_, i) => (
                            <div key={i} className="w-full h-[1px] bg-black/20 dark:bg-white/20"></div>
                         ))}
                      </div>
                   </div>
                   <div className="text-[10px] font-bold text-muted-foreground rotate-0 mt-1">{d.label.substring(0,3)}</div>
                </div>
             )
          })}
       </div>
    </Card>
  )
}

// --- Demo 5: Frequency Bars (Spectrum) ---
function FrequencyBars() {
  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconActivity className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">FREQ.BARS // AUDIO</span>
       </div>
       
       <div className="flex flex-1 items-center justify-center p-6 bg-black/5 dark:bg-white/5">
         <div className="flex gap-1 h-32 items-end">
            {MOCK_DATA.map((d) => {
               // Simulate a "waveform" or "equalizer" look by having mirrored bars or just pixelated bars
               const height = (d.count / 200) * 100 // Max 200 in mock data
               return (
                  <div key={d.label} className="flex flex-col items-center gap-1 group">
                     <div className="w-8 flex flex-col-reverse gap-0.5">
                        {[...Array(12)].map((_, i) => {
                           // Threshold to determine if this "pixel" is lit
                           const isLit = (i / 12) * 100 < height
                           return (
                              <div 
                                 key={i} 
                                 className={cn(
                                    "w-full h-1.5 rounded-[1px] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-300",
                                    isLit ? "opacity-100" : "opacity-10 bg-muted-foreground/20"
                                 )}
                                 style={{ backgroundColor: isLit ? d.fill : undefined }}
                              ></div>
                           )
                        })}
                     </div>
                     <span className="text-[9px] font-mono font-bold text-muted-foreground uppercase">{d.label.substring(0,1)}</span>
                  </div>
               )
            })}
         </div>
       </div>
    </Card>
  )
}

// --- Demo 6: Polar Sector ---
function PolarSector() {
  const data = MOCK_DATA.map(d => ({ name: d.label, value: d.count, fill: d.fill }))
  
  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconRadar className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">POLAR.SECTOR // RADAR</span>
       </div>
       
       <div className="flex-1 relative">
          <ResponsiveContainer width="100%" height="100%">
             <PieChart>
                <Pie
                   data={data}
                   cx="50%"
                   cy="50%"
                   innerRadius={40}
                   outerRadius={80}
                   paddingAngle={5}
                   dataKey="value"
                   stroke="none"
                   startAngle={90}
                   endAngle={-270}
                >
                   {data.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} strokeWidth={0} />
                   ))}
                </Pie>
                <RechartsTooltip 
                   content={({ active, payload }) => {
                      if (active && payload && payload.length) {
                         const d = payload[0].payload
                         return (
                            <div className="bg-background border border-border p-2 text-xs shadow-xl font-mono">
                               <span style={{color: d.fill}} className="font-bold">{d.name}</span>: {d.value}
                            </div>
                         )
                      }
                      return null
                   }}
                />
             </PieChart>
          </ResponsiveContainer>
          
          {/* Center Label */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
             <div className="text-center">
                <div className="text-xs font-mono text-muted-foreground">SCAN</div>
                <div className="font-black text-xl">{TOTAL_COUNT}</div>
             </div>
          </div>
          
          {/* Decorative Rings */}
          <div className="absolute inset-0 border-[20px] border-muted/5 rounded-full pointer-events-none m-8"></div>
       </div>
    </Card>
  )
}

// --- Demo 7: Mondrian Blocks ---
function MondrianBlocks() {
   // Use a simple grid layout to simulate mondrian
   // Total area = 100%. We split it based on proportions roughly.
   // This is a manual artistic approximation for the demo since flex/grid is easier than rect calculation.
   
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconDashboard className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">MONDRIAN.BLOCKS // ART</span>
       </div>
       
       <div className="flex-1 p-4 bg-background">
          <div className="w-full h-full border-4 border-black dark:border-white grid grid-cols-12 grid-rows-6 gap-0 bg-black dark:bg-white">
             {/* Critical - Big Red Block */}
             <div className="col-span-4 row-span-6 bg-red-500 border-r-4 border-black dark:border-white relative group overflow-hidden">
                <span className="absolute bottom-1 right-1 text-xs font-black text-black bg-white/50 px-1 opacity-0 group-hover:opacity-100">CRIT {MOCK_DATA[0].count}</span>
             </div>
             
             {/* High - Orange Block */}
             <div className="col-span-8 row-span-3 bg-orange-500 border-b-4 border-black dark:border-white relative group overflow-hidden">
                <span className="absolute bottom-1 right-1 text-xs font-black text-black bg-white/50 px-1 opacity-0 group-hover:opacity-100">HIGH {MOCK_DATA[1].count}</span>
             </div>
             
             {/* Medium - Yellow */}
             <div className="col-span-3 row-span-3 bg-yellow-400 border-r-4 border-black dark:border-white relative group overflow-hidden">
                <span className="absolute bottom-1 right-1 text-xs font-black text-black bg-white/50 px-1 opacity-0 group-hover:opacity-100">MED {MOCK_DATA[2].count}</span>
             </div>
             
             {/* Low - Green */}
             <div className="col-span-3 row-span-3 bg-green-500 border-r-4 border-black dark:border-white relative group overflow-hidden">
                <span className="absolute bottom-1 right-1 text-xs font-black text-black bg-white/50 px-1 opacity-0 group-hover:opacity-100">LOW {MOCK_DATA[3].count}</span>
             </div>
             
             {/* Info - Blue */}
             <div className="col-span-2 row-span-3 bg-blue-500 relative group overflow-hidden">
                <span className="absolute bottom-1 right-1 text-xs font-black text-black bg-white/50 px-1 opacity-0 group-hover:opacity-100">INFO {MOCK_DATA[4].count}</span>
             </div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 8: Barcode Strip (barcode strip) ---
function BarcodeStrip() {
   // Generate a random-looking strip based on data
   // We flatten the data into an array of "items"
    const barcodeItems = useMemo(() => {
      const items: Array<{ color: string; type: string }> = []
      MOCK_DATA.forEach(d => {
         for(let i=0; i<Math.min(d.count, 50); i++) { // Cap at 50 per category for perf
            items.push({ color: d.fill, type: d.label })
         }
      })
      // Shuffle for random distribution look
      return items.sort(() => Math.random() - 0.5)
   }, [])

   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconCode className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">BARCODE.STRIP // DENSITY</span>
       </div>
       
       <div className="flex-1 flex flex-col p-6 justify-center">
          <div className="flex h-24 w-full items-stretch justify-between overflow-hidden border border-border bg-background">
             {barcodeItems.map((item, i) => (
                <div 
                  key={i} 
                  className="w-[2px] hover:scale-y-110 transition-transform duration-100"
                  style={{ backgroundColor: item.color, opacity: 0.8 }}
                  title={item.type}
                ></div>
             ))}
          </div>
          
          <div className="mt-6 flex justify-between text-xs font-mono text-muted-foreground border-t border-border pt-2">
             <div>SEQ: 2024-X99</div>
             <div>DENSITY: HIGH</div>
             <div>TOTAL: {TOTAL_COUNT} UNITS</div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 9: Hex Hive ---
function HexHive() {
  // Simulate a hex grid using offset rows
  // This is a simplified visual representation using CSS
  return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconShield className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">HEX.HIVE // GRID</span>
       </div>
       
       <div className="flex-1 bg-black/5 dark:bg-white/5 relative overflow-hidden flex items-center justify-center">
          <div className="flex flex-col gap-1 items-center scale-125">
             {[0, 1, 2, 3, 4].map((row, i) => (
                <div key={i} className={cn("flex gap-1", i % 2 === 1 && "pl-4")}>
                   {[0, 1, 2, 3, 4].map((col, j) => {
                      // Randomly assign severity or empty
                      const rand = (i * 5 + j) * 13 % 100
                      let color = "bg-muted/20"
                      let severity = ""
                      
                      if (rand < 10) { color = "bg-red-500"; severity = "CRITICAL" }
                      else if (rand < 30) { color = "bg-orange-500"; severity = "HIGH" }
                      else if (rand < 50) { color = "bg-yellow-500"; severity = "MEDIUM" }
                      else if (rand < 60) { color = "bg-blue-500"; severity = "INFO" }
                      
                      return (
                         <div 
                            key={j} 
                            className={cn(
                               "w-8 h-7 text-[0px] relative clip-path-hex transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:scale-110 hover:z-10",
                               color,
                               severity ? "opacity-90 hover:opacity-100" : "opacity-30"
                            )}
                            style={{
                               clipPath: "polygon(25% 0%, 75% 0%, 100% 50%, 75% 100%, 25% 100%, 0% 50%)"
                            }}
                            title={severity}
                         >
                         </div>
                      )
                   })}
                </div>
             ))}
          </div>
          
          <div className="absolute bottom-4 right-4 text-right">
            <div className="text-xs font-mono text-muted-foreground">CLUSTER STATUS</div>
            <div className="text-lg font-bold">UNSTABLE</div>
          </div>
       </div>
    </Card>
  )
}

// --- Demo 10: Circuit Path ---
function CircuitPath() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconActivity className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">CIRCUIT.PATH // FLOW</span>
       </div>
       
       <div className="flex-1 relative p-6 bg-background">
          <svg className="absolute inset-0 w-full h-full pointer-events-none opacity-20">
             <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
                <path d="M 20 0 L 0 0 0 20" fill="none" stroke="currentColor" strokeWidth="0.5"/>
             </pattern>
             <rect width="100%" height="100%" fill="url(#grid)" />
          </svg>
          
          <div className="relative z-10 flex flex-col justify-between h-full py-4">
             {MOCK_DATA.map((d) => (
                <div key={d.label} className="flex items-center gap-4 group">
                   <div className="w-16 text-right font-mono text-xs font-bold opacity-50 group-hover:opacity-100">{d.label}</div>
                   
                   <div className="flex-1 h-8 relative flex items-center">
                      {/* Trace Line */}
                      <div className="absolute left-0 right-0 h-[2px] bg-muted-foreground/20 group-hover:bg-muted-foreground/40 transition-colors"></div>
                      <div className="absolute left-0 h-[2px] transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 ease-out" 
                           style={{width: `${(d.count / 200) * 100}%`, backgroundColor: d.fill}}></div>
                           
                      {/* Node */}
                      <div className="absolute w-3 h-3 border-2 border-background rounded-full transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500"
                           style={{left: `${(d.count / 200) * 100}%`, backgroundColor: d.fill, transform: 'translateX(-50%)'}}></div>
                   </div>
                   
                   <div className="w-12 font-mono text-sm font-bold text-right">{d.count}</div>
                </div>
             ))}
          </div>
       </div>
    </Card>
   )
}

// --- Demo 11: Tape Reel ---
function TapeReel() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconServer className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">TAPE.REEL // LOG</span>
       </div>
       
       <div className="flex-1 flex flex-col items-center justify-center gap-4 bg-[#f0f0f0] dark:bg-[#1a1a1a]">
          {/* Tape Window */}
          <div className="w-full max-w-[90%] h-32 border-y-4 border-black/80 dark:border-white/20 bg-background relative overflow-hidden flex items-center">
             <div className="absolute inset-0 bg-[url('/images/noise.png')] opacity-10 pointer-events-none"></div>
             
             {/* The Tape Stream */}
             <div className="flex items-center gap-1 animate-marquee whitespace-nowrap px-4">
                {Array.from({length: 20}).map((_, i) => (
                   <div key={i} className="flex gap-1">
                      {MOCK_DATA.map((d, j) => (
                         <div 
                            key={`${i}-${j}`} 
                            className="w-4 h-20 opacity-80"
                            style={{
                               backgroundColor: Math.random() > 0.5 ? d.fill : 'transparent',
                               height: Math.random() * 60 + 20 + 'px'
                            }}
                         ></div>
                      ))}
                   </div>
                ))}
             </div>
             
             {/* Read Head */}
             <div className="absolute left-1/2 top-0 bottom-0 w-1 bg-red-500/50 z-20 pointer-events-none border-x border-red-600"></div>
          </div>
          
          <div className="flex gap-8 text-xs font-mono text-muted-foreground">
             <div>SPEED: 15 IPS</div>
             <div>TRACK: 4</div>
             <div>REC: ON</div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 12: Target Scope ---
function TargetScope() {
   const critical = MOCK_DATA[0]
   
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconRadar className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">TARGET.SCOPE // LOCK</span>
       </div>
       
       <div className="flex-1 relative flex items-center justify-center overflow-hidden bg-black text-green-500 font-mono">
          {/* Grid Lines */}
          <div className="absolute inset-0 opacity-20" 
               style={{backgroundImage: 'linear-gradient(#0f0 1px, transparent 1px), linear-gradient(90deg, #0f0 1px, transparent 1px)', backgroundSize: '50px 50px'}}>
          </div>
          
          {/* Crosshair */}
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none opacity-50">
             <div className="w-full h-[1px] bg-green-500"></div>
             <div className="h-full w-[1px] bg-green-500 absolute"></div>
             <div className="w-64 h-64 border border-green-500 rounded-full flex items-center justify-center">
                <div className="w-48 h-48 border border-green-500/50 rounded-full"></div>
             </div>
          </div>
          
          {/* Main Data in Center */}
          <div className="relative z-10 text-center bg-black/80 p-4 border border-green-500/30 backdrop-blur-sm">
             <div className="text-xs tracking-widest mb-1 text-red-500 animate-pulse">CRITICAL THREATS</div>
             <div className="text-5xl font-black text-white tracking-tighter">{critical.count}</div>
             <div className="text-[10px] text-green-500 mt-2">SECTOR 7G DETECTED</div>
          </div>
          
          {/* Corner Stats */}
          <div className="absolute top-4 left-4 text-xs opacity-70">
             <div>H: {MOCK_DATA[1].count}</div>
             <div>M: {MOCK_DATA[2].count}</div>
          </div>
          <div className="absolute bottom-4 right-4 text-xs opacity-70 text-right">
             <div>SYS: ONLINE</div>
             <div>SCN: ACTIVE</div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 13: Data Stack ---
function DataStack() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-border [[data-theme=bauhaus]_&]:gap-0">
       <div className="bauhaus-kicker flex items-center gap-2 p-2 border-b border-border bg-muted/20">
          <IconServer className="size-4" />
          <span className="font-mono text-xs font-bold tracking-widest">DATA.STACK // RACK</span>
       </div>
       
       <div className="flex-1 p-6 flex flex-col gap-2 justify-center perspective-[500px]">
          {MOCK_DATA.map((d) => {
             const width = (d.count / 200) * 100 + '%'
             return (
                <div key={d.label} className="group relative flex items-center gap-4">
                   {/* Label */}
                   <div className="w-20 text-xs font-mono font-bold text-right text-muted-foreground">{d.label}</div>
                   
                   {/* The 3D-ish Bar */}
                   <div className="flex-1 h-8 bg-muted/10 relative">
                      <div 
                         className="absolute top-0 bottom-0 left-0 bg-gradient-to-r from-transparent to-black/10 dark:to-white/10 transition-[color,background-color,border-color,opacity,transform,box-shadow] duration-500 group-hover:brightness-110"
                         style={{ 
                            width: width,
                            backgroundColor: d.fill,
                            boxShadow: `4px 4px 0px rgba(0,0,0,0.2)`
                         }}
                      >
                         <div className="absolute inset-0 border-t border-white/20"></div>
                         <div className="absolute inset-0 border-b border-black/20"></div>
                         
                         {/* Server Lights */}
                         <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                            <div className="w-1 h-1 rounded-full bg-white/50 animate-pulse"></div>
                            <div className="w-1 h-1 rounded-full bg-white/30"></div>
                         </div>
                      </div>
                   </div>
                   
                   {/* Count */}
                   <div className="w-12 font-black text-lg">{d.count}</div>
                </div>
             )
          })}
       </div>
    </Card>
   )
}

// --- Demo 14: YoRHa Protocol (NieR style) ---
function YorhaProtocol() {
   // Nier palette: #dad4bb (bg), #4e4b42 (text), #cec7b3 (darker bg)
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-0 relative group">
       {/* Background & Scanlines */}
       <div className="absolute inset-0 bg-[#dad4bb] z-0"></div>
       <div className="absolute inset-0 bg-[linear-gradient(rgba(18,16,16,0)_50%,rgba(0,0,0,0.1)_50%),linear-gradient(90deg,rgba(255,0,0,0.03),rgba(0,255,0,0.01),rgba(0,0,255,0.03))] z-[1] bg-[length:100%_4px,3px_100%] pointer-events-none"></div>
       
       <div className="relative z-10 flex flex-col h-full text-[#4e4b42] font-serif">
          {/* Header */}
          <div className="flex justify-between items-center p-4 border-b-2 border-[#b8b29f]">
             <div className="flex items-center gap-2">
                <div className="w-3 h-3 bg-[#4e4b42]"></div>
                <span className="tracking-[0.2em] font-bold text-sm">VULNERABILITY.DAT</span>
             </div>
             <div className="text-xs tracking-widest opacity-70">YoRHa UI // 5.0</div>
          </div>
          
          {/* Content */}
          <div className="flex-1 p-6 flex flex-col justify-center gap-4">
             {MOCK_DATA.slice(0, 4).map((d) => (
                <div key={d.label} className="flex items-center gap-4">
                   <div className="w-24 text-right text-xs font-bold tracking-widest uppercase">{d.label}</div>
                   <div className="flex-1 h-4 bg-[#c0bba6] flex gap-[2px] p-[2px]">
                      {/* Blocky progress bar */}
                      {Array.from({length: 20}).map((_, i) => {
                         const isActive = (i / 20) * 100 < (d.count / 200) * 100
                         return (
                            <div 
                               key={i} 
                               className={cn(
                                  "flex-1 h-full transition-colors duration-300",
                                  isActive ? "bg-[#4e4b42]" : "bg-transparent"
                               )}
                            ></div>
                         )
                      })}
                   </div>
                   <div className="w-8 text-xs font-mono">{d.count}</div>
                </div>
             ))}
          </div>
          
          {/* Footer */}
          <div className="p-2 text-center text-[10px] tracking-[0.3em] opacity-60 border-t border-[#b8b29f]">
             GLORY TO MANKIND
          </div>
       </div>
       
       {/* Decorative corner */}
       <div className="absolute top-0 right-0 w-16 h-16 bg-[#b8b29f] -rotate-45 translate-x-8 -translate-y-8 z-0"></div>
    </Card>
   )
}

// --- Demo 15: Rhodes Access (Arknights style) ---
function RhodesAccess() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-0 relative bg-[#1a1a1a] text-white">
       {/* Slanted decoration */}
       <div className="absolute top-0 right-0 w-[60%] h-full bg-[#222] -skew-x-12 translate-x-12 border-l border-white/10"></div>
       <div className="absolute bottom-0 left-0 w-full h-1 bg-[#F4D03F]"></div>
       
       <div className="relative z-10 flex flex-col h-full p-6">
          <div className="flex justify-between items-start mb-6">
             <div className="flex flex-col">
                <h3 className="text-2xl font-black italic tracking-tighter uppercase">Security<span className="text-[#F4D03F]">.Log</span></h3>
                <span className="text-[10px] text-white/50 tracking-[0.2em]">RHODES ISLAND SYSTEM</span>
             </div>
             <div className="w-12 h-12 border border-white/20 flex items-center justify-center rotate-45 mt-2">
                <div className="w-8 h-8 bg-[#F4D03F]/20 -rotate-45 flex items-center justify-center">
                   <IconShield className="size-4 text-[#F4D03F]" />
                </div>
             </div>
          </div>
          
          <div className="grid grid-cols-2 gap-4 flex-1">
             {/* Big Number */}
             <div className="border border-white/20 p-4 bg-white/5 relative overflow-hidden group hover:bg-[#F4D03F] hover:text-black transition-colors duration-300">
                <div className="absolute top-0 right-0 p-1">
                   <IconAlertTriangle className="size-4 opacity-50" />
                </div>
                <div className="text-xs font-mono opacity-70 mb-1">CRITICAL</div>
                <div className="text-4xl font-black">{MOCK_DATA[0].count}</div>
                <div className="absolute bottom-0 left-0 w-full h-1 bg-[#F4D03F] group-hover:bg-black"></div>
             </div>
             
             {/* List */}
             <div className="flex flex-col justify-between gap-1">
                {MOCK_DATA.slice(1, 4).map(d => (
                   <div key={d.label} className="flex items-center justify-between border-b border-white/10 pb-1">
                      <span className="text-xs font-bold uppercase text-white/70">{d.label}</span>
                      <span className="font-mono text-[#F4D03F]">{d.count}</span>
                   </div>
                ))}
             </div>
          </div>
          
          <div className="mt-4 flex justify-between items-end">
             <div className="h-1 w-24 bg-white/20 flex gap-1">
                <div className="h-full w-1/3 bg-[#F4D03F]"></div>
                <div className="h-full w-1/3 bg-white/50"></div>
             </div>
             <div className="text-[9px] font-mono text-white/30">PRTS TERMINAL_V9</div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 16: Hazard Plate (Heavy Industrial Style) ---
function HazardPlate() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-4 border-black bg-[#EAB308] text-black rounded-none">
       {/* Hazard Stripes Header */}
       <div className="h-6 w-full border-b-4 border-black relative overflow-hidden bg-black">
          <div className="absolute inset-0 flex" style={{width: '200%'}}>
             {/* CSS striped pattern simulation */}
             <div className="w-full h-full" style={{
                backgroundImage: 'repeating-linear-gradient(-45deg, #EAB308, #EAB308 10px, #000 10px, #000 20px)'
             }}></div>
          </div>
       </div>
       
       <div className="flex-1 p-4 flex flex-col relative">
          {/* Bolts */}
          <div className="absolute top-2 left-2 w-3 h-3 rounded-full border-2 border-black flex items-center justify-center"><div className="w-full h-[1px] bg-black rotate-45"></div></div>
          <div className="absolute top-2 right-2 w-3 h-3 rounded-full border-2 border-black flex items-center justify-center"><div className="w-full h-[1px] bg-black rotate-45"></div></div>
          <div className="absolute bottom-2 left-2 w-3 h-3 rounded-full border-2 border-black flex items-center justify-center"><div className="w-full h-[1px] bg-black rotate-45"></div></div>
          <div className="absolute bottom-2 right-2 w-3 h-3 rounded-full border-2 border-black flex items-center justify-center"><div className="w-full h-[1px] bg-black rotate-45"></div></div>
          
          <div className="text-center mb-4 mt-2">
             <div className="text-3xl font-black uppercase tracking-tighter border-b-4 border-black inline-block px-4">Caution</div>
             <div className="text-xs font-bold uppercase mt-1">High Severity Vulnerabilities Detected</div>
          </div>
          
          <div className="flex-1 flex gap-4 items-center justify-center">
             <div className="w-24 h-24 rounded-full border-4 border-black flex items-center justify-center bg-black text-[#EAB308] relative">
                <div className="text-4xl font-black">{MOCK_DATA[0].count}</div>
                <div className="absolute -bottom-6 text-xs font-black text-black uppercase">Crit.</div>
             </div>
             
             <div className="flex flex-col gap-2">
                {MOCK_DATA.slice(1, 4).map(d => (
                   <div key={d.label} className="flex items-center gap-2">
                      <div className="w-24 h-6 border-2 border-black bg-black/10 flex items-center px-1 relative">
                         <div className="h-full bg-black absolute left-0 top-0" style={{width: `${(d.count/150)*100}%`}}></div>
                         <span className="relative z-10 text-xs font-bold text-white mix-blend-difference pl-1">{d.label}</span>
                      </div>
                      <span className="font-black text-lg">{d.count}</span>
                   </div>
                ))}
             </div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 17: Mecha Cockpit (Mecha Cockpit) ---
function MechaCockpit() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border border-cyan-500/50 bg-[#050a10] text-cyan-400 relative">
       {/* HUD Grid */}
       <div className="absolute inset-0 bg-[linear-gradient(rgba(6,182,212,0.1)_1px,transparent_1px),linear-gradient(90deg,rgba(6,182,212,0.1)_1px,transparent_1px)] bg-[size:20px_20px] pointer-events-none"></div>
       <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-cyan-500/10 pointer-events-none"></div>
       
       {/* HUD Corners */}
       <div className="absolute top-2 left-2 w-4 h-4 border-l-2 border-t-2 border-cyan-400"></div>
       <div className="absolute top-2 right-2 w-4 h-4 border-r-2 border-t-2 border-cyan-400"></div>
       <div className="absolute bottom-2 left-2 w-4 h-4 border-l-2 border-b-2 border-cyan-400"></div>
       <div className="absolute bottom-2 right-2 w-4 h-4 border-r-2 border-b-2 border-cyan-400"></div>
       
       <div className="relative z-10 p-6 flex flex-col h-full">
          <div className="flex justify-between items-center text-xs font-mono mb-4">
             <span className="bg-cyan-900/30 px-2 py-0.5 border border-cyan-500/30">SYS.DIAGNOSTIC</span>
             <span className="animate-pulse">ONLINE</span>
          </div>
          
          <div className="flex-1 flex items-center justify-center gap-8">
             {/* Center Circle */}
             <div className="relative w-32 h-32 flex items-center justify-center">
                <div className="absolute inset-0 rounded-full border border-cyan-500/30 animate-[spin_10s_linear_infinite]"></div>
                <div className="absolute inset-2 rounded-full border border-dashed border-cyan-500/50 animate-[spin_15s_linear_infinite_reverse]"></div>
                <div className="text-center">
                   <div className="text-4xl font-bold text-white drop-shadow-[0_0_5px_rgba(34,211,238,0.8)]">{TOTAL_COUNT}</div>
                   <div className="text-[10px] tracking-widest">THREATS</div>
                </div>
             </div>
             
             {/* Side Stats */}
             <div className="flex flex-col gap-3 font-mono text-xs">
                {MOCK_DATA.slice(0, 3).map(d => (
                   <div key={d.label} className="flex items-center gap-2">
                      <div className="w-1 h-1 bg-cyan-400"></div>
                      <div className="w-16 opacity-70">{d.label}</div>
                      <div className="w-24 h-1 bg-cyan-900/50 relative">
                         <div className="absolute h-full bg-cyan-400 box-shadow-[0_0_5px_cyan]" style={{width: `${(d.count/100)*100}%`}}></div>
                      </div>
                      <div className="text-white">{d.count}</div>
                   </div>
                ))}
             </div>
          </div>
       </div>
    </Card>
   )
}

// --- Demo 18: Glitch Console (Glitch Terminal) ---
function GlitchConsole() {
   return (
    <Card className="flex flex-col h-[300px] overflow-hidden border-2 border-red-600 bg-black text-red-500 relative group">
       {/* Glitch Effects */}
       <div className="absolute inset-0 bg-[url('/images/noise.png')] opacity-20 pointer-events-none mix-blend-screen"></div>
       
       <div className="relative z-10 p-6 font-mono flex flex-col h-full">
          <div className="border-b border-red-900 pb-2 mb-4 flex justify-between">
             <h3 className="text-xl font-bold uppercase tracking-widest group-hover:animate-pulse">
                FATAL_ERROR
             </h3>
             <span className="bg-red-600 text-black px-1 font-bold animate-pulse">!!!</span>
          </div>
          
          <div className="flex-1 overflow-hidden relative">
             <div className="absolute top-0 left-0 w-full text-xs space-y-1 opacity-80">
                <div className="flex gap-4"><span className="opacity-50">0x00A1</span> <span>SCANNING SYSTEM SECTORS...</span></div>
                <div className="flex gap-4"><span className="opacity-50">0x00A2</span> <span>VULNERABILITY INDEX OVERFLOW</span></div>
                <div className="flex gap-4"><span className="opacity-50">0x00A3</span> <span className="text-white bg-red-900/50">CRITICAL BREACH: {MOCK_DATA[0].count} UNITS</span></div>
                <div className="flex gap-4"><span className="opacity-50">0x00A4</span> <span>INITIATING CONTAINMENT PROTOCOL</span></div>
                <div className="flex gap-4"><span className="opacity-50">0x00A5</span> <span>...</span></div>
                {MOCK_DATA.slice(1).map((d, i) => (
                   <div key={d.label} className="flex gap-4">
                      <span className="opacity-50">0x00B{i}</span> 
                      <span>DETECTED {d.label} LEVEL THREAT ({d.count})</span>
                   </div>
                ))}
             </div>
             
             {/* Glitch Overlay blocks */}
             <div className="absolute top-1/2 left-10 w-48 h-12 bg-red-500/10 backdrop-blur-sm border border-red-500 flex items-center justify-center animate-bounce">
                <span className="font-black text-2xl tracking-[0.5em] text-white mix-blend-overlay">WARNING</span>
             </div>
          </div>
       </div>
    </Card>
   )
}

// --- Main Page Component ---
export default function VulnDesignsPage() {
  return (
    <div className="p-8 max-w-6xl mx-auto space-y-12">
      <div className="space-y-2">
         <h1 className="text-3xl font-bold tracking-tight">Vulnerability Visualization Concepts</h1>
         <p className="text-muted-foreground">Exploration of severity distribution visualizations for the Bauhaus Dashboard.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option A: Radial Hazard</h3>
            <p className="text-sm text-muted-foreground">Engineered gauge style. Best for emphasizing the &quot;danger level&quot; rather than exact proportions.</p>
            <RadialHazard />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option B: Threat DNA</h3>
            <p className="text-sm text-muted-foreground">Compact horizontal stack. Extremely space-efficient and clean. Fits perfectly under a timeline.</p>
            <ThreatDNA />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option C: Impact Matrix</h3>
            <p className="text-sm text-muted-foreground">10x10 Waffle chart. Digital, retro-tech feel. Makes every unit feel tangible.</p>
            <ImpactMatrix />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option D: Structural Pillar</h3>
            <p className="text-sm text-muted-foreground">Vertical liquid/level indicators. Good for comparing relative heights in a compact width.</p>
            <StructuralPillar />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option E: Frequency Bars</h3>
            <p className="text-sm text-muted-foreground">Digital equalizer style. Segmented bars provide a high-tech, quantized feel.</p>
            <FrequencyBars />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option F: Polar Sector</h3>
            <p className="text-sm text-muted-foreground">Radial radar view. Excellent for cyclical data or just breaking the grid rigidity.</p>
            <PolarSector />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option G: Mondrian Blocks</h3>
            <p className="text-sm text-muted-foreground">Artistic treemap. Uses pure geometry and primary colors to convey proportion.</p>
            <MondrianBlocks />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option H: Barcode Strip</h3>
            <p className="text-sm text-muted-foreground">High-density data strip. Visualizes individual incidents rather than aggregates.</p>
            <BarcodeStrip />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option I: Hex Hive</h3>
            <p className="text-sm text-muted-foreground">Organic grid layout. Perfect for displaying cluster status or network node health.</p>
            <HexHive />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option J: Circuit Path</h3>
            <p className="text-sm text-muted-foreground">PCB trace visualization. Metaphor for data flow and connectivity between nodes.</p>
            <CircuitPath />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option K: Tape Reel</h3>
            <p className="text-sm text-muted-foreground">Retro computing aesthetic. Moving data stream metaphor for logs or real-time events.</p>
            <TapeReel />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option L: Target Scope</h3>
            <p className="text-sm text-muted-foreground">Tactical HUD interface. Focuses purely on the most critical metrics with precision.</p>
            <TargetScope />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option M: Data Stack</h3>
            <p className="text-sm text-muted-foreground">Physical rack metaphor. Skeuomorphic touches giving weight and dimension to the data.</p>
            <DataStack />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option N: YoRHa Protocol</h3>
            <p className="text-sm text-muted-foreground">Nier: Automata inspired. Beige palette, scanlines, and serif typography with blocky UI elements.</p>
            <YorhaProtocol />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option O: Rhodes Access</h3>
            <p className="text-sm text-muted-foreground">Arknights/Tactical inspired. High contrast dark mode with electric yellow accents and slanted geometry.</p>
            <RhodesAccess />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option P: Hazard Plate</h3>
            <p className="text-sm text-muted-foreground">Heavy Industry style. Safety yellow stripes, bolts, and physical warning signage aesthetic.</p>
            <HazardPlate />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option Q: Mecha Cockpit</h3>
            <p className="text-sm text-muted-foreground">Sci-Fi HUD style. Cyan fluorescence, angular brackets, and pilot interface visualization.</p>
            <MechaCockpit />
         </div>

         <div className="space-y-4">
            <h3 className="font-bold text-lg">Option R: Glitch Console</h3>
            <p className="text-sm text-muted-foreground">Cyberpunk terminal style. Distorted text, error codes, and a raw, unstable system aesthetic.</p>
            <GlitchConsole />
         </div>
      </div>
    </div>
  )
}
