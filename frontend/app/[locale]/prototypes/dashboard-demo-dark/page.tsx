"use client"

import type { CSSProperties } from "react"
import {
  ArrowUpRight,
  BarChart3,
  LayoutGrid,
  Layers,
  Sliders,
  ShieldCheck,
  ShieldAlert,
  Activity,
  Waves,
} from "@/components/icons"

const themeVars = {
  "--bg": "#0F1114",
  "--panel": "#161A1F",
  "--panel-muted": "#1D232B",
  "--border": "#2B3138",
  "--border-strong": "#3E4753",
  "--text": "#E8EDF2",
  "--text-2": "#A4ADBA",
  "--muted": "#6B7481",
  "--shadow": "rgba(0, 0, 0, 0.35)",
  "--cut": "10px",
} as CSSProperties

const stats = [
  { label: "Active Assets", value: "1,284", delta: "+4.2%" },
  { label: "Scan Throughput", value: "82%", delta: "+1.1%" },
  { label: "Critical Alerts", value: "12", delta: "-3" },
  { label: "Coverage", value: "96.8%", delta: "+0.6%" },
]

const navItems = [
  { label: "Overview", icon: LayoutGrid },
  { label: "Assets", icon: Layers },
  { label: "Scans", icon: Activity },
  { label: "Threats", icon: ShieldAlert },
  { label: "Settings", icon: Sliders },
]

const timeline = [
  { label: "Ingestion", value: "02:14" },
  { label: "Correlation", value: "04:32" },
  { label: "Mitigation", value: "08:09" },
]

const tableRows = [
  { id: "OP-291", name: "Gateway Sweep", owner: "Ops-7", status: "Stable", score: "A" },
  { id: "OP-377", name: "Credential Drift", owner: "Ops-3", status: "Observe", score: "B" },
  { id: "OP-408", name: "Outbound Flux", owner: "Ops-2", status: "Investigate", score: "A-" },
  { id: "OP-512", name: "Shadow Asset", owner: "Ops-1", status: "Monitor", score: "B+" },
]

const barSeries = [48, 62, 54, 68, 76, 58, 64]
const lineSeries = [12, 24, 18, 36, 30, 44, 38, 52]

