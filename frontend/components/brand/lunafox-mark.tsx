"use client"

import * as React from "react"

import { cn } from "@/lib/utils"

type LunaFoxMarkProps = {
  className?: string
  size?: number
  label?: string
  decorative?: boolean
}

const FAVICON_MASK_DATA_URI =
  "data:image/png;base64,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"
let faviconMaskImagePromise: Promise<HTMLImageElement> | null = null

function loadFaviconMaskImage() {
  if (!faviconMaskImagePromise) {
    faviconMaskImagePromise = new Promise((resolve, reject) => {
      const image = new Image()

      image.onload = () => resolve(image)
      image.onerror = () => reject(new Error("Failed to load LunaFox favicon mask image"))
      image.src = FAVICON_MASK_DATA_URI
    })
  }

  return faviconMaskImagePromise
}

export function LunaFoxMark({
  className,
  size,
  label = "LunaFox Logo",
  decorative = false,
}: LunaFoxMarkProps) {
  const resolvedSize = size ? `${size}px` : undefined

  return (
    <span
      role={decorative ? undefined : "img"}
      aria-label={decorative ? undefined : label}
      aria-hidden={decorative ? "true" : undefined}
      data-slot="lunafox-mark"
      className={cn("lunafox-mark inline-flex shrink-0", className)}
      style={{
        width: resolvedSize,
        height: resolvedSize,
        ["--lunafox-mark-size" as string]: resolvedSize,
      }}
    />
  )
}

export function getLunaFoxFaviconSvg(color: string) {
  return [
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">`,
    `<defs>`,
    `<mask id="fox-mask">`,
    `<rect width="64" height="64" fill="black"/>`,
    `<image href="${FAVICON_MASK_DATA_URI}" width="64" height="64" preserveAspectRatio="xMidYMid meet"/>`,
    `</mask>`,
    `</defs>`,
    `<rect width="64" height="64" fill="${color}" mask="url(#fox-mask)"/>`,
    `</svg>`,
  ].join("")
}

export async function getLunaFoxFaviconPngDataUrl(color: string) {
  const maskImage = await loadFaviconMaskImage()
  const canvas = document.createElement("canvas")
  canvas.width = 64
  canvas.height = 64

  const context = canvas.getContext("2d")
  if (!context) {
    throw new Error("Failed to create LunaFox favicon canvas context")
  }

  context.clearRect(0, 0, 64, 64)
  context.fillStyle = color
  context.fillRect(0, 0, 64, 64)
  context.globalCompositeOperation = "destination-in"
  context.drawImage(maskImage, 0, 0, 64, 64)
  context.globalCompositeOperation = "source-over"

  return canvas.toDataURL("image/png")
}

export async function getLunaFoxBrandedFaviconPngDataUrl({
  background,
  foreground,
}: {
  background: string
  foreground: string
}) {
  const maskImage = await loadFaviconMaskImage()
  const canvas = document.createElement("canvas")
  canvas.width = 64
  canvas.height = 64

  const context = canvas.getContext("2d")
  if (!context) {
    throw new Error("Failed to create LunaFox branded favicon canvas context")
  }

  context.clearRect(0, 0, 64, 64)
  context.fillStyle = background
  context.beginPath()
  context.arc(32, 32, 30, 0, Math.PI * 2)
  context.fill()

  context.save()
  context.fillStyle = foreground
  context.fillRect(12, 12, 40, 40)
  context.globalCompositeOperation = "destination-in"
  context.drawImage(maskImage, 12, 12, 40, 40)
  context.restore()

  return canvas.toDataURL("image/png")
}
