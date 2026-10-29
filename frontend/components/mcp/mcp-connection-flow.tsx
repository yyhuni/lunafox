"use client"

import { useEffect, useId, useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { LunaFoxMark } from "@/components/brand/lunafox-mark"
import { semanticIcons } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { mcpClients, type McpClient } from "./mcp-client-configuration"

const McpIcon = semanticIcons.concept.mcp
const capabilities = ["assets", "risk", "scan"] as const
const capabilityIcons = [semanticIcons.concept.target, semanticIcons.concept.vulnerability, semanticIcons.concept.scan]

export function McpConnectionFlow({ client }: { client: McpClient }) {
  const t = useTranslations("mcpIntegration")
  const titleId = useId()
  const diagramId = useId()
  const svgRef = useRef<SVGSVGElement>(null)
  const [motionVisible, setMotionVisible] = useState(false)
  const selectedClientIndex = mcpClients.indexOf(client)
  const inputY = 38 + selectedClientIndex * 46
  const inputPath = `M148 ${inputY} H214 Q240 ${inputY} 240 110 H330`

  useEffect(() => {
    const svg = svgRef.current
    if (!svg) return
    const media = window.matchMedia("(prefers-reduced-motion: no-preference) and (min-width: 640px)")
    let inView = false
    const updateMotion = () => setMotionVisible(media.matches && inView && !document.hidden)
    const observer = new IntersectionObserver(([entry]) => {
      inView = entry.isIntersecting
      updateMotion()
    })
    observer.observe(svg)
    media.addEventListener("change", updateMotion)
    document.addEventListener("visibilitychange", updateMotion)
    return () => {
      observer.disconnect()
      media.removeEventListener("change", updateMotion)
      document.removeEventListener("visibilitychange", updateMotion)
    }
  }, [])

  return (
    <Card variant="compact">
      <CardContent className="grid items-center gap-4 md:grid-cols-3">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2"><McpIcon className="size-4 text-muted-foreground" aria-hidden="true" /><span className={textRole.monoLabel}>MODEL CONTEXT PROTOCOL</span></div>
          <h2 className={textRole.panelTitle}>{t("page.heroTitle")}</h2>
          <p className={cn("max-w-sm", textRole.bodySubtle)}>{t("page.heroDescription")}</p>
          <Badge variant="outline" size="compact">Streamable HTTP</Badge>
        </div>
        <figure className="m-0 min-w-0 md:col-span-2">
          <svg ref={svgRef} id={diagramId} viewBox="0 0 720 220" className="hidden max-h-48 w-full sm:block" role="img" aria-labelledby={titleId} data-slot="mcp-connection-flow">
            <title id={titleId}>{t("page.flowDescription")}</title>
            <g fill="none" stroke="var(--border)" strokeWidth="1.2">
              {mcpClients.map((option, index) => <path key={option} d={`M148 ${38 + index * 46} H214 Q240 ${38 + index * 46} 240 110 H330`} />)}
              {capabilities.map((capability, index) => <path key={capability} d={`M390 110 H466 Q492 110 492 ${48 + index * 62} H558`} />)}
              <circle cx="360" cy="110" r="44" opacity=".65" /><circle cx="360" cy="110" r="64" opacity=".35" />
            </g>
            <path d={inputPath} fill="none" stroke="var(--interaction-accent)" strokeWidth="1.5" />
            {/* These packets illustrate the relationship; they never represent live requests. */}
            {motionVisible ? <g key={client} className="motion-reduce:hidden" aria-hidden="true" data-slot="mcp-flow-packets">
              {capabilities.map((capability, index) => (
                <g key={capability} fill="var(--interaction-accent)">
                  <circle r="6" opacity=".12" />
                  <circle r="2.8" />
                  <animateMotion dur="3.6s" begin={`${-index * 1.2}s`} repeatCount="indefinite" calcMode="linear" path={`${inputPath} H390 H466 Q492 110 492 ${48 + index * 62} H558`} />
                </g>
              ))}
            </g> : null}
            {mcpClients.map((option, index) => <g key={option}>
              <rect x="12" y={22 + index * 46} width="136" height="32" rx="6" fill="var(--card)" stroke={option === client ? "var(--interaction-accent)" : "var(--border)"} opacity={option === client ? "1" : ".65"} />
              <text x="28" y={43 + index * 46} fill="currentColor" className={option === client ? textRole.bodyStrong : textRole.caption}>{t(`configuration.clients.${option}`)}</text>
              {option === client ? <circle cx="135" cy={38 + index * 46} r="2" fill="var(--interaction-accent)" /> : null}
            </g>)}
            <circle cx="360" cy="110" r="26" fill="var(--card)" />
            <foreignObject x="340" y="90" width="40" height="40"><LunaFoxMark className="size-10" decorative /></foreignObject>
            <text x="360" y="170" textAnchor="middle" fill="currentColor" className={textRole.sectionTitle}>LunaFox MCP</text>
            <text x="360" y="190" textAnchor="middle" fill="currentColor" className={textRole.caption}>Streamable HTTP</text>
            {capabilities.map((capability, index) => {
              const Icon = capabilityIcons[index]
              return <g key={capability}>
                <rect x="558" y={28 + index * 62} width="150" height="40" rx="6" fill="var(--card)" stroke="var(--border)" />
                <Icon x="570" y={40 + index * 62} width="16" height="16" className="text-muted-foreground" />
                <text x="596" y={53 + index * 62} fill="currentColor" className={textRole.bodyStrong}>{t(`page.capabilities.${capability}`)}</text>
              </g>
            })}
          </svg>
          <div className="space-y-3 py-3 sm:hidden" aria-label={t("page.flowDescription")}>
            <div className="flex items-center justify-center gap-4"><span className={textRole.bodyStrong}>{t(`configuration.clients.${client}`)}</span><span className={textRole.caption}>→</span><LunaFoxMark className="size-7" decorative /><span className={textRole.bodyStrong}>LunaFox MCP</span></div>
            <div className="flex flex-wrap justify-center gap-x-4 gap-y-2">{capabilities.map((capability) => <span key={capability} className={textRole.caption}>{t(`page.capabilities.${capability}`)}</span>)}</div>
          </div>
        </figure>
      </CardContent>
    </Card>
  )
}