export default function DashboardDemoPage() {
  return (
    <main className="relative min-h-screen bg-[var(--bg)] text-[var(--text)]" style={themeVars}>
      <div className="pointer-events-none absolute inset-0 demo-grid" aria-hidden />
      <div className="pointer-events-none absolute left-6 top-10 h-24 w-24 demo-ring" aria-hidden />
      <div className="pointer-events-none absolute right-10 top-16 h-10 w-56 demo-beam" aria-hidden />

      <div className="relative mx-auto w-full max-w-7xl px-6 py-8">
        <div className="grid gap-6 lg:grid-cols-[220px_1fr]">
          <aside className="panel demo-sidebar p-4 lg:sticky lg:top-8">
            <div className="space-y-1 border-b border-[var(--border)] pb-4">
              <p className="kicker">LunaFox</p>
              <p className="text-sm font-semibold">Control Hub</p>
            </div>
            <nav className="mt-4 space-y-2">
              {navItems.map((item, index) => (
                <button
                  key={item.label}
                  className={`sidebar-item ${index === 0 ? "active" : ""}`}
                  type="button"
                >
                  <item.icon className="h-4 w-4" />
                  <span>{item.label}</span>
                </button>
              ))}
            </nav>
            <div className="mt-6 panel slab px-3 py-3 text-xs text-[var(--text-2)]">
              <p className="text-[var(--text)]">System Health</p>
              <p className="mt-2">Nominal · 99.1%</p>
            </div>
          </aside>

          <div className="flex flex-col gap-6">
            <header className="panel panel-hero flex flex-col gap-4 p-5 md:flex-row md:items-center md:justify-between">
              <div className="space-y-2">
                <p className="kicker">Operations Dashboard</p>
                <h1 className="text-xl font-semibold tracking-wide">System Overview</h1>
                <p className="text-sm text-[var(--text-2)]">Dark industrial minimal demo based on the current dashboard layout.</p>
              </div>
              <div className="flex flex-wrap gap-3 text-xs text-[var(--text-2)]">
                <span className="tag">Cycle · 14:32</span>
                <span className="tag">Nodes · 48</span>
                <span className="tag">Runtime · Stable</span>
              </div>
              <span className="panel-ornament" aria-hidden />
              <span className="panel-seam" aria-hidden />
            </header>

            <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {stats.map((item) => (
                <div key={item.label} className="panel slab p-4">
                  <p className="text-xs text-[var(--text-2)]">{item.label}</p>
                  <div className="mt-3 flex items-end justify-between">
                    <span className="text-2xl font-semibold">{item.value}</span>
                    <span className="text-xs text-[var(--text-2)]">{item.delta}</span>
                  </div>
                </div>
              ))}
            </section>

            <section className="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
              <div className="panel p-5">
                <span className="panel-ornament" aria-hidden />
                <span className="panel-seam" aria-hidden />
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <p className="kicker">Signal Trend</p>
                    <h2 className="text-base font-semibold">Asset Pulse</h2>
                  </div>
                  <span className="tag">Last 24h</span>
                </div>
                <div className="mt-4 chart-frame">
                  <span className="chart-notch" aria-hidden />
                  <svg viewBox="0 0 400 140" className="h-full w-full">
                    <polyline
                      fill="none"
                      stroke="var(--text)"
                      strokeWidth="2"
                      points={lineSeries
                        .map((value, index) => `${index * 55},${140 - value}`)
                        .join(" ")}
                    />
                  </svg>
                </div>
                <div className="mt-4 flex items-center gap-3 text-xs text-[var(--text-2)]">
                  <Waves className="h-4 w-4" />
                  <span>Variance within expected band.</span>
                </div>
              </div>

              <div className="panel p-5">
                <span className="panel-ornament" aria-hidden />
                <span className="panel-seam" aria-hidden />
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <p className="kicker">Distribution</p>
                    <h2 className="text-base font-semibold">Severity Mix</h2>
                  </div>
                  <span className="tag">Weekly</span>
                </div>
                <div className="mt-4 flex h-36 items-end gap-2">
                  {barSeries.map((value, index) => (
                    <span
                      key={`bar-${index}`}
                      className="bar"
                      style={{ height: `${value}%` }}
                    />
                  ))}
                </div>
                <div className="mt-4 grid grid-cols-3 gap-2 text-xs text-[var(--text-2)]">
                  {[
                    { label: "High", value: "12" },
                    { label: "Medium", value: "38" },
                    { label: "Low", value: "74" },
                  ].map((item) => (
                    <div key={item.label} className="panel slab flex items-center justify-between px-3 py-2">
                      <span>{item.label}</span>
                      <span className="text-[var(--text)]">{item.value}</span>
                    </div>
                  ))}
                </div>
              </div>
            </section>

            <section className="grid gap-4 lg:grid-cols-[1.4fr_0.8fr]">
              <div className="panel p-5">
                <span className="panel-ornament" aria-hidden />
                <span className="panel-seam" aria-hidden />
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <p className="kicker">Operations</p>
                    <h2 className="text-base font-semibold">Recent Activity</h2>
                  </div>
                  <span className="tag">Updated 2m ago</span>
                </div>
                <div className="mt-4 overflow-hidden rounded-none border border-[var(--border)]">
                  <table className="w-full text-sm">
                    <thead className="bg-[var(--panel-muted)] text-xs text-[var(--text-2)]">
                      <tr>
                        <th className="px-4 py-3 text-left">ID</th>
                        <th className="px-4 py-3 text-left">Task</th>
                        <th className="px-4 py-3 text-left">Owner</th>
                        <th className="px-4 py-3 text-left">Status</th>
                        <th className="px-4 py-3 text-right">Score</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tableRows.map((row) => (
                        <tr key={row.id} className="border-t border-[var(--border)]">
                          <td className="px-4 py-3 text-xs text-[var(--text-2)]">{row.id}</td>
                          <td className="px-4 py-3">{row.name}</td>
                          <td className="px-4 py-3 text-xs text-[var(--text-2)]">{row.owner}</td>
                          <td className="px-4 py-3 text-xs">
                            <span className="tag muted">{row.status}</span>
                          </td>
                          <td className="px-4 py-3 text-right">{row.score}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              <div className="panel p-5">
                <span className="panel-ornament" aria-hidden />
                <span className="panel-seam" aria-hidden />
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <p className="kicker">Response Loop</p>
                    <h2 className="text-base font-semibold">Cycle Timing</h2>
                  </div>
                  <ShieldCheck className="h-5 w-5" />
                </div>
                <div className="mt-4 space-y-3">
                  {timeline.map((item) => (
                    <div key={item.label} className="panel slab flex items-center justify-between px-4 py-3">
                      <span className="text-sm">{item.label}</span>
                      <span className="text-sm text-[var(--text-2)]">{item.value}</span>
                    </div>
                  ))}
                </div>
                <div className="mt-4 panel slab flex items-center justify-between px-4 py-3 text-xs text-[var(--text-2)]">
                  <span className="flex items-center gap-2">
                    <BarChart3 className="h-4 w-4" />
                    Output band stable
                  </span>
                  <ArrowUpRight className="h-4 w-4" />
                </div>
              </div>
            </section>
          </div>
        </div>
      </div>

      <style jsx>{`
        .demo-grid {
          background-image: none;
        }

        .demo-ring {
          border-radius: 999px;
          border: 2px solid var(--border-strong);
          background: rgba(232, 237, 242, 0.04);
        }

        .demo-beam {
          border-top: 2px solid var(--border-strong);
          border-bottom: 1px solid var(--border);
          opacity: 0.7;
        }

        .panel {
          position: relative;
          background: var(--panel);
          border: 1px solid var(--border);
          box-shadow: 0 6px 12px var(--shadow);
          clip-path: polygon(0 0, calc(100% - var(--cut)) 0, 100% var(--cut), 100% 100%, var(--cut) 100%, 0 calc(100% - var(--cut)));
        }

        .panel::before {
          content: "";
          position: absolute;
          left: 0;
          top: 0;
          width: 6px;
          height: 100%;
          background: rgba(232, 237, 242, 0.12);
          pointer-events: none;
        }

        .panel-hero::before {
          width: 8px;
          background: rgba(232, 237, 242, 0.18);
        }

        .panel-ornament {
          position: absolute;
          right: 14px;
          top: 14px;
          width: 28px;
          height: 14px;
          border-top: 2px solid var(--border-strong);
          border-right: 2px solid var(--border-strong);
          opacity: 0.6;
          pointer-events: none;
        }

        .panel-ornament::after {
          content: "";
          position: absolute;
          right: -6px;
          top: 6px;
          width: 12px;
          height: 2px;
          background: var(--border-strong);
        }

        .panel-seam {
          position: absolute;
          left: 16px;
          bottom: 12px;
          width: 140px;
          height: 0;
          border-top: 2px dashed rgba(232, 237, 242, 0.14);
          opacity: 0.6;
        }

        .panel-seam::after {
          content: "";
          position: absolute;
          right: -10px;
          top: -3px;
          width: 2px;
          height: 8px;
          background: rgba(232, 237, 242, 0.18);
        }

        .demo-sidebar {
          display: flex;
          flex-direction: column;
          gap: 16px;
          min-height: 520px;
        }

        .sidebar-item {
          display: flex;
          align-items: center;
          gap: 10px;
          width: 100%;
          padding: 8px 10px;
          border: 1px solid transparent;
          color: var(--text-2);
          background: transparent;
          font-size: 12px;
          letter-spacing: 0.12em;
          text-transform: uppercase;
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
          transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
          clip-path: polygon(0 0, calc(100% - 12px) 0, 100% 12px, 100% 100%, 12px 100%, 0 calc(100% - 12px));
          position: relative;
        }

        .sidebar-item::after {
          content: "";
          position: absolute;
          right: 10px;
          top: 50%;
          width: 6px;
          height: 6px;
          background: var(--border-strong);
          transform: translateY(-50%) rotate(45deg);
        }

        .sidebar-item:hover {
          border-color: var(--border);
          color: var(--text);
          background: var(--panel-muted);
        }

        .sidebar-item.active {
          border-color: var(--border-strong);
          color: var(--text);
          background: var(--panel-muted);
        }

        .sidebar-item.active::after {
          background: var(--text);
        }

        .panel::after {
          content: "";
          position: absolute;
          inset: 10px;
          border: 1px solid rgba(232, 237, 242, 0.06);
          pointer-events: none;
          clip-path: polygon(0 0, calc(100% - 8px) 0, 100% 8px, 100% 100%, 8px 100%, 0 calc(100% - 8px));
        }

        .slab {
          border-top: 2px solid var(--border-strong);
          box-shadow: none;
          clip-path: polygon(0 0, calc(100% - 8px) 0, 100% 8px, 100% 100%, 8px 100%, 0 calc(100% - 8px));
        }

        .kicker {
          font-size: 10px;
          letter-spacing: 0.3em;
          text-transform: uppercase;
          color: var(--text-2);
          font-weight: 600;
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
        }

        .tag {
          display: inline-flex;
          align-items: center;
          border: 1px solid var(--border);
          padding: 2px 10px;
          font-size: 10px;
          letter-spacing: 0.2em;
          text-transform: uppercase;
          color: var(--text-2);
          background: var(--panel-muted);
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
          clip-path: polygon(0 0, calc(100% - 8px) 0, 100% 8px, 100% 100%, 8px 100%, 0 calc(100% - 8px));
        }

        .tag.muted {
          background: transparent;
          border-color: var(--border-strong);
        }

        .chart-frame {
          height: 160px;
          border: 1px solid var(--border);
          background: var(--panel-muted);
          position: relative;
        }

        .chart-notch {
          position: absolute;
          right: 10px;
          top: 10px;
          width: 14px;
          height: 14px;
          border-top: 2px solid var(--border-strong);
          border-right: 2px solid var(--border-strong);
          opacity: 0.6;
        }

        .bar {
          flex: 1;
          background: var(--text);
          opacity: 0.18;
        }
      `}</style>
    </main>
  )
}
