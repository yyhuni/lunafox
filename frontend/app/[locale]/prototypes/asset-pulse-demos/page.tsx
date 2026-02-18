"use client"

import { PageHeader } from "@/components/common/page-header"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { IconActivity, IconRadar, IconScan, Signal, Target, Waves, Cpu, HardDrive, ShieldAlert, Database, Crosshair } from "@/components/icons"

const SERIES = {
  total: [180, 192, 205, 198, 222, 238, 256],
  subdomains: [82, 88, 93, 91, 98, 104, 112],
  websites: [28, 31, 34, 33, 37, 40, 44],
  ips: [24, 26, 25, 27, 30, 32, 35],
  endpoints: [46, 52, 57, 55, 61, 67, 76],
}

const DAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

const LEDGER_ITEMS = [
  { id: "GW-12", label: "Gateway Drift", status: "Elevated", series: SERIES.websites, tone: "bg-amber-500/70" },
  { id: "API-07", label: "API Exposure", status: "Watch", series: SERIES.endpoints, tone: "bg-blue-500/70" },
  { id: "SUB-31", label: "Subdomain Burst", status: "Stable", series: SERIES.subdomains, tone: "bg-emerald-500/70" },
]

const TILE_STATS = [
  { label: "Total", value: SERIES.total.at(-1)!, pct: 76, color: "rgba(34,197,94,0.9)" },
  { label: "Subdomains", value: SERIES.subdomains.at(-1)!, pct: 64, color: "rgba(59,130,246,0.9)" },
  { label: "Websites", value: SERIES.websites.at(-1)!, pct: 52, color: "rgba(16,185,129,0.9)" },
  { label: "Endpoints", value: SERIES.endpoints.at(-1)!, pct: 71, color: "rgba(234,179,8,0.9)" },
]

const ZONES = [
  { label: "Edge Perimeter", value: 74, tone: "bg-emerald-500/70" },
  { label: "Gateway Mesh", value: 61, tone: "bg-blue-500/70" },
  { label: "Core API", value: 48, tone: "bg-amber-500/70" },
  { label: "Watchlist", value: 29, tone: "bg-red-500/60" },
]

const METRICS = [
  { label: "Scan Throughput", value: "82%", delta: "+3.1%", tone: "text-emerald-500" },
  { label: "Detection Drift", value: "0.42", delta: "-0.03", tone: "text-emerald-500" },
  { label: "Exposure Index", value: "17.6", delta: "+1.2", tone: "text-amber-500" },
]

const TIMELINE_POINTS = [12, 18, 22, 16, 26, 30, 28]

const CONSTELLATION = [
  { x: "18%", y: "22%", size: "h-2 w-2" },
  { x: "42%", y: "30%", size: "h-1.5 w-1.5" },
  { x: "68%", y: "24%", size: "h-2 w-2" },
  { x: "28%", y: "58%", size: "h-1.5 w-1.5" },
  { x: "52%", y: "64%", size: "h-2 w-2" },
  { x: "74%", y: "58%", size: "h-1.5 w-1.5" },
]

const RISK_LEVELS = [
  { label: "Low", value: 62, tone: "from-emerald-500/70 to-emerald-500/20" },
  { label: "Medium", value: 38, tone: "from-amber-500/70 to-amber-500/20" },
  { label: "High", value: 21, tone: "from-orange-500/70 to-orange-500/20" },
  { label: "Critical", value: 9, tone: "from-red-500/70 to-red-500/20" },
]

const SHARD_ROWS = [
  { id: "GW-21", label: "Gateway Drift", series: SERIES.websites },
  { id: "API-08", label: "Endpoint Surge", series: SERIES.endpoints },
  { id: "SUB-33", label: "Subdomain Bloom", series: SERIES.subdomains },
  { id: "IP-11", label: "Edge Spread", series: SERIES.ips },
]

const TAG_ITEMS = [
  { label: "Active Targets", value: 128 },
  { label: "Staged Scans", value: 24 },
  { label: "Watchlist", value: 9 },
  { label: "Blocked", value: 3 },
  { label: "Queued", value: 17 },
  { label: "Verified", value: 64 },
]

const DISPATCH_STEPS = [
  { label: "Queue", value: 28 },
  { label: "Analyze", value: 52 },
  { label: "Patch", value: 64 },
  { label: "Verify", value: 81 },
]

