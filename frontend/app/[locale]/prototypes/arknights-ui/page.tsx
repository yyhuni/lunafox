"use client"

import type { CSSProperties } from "react"
import {
  Activity,
  AlarmClock,
  AlignLeft,
  Crosshair,
  Database,
  Hexagon,
  Radar,
  Shield,
  Target,
  Zap,
} from "@/components/icons"

const themeVars = {
  "--ark-bg": "#FFFFFF",
  "--ark-bg-2": "#FFFFFF",
  "--ark-panel": "#FFFFFF",
  "--ark-panel-2": "#F6F7F9",
  "--ark-border": "#E4E8ED",
  "--ark-border-strong": "#CCD3DC",
  "--ark-text": "#20252B",
  "--ark-text-2": "#6E7682",
  "--ark-muted": "#A0A7B2",
  "--ark-accent": "#20252B",
  "--ark-accent-2": "#20252B",
  "--ark-warn": "#20252B",
  "--ark-danger": "#20252B",
  fontFamily: "\"HarmonyOS Sans SC\", \"IBM Plex Sans\", \"Segoe UI\", sans-serif",
} as CSSProperties

const quickStats = [
  { label: "Signal Integrity", value: "98.2%", delta: "+1.4%" },
  { label: "Relay Latency", value: "19 ms", delta: "-4 ms" },
  { label: "Threat Index", value: "0.32", delta: "Stable" },
]

const modules = [
  {
    title: "Operation Overview",
    desc: "Live telemetry for core objectives and readiness.",
    status: "Active",
    accent: "var(--ark-accent)",
  },
  {
    title: "Asset Recon",
    desc: "27 assets scanned · 6 elevated risk vectors.",
    status: "Recon",
    accent: "var(--ark-accent-2)",
  },
  {
    title: "Defense Matrix",
    desc: "Critical services hardened · 2 patches pending.",
    status: "Shield",
    accent: "var(--ark-muted)",
  },
]

const queue = [
  { id: "RX-102", name: "Perimeter Sweep", status: "Queued" },
  { id: "LF-551", name: "Payload Trace", status: "Executing" },
  { id: "LX-209", name: "Credential Drift", status: "Hold" },
]

const alerts = [
  {
    title: "Outbound anomaly on Node A-3",
    meta: "Packet ratio +18% · 2m ago",
    level: "High",
  },
  {
    title: "New fingerprint detected",
    meta: "Signature: ALB-17 · 11m ago",
    level: "Medium",
  },
  {
    title: "Backup cycle complete",
    meta: "Snapshot #311 · 28m ago",
    level: "Low",
  },
]

const assets = [
  { id: "S-14", type: "Web Gateway", status: "Stable", score: "A" },
  { id: "K-07", type: "Data Relay", status: "Monitor", score: "B" },
  { id: "P-21", type: "API Node", status: "Harden", score: "A-" },
  { id: "R-02", type: "Edge Cache", status: "Observe", score: "C+" },
]

const navItems = [
  { id: "radar", Icon: Radar },
  { id: "target", Icon: Target },
  { id: "crosshair", Icon: Crosshair },
  { id: "shield", Icon: Shield },
  { id: "database", Icon: Database },
]

