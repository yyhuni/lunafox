"use client"

import type { CSSProperties } from "react"
import {
  Activity,
  Box,
  Hexagon,
  LayoutGrid,
  Layers,
  Settings,
  Shield,
  ShieldAlert,
  Signal,
  Target,
  Terminal,
  Zap,
} from "@/components/icons"

// Reuse theme variables from Arknights UI
const themeVars = {
  "--ark-bg": "#F4F5F7",
  "--ark-bg-2": "#FFFFFF",
  "--ark-panel": "#FFFFFF",
  "--ark-panel-2": "#F8F9FA",
  "--ark-border": "#E1E5EA",
  "--ark-border-strong": "#C5CBD3",
  "--ark-text": "#1A1D21",
  "--ark-text-2": "#6F7681",
  "--ark-muted": "#9CA3AF",
  "--ark-accent": "#1A1D21",     // Deep black as the main accent color
  "--ark-accent-2": "#4C5159",   // secondary emphasis
  "--ark-active": "#2563EB",     // Activated state (optional blue)
  fontFamily: "\"HarmonyOS Sans SC\", \"IBM Plex Sans\", \"Segoe UI\", sans-serif",
} as CSSProperties

const stats = [
  { id: "01", label: "Active Assets", value: "1,284", delta: "+4.2%", icon: Box },
  { id: "02", label: "Scan Throughput", value: "82%", delta: "+1.1%", icon: Activity },
  { id: "03", label: "Critical Alerts", value: "12", delta: "-3", icon: Zap },
  { id: "04", label: "Coverage", value: "96.8%", delta: "+0.6%", icon: Target },
]

const navItems = [
  { label: "Overview", icon: LayoutGrid, active: true },
  { label: "Assets", icon: Layers },
  { label: "Scans", icon: Activity },
  { label: "Threats", icon: ShieldAlert },
  { label: "Settings", icon: Settings },
]

const timeline = [
  { label: "Ingestion", value: "02:14", status: "Done" },
  { label: "Correlation", value: "04:32", status: "Done" },
  { label: "Mitigation", value: "08:09", status: "Active" },
]

const tableRows = [
  { id: "OP-291", name: "Gateway Sweep", owner: "Ops-7", status: "STABLE", score: "A" },
  { id: "OP-377", name: "Credential Drift", owner: "Ops-3", status: "OBSERVE", score: "B" },
  { id: "OP-408", name: "Outbound Flux", owner: "Ops-2", status: "INVESTIGATE", score: "A-" },
  { id: "OP-512", name: "Shadow Asset", owner: "Ops-1", status: "MONITOR", score: "B+" },
]

const barSeries = [48, 62, 54, 68, 76, 58, 64]
const lineSeries = [12, 24, 18, 36, 30, 44, 38, 52]