function sparklinePath(data: number[], width: number, height: number, padding = 2) {
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

function areaPath(data: number[], width: number, height: number, padding = 2) {
  const line = sparklinePath(data, width, height, padding)
  const endX = width - padding
  const baseY = height - padding
  return `${line} L ${endX} ${baseY} L ${padding} ${baseY} Z`
}

function Sparkline({
  data,
  width = 140,
  height = 36,
  color,
  fill,
}: {
  data: number[]
  width?: number
  height?: number
  color: string
  fill?: string
}) {
  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className="block"
      aria-hidden
    >
      {fill && (
        <path
          d={areaPath(data, width, height, 2)}
          fill={fill}
          opacity={0.25}
        />
      )}
      <path
        d={sparklinePath(data, width, height, 2)}
        fill="none"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

export default function AssetPulseDemosPage() {
  const delta = SERIES.total[SERIES.total.length - 1] - SERIES.total[SERIES.total.length - 2]
  const deltaSign = delta >= 0 ? "+" : ""
  const stackTotals = DAYS.map((_, i) =>
    SERIES.subdomains[i] + SERIES.websites[i] + SERIES.ips[i] + SERIES.endpoints[i]
  )
  const maxStack = Math.max(...stackTotals)

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <PageHeader
        code="DASH-PULSE"
        title="Asset Pulse Demos"
        description="与当前主题一致的 ASSET PULSE 视觉方案（30 套）"
      />

      <div className="grid gap-6 px-4 lg:px-6 md:grid-cols-2">
        {/* Demo 1: Pulse Rail */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-mono tracking-[0.2em] text-muted-foreground">
              <IconActivity className="h-4 w-4 text-primary" />
              ASSET PULSE
            </div>
            <span className="text-xs text-muted-foreground">Last 7 days</span>
          </CardHeader>
          <CardContent className="relative">
            <div className="flex items-end justify-between gap-6">
              <div>
                <p className="text-sm text-muted-foreground">Total Assets</p>
                <div className="text-3xl font-semibold">{SERIES.total.at(-1)}</div>
                <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="text-emerald-500">{deltaSign}{delta}</span>
                  <span>since yesterday</span>
                </div>
              </div>
              <div className="flex flex-col items-end gap-2">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                  SIGNAL STABLE
                </div>
                <Sparkline data={SERIES.total} width={200} height={46} color="var(--primary)" fill="var(--primary)" />
              </div>
            </div>
            <div className="mt-4 flex items-center gap-3 text-xs text-muted-foreground">
              <span className="px-2 py-1 rounded-md bg-muted/60">Peak {Math.max(...SERIES.total)}</span>
              <span className="px-2 py-1 rounded-md bg-muted/60">Min {Math.min(...SERIES.total)}</span>
              <span className="px-2 py-1 rounded-md bg-muted/60">Δ 7D {SERIES.total.at(-1)! - SERIES.total[0]}</span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 2: Signal Matrix */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Signal className="h-4 w-4 text-primary" />
              Signal Matrix
            </CardTitle>
            <span className="text-xs text-muted-foreground">Weekly micro‑trends</span>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-3">
              {[
                { label: "Subdomains", data: SERIES.subdomains, color: "#3b82f6" },
                { label: "Websites", data: SERIES.websites, color: "#22c55e" },
                { label: "IPs", data: SERIES.ips, color: "#f97316" },
                { label: "Endpoints", data: SERIES.endpoints, color: "#eab308" },
              ].map((item) => (
                <div
                  key={item.label}
                  className="rounded-lg border border-border/60 bg-muted/20 p-3 flex flex-col gap-2"
                >
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>{item.label}</span>
                    <span className="font-medium text-foreground/80">{item.data.at(-1)}</span>
                  </div>
                  <Sparkline data={item.data} width={140} height={32} color={item.color} />
                </div>
              ))}
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Hover in main chart to sync focus</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-primary animate-ping" />
                LIVE FEED
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 3: Dual‑Track */}
        <Card className="relative overflow-hidden md:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconRadar className="h-4 w-4 text-primary" />
              Dual‑Track Pulse
            </CardTitle>
            <span className="text-xs text-muted-foreground">Structure + growth</span>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between gap-6">
              <div>
                <p className="text-sm text-muted-foreground">Total Assets</p>
                <div className="text-3xl font-semibold">{SERIES.total.at(-1)}</div>
                <div className="mt-1 text-xs text-muted-foreground">
                  7D trend: {SERIES.total[0]} → {SERIES.total.at(-1)}
                </div>
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <IconScan className="h-4 w-4 text-primary" />
                Pulse window: 7D
              </div>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/10 p-3">
              <Sparkline
                data={SERIES.total}
                width={520}
                height={60}
                color="var(--primary)"
                fill="var(--primary)"
              />
            </div>
            <div className="flex items-end justify-between gap-3">
              {DAYS.map((day, i) => {
                const total = stackTotals[i]
                const height = Math.max(36, Math.round((total / maxStack) * 70))
                return (
                  <div key={day} className="flex flex-col items-center gap-2">
                    <div className="flex w-6 flex-col justify-end rounded-md border border-border/60 bg-muted/20 overflow-hidden" style={{ height }}>
                      <div style={{ height: `${(SERIES.subdomains[i] / total) * 100}%` }} className="bg-blue-500/70" />
                      <div style={{ height: `${(SERIES.websites[i] / total) * 100}%` }} className="bg-emerald-500/70" />
                      <div style={{ height: `${(SERIES.ips[i] / total) * 100}%` }} className="bg-orange-500/70" />
                      <div style={{ height: `${(SERIES.endpoints[i] / total) * 100}%` }} className="bg-yellow-500/70" />
                    </div>
                    <span className="text-[10px] text-muted-foreground">{day}</span>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>

        {/* Demo 4: Radar Sweep */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconRadar className="h-4 w-4 text-primary" />
              Radar Sweep
            </CardTitle>
            <span className="text-xs text-muted-foreground">Live perimeter</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-52 rounded-lg border border-border/60 bg-muted/10 overflow-hidden">
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(34,197,94,0.12)_1px,transparent_1px)] [background-size:12px_12px]" />
              <div className="absolute inset-6 rounded-full border border-emerald-500/20" />
              <div className="absolute inset-12 rounded-full border border-emerald-500/20" />
              <div className="absolute inset-20 rounded-full border border-emerald-500/20" />
              <div className="absolute inset-0 flex items-center justify-center">
                <Target className="h-6 w-6 text-emerald-400/80" />
              </div>
              <div className="absolute inset-0 radar-sweep" />
              <div className="absolute left-[20%] top-[30%] h-2 w-2 rounded-full bg-emerald-400/70 animate-ping" />
              <div className="absolute left-[65%] top-[40%] h-1.5 w-1.5 rounded-full bg-emerald-400/70 animate-ping [animation-delay:0.8s]" />
              <div className="absolute left-[55%] top-[70%] h-2 w-2 rounded-full bg-emerald-400/70 animate-ping [animation-delay:1.6s]" />
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Contacts: 18</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                SCAN ACTIVE
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 5: Pulse Lanes */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Waves className="h-4 w-4 text-primary" />
              Pulse Lanes
            </CardTitle>
            <span className="text-xs text-muted-foreground">Traffic bands</span>
          </CardHeader>
          <CardContent className="space-y-4">
            {[
              { label: "Subdomains", value: SERIES.subdomains.at(-1)!, tone: "bg-blue-500/60" },
              { label: "Websites", value: SERIES.websites.at(-1)!, tone: "bg-emerald-500/60" },
              { label: "IPs", value: SERIES.ips.at(-1)!, tone: "bg-orange-500/60" },
              { label: "Endpoints", value: SERIES.endpoints.at(-1)!, tone: "bg-yellow-500/60" },
            ].map((row, index) => (
              <div key={row.label} className="space-y-2">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{row.label}</span>
                  <span className="font-medium text-foreground/80">{row.value}</span>
                </div>
                <div className="relative h-2 rounded-full bg-muted/60 overflow-hidden border border-border/50">
                  <div className={`absolute inset-y-0 left-0 ${row.tone}`} style={{ width: `${Math.min(100, row.value)}%` }} />
                  <div className="absolute inset-y-0 left-0 lane-sweep" style={{ animationDelay: `${index * 0.35}s` }} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Demo 6: Grid Scan */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Cpu className="h-4 w-4 text-primary" />
              Grid Scan
            </CardTitle>
            <span className="text-xs text-muted-foreground">Sector sweep</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-48 rounded-lg border border-border/60 bg-muted/10 overflow-hidden">
              <div className="absolute inset-0 grid grid-cols-8 grid-rows-6 gap-1 p-3">
                {Array.from({ length: 48 }).map((_, i) => (
                  <div
                    key={i}
                    className={`rounded-sm ${i % 7 === 0 ? "bg-emerald-500/30" : "bg-muted/50"} grid-blip`}
                    style={{ animationDelay: `${(i % 12) * 0.2}s` }}
                  />
                ))}
              </div>
              <div className="absolute inset-0 grid-scan-line" />
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Nodes: 48</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                GRID ACTIVE
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 7: Lockstep KPIs */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <HardDrive className="h-4 w-4 text-primary" />
              Lockstep KPIs
            </CardTitle>
            <span className="text-xs text-muted-foreground">Sync ratio</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {[
              { label: "Indexed", value: 92, tone: "bg-emerald-500/70" },
              { label: "Queued", value: 68, tone: "bg-blue-500/70" },
              { label: "Processed", value: 76, tone: "bg-yellow-500/70" },
              { label: "Errored", value: 12, tone: "bg-red-500/60" },
            ].map((row) => (
              <div key={row.label} className="space-y-2">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{row.label}</span>
                  <span className="font-medium text-foreground/80">{row.value}%</span>
                </div>
                <div className="relative h-2 rounded-full bg-muted/60 overflow-hidden border border-border/50">
                  <div className={`absolute inset-y-0 left-0 ${row.tone}`} style={{ width: `${row.value}%` }} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Demo 8: Thermal Columns */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Database className="h-4 w-4 text-primary" />
              Thermal Columns
            </CardTitle>
            <span className="text-xs text-muted-foreground">Heat load</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-48 rounded-lg border border-border/60 bg-muted/10 overflow-hidden p-4 flex items-end gap-2">
              {stackTotals.map((value, index) => {
                const height = Math.max(20, Math.round((value / maxStack) * 120))
                return (
                  <div key={`thermal-${index}`} className="flex-1 flex flex-col items-center gap-2">
                    <div
                      className="w-full rounded-sm bg-gradient-to-t from-emerald-500/10 via-emerald-500/50 to-emerald-500/90"
                      style={{ height }}
                    />
                    <span className="text-[10px] text-muted-foreground">{DAYS[index]}</span>
                  </div>
                )
              })}
              <div className="absolute inset-0 thermal-sweep" />
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Peak load {Math.max(...stackTotals)}</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                THERMAL LOCK
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 9: Anomaly Ledger */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-primary" />
              Anomaly Ledger
            </CardTitle>
            <span className="text-xs text-muted-foreground">Signal drift</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {LEDGER_ITEMS.map((item) => (
              <div key={item.id} className="rounded-lg border border-border/60 bg-muted/20 p-3 flex items-center justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="font-mono text-[10px]">{item.id}</span>
                    <span className={`h-1.5 w-1.5 rounded-full ${item.tone}`} />
                    <span>{item.status}</span>
                  </div>
                  <div className="mt-1 text-sm font-medium">{item.label}</div>
                </div>
                <Sparkline data={item.series} width={120} height={26} color="var(--primary)" />
              </div>
            ))}
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>3 of 12 signals elevated</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-amber-500 animate-pulse" />
                WATCH MODE
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 10: Pulse Tiles */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Crosshair className="h-4 w-4 text-primary" />
              Pulse Tiles
            </CardTitle>
            <span className="text-xs text-muted-foreground">Quick telemetry</span>
          </CardHeader>
          <CardContent className="grid gap-3 sm:grid-cols-2">
            {TILE_STATS.map((tile) => (
              <div key={tile.label} className="rounded-lg border border-border/60 bg-muted/20 p-3 flex items-center gap-3">
                <div className="relative h-12 w-12">
                  <div
                    className="absolute inset-0 rounded-full tile-ring"
                    style={{
                      background: `conic-gradient(from -90deg, ${tile.color} 0 ${tile.pct}%, rgba(0,0,0,0.08) ${tile.pct}% 100%)`,
                    }}
                  />
                  <div className="absolute inset-1 rounded-full bg-background border border-border/60" />
                  <div className="absolute inset-0 flex items-center justify-center text-[10px] font-semibold text-foreground/80">
                    {tile.pct}%
                  </div>
                </div>
                <div>
                  <div className="text-xs text-muted-foreground">{tile.label}</div>
                  <div className="text-lg font-semibold">{tile.value}</div>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Demo 11: Sector Ring */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Target className="h-4 w-4 text-primary" />
              Sector Ring
            </CardTitle>
            <span className="text-xs text-muted-foreground">Zone health</span>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="relative mx-auto h-44 w-44">
              <div className="absolute inset-0 rounded-full ring-sweep" />
              <div className="absolute inset-6 rounded-full bg-background border border-border/60" />
              <div className="absolute inset-0 flex items-center justify-center text-center">
                <div>
                  <div className="text-2xl font-semibold">86%</div>
                  <div className="text-[10px] text-muted-foreground">Zone Integrity</div>
                </div>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
              {ZONES.map((zone) => (
                <div key={zone.label} className="flex items-center gap-2 rounded-md border border-border/60 bg-muted/20 px-2 py-1">
                  <span className={`h-2 w-2 rounded-full ${zone.tone}`} />
                  <span className="flex-1">{zone.label}</span>
                  <span className="font-medium text-foreground/80">{zone.value}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Demo 12: Trace Columns */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Cpu className="h-4 w-4 text-primary" />
              Trace Columns
            </CardTitle>
            <span className="text-xs text-muted-foreground">Footprint mix</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {DAYS.map((day, index) => {
              const total = stackTotals[index]
              const height = Math.max(28, Math.round((total / maxStack) * 80))
              return (
                <div key={`trace-${day}`} className="flex items-center gap-3">
                  <span className="w-8 text-[10px] text-muted-foreground">{day}</span>
                  <div className="flex-1 rounded-full border border-border/60 bg-muted/30 overflow-hidden h-2.5">
                    <div className="h-full bg-gradient-to-r from-blue-500/70 via-emerald-500/70 to-amber-500/70" style={{ width: `${Math.min(100, total)}%` }} />
                  </div>
                  <span className="text-xs text-muted-foreground w-10 text-right">{height}</span>
                </div>
              )
            })}
            <div className="pt-2 text-xs text-muted-foreground">Gradient represents mix of assets per day.</div>
          </CardContent>
        </Card>

        {/* Demo 13: Signal Capsules */}
        <Card className="relative overflow-hidden md:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Signal className="h-4 w-4 text-primary" />
              Signal Capsules
            </CardTitle>
            <span className="text-xs text-muted-foreground">Live metrics</span>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 md:grid-cols-3">
              {METRICS.map((metric) => (
                <div key={metric.label} className="rounded-lg border border-border/60 bg-muted/20 p-3 space-y-2">
                  <div className="text-xs text-muted-foreground">{metric.label}</div>
                  <div className="flex items-end justify-between">
                    <div className="text-2xl font-semibold">{metric.value}</div>
                    <span className={`text-xs font-medium ${metric.tone}`}>{metric.delta}</span>
                  </div>
                  <div className="relative h-1.5 rounded-full bg-muted/60 overflow-hidden">
                    <div className="absolute inset-y-0 left-0 capsule-sweep" />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Demo 14: Pulse Timeline */}
        <Card className="relative overflow-hidden md:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconActivity className="h-4 w-4 text-primary" />
              Pulse Timeline
            </CardTitle>
            <span className="text-xs text-muted-foreground">Daily checkpoints</span>
          </CardHeader>
          <CardContent>
            <div className="relative rounded-lg border border-border/60 bg-muted/10 px-4 py-6">
              <div className="absolute left-4 right-4 top-1/2 h-px -translate-y-1/2 bg-border/70" />
              <div className="relative flex items-center justify-between">
                {TIMELINE_POINTS.map((value, index) => (
                  <div key={`tl-${index}`} className="flex flex-col items-center gap-2">
                    <span
                      className="timeline-dot"
                      style={{ animationDelay: `${index * 0.2}s` }}
                    />
                    <span className="text-[10px] text-muted-foreground">{value}%</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Pulse variance 6.4%</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                STABLE
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 15: Node Constellation */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Target className="h-4 w-4 text-primary" />
              Node Constellation
            </CardTitle>
            <span className="text-xs text-muted-foreground">Link mesh</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-52 rounded-lg border border-border/60 bg-muted/10 overflow-hidden">
              <div className="absolute inset-0 constellation-grid" />
              <div className="absolute inset-0 constellation-beam" />
              {CONSTELLATION.map((node, index) => (
                <span
                  key={`node-${index}`}
                  className={`absolute rounded-full bg-primary/70 ${node.size} node-blip`}
                  style={{ left: node.x, top: node.y, animationDelay: `${index * 0.35}s` }}
                />
              ))}
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Active nodes: 6</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-primary animate-pulse" />
                LINKED
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 16: Signal Bars */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Signal className="h-4 w-4 text-primary" />
              Signal Bars
            </CardTitle>
            <span className="text-xs text-muted-foreground">Amplitude</span>
          </CardHeader>
          <CardContent>
            <div className="flex items-end justify-between gap-2 h-40 rounded-lg border border-border/60 bg-muted/10 px-3 py-4">
              {SERIES.total.map((value, index) => {
                const height = Math.max(22, Math.round((value / SERIES.total.at(-1)!) * 120))
                return (
                  <div key={`bar-${index}`} className="flex flex-col items-center gap-2">
                    <span
                      className="signal-bar"
                      style={{ height, animationDelay: `${index * 0.15}s` }}
                    />
                    <span className="text-[10px] text-muted-foreground">{DAYS[index]}</span>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>

        {/* Demo 17: Scan Ribbon */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconScan className="h-4 w-4 text-primary" />
              Scan Ribbon
            </CardTitle>
            <span className="text-xs text-muted-foreground">Flow channel</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-20 rounded-lg border border-border/60 bg-muted/10 overflow-hidden">
              <div className="absolute inset-0 ribbon-sweep" />
              <div className="absolute inset-0 flex items-center justify-between px-4 text-xs text-muted-foreground">
                <span>Ingress</span>
                <span>Core</span>
                <span>Egress</span>
              </div>
            </div>
            <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground">
              <span>Ribbon load 71%</span>
              <span className="flex items-center gap-2">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                ROUTING
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Demo 18: Risk Histogram */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-primary" />
              Risk Histogram
            </CardTitle>
            <span className="text-xs text-muted-foreground">Severity mix</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {RISK_LEVELS.map((level) => (
              <div key={level.label} className="space-y-1">
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{level.label}</span>
                  <span className="font-medium text-foreground/80">{level.value}</span>
                </div>
                <div className="h-2 rounded-full bg-muted/60 overflow-hidden border border-border/50">
                  <div className={`h-full bg-gradient-to-r ${level.tone}`} style={{ width: `${level.value}%` }} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Demo 19: Latency Ladder */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Cpu className="h-4 w-4 text-primary" />
              Latency Ladder
            </CardTitle>
            <span className="text-xs text-muted-foreground">Stage hops</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {[
              { label: "Ingress", value: "12ms" },
              { label: "Parse", value: "18ms" },
              { label: "Resolve", value: "24ms" },
              { label: "Enrich", value: "31ms" },
            ].map((row, index) => (
              <div key={row.label} className="flex items-center gap-3">
                <span className="w-16 text-xs text-muted-foreground">{row.label}</span>
                <div className="flex-1 h-2 rounded-full bg-muted/60 border border-border/50 relative overflow-hidden">
                  <div className="absolute inset-y-0 left-0 ladder-bar" style={{ width: `${(index + 1) * 20}%` }} />
                </div>
                <span className="text-xs text-muted-foreground w-12 text-right">{row.value}</span>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Demo 20: Sweep Field */}
        <Card className="relative overflow-hidden md:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconRadar className="h-4 w-4 text-primary" />
              Sweep Field
            </CardTitle>
            <span className="text-xs text-muted-foreground">Wide scan</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-40 rounded-lg border border-border/60 bg-muted/10 overflow-hidden sweep-grid">
              <div className="absolute inset-0 sweep-line" />
              <div className="absolute inset-0 flex items-center justify-center text-xs text-muted-foreground">
                Field coverage 92%
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Demo 21: Mini Radar */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconRadar className="h-4 w-4 text-primary" />
              Mini Radar
            </CardTitle>
            <span className="text-xs text-muted-foreground">Compact sweep</span>
          </CardHeader>
          <CardContent>
            <div className="relative mx-auto h-36 w-36 rounded-full border border-border/60 bg-muted/10 overflow-hidden">
              <div className="absolute inset-3 rounded-full border border-border/60" />
              <div className="absolute inset-8 rounded-full border border-border/60" />
              <div className="absolute inset-0 mini-radar-sweep" />
              <span className="absolute left-[30%] top-[32%] h-2 w-2 rounded-full bg-primary/70 animate-ping" />
              <span className="absolute left-[62%] top-[52%] h-1.5 w-1.5 rounded-full bg-primary/70 animate-ping [animation-delay:1s]" />
            </div>
          </CardContent>
        </Card>

        {/* Demo 22: Shard Rows */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Database className="h-4 w-4 text-primary" />
              Shard Rows
            </CardTitle>
            <span className="text-xs text-muted-foreground">Micro pulses</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {SHARD_ROWS.map((row) => (
              <div key={row.id} className="flex items-center justify-between rounded-lg border border-border/60 bg-muted/20 px-3 py-2">
                <div>
                  <div className="text-xs text-muted-foreground">{row.id}</div>
                  <div className="text-sm font-medium">{row.label}</div>
                </div>
                <Sparkline data={row.series} width={110} height={26} color="var(--primary)" />
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Demo 23: Wave Stack */}
        <Card className="relative overflow-hidden md:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Waves className="h-4 w-4 text-primary" />
              Wave Stack
            </CardTitle>
            <span className="text-xs text-muted-foreground">Multi-channel</span>
          </CardHeader>
          <CardContent>
            <div className="rounded-lg border border-border/60 bg-muted/10 p-3">
              <svg viewBox="0 0 520 120" className="w-full h-[120px]" aria-hidden>
                <path d={sparklinePath(SERIES.subdomains, 520, 120, 6)} stroke="rgba(59,130,246,0.8)" strokeWidth="2" fill="none" />
                <path d={sparklinePath(SERIES.websites, 520, 120, 6)} stroke="rgba(16,185,129,0.8)" strokeWidth="2" fill="none" />
                <path d={sparklinePath(SERIES.ips, 520, 120, 6)} stroke="rgba(249,115,22,0.8)" strokeWidth="2" fill="none" />
                <path d={sparklinePath(SERIES.endpoints, 520, 120, 6)} stroke="rgba(234,179,8,0.8)" strokeWidth="2" fill="none" />
              </svg>
            </div>
          </CardContent>
        </Card>

        {/* Demo 24: Pulse Tags */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Signal className="h-4 w-4 text-primary" />
              Pulse Tags
            </CardTitle>
            <span className="text-xs text-muted-foreground">Counters</span>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {TAG_ITEMS.map((tag, index) => (
                <span key={tag.label} className="pulse-tag">
                  <span className="text-xs">{tag.label}</span>
                  <span className="text-xs font-semibold">{tag.value}</span>
                  <span className="tag-sweep" style={{ animationDelay: `${index * 0.3}s` }} />
                </span>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Demo 25: Pulse Cluster */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Target className="h-4 w-4 text-primary" />
              Pulse Cluster
            </CardTitle>
            <span className="text-xs text-muted-foreground">Node grouping</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-40 rounded-lg border border-border/60 bg-muted/10 overflow-hidden">
              <div className="absolute inset-6 rounded-full cluster-ring" />
              <span className="absolute left-[35%] top-[38%] h-2 w-2 rounded-full bg-primary/70 animate-ping" />
              <span className="absolute left-[52%] top-[46%] h-1.5 w-1.5 rounded-full bg-primary/70 animate-ping [animation-delay:0.6s]" />
              <span className="absolute left-[45%] top-[62%] h-2 w-2 rounded-full bg-primary/70 animate-ping [animation-delay:1.2s]" />
            </div>
          </CardContent>
        </Card>

        {/* Demo 26: Trace Gauge */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Crosshair className="h-4 w-4 text-primary" />
              Trace Gauge
            </CardTitle>
            <span className="text-xs text-muted-foreground">Integrity</span>
          </CardHeader>
          <CardContent className="flex items-center justify-center">
            <svg viewBox="0 0 140 80" className="h-28 w-48" aria-hidden>
              <path d="M10 70 A60 60 0 0 1 130 70" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="10" />
              <path d="M10 70 A60 60 0 0 1 110 70" fill="none" stroke="rgba(34,197,94,0.8)" strokeWidth="10" strokeLinecap="round" />
              <circle cx="70" cy="70" r="6" fill="rgba(34,197,94,0.9)" />
              <text x="70" y="52" textAnchor="middle" fontSize="14" fill="currentColor">78%</text>
            </svg>
          </CardContent>
        </Card>

        {/* Demo 27: Timeline Matrix */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Database className="h-4 w-4 text-primary" />
              Timeline Matrix
            </CardTitle>
            <span className="text-xs text-muted-foreground">Grid drift</span>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-8 gap-1 rounded-lg border border-border/60 bg-muted/10 p-3">
              {Array.from({ length: 64 }).map((_, index) => {
                const active = index % 9 === 0
                return (
                  <span
                    key={`mx-${index}`}
                    className={`matrix-cell ${active ? "matrix-active" : ""}`}
                    style={{ animationDelay: `${(index % 12) * 0.2}s` }}
                  />
                )
              })}
            </div>
          </CardContent>
        </Card>

        {/* Demo 28: Echo Rings */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <IconRadar className="h-4 w-4 text-primary" />
              Echo Rings
            </CardTitle>
            <span className="text-xs text-muted-foreground">Pulse echo</span>
          </CardHeader>
          <CardContent>
            <div className="relative h-40 rounded-lg border border-border/60 bg-muted/10 flex items-center justify-center">
              <span className="echo-ring echo-1" />
              <span className="echo-ring echo-2" />
              <span className="echo-ring echo-3" />
              <span className="h-3 w-3 rounded-full bg-primary/80" />
            </div>
          </CardContent>
        </Card>

        {/* Demo 29: Signal Dispatch */}
        <Card className="relative overflow-hidden md:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <HardDrive className="h-4 w-4 text-primary" />
              Signal Dispatch
            </CardTitle>
            <span className="text-xs text-muted-foreground">Pipeline</span>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              {DISPATCH_STEPS.map((step, index) => (
                <div key={step.label} className="flex flex-col items-center gap-2">
                  <span className="h-9 w-9 rounded-full border border-border/60 bg-muted/20 flex items-center justify-center text-xs">
                    {index + 1}
                  </span>
                  <span className="text-[10px] text-muted-foreground">{step.label}</span>
                </div>
              ))}
            </div>
            <div className="h-2 rounded-full bg-muted/60 overflow-hidden border border-border/60">
              <div className="dispatch-scan" />
            </div>
          </CardContent>
        </Card>

        {/* Demo 30: Pulse Ladder */}
        <Card className="relative overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Cpu className="h-4 w-4 text-primary" />
              Pulse Ladder
            </CardTitle>
            <span className="text-xs text-muted-foreground">Ranked</span>
          </CardHeader>
          <CardContent className="space-y-3">
            {[
              { label: "Tier A", value: 86 },
              { label: "Tier B", value: 64 },
              { label: "Tier C", value: 42 },
              { label: "Tier D", value: 23 },
            ].map((tier) => (
              <div key={tier.label} className="flex items-center gap-3">
                <span className="w-16 text-xs text-muted-foreground">{tier.label}</span>
                <div className="flex-1 h-2 rounded-full bg-muted/60 border border-border/50 overflow-hidden">
                  <div className="h-full bg-primary/70" style={{ width: `${tier.value}%` }} />
                </div>
                <span className="text-xs text-muted-foreground w-10 text-right">{tier.value}%</span>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
      <style jsx>{`
        .radar-sweep {
          background: conic-gradient(from 180deg, rgba(16, 185, 129, 0), rgba(16, 185, 129, 0.35), rgba(16, 185, 129, 0));
          animation: radar 8s linear infinite;
        }
        @keyframes radar {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        .lane-sweep {
          width: 40%;
          background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.5), transparent);
          animation: lane 2.6s linear infinite;
          opacity: 0.6;
        }
        @keyframes lane {
          from { transform: translateX(-60%); }
          to { transform: translateX(220%); }
        }
        .grid-scan-line {
          background: linear-gradient(90deg, transparent, rgba(16, 185, 129, 0.5), transparent);
          height: 100%;
          width: 30%;
          animation: gridline 3.2s linear infinite;
          opacity: 0.5;
        }
        @keyframes gridline {
          from { transform: translateX(-80%); }
          to { transform: translateX(240%); }
        }
        .grid-blip {
          animation: blip 2.8s ease-in-out infinite;
        }
        @keyframes blip {
          0%, 100% { opacity: 0.4; }
          50% { opacity: 1; }
        }
        .thermal-sweep {
          background: linear-gradient(90deg, transparent, rgba(16, 185, 129, 0.35), transparent);
          height: 100%;
          width: 25%;
          animation: thermal 4s linear infinite;
          opacity: 0.45;
        }
        @keyframes thermal {
          from { transform: translateX(-80%); }
          to { transform: translateX(240%); }
        }
        .tile-ring {
          box-shadow: 0 0 18px rgba(34, 197, 94, 0.15);
        }
        .ring-sweep {
          background: conic-gradient(from -90deg, rgba(59, 130, 246, 0.7), rgba(16, 185, 129, 0.7), rgba(234, 179, 8, 0.7), rgba(239, 68, 68, 0.7));
          animation: ring 8s linear infinite;
          border-radius: 999px;
          opacity: 0.8;
        }
        @keyframes ring {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
        .capsule-sweep {
          width: 45%;
          background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.7), transparent);
          animation: capsule 2.8s linear infinite;
          opacity: 0.5;
        }
        @keyframes capsule {
          from { transform: translateX(-60%); }
          to { transform: translateX(220%); }
        }
        .timeline-dot {
          height: 10px;
          width: 10px;
          border-radius: 999px;
          background: rgba(34, 197, 94, 0.6);
          box-shadow: 0 0 0 6px rgba(34, 197, 94, 0.08);
          animation: timeline 2.4s ease-in-out infinite;
        }
        @keyframes timeline {
          0%, 100% { transform: scale(1); opacity: 0.7; }
          50% { transform: scale(1.4); opacity: 1; }
        }
        .constellation-grid {
          background-image:
            linear-gradient(90deg, rgba(148, 163, 184, 0.18) 1px, transparent 1px),
            linear-gradient(180deg, rgba(148, 163, 184, 0.18) 1px, transparent 1px);
          background-size: 24px 24px;
          opacity: 0.35;
        }
        .constellation-beam {
          background: linear-gradient(120deg, transparent, rgba(59, 130, 246, 0.35), transparent);
          animation: beam 6s linear infinite;
          opacity: 0.4;
        }
        @keyframes beam {
          from { transform: translateX(-60%); }
          to { transform: translateX(220%); }
        }
        .node-blip {
          animation: node 3s ease-in-out infinite;
        }
        @keyframes node {
          0%, 100% { opacity: 0.5; }
          50% { opacity: 1; }
        }
        .signal-bar {
          width: 10px;
          border-radius: 6px;
          background: linear-gradient(180deg, rgba(59, 130, 246, 0.85), rgba(59, 130, 246, 0.25));
          animation: bar 2.4s ease-in-out infinite;
        }
        @keyframes bar {
          0%, 100% { transform: scaleY(0.9); }
          50% { transform: scaleY(1.05); }
        }
        .ribbon-sweep {
          background: linear-gradient(90deg, transparent, rgba(59, 130, 246, 0.45), transparent);
          animation: ribbon 3.4s linear infinite;
          opacity: 0.6;
        }
        @keyframes ribbon {
          from { transform: translateX(-80%); }
          to { transform: translateX(240%); }
        }
        .ladder-bar {
          background: linear-gradient(90deg, rgba(59, 130, 246, 0.7), rgba(59, 130, 246, 0.15));
        }
        .sweep-grid {
          background-image:
            linear-gradient(90deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px),
            linear-gradient(180deg, rgba(148, 163, 184, 0.12) 1px, transparent 1px);
          background-size: 18px 18px;
        }
        .sweep-line {
          width: 22%;
          background: linear-gradient(90deg, transparent, rgba(59, 130, 246, 0.4), transparent);
          animation: sweep 3.2s linear infinite;
          opacity: 0.5;
        }
        @keyframes sweep {
          from { transform: translateX(-70%); }
          to { transform: translateX(240%); }
        }
        .mini-radar-sweep {
          background: conic-gradient(from 180deg, rgba(59, 130, 246, 0), rgba(59, 130, 246, 0.35), rgba(59, 130, 246, 0));
          animation: radar 6s linear infinite;
        }
        .pulse-tag {
          position: relative;
          display: inline-flex;
          align-items: center;
          gap: 8px;
          padding: 6px 10px;
          border-radius: 999px;
          border: 1px solid rgba(148, 163, 184, 0.4);
          background: rgba(148, 163, 184, 0.12);
          overflow: hidden;
        }
        .tag-sweep {
          position: absolute;
          inset: 0;
          background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.6), transparent);
          animation: tag 2.8s linear infinite;
          opacity: 0.3;
        }
        @keyframes tag {
          from { transform: translateX(-80%); }
          to { transform: translateX(240%); }
        }
        .cluster-ring {
          border: 1px dashed rgba(59, 130, 246, 0.5);
          box-shadow: 0 0 0 6px rgba(59, 130, 246, 0.08);
        }
        .matrix-cell {
          height: 12px;
          border-radius: 3px;
          background: rgba(148, 163, 184, 0.2);
        }
        .matrix-active {
          background: rgba(59, 130, 246, 0.55);
          animation: matrix 2.6s ease-in-out infinite;
        }
        @keyframes matrix {
          0%, 100% { opacity: 0.4; }
          50% { opacity: 1; }
        }
        .echo-ring {
          position: absolute;
          border-radius: 999px;
          border: 1px solid rgba(59, 130, 246, 0.4);
          animation: echo 2.8s ease-in-out infinite;
          opacity: 0.6;
        }
        .echo-1 { width: 60px; height: 60px; }
        .echo-2 { width: 90px; height: 90px; animation-delay: 0.6s; }
        .echo-3 { width: 120px; height: 120px; animation-delay: 1.2s; }
        @keyframes echo {
          0%, 100% { transform: scale(0.9); opacity: 0.35; }
          50% { transform: scale(1.05); opacity: 0.8; }
        }
        .dispatch-scan {
          height: 100%;
          width: 35%;
          background: linear-gradient(90deg, transparent, rgba(59, 130, 246, 0.5), transparent);
          animation: dispatch 3.6s linear infinite;
          opacity: 0.5;
        }
        @keyframes dispatch {
          from { transform: translateX(-80%); }
          to { transform: translateX(240%); }
        }
      `}</style>
    </div>
  )
}