export default function ArknightsUiPrototypePage() {
  return (
    <main
      className="relative min-h-screen overflow-hidden bg-[var(--ark-bg)] text-[var(--ark-text)]"
      style={themeVars}
    >
      <div className="pointer-events-none absolute inset-0 ark-grid" aria-hidden />
      <div className="pointer-events-none absolute left-8 top-14 h-40 w-40 industrial-ring" aria-hidden />
      <div className="pointer-events-none absolute right-12 top-24 h-20 w-64 industrial-slab" aria-hidden />
      <div className="pointer-events-none absolute left-16 bottom-16 h-14 w-72 industrial-slab thin" aria-hidden />
      <div className="pointer-events-none absolute left-12 top-40 h-[3px] w-[260px] industrial-beam" aria-hidden />

      <div className="relative mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-8 lg:px-10">
        <header className="ark-panel ark-hero flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center border border-[var(--ark-border)] bg-[var(--ark-panel-2)]">
              <Hexagon className="h-6 w-6 text-[var(--ark-accent)]" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-[var(--ark-text-2)]">LunaFox Bauhaus Console</p>
              <h1 className="text-xl font-semibold tracking-wide">Bauhaus / Industrial UI</h1>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-xs text-[var(--ark-text-2)]">
            <span className="ark-tag">ZONE-09</span>
            <span className="ark-tag">OPERATOR READY</span>
            <span className="ark-tag">
              <AlarmClock className="h-3.5 w-3.5 text-[var(--ark-accent)]" />
              21:06:44 UTC
            </span>
          </div>
        </header>

        <section className="grid grid-cols-12 gap-4">
          <aside className="col-span-12 flex gap-3 lg:col-span-2 lg:flex-col">
            {navItems.map(({ id, Icon }) => (
              <button
                key={id}
                className="ark-nav"
                type="button"
                aria-label={`Open ${id}`}
              >
                <Icon className="h-5 w-5" />
              </button>
            ))}
          </aside>

          <div className="col-span-12 grid gap-4 lg:col-span-7">
            <div className="ark-panel">
              <div className="flex flex-col gap-4 border-b border-[var(--ark-border)] p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="ark-kicker">PRIMARY STATUS</p>
                    <h2 className="text-lg font-semibold">Operation Mercury</h2>
                  </div>
                  <span className="ark-tag">STAGE 3 · ACTIVE</span>
                </div>
                <div className="grid gap-3 sm:grid-cols-3">
                  {quickStats.map((stat) => (
                    <div key={stat.label} className="ark-slab p-3">
                      <p className="text-xs text-[var(--ark-text-2)]">{stat.label}</p>
                      <div className="mt-2 flex items-center justify-between">
                        <span className="text-lg font-semibold text-[var(--ark-accent)]">{stat.value}</span>
                        <span className="text-xs text-[var(--ark-text-2)]">{stat.delta}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              <div className="grid gap-4 p-4 lg:grid-cols-[1.2fr_0.8fr]">
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <p className="ark-kicker">TACTICAL FEED</p>
                    <button className="ark-button" type="button">Deploy</button>
                  </div>
                  <div className="ark-slab ark-graph h-36 p-3">
                    <div className="flex h-full items-end gap-2">
                      {Array.from({ length: 18 }).map((_, index) => (
                        <span
                          key={`bar-${index}`}
                          className="w-full rounded-sm"
                          style={{
                            height: `${20 + (index % 6) * 12}%`,
                            background: "linear-gradient(180deg, rgba(60,66,74,0.85), rgba(60,66,74,0.2))",
                          }}
                        />
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center justify-between text-xs text-[var(--ark-text-2)]">
                    <span>Last sync 42s ago</span>
                    <span className="flex items-center gap-2 text-[var(--ark-text-2)]">
                      <Activity className="h-3.5 w-3.5" />
                      Live streaming
                    </span>
                  </div>
                </div>
                <div className="space-y-3">
                  <p className="ark-kicker">QUEUE</p>
                  <div className="space-y-2">
                    {queue.map((item) => (
                      <div key={item.id} className="ark-slab flex items-center justify-between px-3 py-2">
                        <div>
                          <p className="text-sm font-medium">{item.name}</p>
                          <p className="text-xs text-[var(--ark-text-2)]">{item.id}</p>
                        </div>
                        <span className="ark-chip">{item.status}</span>
                      </div>
                    ))}
                  </div>
                  <div className="ark-slab p-3 text-xs text-[var(--ark-text-2)]">
                    Next dispatch in 06:12 · Priority window open.
                  </div>
                </div>
              </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-3">
              {modules.map((module) => (
                <div key={module.title} className="ark-panel">
                  <div className="border-b border-[var(--ark-border)] p-4">
                    <div className="flex items-center justify-between">
                      <p className="ark-kicker">MODULE</p>
                      <span className="ark-dot" style={{ background: module.accent }} />
                    </div>
                    <h3 className="mt-2 text-base font-semibold">{module.title}</h3>
                  </div>
                  <div className="space-y-3 p-4 text-sm text-[var(--ark-text-2)]">
                    <p>{module.desc}</p>
                    <div className="flex items-center justify-between text-xs">
                      <span>Status</span>
                      <span className="text-[var(--ark-text)]">{module.status}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <aside className="col-span-12 grid gap-4 lg:col-span-3">
            <div className="ark-panel">
              <div className="border-b border-[var(--ark-border)] p-4">
                <div className="flex items-center justify-between">
                  <p className="ark-kicker">SYSTEM ALERTS</p>
                  <Zap className="h-4 w-4 text-[var(--ark-text-2)]" />
                </div>
              </div>
              <div className="space-y-3 p-4">
                {alerts.map((alert) => (
                  <div key={alert.title} className="ark-slab p-3">
                    <div className="flex items-center justify-between">
                      <p className="text-sm font-medium">{alert.title}</p>
                      <span className="ark-chip">{alert.level}</span>
                    </div>
                    <p className="mt-2 text-xs text-[var(--ark-text-2)]">{alert.meta}</p>
                  </div>
                ))}
              </div>
            </div>

            <div className="ark-panel">
              <div className="border-b border-[var(--ark-border)] p-4">
                <div className="flex items-center justify-between">
                  <p className="ark-kicker">ASSET MATRIX</p>
                  <AlignLeft className="h-4 w-4 text-[var(--ark-accent)]" />
                </div>
              </div>
              <div className="space-y-2 p-4 text-xs">
                {assets.map((asset) => (
                  <div key={asset.id} className="ark-slab grid grid-cols-[0.6fr_1.4fr_0.8fr_0.4fr] items-center gap-2 px-2 py-2">
                    <span className="text-[var(--ark-accent)]">{asset.id}</span>
                    <span>{asset.type}</span>
                    <span className="text-[var(--ark-text-2)]">{asset.status}</span>
                    <span className="text-right text-[var(--ark-text)]">{asset.score}</span>
                  </div>
                ))}
              </div>
            </div>
          </aside>
        </section>

        <section className="ark-panel">
          <div className="flex flex-col gap-3 border-b border-[var(--ark-border)] p-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p className="ark-kicker">TACTICAL NOTES</p>
              <h2 className="text-lg font-semibold">Operator Briefing</h2>
            </div>
            <button className="ark-button ghost" type="button">View Log</button>
          </div>
          <div className="grid gap-4 p-4 lg:grid-cols-[1.3fr_0.7fr]">
            <div className="space-y-3 text-sm text-[var(--ark-text-2)]">
              <p>
                Signal mesh stabilized. Maintain perimeter scans and isolate anomalous traffic.
                Deploy countermeasures on request from Sector-7 control.
              </p>
              <div className="flex flex-wrap gap-2 text-xs">
                {[
                  "Clearance: Sigma",
                  "Auto-patch enabled",
                  "Fallback route locked",
                  "Ops window: 03:00",
                ].map((tag) => (
                  <span key={tag} className="ark-tag">{tag}</span>
                ))}
              </div>
            </div>
            <div className="ark-slab space-y-2 p-4 text-xs text-[var(--ark-text-2)]">
              <p className="flex items-center justify-between">
                <span>Resupply ETA</span>
                <span className="text-[var(--ark-text)]">00:28:15</span>
              </p>
              <p className="flex items-center justify-between">
                <span>Shield sync</span>
                <span className="text-[var(--ark-text)]">96.4%</span>
              </p>
              <p className="flex items-center justify-between">
                <span>Command link</span>
                <span className="text-[var(--ark-accent)]">Stable</span>
              </p>
            </div>
          </div>
        </section>
      </div>

      <style jsx>{`
        .ark-grid {
          background-image:
            linear-gradient(rgba(32, 37, 43, 0.045) 1px, transparent 1px),
            linear-gradient(90deg, rgba(32, 37, 43, 0.045) 1px, transparent 1px),
            linear-gradient(rgba(32, 37, 43, 0.02) 1px, transparent 1px),
            linear-gradient(90deg, rgba(32, 37, 43, 0.02) 1px, transparent 1px);
          background-size: 120px 120px, 120px 120px, 24px 24px, 24px 24px;
          mask-image: linear-gradient(180deg, transparent 0%, black 8%, black 92%, transparent 100%);
        }

        .industrial-ring {
          border-radius: 999px;
          border: 2px solid var(--ark-border-strong);
          background: rgba(32, 37, 43, 0.015);
          box-shadow: inset 0 0 0 10px rgba(32, 37, 43, 0.01);
          opacity: 0.7;
        }

        .industrial-slab {
          border: 2px solid var(--ark-border);
          background: var(--ark-panel);
          box-shadow: 0 4px 10px rgba(21, 26, 32, 0.025);
          opacity: 0.85;
        }

        .industrial-slab.thin {
          border-color: var(--ark-border-strong);
          background: var(--ark-panel-2);
          box-shadow: 0 4px 8px rgba(21, 26, 32, 0.02);
        }

        .industrial-beam {
          background: linear-gradient(90deg, rgba(32, 37, 43, 0.25), transparent);
          opacity: 0.7;
        }

        .ark-panel {
          position: relative;
          background: var(--ark-panel);
          border: 1px solid var(--ark-border);
          border-top: 2px solid var(--ark-border-strong);
          box-shadow: 0 6px 12px rgba(21, 26, 32, 0.035);
        }

        .ark-panel::after {
          content: "";
          position: absolute;
          inset: 10px;
          border: 1px solid rgba(32, 37, 43, 0.03);
          pointer-events: none;
        }

        .ark-panel.ark-hero::before {
          content: "";
          position: absolute;
          left: 0;
          top: 0;
          width: 4px;
          height: 100%;
          background: var(--ark-border-strong);
          pointer-events: none;
        }

        .ark-slab {
          position: relative;
          background: var(--ark-panel-2);
          border: 1px solid var(--ark-border);
          border-top: 2px solid rgba(32, 37, 43, 0.12);
        }

        .ark-slab::after {
          content: "";
          position: absolute;
          inset: 6px;
          border: 1px solid rgba(32, 37, 43, 0.025);
          pointer-events: none;
        }

        .ark-graph {
          background-image:
            linear-gradient(90deg, rgba(32, 37, 43, 0.07) 1px, transparent 1px),
            linear-gradient(180deg, rgba(32, 37, 43, 0.07) 1px, transparent 1px);
          background-size: 16px 16px;
        }

        .ark-nav {
          position: relative;
          display: inline-flex;
          align-items: center;
          justify-content: center;
          height: 52px;
          border: 1px solid var(--ark-border);
          border-left: 3px solid var(--ark-border-strong);
          background: var(--ark-panel);
          color: var(--ark-text-2);
          transition: border-color 0.2s ease, color 0.2s ease;
        }

        .ark-nav:hover {
          color: var(--ark-text);
          border-color: var(--ark-border-strong);
        }

        .ark-kicker {
          font-size: 10px;
          letter-spacing: 0.3em;
          text-transform: uppercase;
          color: var(--ark-text-2);
          font-weight: 600;
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
        }

        .ark-tag {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          border: 1px solid var(--ark-border);
          padding: 3px 10px;
          font-size: 10px;
          letter-spacing: 0.22em;
          text-transform: uppercase;
          color: var(--ark-text-2);
          background: var(--ark-bg-2);
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
          box-shadow: inset 0 0 0 1px rgba(29, 36, 44, 0.04);
        }

        .ark-chip {
          border: 1px solid var(--ark-border);
          padding: 2px 8px;
          font-size: 10px;
          text-transform: uppercase;
          letter-spacing: 0.2em;
          color: var(--ark-text-2);
          background: transparent;
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
        }

        .ark-button {
          border: 2px solid var(--ark-border-strong);
          background: var(--ark-accent);
          color: #ffffff;
          padding: 6px 14px;
          font-size: 12px;
          text-transform: uppercase;
          letter-spacing: 0.22em;
          font-family: "IBM Plex Mono", "HarmonyOS Sans SC", sans-serif;
          transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
          box-shadow: 0 4px 8px rgba(21, 26, 32, 0.08);
        }

        .ark-button:hover {
          background: #14181e;
        }

        .ark-button.ghost {
          border-color: var(--ark-border-strong);
          background: transparent;
          color: var(--ark-text-2);
        }

        .ark-dot {
          width: 10px;
          height: 10px;
          border-radius: 999px;
        }
      `}</style>
    </main>
  )
}
