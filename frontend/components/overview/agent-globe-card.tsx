"use client"

import { useEffect, useMemo, useRef } from "react"
import { useTranslations } from "next-intl"
import createGlobe from "cobe"
import type { COBEOptions } from "cobe"
import { Skeleton } from "@/components/ui/skeleton"
import { AlertTriangle, CheckCircle2 } from "@/components/icons"
import { OverviewSectionPanel } from "@/components/overview/overview-section-layouts"
import { useAgents } from "@/hooks/use-agents"
import { useColorTheme } from "@/hooks/use-color-theme"
import type { ColorTheme } from "@/lib/color-themes"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { buildAgentSummary } from "./agent-globe-card-state"

const PREVIEW_MARKERS = [
  { location: [39.9042, 116.4074] as [number, number], size: 0.04, id: "pek" },
  { location: [35.6762, 139.6503] as [number, number], size: 0.05, id: "nrt" },
  { location: [37.7595, -122.4367] as [number, number], size: 0.05, id: "sfo" },
  { location: [40.7128, -74.006] as [number, number], size: 0.05, id: "iad" },
  { location: [51.5072, -0.1276] as [number, number], size: 0.04, id: "lon" },
  { location: [-23.5505, -46.6333] as [number, number], size: 0.05, id: "gru" },
]

const PREVIEW_ARCS = [
  { from: [37.7595, -122.4367] as [number, number], to: [40.7128, -74.006] as [number, number] },
  { from: [40.7128, -74.006] as [number, number], to: [51.5072, -0.1276] as [number, number] },
  { from: [51.5072, -0.1276] as [number, number], to: [35.6762, 139.6503] as [number, number] },
  { from: [-23.5505, -46.6333] as [number, number], to: [37.7595, -122.4367] as [number, number] },
]

const PREVIEW_LABELS = [
  { id: "pek", label: "dxb1" },
  { id: "nrt", label: "bom1" },
  { id: "sfo", label: "iad1" },
  { id: "iad", label: "cdg1" },
  { id: "lon", label: "arn1" },
  { id: "gru", label: "gru1" },
]

type CobeColor = [number, number, number]

type CobeGlobeTheme = {
  theta: number
  dark: 0 | 1
  diffuse: number
  mapBrightness: number
  baseColor: CobeColor
  markerColor: CobeColor
  glowColor: CobeColor
  arcColor: CobeColor
  markerElevation: number
}

function hexToCobeColor(hexColor: string | undefined, fallback: CobeColor): CobeColor {
  const match = hexColor?.match(/^#?([\da-f]{2})([\da-f]{2})([\da-f]{2})$/i)

  if (!match) {
    return fallback
  }

  return [
    Number.parseInt(match[1], 16) / 255,
    Number.parseInt(match[2], 16) / 255,
    Number.parseInt(match[3], 16) / 255,
  ]
}

function rgbStringToCobeColor(color: string | undefined, fallback: CobeColor): CobeColor {
  const match = color?.match(/^rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)/i)

  if (!match) {
    return fallback
  }

  return [
    Number.parseFloat(match[1]) / 255,
    Number.parseFloat(match[2]) / 255,
    Number.parseFloat(match[3]) / 255,
  ]
}

