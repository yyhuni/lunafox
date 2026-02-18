"use client"

import type { CSSProperties } from "react"
import {
  Activity,
  Crosshair,
  Database,
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
  fontFamily: "\"HarmonyOS Sans SC\", \"IBM Plex Sans\", \"Segoe UI\", sans-serif",
} as CSSProperties

const pulseSeries = [182, 196, 210, 204, 228, 246, 260]
const dayLabels = ["M", "T", "W", "T", "F", "S", "S"]
const laneSeries = {
  subdomains: [76, 82, 88, 90, 96, 101, 108],
  websites: [28, 32, 36, 35, 39, 43, 47],
  ips: [22, 24, 25, 27, 30, 33, 36],
  endpoints: [54, 58, 63, 66, 70, 74, 79],
}

function sparkPath(data: number[], width: number, height: number, padding = 2) {
  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = Math.max(1, max - min)
  const step = (width - padding * 2) / (data.length - 1)
  return data
    .map((value, index) => {
      const x = padding + step * index
      const y = height - padding - ((value - min) / range) * (height - padding * 2)
      return `${index === 0 ? "M" : "L"}${x} ${y}`
    })
    .join(" ")
}

function Sparkline({
  data,
  width = 180,
  height = 46,
}: {
  data: number[]
  width?: number
  height?: number
}) {
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} aria-hidden>
      <path
        d={sparkPath(data, width, height, 2)}
        fill="none"
        stroke="var(--ark-accent)"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