export default function DashboardDemoPage() {
  return (
    <main className="relative min-h-screen bg-[var(--ark-bg)] text-[var(--ark-text)] overflow-hidden" style={themeVars}>
      {/* Industrial style background grid and decoration */}
      <div className="pointer-events-none absolute inset-0 ark-grid" aria-hidden />
      
      {/* Top decorative strip */}
      <div className="fixed top-0 left-0 right-0 h-1 bg-[var(--ark-accent)] z-50 opacity-80" />
      
      <div className="relative mx-auto w-full max-w-[1600px] p-6 lg:p-8">
        <div className="grid gap-6 lg:grid-cols-[240px_1fr]">
          
          {/* Sidebar */}
          <aside className="flex flex-col gap-6 lg:sticky lg:top-8 lg:h-[calc(100vh-4rem)]">
            <div className="ark-panel p-4 flex items-center gap-3">
              <div className="h-10 w-10 flex items-center justify-center bg-[var(--ark-text)] text-white">
                <Hexagon className="h-6 w-6" />
              </div>
              <div>
                <div className="text-[10px] tracking-widest text-[var(--ark-text-2)] uppercase">Console</div>
                <div className="font-bold tracking-tight">LUNAFOX</div>
              </div>
            </div>

            <nav className="flex-1 space-y-2">
              {navItems.map((item) => (
                <button
                  key={item.label}
                  className={`ark-nav-item w-full ${item.active ? "active" : ""}`}
                >
                  <item.icon className="h-4 w-4" />
                  <span>{item.label}</span>
                  {item.active && <span className="ml-auto w-1.5 h-1.5 bg-[var(--ark-active)] rounded-full" />}
                </button>
              ))}
            </nav>

            <div className="ark-panel mt-auto p-4 space-y-3">
              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--ark-text-2)]">SYSTEM STATUS</span>
                <span className="h-2 w-2 rounded-full bg-[var(--success)] animate-pulse" />
              </div>
              <div className="space-y-1">
                <div className="flex justify-between text-xs font-mono">
                  <span>CPU</span>
                  <span>12%</span>
                </div>
                <div className="h-1 w-full bg-[var(--ark-border)] overflow-hidden">
                  <div className="h-full bg-[var(--ark-text)] w-[12%]" />
                </div>
              </div>
              <div className="space-y-1">
                <div className="flex justify-between text-xs font-mono">
                  <span>MEM</span>
                  <span>48%</span>
                </div>
                <div className="h-1 w-full bg-[var(--ark-border)] overflow-hidden">
                  <div className="h-full bg-[var(--ark-text)] w-[48%]" />
                </div>
              </div>
            </div>
          </aside>

          {/* Main content area */}
          <div className="flex flex-col gap-6">
            
            {/* Header */}
            <header className="ark-panel p-5 flex flex-col md:flex-row md:items-end justify-between gap-4">
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <span className="px-1.5 py-0.5 text-[10px] font-mono bg-[var(--ark-text)] text-white">DASH-01</span>
                  <p className="text-xs tracking-[0.2em] text-[var(--ark-text-2)] uppercase">Operations Command</p>
                </div>
                <h1 className="text-2xl font-bold tracking-tight uppercase">System Overview</h1>
              </div>
              <div className="flex gap-2">
                <div className="ark-slab px-3 py-1.5 flex items-center gap-2 text-xs font-mono">
                  <span className="w-1.5 h-1.5 bg-[var(--success)] rounded-sm" />
                  NETWORK: ONLINE
                </div>
                <div className="ark-slab px-3 py-1.5 flex items-center gap-2 text-xs font-mono">
                  <span className="text-[var(--ark-text-2)]">CYCLE:</span>
                  14:32:09
                </div>
              </div>
            </header>

            {/* Stats Grid */}
            <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {stats.map((item) => (
                <div key={item.label} className="ark-panel group relative overflow-hidden transition-[color,background-color,border-color,opacity,transform,box-shadow] hover:-translate-y-1">
                  <div className="absolute top-0 left-0 p-2 text-[10px] font-mono text-[var(--ark-muted)] opacity-50">
                    {item.id} {"//"}
                  </div>
                  <div className="p-5 pt-8 space-y-4">
                    <div className="flex justify-between items-start">
                      <div className="p-2 bg-[var(--ark-panel-2)] border border-[var(--ark-border)]">
                        <item.icon className="h-5 w-5 text-[var(--ark-text-2)]" />
                      </div>
                      <span className={`text-xs font-mono px-1.5 py-0.5 border ${item.delta.startsWith('+') ? 'border-[var(--success)]/30 bg-[var(--success)]/5 text-[var(--success)]' : 'border-[var(--error)]/30 bg-[var(--error)]/5 text-[var(--error)]'}`}>
                        {item.delta}
                      </span>
                    </div>
                    <div>
                      <div className="text-2xl font-bold tracking-tight">{item.value}</div>
                      <div className="text-xs text-[var(--ark-text-2)] uppercase tracking-wider mt-1">{item.label}</div>
                    </div>
                  </div>
                  <div className="absolute bottom-0 right-0 h-8 w-8 bg-[linear-gradient(135deg,transparent_50%,var(--ark-border)_50%)] opacity-20" />
                </div>
              ))}
            </section>

            {/* Charts Section */}
            <section className="grid gap-6 lg:grid-cols-[1.6fr_1fr]">
              <div className="ark-panel flex flex-col">
                <div className="border-b border-[var(--ark-border)] p-4 flex items-center justify-between bg-[var(--ark-panel-2)]">
                  <div className="flex items-center gap-2">
                    <Signal className="h-4 w-4" />
                    <span className="ark-kicker">ASSET PULSE</span>
                  </div>
                  <div className="flex gap-2">
                    <button className="text-xs px-2 py-1 bg-white border border-[var(--ark-border)] hover:border-[var(--ark-text)] transition-colors">24H</button>
                    <button className="text-xs px-2 py-1 bg-transparent border border-transparent text-[var(--ark-text-2)] hover:text-[var(--ark-text)]">7D</button>
                  </div>
                </div>
                <div className="p-6 flex-1 min-h-[240px] relative">
                  {/* Grid background decoration */}
                  <div className="absolute inset-6 border-l border-b border-[var(--ark-border)] opacity-50 pointer-events-none" />
                  <div className="absolute inset-0 flex items-center justify-center">
                     <svg viewBox="0 0 400 140" className="h-[70%] w-full max-w-2xl overflow-visible">
                        {/* Decorative grid lines */}
                        <pattern id="grid" width="40" height="140" patternUnits="userSpaceOnUse">
                          <line x1="0" y1="0" x2="0" y2="140" stroke="var(--ark-border)" strokeWidth="1" strokeDasharray="2 2" />
                        </pattern>
                        <rect width="400" height="140" fill="url(#grid)" opacity="0.3" />
                        
                        <polyline
                          fill="none"
                          stroke="var(--ark-text)"
                          strokeWidth="2"
                          points={lineSeries
                            .map((value, index) => `${index * 55},${140 - value}`)
                            .join(" ")}
                          vectorEffect="non-scaling-stroke"
                        />
                        {/* data points */}
                        {lineSeries.map((value, index) => (
                           <g key={index}>
                             <circle cx={index * 55} cy={140 - value} r="3" fill="var(--ark-bg)" stroke="var(--ark-text)" strokeWidth="2" />
                           </g>
                        ))}
                      </svg>
                  </div>
                </div>
                <div className="border-t border-[var(--ark-border)] p-3 px-6 flex items-center gap-4 text-xs font-mono text-[var(--ark-text-2)]">
                  <span>MIN: 12ms</span>
                  <span>MAX: 52ms</span>
                  <span>AVG: 33ms</span>
                  <span className="ml-auto text-[var(--success)] flex items-center gap-1">
                    <div className="w-1.5 h-1.5 bg-[var(--success)] rounded-full animate-pulse" />
                    SIGNAL STABLE
                  </span>
                </div>
              </div>

              <div className="ark-panel flex flex-col">
                <div className="border-b border-[var(--ark-border)] p-4 flex items-center justify-between bg-[var(--ark-panel-2)]">
                  <div className="flex items-center gap-2">
                    <Shield className="h-4 w-4" />
                    <span className="ark-kicker">SEVERITY MIX</span>
                  </div>
                </div>
                <div className="p-6 flex-1 flex flex-col justify-end gap-4 min-h-[240px]">
                  <div className="flex items-end gap-2 h-40">
                    {barSeries.map((value, index) => (
                      <div key={index} className="flex-1 flex flex-col justify-end group h-full gap-2">
                        <div 
                          className="w-full bg-[var(--ark-text-2)] opacity-20 group-hover:opacity-40 transition-opacity relative"
                          style={{ height: `${value}%` }}
                        >
                           {/* Top decorative line */}
                           <div className="absolute top-0 left-0 right-0 h-[2px] bg-[var(--ark-text)] opacity-0 group-hover:opacity-100 transition-opacity" />
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className="grid grid-cols-3 gap-2 pt-4 border-t border-[var(--ark-border)]">
                    {[
                      { l: "HIGH", v: 12, c: "text-[var(--error)]" },
                      { l: "MED", v: 38, c: "text-[var(--warning)]" },
                      { l: "LOW", v: 74, c: "text-[var(--info)]" }
                    ].map((item) => (
                      <div key={item.l} className="ark-slab p-2 text-center">
                        <div className={`text-xs font-bold ${item.c}`}>{item.v}</div>
                        <div className="text-[10px] text-[var(--ark-text-2)]">{item.l}</div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </section>

            {/* Bottom Section */}
            <section className="grid gap-6 lg:grid-cols-[2fr_1fr]">
              <div className="ark-panel p-0">
                <div className="p-4 border-b border-[var(--ark-border)] flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Terminal className="h-4 w-4" />
                    <span className="ark-kicker">RECENT OPERATIONS</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-[var(--ark-text-2)]">
                     <span className="w-2 h-2 border border-[var(--ark-text-2)] opacity-50" />
                     LIVE FEED
                  </div>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm text-left">
                    <thead className="bg-[var(--ark-panel-2)] text-xs text-[var(--ark-text-2)] font-mono border-b border-[var(--ark-border)]">
                      <tr>
                        <th className="px-6 py-3 font-medium">ID</th>
                        <th className="px-6 py-3 font-medium">TASK</th>
                        <th className="px-6 py-3 font-medium">OWNER</th>
                        <th className="px-6 py-3 font-medium">STATUS</th>
                        <th className="px-6 py-3 text-right font-medium">SCORE</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[var(--ark-border)]">
                      {tableRows.map((row) => (
                        <tr key={row.id} className="hover:bg-[var(--ark-bg)] transition-colors group">
                          <td className="px-6 py-3 font-mono text-xs text-[var(--ark-text-2)] group-hover:text-[var(--ark-active)] transition-colors">{row.id}</td>
                          <td className="px-6 py-3 font-medium">{row.name}</td>
                          <td className="px-6 py-3 text-xs text-[var(--ark-text-2)]">{row.owner}</td>
                          <td className="px-6 py-3">
                            <span className="ark-chip text-[10px]">{row.status}</span>
                          </td>
                          <td className="px-6 py-3 text-right font-mono">{row.score}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              <div className="ark-panel flex flex-col">
                <div className="p-4 border-b border-[var(--ark-border)]">
                  <span className="ark-kicker">CYCLE TIMING</span>
                </div>
                <div className="p-4 flex-1 space-y-4">
                  {timeline.map((item, i) => (
                    <div key={item.label} className="relative pl-6 pb-4 last:pb-0 border-l border-[var(--ark-border)] last:border-0">
                      <div className={`absolute left-[-5px] top-0 h-2.5 w-2.5 rounded-full border-2 border-[var(--ark-bg)] ${i===2 ? 'bg-[var(--info)] animate-pulse' : 'bg-[var(--ark-text-2)]'}`} />
                      <div className="ark-slab p-3 flex justify-between items-center -mt-2">
                        <span className="text-sm font-medium">{item.label}</span>
                        <span className="text-xs font-mono text-[var(--ark-text-2)]">{item.value}</span>
                      </div>
                    </div>
                  ))}
                </div>
                <div className="p-3 border-t border-[var(--ark-border)] text-xs text-center text-[var(--ark-text-2)] font-mono">
                  ALL SYSTEMS NOMINAL
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
          opacity: 0.15;
        }

        .ark-panel {
          background: var(--ark-panel);
          border: 1px solid var(--ark-border);
          border-top: 2px solid var(--ark-text); /* Top Accent Border */
          box-shadow: 0 1px 3px rgba(0,0,0,0.02);
        }

        .ark-slab {
          background: var(--ark-panel-2);
          border: 1px solid var(--ark-border);
        }

        .ark-kicker {
          font-size: 11px;
          letter-spacing: 0.15em;
          font-weight: 600;
          color: var(--ark-text-2);
          text-transform: uppercase;
        }

        .ark-nav-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 10px 16px;
          font-size: 13px;
          font-weight: 500;
          color: var(--ark-text-2);
          transition: color 0.2s, background-color 0.2s, border-color 0.2s, opacity 0.2s, transform 0.2s, box-shadow 0.2s;
          border-left: 2px solid transparent;
        }

        .ark-nav-item:hover {
          color: var(--ark-text);
          background: var(--ark-panel-2);
        }

        .ark-nav-item.active {
          color: var(--ark-text);
          background: var(--ark-panel-2);
          border-left-color: var(--ark-active);
        }

        .ark-chip {
          display: inline-block;
          padding: 2px 8px;
          border: 1px solid var(--ark-border);
          background: var(--ark-panel-2);
          color: var(--ark-text);
          font-family: var(--font-mono);
          letter-spacing: 0.05em;
        }
      `}</style>
    </main>
  )
}