function oklchStringToCobeColor(color: string | undefined, fallback: CobeColor): CobeColor {
  const match = color?.match(/^oklch\(\s*([\d.]+%?)\s+([\d.]+)\s+([-\d.]+)/i)

  if (!match) {
    return fallback
  }

  const lightness = match[1].endsWith("%")
    ? Number.parseFloat(match[1]) / 100
    : Number.parseFloat(match[1])
  const chroma = Number.parseFloat(match[2])
  const hue = Number.parseFloat(match[3]) * (Math.PI / 180)

  if (!Number.isFinite(lightness) || !Number.isFinite(chroma) || !Number.isFinite(hue)) {
    return fallback
  }

  const a = chroma * Math.cos(hue)
  const b = chroma * Math.sin(hue)
  const l = lightness + 0.3963377774 * a + 0.2158037573 * b
  const m = lightness - 0.1055613458 * a - 0.0638541728 * b
  const s = lightness - 0.0894841775 * a - 1.291485548 * b

  const l3 = l * l * l
  const m3 = m * m * m
  const s3 = s * s * s

  const linearRgb: CobeColor = [
    4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3,
    -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3,
    -0.0041960863 * l3 - 0.7034186147 * m3 + 1.707614701 * s3,
  ]

  return linearRgb.map((channel) => {
    const clamped = Math.min(1, Math.max(0, channel))

    return clamped >= 0.0031308
      ? 1.055 * Math.pow(clamped, 1 / 2.4) - 0.055
      : 12.92 * clamped
  }) as CobeColor
}

function cssLabChannelToNumber(value: string, scalePercentTo: number): number {
  return value.endsWith("%")
    ? (Number.parseFloat(value) / 100) * scalePercentTo
    : Number.parseFloat(value)
}

function labStringToCobeColor(color: string | undefined, fallback: CobeColor): CobeColor {
  const match = color?.match(/^lab\(\s*([-\d.]+%?)\s+([-\d.]+%?)\s+([-\d.]+%?)/i)

  if (!match) {
    return fallback
  }

  const lightness = cssLabChannelToNumber(match[1], 100)
  const a = cssLabChannelToNumber(match[2], 125)
  const b = cssLabChannelToNumber(match[3], 125)

  if (!Number.isFinite(lightness) || !Number.isFinite(a) || !Number.isFinite(b)) {
    return fallback
  }

  const fy = (lightness + 16) / 116
  const fx = fy + a / 500
  const fz = fy - b / 200
  const delta = 6 / 29
  const labInverse = (value: number) => (
    value > delta
      ? value ** 3
      : 3 * delta ** 2 * (value - 4 / 29)
  )

  const xD50 = 0.96422 * labInverse(fx)
  const yD50 = labInverse(fy)
  const zD50 = 0.82521 * labInverse(fz)

  const xD65 = 0.9554734 * xD50 - 0.0230985 * yD50 + 0.0632593 * zD50
  const yD65 = -0.0283697 * xD50 + 1.0099956 * yD50 + 0.0210414 * zD50
  const zD65 = 0.012314 * xD50 - 0.0205077 * yD50 + 1.3303659 * zD50

  const linearRgb: CobeColor = [
    3.2404542 * xD65 - 1.5371385 * yD65 - 0.4985314 * zD65,
    -0.969266 * xD65 + 1.8760108 * yD65 + 0.041556 * zD65,
    0.0556434 * xD65 - 0.2040259 * yD65 + 1.0572252 * zD65,
  ]

  return linearRgb.map((channel) => {
    const clamped = Math.min(1, Math.max(0, channel))

    return clamped >= 0.0031308
      ? 1.055 * Math.pow(clamped, 1 / 2.4) - 0.055
      : 12.92 * clamped
  }) as CobeColor
}

function resolveThemeColorToCobeColor(color: string | undefined, fallback: CobeColor): CobeColor {
  const directHexColor = hexToCobeColor(color, fallback)
  if (directHexColor !== fallback) {
    return directHexColor
  }

  if (typeof document === "undefined" || !color) {
    return fallback
  }

  const probe = document.createElement("span")
  probe.style.color = color
  probe.style.position = "absolute"
  probe.style.opacity = "0"
  probe.style.pointerEvents = "none"
  document.body.appendChild(probe)

  const resolved = getComputedStyle(probe).color || color
  probe.remove()

  const resolvedRgbColor = rgbStringToCobeColor(resolved, fallback)
  if (resolvedRgbColor !== fallback) {
    return resolvedRgbColor
  }

  const resolvedLabColor = labStringToCobeColor(resolved, fallback)
  if (resolvedLabColor !== fallback) {
    return resolvedLabColor
  }

  return oklchStringToCobeColor(resolved, fallback)
}

function mixCobeColor(from: CobeColor, to: CobeColor, amount: number): CobeColor {
  return [
    from[0] + (to[0] - from[0]) * amount,
    from[1] + (to[1] - from[1]) * amount,
    from[2] + (to[2] - from[2]) * amount,
  ]
}

export function buildThemeSyncedCobeGlobeTheme(theme: ColorTheme): CobeGlobeTheme {
  const accentColor = resolveThemeColorToCobeColor(theme.color, theme.isDark ? [0.92, 0.92, 0.92] : [0.1, 0.1, 0.1])
  const backgroundColor = resolveThemeColorToCobeColor(theme.colors[0], theme.isDark ? [0.145, 0.145, 0.145] : [1, 1, 1])
  const neutralBase: CobeColor = theme.isDark ? [0.5, 0.5, 0.5] : [1, 1, 1]
  const baseMixAmount = theme.isDark ? 0.22 : 0.08
  const glowMixAmount = theme.isDark ? 0.44 : 0.3

  return {
    theta: 0.2,
    dark: theme.isDark ? 1 : 0,
    diffuse: 1.5,
    mapBrightness: theme.isDark ? 10 : 11,
    baseColor: mixCobeColor(neutralBase, accentColor, baseMixAmount),
    markerColor: accentColor,
    glowColor: mixCobeColor(backgroundColor, accentColor, glowMixAmount),
    arcColor: accentColor,
    markerElevation: theme.isDark ? 0 : 0.01,
  }
}


export function AgentGlobeCard() {
  const t = useTranslations("overview.agentGlobe")
  const { data, isLoading } = useAgents({ pageSize: 100, orderBy: "createdAt desc" })
  const { currentTheme } = useColorTheme()
  const nodes = useMemo(() => data?.results ?? [], [data?.results])
  const summary = useMemo(() => buildAgentSummary(nodes), [nodes])
  const totalNodes = summary.total
  const abnormalNodes = Math.max(totalNodes - summary.alive, 0)
  const metricItems = [
    {
      label: t("metrics.onlineNodes.label"),
      value: summary.alive,
      helper: totalNodes > 0 ? t("metrics.onlineNodes.helper") : t("metrics.onlineNodes.waitingHeartbeat"),
      icon: CheckCircle2,
    },
    {
      label: t("metrics.abnormalNodes.label"),
      value: abnormalNodes,
      helper: totalNodes > 0 ? t("metrics.abnormalNodes.helper") : t("metrics.abnormalNodes.waitingHeartbeat"),
      icon: AlertTriangle,
    },
  ] as const

  return (
    <OverviewSectionPanel
      title={t("title")}
      className="overview-agent-globe-shell overview-reveal relative h-full overflow-hidden bg-card text-card-foreground"
      contentClassName="relative z-10 grid flex-1 grid-rows-[auto_1fr] gap-3 pt-1"
    >
        <div className="grid grid-cols-2 border-b pb-4">
          {metricItems.map((item) => {
            const MetricIcon = item.icon

            return (
              <div key={item.label} className="border-r pr-4 last:border-r-0 last:pl-4 last:pr-0">
                <div className="flex items-center gap-3">
                  <span className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <MetricIcon className="size-5" aria-hidden="true" />
                  </span>
                  <span className={textRole.bodyStrong}>{item.label}</span>
                </div>
                {isLoading ? (
                  <Skeleton className="mt-4 h-5 w-14" />
                ) : (
                  <div key={item.value} className={cn(textRole.metricValueDisplay, "overview-number-swap mt-4 text-card-foreground")}>{item.value}</div>
                )}
                <div className={cn("mt-3", textRole.bodySubtle)}>{item.helper}</div>
              </div>
            )
          })}
        </div>

        <div className="flex min-w-0 items-center justify-center">
          <div className="overview-agent-globe-shell relative flex aspect-square w-full max-w-[240px] items-center justify-center justify-self-center overflow-visible rounded-full">
            <CobePreviewGlobe ariaLabel={t("aria.globePreview")} theme={currentTheme} />
            {PREVIEW_LABELS.map((item) => (
              <span
                key={item.id}
                className="pointer-events-none absolute mb-2 -translate-x-1/2 whitespace-nowrap rounded border border-border bg-popover px-1.5 py-0.5 font-mono text-[10px] leading-none text-popover-foreground opacity-[var(--cobe-visible,0)] shadow-sm transition-opacity"
                style={{
                  positionAnchor: `--cobe-${item.id}`,
                  bottom: "anchor(top)",
                  left: "anchor(center)",
                  opacity: `var(--cobe-visible-${item.id}, 0)`,
                } as React.CSSProperties}
              >
                {item.label}
              </span>
            ))}
          </div>
        </div>
    </OverviewSectionPanel>
  )
}

function CobePreviewGlobe({
  ariaLabel,
  theme,
}: {
  ariaLabel: string
  theme: ColorTheme
}) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || typeof ResizeObserver === "undefined") {
      return undefined
    }

    let width = 0
    let phi = 0
    let animationFrame = 0
    let globe: ReturnType<typeof createGlobe> | null = null
    const cobeTheme = buildThemeSyncedCobeGlobeTheme(theme)
    const buildOptions = (nextWidth: number): COBEOptions => ({
      devicePixelRatio: 2,
      width: nextWidth * 2,
      height: nextWidth * 2,
      phi: 0,
      theta: cobeTheme.theta,
      dark: cobeTheme.dark,
      diffuse: cobeTheme.diffuse,
      scale: 1,
      mapSamples: 16000,
      mapBrightness: cobeTheme.mapBrightness,
      baseColor: cobeTheme.baseColor,
      markerColor: cobeTheme.markerColor,
      glowColor: cobeTheme.glowColor,
      offset: [0, 0],
      markers: PREVIEW_MARKERS,
      arcs: PREVIEW_ARCS,
      arcColor: cobeTheme.arcColor,
      arcWidth: 0.5,
      arcHeight: 0.25,
      markerElevation: cobeTheme.markerElevation,
      opacity: 0.7,
    })

    const startGlobe = () => {
      if (globe || width <= 0) {
        return
      }

      globe = createGlobe(canvas, buildOptions(width))
      animate()
    }

    function animate() {
      if (!globe) {
        return
      }

      phi += 0.005
      globe.update({
        phi,
        width: width * 2,
        height: width * 2,
      })
      animationFrame = requestAnimationFrame(animate)
    }

    const resizeObserver = new ResizeObserver((entries) => {
      const entry = entries[0]
      const nextWidth = Math.round(entry.contentRect.width)
      if (nextWidth <= 0) {
        return
      }

      width = nextWidth
      startGlobe()
    })

    resizeObserver.observe(canvas)

    return () => {
      cancelAnimationFrame(animationFrame)
      resizeObserver.disconnect()
      globe?.destroy()
      globe = null
    }
  }, [theme])

  return (
    <canvas
      ref={canvasRef}
      data-cobe-theme={theme.id}
      className="lunafox-cobe-canvas aspect-square w-full max-w-[240px] opacity-95"
      aria-label={ariaLabel}
    />
  )
}
