"use client"

import { Layers, semanticIcons } from "@/components/icons"
import { DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { getArchitectureFlowRoleDotClassName } from "@/lib/chart-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { ArchitectureFlow } from "./architecture-flow"
import {
  agentFrames,
  ArchitectureFlowShell,
  ArchitectureFlowViewport,
  type FlowFrame,
  groupFrames,
  serverFrame,
  engineFrames,
} from "./architecture-flow-layout"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface ArchitectureDialogHeaderProps {
  t: TranslationFn
}

export function ArchitectureDialogHeader({ t }: ArchitectureDialogHeaderProps) {
  return (
    <DialogHeader className="min-w-0">
      <DialogTitle>{t("flowTitle")}</DialogTitle>
      <DialogDescription>{t("flowDesc")}</DialogDescription>
    </DialogHeader>
  )
}

function SummaryItem({
  title,
  description,
  icon: Icon,
}: {
  title: string
  description: string
  icon: typeof semanticIcons.concept.server
}) {
  return (
    <div className="flex gap-3">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-full border bg-background/60 text-muted-foreground">
        <Icon className="size-4" />
      </span>
      <div className="min-w-0 space-y-1">
        <div className="flex items-center gap-2">
          <span className={cn("size-2 rounded-full", getArchitectureFlowRoleDotClassName("agent"))} />
          <p className={textRole.bodyStrong}>{title}</p>
        </div>
        <p className={textRole.helperText}>{description}</p>
      </div>
    </div>
  )
}

function LegendLine({
  title,
  description,
  tone,
}: {
  title: string
  description: string
  tone: "control" | "local" | "result"
}) {
  const lineStyle = tone === "control"
    ? { borderColor: "var(--chart-2)" }
    : tone === "local"
      ? { borderColor: "var(--chart-1)" }
      : undefined

  return (
    <div className="flex gap-3">
      <span
        className={cn(
          "mt-2 h-0 w-12 shrink-0 border-t-2",
          tone === "result" ? "border-muted-foreground" : "border-dashed"
        )}
        style={lineStyle}
      />
      <div className="min-w-0 space-y-1">
        <p className={textRole.bodyStrong}>{title}</p>
        <p className={textRole.helperText}>{description}</p>
      </div>
    </div>
  )
}

interface ArchitectureSummaryPanelProps {
  t: TranslationFn
}

export function ArchitectureSummaryPanel({ t }: ArchitectureSummaryPanelProps) {
  const summaryItems = [
    {
      title: t("flowSummaryControlTitle"),
      description: t("flowSummaryControlDesc"),
      icon: Layers,
    },
    {
      title: t("flowSummaryAgentTitle"),
      description: t("flowSummaryAgentDesc"),
      icon: semanticIcons.concept.agent,
    },
    {
      title: t("flowSummaryEngineTitle"),
      description: t("flowSummaryEngineDesc"),
      icon: semanticIcons.concept.engine,
    },
  ]

  return (
    <aside className="min-h-0 border-r">
      <ScrollArea className="h-full">
        <div className="space-y-6 p-6">
          <section className="space-y-4">
            <h3 className={textRole.sectionTitle}>{t("flowSummaryTitle")}</h3>
            <div className="space-y-6">
              {summaryItems.map((item) => (
                <SummaryItem key={item.title} {...item} />
              ))}
            </div>
          </section>

          <section className="space-y-4 border-t pt-6">
            <h3 className={textRole.sectionTitle}>{t("flowLegendTitle")}</h3>
            <div className="space-y-5">
              <LegendLine
                description={t("flowLegendControlDesc")}
                title={t("flowLegendControlTitle")}
                tone="control"
              />
              <LegendLine
                description={t("flowLegendLocalDesc")}
                title={t("flowLegendLocalTitle")}
                tone="local"
              />
              <LegendLine
                description={t("flowLegendResultDesc")}
                title={t("flowLegendResultTitle")}
                tone="result"
              />
            </div>
          </section>
        </div>
      </ScrollArea>
    </aside>
  )
}

function ArchitectureFlowCanvasLoadingState({ fillAvailableSpace = false }: { fillAvailableSpace?: boolean }) {
  return (
    <ArchitectureFlowShell fillAvailableSpace={fillAvailableSpace}>
      <ArchitectureFlowViewport fillAvailableSpace={fillAvailableSpace}>
        {groupFrames.map((frame, index) => (
          <div
            key={`group-${index}`}
            className="absolute rounded-lg border border-dashed bg-muted/10"
            style={frameStyle(frame)}
          >
            <div className="flex items-center gap-2 px-4 py-3">
              <Skeleton className="h-4 w-24" />
              <span className="h-4 w-px bg-muted-foreground/20" />
              <Skeleton className="h-4 w-28" />
            </div>
          </div>
        ))}
        <Skeleton className="absolute rounded-md" style={frameStyle(serverFrame)} />
        {agentFrames.map((frame, index) => (
          <Skeleton key={`agent-${index}`} className="absolute rounded-md" style={frameStyle(frame)} />
        ))}
        {engineFrames.flatMap((frames, groupIndex) =>
          frames.map((frame, engineIndex) => (
            <Skeleton
              key={`engine-${groupIndex}-${engineIndex}`}
              className="absolute rounded-md"
              style={frameStyle(frame)}
            />
          ))
        )}
      </ArchitectureFlowViewport>
    </ArchitectureFlowShell>
  )
}

function frameStyle(frame: FlowFrame) {
  return {
    height: frame.height,
    left: frame.x,
    top: frame.y,
    width: frame.width,
  }
}

interface ArchitectureCommandCenterProps {
  t: TranslationFn
  isOpen: boolean
}

export function ArchitectureCommandCenter({
  t,
  isOpen,
}: ArchitectureCommandCenterProps) {
  return (
    <div className="grid min-h-0 flex-1">
      <div
        className="grid min-h-0"
        style={{ gridTemplateColumns: "264px minmax(0, 1fr)" }}
      >
        <ArchitectureSummaryPanel t={t} />
        <main className="min-h-0 overflow-hidden">
          {isOpen
            ? <ArchitectureFlow fillAvailableSpace showDiagramNote />
            : <ArchitectureFlowCanvasLoadingState fillAvailableSpace />}
        </main>
      </div>
    </div>
  )
}

export { ScrollArea }
