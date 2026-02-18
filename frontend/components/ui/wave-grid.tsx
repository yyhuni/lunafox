"use client"

import React, { useCallback, useEffect, useRef, useMemo } from "react"
import { cn } from "@/lib/utils"

interface WaveGridProps {
  /** Size of each grid square in pixels */
  squareSize?: number
  /** Gap between squares in pixels */
  gridGap?: number
  /** Color of the squares (hex or rgb) */
  color?: string
  /** Maximum opacity of squares (0-1) */
  maxOpacity?: number
  /** Minimum opacity of squares (0-1) */
  minOpacity?: number
  /** Wave speed multiplier */
  waveSpeed?: number
  /** Wave length in pixels */
  waveLength?: number
  /** Wave direction: 'horizontal' | 'vertical' | 'diagonal' | 'radial' | 'random' */
  waveDirection?: "horizontal" | "vertical" | "diagonal" | "radial" | "random"
  /** Flicker chance per frame for random mode (0-1) */
  flickerChance?: number
  /** Width of the grid */
  width?: number
  /** Height of the grid */
  height?: number
  /** Additional class names */
  className?: string
}

function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16),
      }
    : null
}

export function WaveGrid({
  squareSize = 4,
  gridGap = 6,
  color = "#6B7280",
  maxOpacity = 0.5,
  minOpacity = 0.1,
  waveSpeed = 0.03,
  waveLength = 100,
  waveDirection = "diagonal",
  flickerChance = 0.03,
  width = 800,
  height = 800,
  className,
}: WaveGridProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const animationRef = useRef<number>(0)
  const timeRef = useRef<number>(0)
  const isActiveRef = useRef(true)
  const isIntersectingRef = useRef(true)
  const prefersReducedMotionRef = useRef(false)

  const rgb = useMemo(() => hexToRgb(color) || { r: 107, g: 114, b: 128 }, [color])

  const cols = useMemo(() => Math.ceil(width / (squareSize + gridGap)), [width, squareSize, gridGap])
  const rows = useMemo(() => Math.ceil(height / (squareSize + gridGap)), [height, squareSize, gridGap])
  const centerX = useMemo(() => (cols * (squareSize + gridGap)) / 2, [cols, squareSize, gridGap])
  const centerY = useMemo(() => (rows * (squareSize + gridGap)) / 2, [rows, squareSize, gridGap])

  // Store opacity values for random mode
  const opacitiesRef = useRef<Float32Array | null>(null)
  const targetOpacitiesRef = useRef<Float32Array | null>(null)

  // Initialize opacity arrays for random mode
  useEffect(() => {
    if (waveDirection === "random") {
      const totalCells = rows * cols
      opacitiesRef.current = new Float32Array(totalCells)
      targetOpacitiesRef.current = new Float32Array(totalCells)
      // Initialize with random values
      for (let i = 0; i < totalCells; i++) {
        const randomOpacity = minOpacity + Math.random() * (maxOpacity - minOpacity)
        opacitiesRef.current[i] = randomOpacity
        targetOpacitiesRef.current[i] = randomOpacity
      }
    }
  }, [waveDirection, rows, cols, minOpacity, maxOpacity])

  const getWaveOpacity = useCallback(
    (x: number, y: number, time: number): number => {
      let phase: number

      switch (waveDirection) {
        case "horizontal":
          phase = x / waveLength
          break
        case "vertical":
          phase = y / waveLength
          break
        case "diagonal":
          phase = (x + y) / waveLength
          break
        case "radial":
          const dx = x - centerX
          const dy = y - centerY
          phase = Math.sqrt(dx * dx + dy * dy) / waveLength
          break
        default:
          phase = (x + y) / waveLength
      }

      // Sine wave oscillates between -1 and 1, normalize to 0-1
      const normalizedSin = (Math.sin(phase - time * waveSpeed) + 1) / 2
      // Map to opacity range
      return minOpacity + normalizedSin * (maxOpacity - minOpacity)
    },
    [waveDirection, waveLength, waveSpeed, minOpacity, maxOpacity, centerX, centerY]
  )

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext("2d")
    if (!ctx) return
    if (!isActiveRef.current) {
      animationRef.current = requestAnimationFrame(draw)
      return
    }

    ctx.clearRect(0, 0, width, height)

    if (waveDirection === "random" && opacitiesRef.current && targetOpacitiesRef.current) {
      // Random flickering mode
      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          const index = row * cols + col
          const x = col * (squareSize + gridGap)
          const y = row * (squareSize + gridGap)

          // Randomly decide to change target opacity
          if (Math.random() < flickerChance) {
            targetOpacitiesRef.current[index] = minOpacity + Math.random() * (maxOpacity - minOpacity)
          }

          // Smoothly interpolate towards target
          const current = opacitiesRef.current[index]
          const target = targetOpacitiesRef.current[index]
          opacitiesRef.current[index] = current + (target - current) * 0.1

          ctx.fillStyle = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${opacitiesRef.current[index]})`
          ctx.fillRect(x, y, squareSize, squareSize)
        }
      }
    } else {
      // Wave mode
      for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
          const x = col * (squareSize + gridGap)
          const y = row * (squareSize + gridGap)
          const opacity = getWaveOpacity(x, y, timeRef.current)

          ctx.fillStyle = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${opacity})`
          ctx.fillRect(x, y, squareSize, squareSize)
        }
      }
    }

    timeRef.current += 1
    animationRef.current = requestAnimationFrame(draw)
  }, [cols, rows, squareSize, gridGap, width, height, rgb, getWaveOpacity, waveDirection, flickerChance, minOpacity, maxOpacity])

  const updateActiveState = useCallback(() => {
    isActiveRef.current =
      !prefersReducedMotionRef.current &&
      !document.hidden &&
      isIntersectingRef.current
  }, [])

  useEffect(() => {
    if (typeof window === "undefined") return
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)")
    const handleReduceMotion = (e: MediaQueryListEvent | MediaQueryList) => {
      prefersReducedMotionRef.current = "matches" in e ? e.matches : mq.matches
      updateActiveState()
    }
    handleReduceMotion(mq)
    mq.addEventListener("change", handleReduceMotion)
    return () => mq.removeEventListener("change", handleReduceMotion)
  }, [updateActiveState])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const handleVisibility = () => {
      updateActiveState()
    }

    const io = new IntersectionObserver(
      ([entry]) => {
        isIntersectingRef.current = entry.isIntersecting
        updateActiveState()
      },
      { threshold: 0 }
    )

    io.observe(canvas)
    document.addEventListener("visibilitychange", handleVisibility)
    updateActiveState()

    return () => {
      io.disconnect()
      document.removeEventListener("visibilitychange", handleVisibility)
    }
  }, [updateActiveState])

  useEffect(() => {
    draw()
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [draw])

  return (
    <canvas
      ref={canvasRef}
      width={width}
      height={height}
      aria-hidden="true"
      role="presentation"
      className={cn("pointer-events-none", className)}
    />
  )
}
