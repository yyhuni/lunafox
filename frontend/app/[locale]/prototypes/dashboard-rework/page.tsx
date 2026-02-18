"use client"

import type { CSSProperties } from "react"
import {
  Activity,
  BarChart3,
  ChevronRight,
  Hexagon,
  LayoutGrid,
  Layers,
  Radar,
  Search,
  Settings,
  ShieldAlert,
  Sliders,
  Terminal,
  Zap,
} from "@/components/icons"

// Terminus/Arknights style theme variables
const themeVars = {
  "--ark-bg": "#F2F3F5", // Light gray background, more texture
  "--ark-bg-2": "#FFFFFF",
  "--ark-panel": "#FFFFFF",
  "--ark-panel-2": "#F8F9FB", // Slightly darker panel background
  "--ark-border": "#DCDFE6",
  "--ark-border-strong": "#181A1F", // strong dark border
  "--ark-text": "#181A1F", // dark black text
  "--ark-text-2": "#6B7280", // secondary text
  "--ark-accent": "#181A1F", // Use dark black as an accent color
  "--ark-accent-light": "rgba(24, 26, 31, 0.05)",
  "--ark-highlight": "#FF4C24", // Orange-red as embellishment (similar to the terminal logo color)
  "--font-mono": "\"IBM Plex Mono\", \"JetBrains Mono\", monospace",
  "--font-sans": "\"HarmonyOS Sans SC\", \"Inter\", sans-serif",
} as CSSProperties

const stats = [
  { label: "Active Assets", value: "1,284", delta: "+4.2%", id: "01" },
  { label: "Scan Throughput", value: "82%", delta: "+1.1%", id: "02" },
  { label: "Critical Alerts", value: "12", delta: "-3", id: "03", alert: true },
  { label: "Coverage", value: "96.8%", delta: "+0.6%", id: "04" },
]

const navItems = [
  { label: "Overview", icon: LayoutGrid, active: true },
  { label: "Assets", icon: Layers },
  { label: "Scans", icon: Radar },
  { label: "Threats", icon: ShieldAlert },
  { label: "Settings", icon: Sliders },
]

const timeline = [
  { label: "Ingestion", value: "02:14", status: "Done" },
  { label: "Correlation", value: "04:32", status: "Done" },
  { label: "Mitigation", value: "08:09", status: "Active" },
]

const tableRows = [
  { id: "OP-291", name: "Gateway Sweep", owner: "Ops-7", status: "Stable", score: "A" },
  { id: "OP-377", name: "Credential Drift", owner: "Ops-3", status: "Observe", score: "B" },
  { id: "OP-408", name: "Outbound Flux", owner: "Ops-2", status: "Investigate", score: "A-" },
  { id: "OP-512", name: "Shadow Asset", owner: "Ops-1", status: "Monitor", score: "B+" },
]

const barSeries = [48, 62, 54, 68, 76, 58, 64]
const lineSeries = [12, 24, 18, 36, 30, 44, 38, 52]