export default function AssetPulseArknightsPage() {
  const stackTotals = dayLabels.map((_, i) =>
    laneSeries.subdomains[i] + laneSeries.websites[i] + laneSeries.ips[i] + laneSeries.endpoints[i]
  )
  const maxStack = Math.max(...stackTotals)
  const trendDelta = pulseSeries.at(-1)! - pulseSeries[0]

  return (
    <main
      className="relative min-h-screen overflow-hidden bg-[var(--ark-bg)] text-[var(--ark-text)]"
      style={themeVars}
    >
      <div className="pointer-events-none absolute inset-0 ark-grid" aria-hidden />
      <div className="pointer-events-none absolute left-10 top-16 h-32 w-32 industrial-ring" aria-hidden />
      <div className="pointer-events-none absolute right-12 top-20 h-16 w-56 industrial-slab" aria-hidden />
      <div className="pointer-events-none absolute left-10 top-40 h-[3px] w-[240px] industrial-beam" aria-hidden />

      <div className="relative mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-8 lg:px-10">
        <header className="ark-panel ark-hero flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center border border-[var(--ark-border)] bg-[var(--ark-panel-2)]">
              <Target className="h-6 w-6 text-[var(--ark-accent)]" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-[var(--ark-text-2)]">ASSET PULSE</p>
              <h1 className="text-xl font-semibold tracking-wide">Arknights Style Variants</h1>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-xs text-[var(--ark-text-2)]">
            <span className="ark-tag">ZONE-09</span>
            <span className="ark-tag">SIGNAL STABLE</span>
            <span className="ark-tag">
              <Activity className="h-3.5 w-3.5 text-[var(--ark-accent)]" />
              LIVE FEED
            </span>
          </div>
        </header>

        <section className="grid grid-cols-12 gap-4">
          {/* Demo 1: Pulse Rail */}
          <div className="ark-panel col-span-12 lg:col-span-7">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">PULSE RAIL</p>
                <span className="ark-chip">ACTIVE</span>
              </div>
              <div className="mt-3 flex items-end justify-between">
                <div>
                  <p className="text-xs text-[var(--ark-text-2)]">Total Assets</p>
                  <div className="text-3xl font-semibold">{pulseSeries.at(-1)}</div>
                  <p className="mt-1 text-xs text-[var(--ark-text-2)]">Δ 7D +{pulseSeries.at(-1)! - pulseSeries[0]}</p>
                </div>
                <div className="ark-slab px-3 py-2">
                  <Sparkline data={pulseSeries} width={200} height={50} />
                </div>
              </div>
            </div>
            <div className="grid gap-3 p-4 sm:grid-cols-3">
              {[
                { label: "Subdomains", value: laneSeries.subdomains.at(-1) },
                { label: "Websites", value: laneSeries.websites.at(-1) },
                { label: "Endpoints", value: laneSeries.endpoints.at(-1) },
              ].map((stat) => (
                <div key={stat.label} className="ark-slab p-3">
                  <p className="text-xs text-[var(--ark-text-2)]">{stat.label}</p>
                  <div className="mt-2 text-lg font-semibold">{stat.value}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Demo 2: Signal Matrix */}
          <div className="ark-panel col-span-12 lg:col-span-5">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">SIGNAL MATRIX</p>
                <Radar className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="grid gap-3 p-4 sm:grid-cols-2">
              {Object.entries(laneSeries).map(([label, series]) => (
                <div key={label} className="ark-slab p-3">
                  <div className="flex items-center justify-between text-xs text-[var(--ark-text-2)]">
                    <span className="capitalize">{label}</span>
                    <span className="text-[var(--ark-text)]">{series.at(-1)}</span>
                  </div>
                  <div className="mt-2">
                    <Sparkline data={series} width={140} height={34} />
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Demo 3: Radar Sweep */}
          <div className="ark-panel col-span-12 lg:col-span-4">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">RADAR SWEEP</p>
                <Crosshair className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="p-4">
              <div className="relative h-44 rounded-md border border-[var(--ark-border)] bg-[var(--ark-panel-2)]">
                <div className="absolute inset-4 rounded-full border border-[var(--ark-border)]" />
                <div className="absolute inset-10 rounded-full border border-[var(--ark-border)]" />
                <div className="absolute inset-16 rounded-full border border-[var(--ark-border)]" />
                <div className="absolute inset-0 arkp-radar" />
                <div className="absolute left-[25%] top-[30%] h-2 w-2 rounded-full bg-[var(--ark-accent)]/70 animate-ping" />
                <div className="absolute left-[60%] top-[55%] h-1.5 w-1.5 rounded-full bg-[var(--ark-accent)]/70 animate-ping [animation-delay:0.8s]" />
                <div className="absolute left-[50%] top-[70%] h-2 w-2 rounded-full bg-[var(--ark-accent)]/70 animate-ping [animation-delay:1.6s]" />
              </div>
              <div className="mt-3 flex items-center justify-between text-xs text-[var(--ark-text-2)]">
                <span>Contacts: 18</span>
                <span className="ark-chip">ACTIVE</span>
              </div>
            </div>
          </div>

          {/* Demo 4: Ops Queue */}
          <div className="ark-panel col-span-12 lg:col-span-4">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">OPS QUEUE</p>
                <Shield className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="space-y-2 p-4 text-sm">
              {[
                { id: "RX-112", name: "Perimeter Sweep", status: "Queued" },
                { id: "LF-551", name: "Ingress Trace", status: "Executing" },
                { id: "LK-044", name: "Key Rotation", status: "Hold" },
              ].map((item) => (
                <div key={item.id} className="ark-slab flex items-center justify-between px-3 py-2">
                  <div>
                    <p className="text-sm font-medium">{item.name}</p>
                    <p className="text-xs text-[var(--ark-text-2)]">{item.id}</p>
                  </div>
                  <span className="ark-chip">{item.status}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Demo 5: Asset Lattice */}
          <div className="ark-panel col-span-12 lg:col-span-4">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">ASSET LATTICE</p>
                <Database className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="p-4">
              <div className="grid grid-cols-8 gap-1">
                {Array.from({ length: 40 }).map((_, index) => (
                  <div
                    key={index}
                    className={`h-3 rounded-sm ${index % 6 === 0 ? "bg-[var(--ark-accent)]/40" : "bg-[var(--ark-border)]"}`}
                  />
                ))}
              </div>
              <div className="mt-3 flex items-center justify-between text-xs text-[var(--ark-text-2)]">
                <span>Nodes: 40</span>
                <span className="flex items-center gap-2">
                  <span className="h-1.5 w-1.5 rounded-full bg-[var(--ark-accent)] animate-pulse" />
                  SYNCED
                </span>
              </div>
            </div>
          </div>
        </section>

        <section className="grid grid-cols-12 gap-4">
          {/* Demo 6: Telemetry Strip */}
          <div className="ark-panel col-span-12 lg:col-span-5">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">TELEMETRY STRIP</p>
                <Zap className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="p-4">
              <div className="relative overflow-hidden rounded-md border border-[var(--ark-border)] bg-[var(--ark-panel-2)] px-3 py-4">
                <div className="absolute inset-y-0 left-0 arkp-strip-line" />
                <div className="flex items-end justify-between gap-2">
                  {pulseSeries.map((value, index) => (
                    <div key={`strip-${index}`} className="flex flex-col items-center gap-2">
                      <div
                        className="w-2 rounded-sm bg-[var(--ark-accent)]/40"
                        style={{ height: `${12 + (value / pulseSeries.at(-1)!) * 26}px` }}
                      />
                      <span className="text-[10px] text-[var(--ark-text-2)]">{dayLabels[index]}</span>
                    </div>
                  ))}
                </div>
              </div>
              <div className="mt-3 flex items-center justify-between text-xs text-[var(--ark-text-2)]">
                <span>Trend Δ 7D +{trendDelta}</span>
                <span className="ark-chip">SYNCED</span>
              </div>
            </div>
          </div>

          {/* Demo 7: Pulse Dial */}
          <div className="ark-panel col-span-12 lg:col-span-3">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">PULSE DIAL</p>
                <Target className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="p-4 flex flex-col items-center gap-3">
              <div className="relative h-36 w-36">
                <div className="absolute inset-0 rounded-full arkp-dial" />
                <div className="absolute inset-4 rounded-full bg-[var(--ark-panel)] border border-[var(--ark-border)]" />
                <div className="absolute inset-0 flex items-center justify-center">
                  <div className="text-center">
                    <div className="text-2xl font-semibold">74%</div>
                    <div className="text-[10px] text-[var(--ark-text-2)]">Integrity</div>
                  </div>
                </div>
              </div>
              <div className="text-xs text-[var(--ark-text-2)]">Index stable · 4.2%</div>
            </div>
          </div>

          {/* Demo 8: Delta Stack */}
          <div className="ark-panel col-span-12 lg:col-span-4">
            <div className="border-b border-[var(--ark-border)] p-4">
              <div className="flex items-center justify-between">
                <p className="ark-kicker">DELTA STACK</p>
                <Database className="h-4 w-4 text-[var(--ark-text-2)]" />
              </div>
            </div>
            <div className="p-4">
              <div className="flex items-end justify-between gap-3">
                {dayLabels.map((day, index) => {
                  const total = stackTotals[index]
                  const height = Math.max(36, Math.round((total / maxStack) * 90))
                  return (
                    <div key={`stack-${day}-${index}`} className="flex flex-col items-center gap-2">
                      <div className="flex w-6 flex-col justify-end rounded-md border border-[var(--ark-border)] bg-[var(--ark-panel-2)] overflow-hidden" style={{ height }}>
                        <div style={{ height: `${(laneSeries.subdomains[index] / total) * 100}%` }} className="bg-[var(--ark-accent)]/35" />
                        <div style={{ height: `${(laneSeries.websites[index] / total) * 100}%` }} className="bg-[var(--ark-accent)]/25" />
                        <div style={{ height: `${(laneSeries.ips[index] / total) * 100}%` }} className="bg-[var(--ark-accent)]/2" />
                        <div style={{ height: `${(laneSeries.endpoints[index] / total) * 100}%` }} className="bg-[var(--ark-accent)]/45" />
                      </div>
                      <span className="text-[10px] text-[var(--ark-text-2)]">{day}</span>
                    </div>
                  )
                })}
              </div>
              <div className="mt-3 flex items-center justify-between text-xs text-[var(--ark-text-2)]">
                <span>Mix stability 0.94</span>
                <span className="ark-chip">LOCKED</span>
              </div>
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

        .arkp-radar {
          background: conic-gradient(from 180deg, rgba(32, 37, 43, 0), rgba(32, 37, 43, 0.25), rgba(32, 37, 43, 0));
          animation: radar 7.5s linear infinite;
        }

        @keyframes radar {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }

        .arkp-strip-line {
          width: 28%;
          background: linear-gradient(90deg, transparent, rgba(32, 37, 43, 0.35), transparent);
          animation: strip 3.6s linear infinite;
          opacity: 0.55;
        }

        @keyframes strip {
          from { transform: translateX(-70%); }
          to { transform: translateX(240%); }
        }

        .arkp-dial {
          background: conic-gradient(
            from -90deg,
            rgba(32, 37, 43, 0.8) 0deg,
            rgba(32, 37, 43, 0.8) 266deg,
            rgba(32, 37, 43, 0.15) 266deg,
            rgba(32, 37, 43, 0.15) 360deg
          );
          border-radius: 999px;
        }
      `}</style>
    </main>
  )
}
