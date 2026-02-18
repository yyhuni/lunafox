"use client"

import { useEffect, useRef, useState } from "react"
import type mermaidType from "mermaid"

interface MermaidDiagramProps {
  chart: string
  className?: string
}

let mermaidLoader: Promise<typeof mermaidType> | null = null

function loadMermaid() {
  if (!mermaidLoader) {
    mermaidLoader = import("mermaid").then((mod) => mod.default)
  }
  return mermaidLoader
}

export function MermaidDiagram({ chart, className = "" }: MermaidDiagramProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svg, setSvg] = useState<string>("")

  useEffect(() => {
    let cancelled = false

    const renderDiagram = async () => {
      if (!containerRef.current) return

      try {
        const mermaid = await loadMermaid()
        if (cancelled) return

        const rootStyle = getComputedStyle(document.documentElement)
        const resolveColor = (value: string, fallback: string) => {
          if (!value || !document.body) return fallback
          const probe = document.createElement("span")
          probe.style.color = value
          probe.style.position = "absolute"
          probe.style.opacity = "0"
          probe.style.pointerEvents = "none"
          document.body.appendChild(probe)
          const resolved = getComputedStyle(probe).color || fallback
          probe.remove()
          return resolved
        }
        const colorFromVar = (name: string, fallback: string) => {
          const raw = rootStyle.getPropertyValue(name).trim()
          if (!raw) return fallback
          const value = raw.includes("(") ? raw : `hsl(${raw})`
          return resolveColor(value, fallback)
        }

        const background = colorFromVar("--background", "#ffffff")
        const card = colorFromVar("--card", "#ffffff")
        const foreground = colorFromVar("--foreground", "#111827")
        const mutedForeground = colorFromVar("--muted-foreground", "#6b7280")
        const border = colorFromVar("--border", "#e5e7eb")

        // Configure Mermaid
        mermaid.initialize({
          startOnLoad: false,
          theme: "base",
          themeVariables: {
            fontFamily:
              "var(--font-sans, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif)",
            primaryColor: card,
            primaryTextColor: foreground,
            primaryBorderColor: border,
            lineColor: border,
            secondaryColor: card,
            tertiaryColor: card,
            background: "transparent",
            mainBkg: card,
            secondBkg: card,
            tertiaryBkg: card,
            nodeBorder: border,
            nodeTextColor: foreground,
            clusterBkg: background,
            clusterBorder: border,
            defaultLinkColor: border,
            titleColor: foreground,
            edgeLabelBackground: background,
            textColor: mutedForeground,
          },
          flowchart: {
            useMaxWidth: true,
            htmlLabels: false,
            curve: "linear",
          },
        })

        // Generate unique ID
        const id =
          typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
            ? `mermaid-${crypto.randomUUID()}`
            : `mermaid-${Math.random().toString(36).slice(2, 11)}`

        // Render chart
        const { svg: renderedSvg } = await mermaid.render(id, chart)
        if (cancelled) return
        setSvg(renderedSvg)
      } catch (error) {
        void error
      }
    }

    void renderDiagram()
    return () => {
      cancelled = true
    }
  }, [chart])

  return (
    <div
      ref={containerRef}
      className={`mermaid-container ${className}`}
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}