export default function DashboardReworkPage() {
  return (
    <main className="relative min-h-screen bg-[var(--ark-bg)] text-[var(--ark-text)] font-sans" style={themeVars}>
      {/* Background grid decoration */}
      <div className="pointer-events-none absolute inset-0 ark-grid" aria-hidden />
      
      {/* Top decorative strip */}
      <div className="fixed top-0 left-0 right-0 h-1 bg-[var(--ark-border-strong)] z-50" />
      
      <div className="relative mx-auto w-full max-w-[1600px] p-6 lg:p-10">
        <div className="grid gap-8 lg:grid-cols-[260px_1fr]">
          
          {/* Sidebar */}
          <aside className="flex flex-col gap-6 lg:sticky lg:top-10 h-fit">
            <div className="ark-panel p-6 flex flex-col gap-4">
              <div className="flex items-center gap-3 border-b-2 border-[var(--ark-border-strong)] pb-4">
                <div className="bg-[var(--ark-accent)] text-white p-1.5">
                  <Hexagon className="h-5 w-5 fill-current" />
                </div>
                <div>
                  <h1 className="font-bold text-lg leading-none tracking-tight">LUNAFOX</h1>
                  <p className="text-[10px] text-[var(--ark-text-2)] font-mono mt-1 tracking-widest">CONTROL HUB</p>
                </div>
              </div>
              
              <nav className="flex flex-col gap-1">
                {navItems.map((item) => (
                  <button
                    key={item.label}
                    className={`ark-nav-item ${item.active ? "active" : ""}`}
                    type="button"
                  >
                    <span className="w-1 h-full absolute left-0 top-0 bg-[var(--ark-accent)] opacity-0 transition-opacity indicator" />
                    <item.icon className="h-4 w-4" />
                    <span className="font-medium tracking-wide text-sm">{item.label}</span>
                    {item.active && <ChevronRight className="h-3 w-3 ml-auto opacity-50" />}
                  </button>
                ))}
              </nav>
            </div>

            <div className="ark-panel p-5 bg-[var(--ark-accent)] text-white relative overflow-hidden">
              <div className="relative z-10">
                <p className="text-[10px] font-mono opacity-60 mb-1">SYSTEM STATUS</p>
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-[#00FF94] animate-pulse" />
                  <span className="font-bold tracking-wider">NOMINAL</span>
                </div>
                <p className="text-xs mt-3 font-mono opacity-80">UPTIME: 99.98%</p>
              </div>
              <Activity className="absolute -right-4 -bottom-4 w-24 h-24 opacity-10" />
            </div>
          </aside>

          {/* Main content area */}
          <div className="flex flex-col gap-6">
            
            {/* Top Header */}
            <header className="ark-panel p-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-[var(--ark-text-2)]">
                  <Terminal className="h-4 w-4" />
                  <span className="text-xs font-mono">/ DASHBOARD / OVERVIEW</span>
                </div>
                <h2 className="text-2xl font-bold uppercase tracking-tight">System Operations</h2>
              </div>
              <div className="flex gap-4 font-mono text-xs">
                <div className="ark-tag">
                  <span className="opacity-50">CYCLE:</span>
                  <span className="font-bold">14:32</span>
                </div>
                <div className="ark-tag">
                  <span className="opacity-50">NODES:</span>
                  <span className="font-bold">48</span>
                </div>
              </div>
            </header>

            {/* Statistics card */}
            <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {stats.map((item) => (
                <div key={item.label} className="ark-card p-5 group">
                  <div className="flex justify-between items-start mb-4">
                    <p className="text-xs font-mono text-[var(--ark-text-2)] uppercase">{item.label}</p>
                    <span className="text-[10px] font-mono opacity-30 group-hover:opacity-100 transition-opacity">
                      {item.id}
                    </span>
                  </div>
                  <div className="flex items-baseline gap-2">
                    <span className="text-3xl font-bold tracking-tight">{item.value}</span>
                  </div>
                  <div className={`mt-2 text-xs font-mono inline-flex px-1.5 py-0.5 border ${
                    item.alert 
                      ? "border-[var(--ark-highlight)] text-[var(--ark-highlight)] bg-[rgba(255,76,36,0.05)]" 
                      : "border-[var(--ark-border)] text-[var(--ark-text-2)]"
                  }`}>
                    {item.delta}
                  </div>
                  {/* Decorative corner mark */}
                  <div className="absolute bottom-0 right-0 w-3 h-3 border-b-2 border-r-2 border-[var(--ark-border-strong)] opacity-20" />
                </div>
              ))}
            </section>

            {/* Chart area */}
            <section className="grid gap-6 lg:grid-cols-[1.6fr_1fr]">
              {/* Line chart */}
              <div className="ark-panel p-6 relative">
                <div className="flex justify-between items-center mb-6 border-b border-[var(--ark-border)] pb-4">
                  <div>
                    <h3 className="font-bold text-sm uppercase flex items-center gap-2">
                      <Activity className="h-4 w-4" />
                      Asset Pulse
                    </h3>
                  </div>
                  <span className="text-xs font-mono bg-[var(--ark-accent)] text-white px-2 py-1">LIVE</span>
                </div>
                
                <div className="h-[200px] w-full relative border border-[var(--ark-border)] bg-[var(--ark-panel-2)] p-4">
                  {/* Grid background */}
                  <div className="absolute inset-0 grid grid-cols-6 grid-rows-4 pointer-events-none">
                    {Array.from({ length: 24 }).map((_, i) => (
                      <div key={i} className="border-r border-b border-[var(--ark-border)] opacity-30" />
                    ))}
                  </div>
                  
                  {/* SVG chart */}
                  <svg viewBox="0 0 400 140" className="h-full w-full relative z-10 overflow-visible">
                    <defs>
                      <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                        <path d="M 40 0 L 0 0 0 40" fill="none" stroke="currentColor" strokeOpacity="0.1" />
                      </pattern>
                    </defs>
                    <polyline
                      fill="none"
                      stroke="var(--ark-accent)"
                      strokeWidth="2"
                      points={lineSeries
                        .map((value, index) => `${index * 55},${140 - value}`)
                        .join(" ")}
                      vectorEffect="non-scaling-stroke"
                    />
                    {/* data points */}
                    {lineSeries.map((value, index) => (
                      <circle 
                        key={index}
                        cx={index * 55} 
                        cy={140 - value} 
                        r="3" 
                        fill="var(--ark-bg)" 
                        stroke="var(--ark-accent)" 
                        strokeWidth="2"
                      />
                    ))}
                  </svg>
                </div>
                
                <div className="mt-4 flex gap-4 text-xs font-mono text-[var(--ark-text-2)]">
                  <span>MIN: 12ms</span>
                  <span>MAX: 52ms</span>
                  <span>AVG: 34ms</span>
                </div>
              </div>

              {/* Bar chart */}
              <div className="ark-panel p-6">
                <div className="flex justify-between items-center mb-6 border-b border-[var(--ark-border)] pb-4">
                  <h3 className="font-bold text-sm uppercase flex items-center gap-2">
                    <BarChart3 className="h-4 w-4" />
                    Severity Mix
                  </h3>
                  <Sliders className="h-4 w-4 text-[var(--ark-text-2)]" />
                </div>

                <div className="flex h-[180px] items-end gap-3 px-2">
                  {barSeries.map((value, index) => (
                    <div key={index} className="flex-1 flex flex-col gap-2 group">
                      <div 
                        className="w-full bg-[var(--ark-accent)] opacity-80 group-hover:opacity-100 transition-opacity relative"
                        style={{ height: `${value}%` }}
                      >
                        {/* Top highlight line */}
                        <div className="absolute top-0 left-0 right-0 h-[2px] bg-[var(--ark-highlight)] opacity-0 group-hover:opacity-100" />
                      </div>
                      <span className="text-[10px] font-mono text-center text-[var(--ark-text-2)]">0{index + 1}</span>
                    </div>
                  ))}
                </div>

                <div className="mt-6 grid grid-cols-3 gap-2">
                  {[
                    { label: "HIGH", count: 12, color: "var(--ark-highlight)" },
                    { label: "MED", count: 38, color: "var(--ark-text)" },
                    { label: "LOW", count: 74, color: "var(--ark-text-2)" },
                  ].map(stat => (
                    <div key={stat.label} className="border border-[var(--ark-border)] p-2 text-center">
                      <div className="text-[10px] font-mono mb-1" style={{ color: stat.color }}>{stat.label}</div>
                      <div className="font-bold text-lg">{stat.count}</div>
                    </div>
                  ))}
                </div>
              </div>
            </section>

            {/* Bottom area */}
            <section className="grid gap-6 lg:grid-cols-[1.6fr_1fr]">
              {/* sheet */}
              <div className="ark-panel p-0 overflow-hidden">
                <div className="p-4 border-b border-[var(--ark-border)] bg-[var(--ark-panel-2)] flex justify-between items-center">
                  <h3 className="font-bold text-sm uppercase">Recent Activity</h3>
                  <div className="flex gap-2">
                    <button className="p-1 hover:bg-white border border-transparent hover:border-[var(--ark-border)]" aria-label="Search activity">
                      <Search className="h-4 w-4" />
                    </button>
                    <button className="p-1 hover:bg-white border border-transparent hover:border-[var(--ark-border)]" aria-label="Open settings">
                      <Settings className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm text-left">
                    <thead className="text-xs text-[var(--ark-text-2)] font-mono uppercase bg-[var(--ark-bg)] border-b border-[var(--ark-border)]">
                      <tr>
                        <th className="px-6 py-3 font-medium">ID</th>
                        <th className="px-6 py-3 font-medium">Task Name</th>
                        <th className="px-6 py-3 font-medium">Owner</th>
                        <th className="px-6 py-3 font-medium">Status</th>
                        <th className="px-6 py-3 font-medium text-right">Score</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tableRows.map((row, i) => (
                        <tr key={row.id} className="border-b border-[var(--ark-border)] hover:bg-[var(--ark-panel-2)] transition-colors group">
                          <td className="px-6 py-4 font-mono text-xs">{row.id}</td>
                          <td className="px-6 py-4 font-medium">{row.name}</td>
                          <td className="px-6 py-4 text-[var(--ark-text-2)] text-xs">{row.owner}</td>
                          <td className="px-6 py-4">
                            <span className="inline-flex items-center gap-1.5 px-2 py-1 border border-[var(--ark-border)] text-xs font-mono uppercase bg-white">
                              <span className={`w-1.5 h-1.5 rounded-full ${i === 0 ? 'bg-green-500' : 'bg-yellow-500'}`} />
                              {row.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-right font-mono font-bold">{row.score}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Timeline/Status */}
              <div className="ark-panel p-6 flex flex-col h-full">
                <div className="flex justify-between items-center mb-6">
                  <h3 className="font-bold text-sm uppercase">Cycle Timing</h3>
                  <div className="w-2 h-2 bg-[var(--ark-accent)] animate-pulse" />
                </div>
                
                <div className="flex-1 relative pl-4 space-y-8 before:absolute before:left-0 before:top-2 before:bottom-2 before:w-[2px] before:bg-[var(--ark-border)]">
                  {timeline.map((item, index) => (
                    <div key={item.label} className="relative pl-6">
                      {/* Timeline point */}
                      <div className={`absolute left-[-5px] top-1.5 w-2.5 h-2.5 border-2 border-[var(--ark-bg)] outline outline-1 outline-[var(--ark-border-strong)] ${
                        index === 2 ? 'bg-[var(--ark-highlight)]' : 'bg-[var(--ark-accent)]'
                      }`} />
                      
                      <div className="flex justify-between items-baseline">
                        <span className="font-bold text-sm">{item.label}</span>
                        <span className="font-mono text-xs text-[var(--ark-text-2)]">{item.value}</span>
                      </div>
                      <div className="mt-1 text-xs text-[var(--ark-text-2)] font-mono uppercase tracking-wider">
                        STATUS: {item.status}
                      </div>
                    </div>
                  ))}
                </div>

                <div className="mt-6 pt-4 border-t border-[var(--ark-border)]">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 border border-[var(--ark-border)] flex items-center justify-center bg-[var(--ark-panel-2)]">
                      <Zap className="h-5 w-5" />
                    </div>
                    <div>
                      <div className="text-xs font-mono text-[var(--ark-text-2)]">POWER OUTPUT</div>
                      <div className="font-bold">STABLE / 98%</div>
                    </div>
                  </div>
                </div>
              </div>
            </section>
          </div>
        </div>
      </div>

      <style jsx>{`
        .ark-grid {
          background-image: 
            linear-gradient(var(--ark-border) 1px, transparent 1px),
            linear-gradient(90deg, var(--ark-border) 1px, transparent 1px);
          background-size: 40px 40px;
          opacity: 0.3;
        }

        .ark-panel {
          background: var(--ark-panel);
          border: 1px solid var(--ark-border);
          box-shadow: 0 2px 4px rgba(0,0,0,0.02);
          position: relative;
        }
        
        /* Top bold border decoration */
        .ark-panel::before {
          content: "";
          position: absolute;
          top: -1px;
          left: -1px;
          right: -1px;
          height: 3px;
          background: var(--ark-border-strong);
          opacity: 0;
          transition: opacity 0.3s;
        }
        
        .ark-panel:hover::before {
          opacity: 1;
        }

        .ark-card {
          background: var(--ark-panel);
          border: 1px solid var(--ark-border);
          position: relative;
          transition: color 0.2s, background-color 0.2s, border-color 0.2s, opacity 0.2s, transform 0.2s, box-shadow 0.2s;
        }
        
        .ark-card:hover {
          border-color: var(--ark-border-strong);
          transform: translateY(-2px);
          box-shadow: 4px 4px 0 rgba(24, 26, 31, 0.05);
        }

        .ark-nav-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 12px 16px;
          color: var(--ark-text-2);
          transition: color 0.2s, background-color 0.2s, border-color 0.2s, opacity 0.2s, transform 0.2s, box-shadow 0.2s;
          position: relative;
          border: 1px solid transparent;
        }

        .ark-nav-item:hover {
          background: var(--ark-panel-2);
          color: var(--ark-text);
        }

        .ark-nav-item.active {
          background: var(--ark-accent-light);
          color: var(--ark-text);
          border-color: var(--ark-border);
        }
        
        .ark-nav-item.active .indicator {
          opacity: 1;
        }

        .ark-tag {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 4px 8px;
          background: var(--ark-panel-2);
          border: 1px solid var(--ark-border);
          font-family: var(--font-mono);
        }
      `}</style>
    </main>
  )
}
