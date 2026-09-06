"use client"

import * as React from "react"
import { cn } from "@/lib/utils"

type LoginAsciiBackgroundProps = React.HTMLAttributes<HTMLDivElement>

const FRAME_INTERVAL_MS = 1000 / 24
const ANIMATION_TIME_SCALE = 0.6
const PARTICLE_ALPHA = 0.32
const PARTICLE_SIZE_RATIO = 0.48

type PixelPalette = {
  particle: string
}

type PixelCell = {
  fineTurbulence: number
  nx: number
  ny: number
  quietCenter: number
  seed: number
  shimmer: number
  threshold: number
  turbulence: number
  visible: boolean
  x: number
  y: number
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function fract(value: number) {
  return value - Math.floor(value)
}

function hash2(x: number, y: number) {
  let hash = Math.imul(Math.floor(x), 0x9e3779b1) ^ Math.imul(Math.floor(y), 0x85ebca6b)
  hash ^= hash >>> 16
  hash = Math.imul(hash, 0x7feb352d)
  hash ^= hash >>> 15
  hash = Math.imul(hash, 0x846ca68b)
  hash ^= hash >>> 16
  return (hash >>> 0) / 0x100000000
}

function lerp(from: number, to: number, amount: number) {
  return from + (to - from) * amount
}

function smoothstep(value: number) {
  return value * value * (3 - 2 * value)
}

function valueNoise(x: number, y: number) {
  const ix = Math.floor(x)
  const iy = Math.floor(y)
  const fx = smoothstep(x - ix)
  const fy = smoothstep(y - iy)
  const top = lerp(hash2(ix, iy), hash2(ix + 1, iy), fx)
  const bottom = lerp(hash2(ix, iy + 1), hash2(ix + 1, iy + 1), fx)
  return lerp(top, bottom, fy)
}

function fbmNoise(x: number, y: number) {
  let amplitude = 0.5
  let frequency = 1
  let total = 0

  for (let octave = 0; octave < 4; octave += 1) {
    total += valueNoise(x * frequency, y * frequency) * amplitude
    frequency *= 2
    amplitude *= 0.5
  }

  return total / 0.9375
}

function ribbonEnergy(nx: number, ny: number, offset: number, slope: number, width: number, time: number) {
  const wave = Math.sin(nx * 9.5 + time * 0.42 + offset * 5) * 0.08
    + Math.sin(nx * 21 - time * 0.26 + offset * 9) * 0.035
  const center = offset + slope * (nx - 0.5) + wave
  return clamp(1 - Math.abs(ny - center) / width, 0, 1)
}

function fieldEnergy(cell: PixelCell, time: number) {
  const { fineTurbulence, nx, ny, quietCenter, seed, shimmer, turbulence } = cell
  const drift = time * 0.055
  const timeWarp = Math.sin(time * 0.28 + seed * 9) * 0.045
  const warpedX = nx + (turbulence - 0.5) * 0.16 + timeWarp + Math.sin(ny * 7.5 + time * 0.22) * 0.035
  const warpedY = ny + (fineTurbulence - 0.5) * 0.12 - timeWarp * 0.6 + Math.sin(nx * 6.5 - time * 0.3) * 0.03
  const bandA = ribbonEnergy(warpedX, warpedY, 0.24 + Math.sin(drift + 0.4) * 0.18, 0.58, 0.16, time)
  const bandB = ribbonEnergy(warpedX, warpedY, 0.58 + Math.sin(drift * 0.8 + 2.1) * 0.16, -0.42, 0.18, time + 4)
  const bandC = ribbonEnergy(warpedX, warpedY, 0.86 - fract(drift * 0.42) * 1.25, 0.36, 0.13, time + 8)
  const liveShimmer = Math.sin(seed * 17 + warpedX * 28 - warpedY * 19 + time * 0.38) * 0.5 + 0.5
  const breath = Math.sin(time * 0.55 + seed * 8) * 0.5 + 0.5
  const mixedShimmer = shimmer * 0.84 + liveShimmer * 0.16

  return (Math.max(bandA, bandB, bandC) * 0.72 + mixedShimmer * breath * 0.28) * (0.58 + quietCenter * 0.42)
}

function cellPosition(item: PixelCell, cell: number) {
  return {
    x: item.x * cell,
    y: item.y * cell,
  }
}

function crispRect(position: { x: number; y: number }, size: number) {
  return {
    size: Math.max(2, Math.round(size)),
    x: Math.round(position.x),
    y: Math.round(position.y),
  }
}

export function LoginAsciiBackground({ className, ...props }: LoginAsciiBackgroundProps) {
  const canvasRef = React.useRef<HTMLCanvasElement>(null)
  const cellsRef = React.useRef<PixelCell[]>([])
  const frameRef = React.useRef<number | null>(null)
  const lastFrameRef = React.useRef(0)
  const sizeRef = React.useRef({ cell: 12, height: 1, width: 1 })

  React.useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const context = canvas.getContext("2d")
    if (!context) return

    context.imageSmoothingEnabled = false

    const root = canvas.parentElement
    if (!root) return

    const motionQuery = window.matchMedia("(prefers-reduced-motion: reduce)")
    let reducedMotion = motionQuery.matches
    let visible = document.visibilityState === "visible"
    let palette: PixelPalette = { particle: "" }

    const resolvePalette = () => {
      const styles = getComputedStyle(root)
      const rootStyles = getComputedStyle(document.documentElement)
      const particle = rootStyles.getPropertyValue("--primary").trim() || styles.color
      return { particle }
    }

    const rebuildCells = () => {
      const rect = root.getBoundingClientRect()
      const width = Math.max(1, Math.floor(rect.width))
      const height = Math.max(1, Math.floor(rect.height))
      const dpr = Math.min(window.devicePixelRatio || 1, 1.5)
      const cell = width < 720 ? 10 : 14
      const columns = Math.ceil(width / cell)
      const rows = Math.ceil(height / cell)
      const cells: PixelCell[] = []

      palette = resolvePalette()
      canvas.width = Math.floor(width * dpr)
      canvas.height = Math.floor(height * dpr)
      context.imageSmoothingEnabled = false
      context.setTransform(dpr, 0, 0, dpr, 0, 0)
      sizeRef.current = { cell, height, width }

      for (let y = 0; y < rows; y += 1) {
        for (let x = 0; x < columns; x += 1) {
          const seed = hash2(x, y)
          const nx = columns > 0 ? x / columns : 0
          const ny = height > 0 ? (y * cell) / height : 0
          const quietCenter = clamp(Math.hypot((nx - 0.5) / 0.34, (ny - 0.5) / 0.26) - 0.54, 0, 1)
          cells.push({
            fineTurbulence: fbmNoise(nx * 8.2 + seed, ny * 8.2),
            nx,
            ny,
            quietCenter,
            seed,
            shimmer: fbmNoise(nx * 12 + seed * 3, ny * 12 - seed * 2),
            threshold: 0.44 + hash2(x + 7, y - 5) * 0.22,
            turbulence: fbmNoise(nx * 3.4, ny * 3.4),
            visible: false,
            x,
            y,
          })
        }
      }

      cellsRef.current = cells
    }

    const draw = (timestamp: number) => {
      const { cell, height, width } = sizeRef.current
      const time = reducedMotion ? 10 : timestamp * 0.001 * ANIMATION_TIME_SCALE

      context.globalAlpha = 1
      context.clearRect(0, 0, width, height)

      for (const item of cellsRef.current) {
        const value = fieldEnergy(item, time)
        const ambient = item.shimmer * 0.14 + item.quietCenter * 0.08
        const signal = value * 0.82 + ambient
        const threshold = item.threshold + Math.sin(time * 0.16 + item.seed * 12) * 0.012

        if (item.visible) {
          item.visible = signal >= threshold - 0.035
        } else {
          item.visible = signal >= threshold + 0.045
        }

        if (!item.visible) continue

        const position = cellPosition(item, cell)
        const size = clamp(cell * PARTICLE_SIZE_RATIO, 3, cell - 2)
        const rect = crispRect(position, size)

        context.globalAlpha = PARTICLE_ALPHA
        context.fillStyle = palette.particle
        context.fillRect(rect.x, rect.y, rect.size, rect.size)
      }

      context.globalAlpha = 1
    }

    const stop = () => {
      if (frameRef.current !== null) {
        window.cancelAnimationFrame(frameRef.current)
        frameRef.current = null
      }
    }

    const tick = (timestamp: number) => {
      if (!visible || reducedMotion) {
        frameRef.current = null
        return
      }

      if (timestamp - lastFrameRef.current >= FRAME_INTERVAL_MS) {
        draw(timestamp)
        lastFrameRef.current = timestamp
      }

      frameRef.current = window.requestAnimationFrame(tick)
    }

    const start = () => {
      stop()
      if (reducedMotion || !visible) {
        draw(0)
        return
      }
      frameRef.current = window.requestAnimationFrame(tick)
    }

    const handleMotionChange = (event: MediaQueryListEvent) => {
      reducedMotion = event.matches
      start()
    }

    const handleVisibilityChange = () => {
      visible = document.visibilityState === "visible"
      start()
    }

    const handleThemeChange = () => {
      palette = resolvePalette()
      if (!visible || reducedMotion) {
        draw(0)
      }
    }

    const resizeObserver = new ResizeObserver(() => {
      rebuildCells()
      start()
    })
    const themeObserver = new MutationObserver(handleThemeChange)

    rebuildCells()
    resizeObserver.observe(root)
    themeObserver.observe(document.documentElement, {
      attributeFilter: ["class", "style", "data-theme"],
      attributes: true,
    })
    motionQuery.addEventListener("change", handleMotionChange)
    document.addEventListener("visibilitychange", handleVisibilityChange)
    start()

    return () => {
      stop()
      resizeObserver.disconnect()
      themeObserver.disconnect()
      motionQuery.removeEventListener("change", handleMotionChange)
      document.removeEventListener("visibilitychange", handleVisibilityChange)
    }
  }, [])

  return (
    <div
      aria-hidden="true"
      className={cn("login-ascii-background pointer-events-none absolute inset-0 size-full text-primary", className)}
      {...props}
    >
      <canvas ref={canvasRef} className="absolute inset-0 size-full" />
    </div>
  )
}
