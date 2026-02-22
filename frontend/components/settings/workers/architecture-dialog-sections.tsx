"use client"

import dynamic from "next/dynamic"
import { DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import type { ArchitectureLabels, ArchitectureRoleDetail } from "@/components/settings/workers/architecture-dialog-state"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

const ArchitectureFlow = dynamic(
  () => import("./architecture-flow").then((mod) => mod.ArchitectureFlow),
  {
    ssr: false,
    loading: () => <Skeleton className="h-[360px] w-full" />,
  }
)

interface ArchitectureDialogHeaderProps {
  t: TranslationFn
}

export function ArchitectureDialogHeader({ t }: ArchitectureDialogHeaderProps) {
  return (
    <DialogHeader>
      <DialogTitle>{t("flowTitle")}</DialogTitle>
      <DialogDescription>{t("flowDesc")}</DialogDescription>
    </DialogHeader>
  )
}

interface ArchitectureFlowSectionProps {
  t: TranslationFn
  isOpen: boolean
}

export function ArchitectureFlowSection({ t, isOpen }: ArchitectureFlowSectionProps) {
  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <p className="text-sm font-medium">{t("flowDiagramTitle")}</p>
        <p className="text-xs text-muted-foreground leading-5">
          {t("flowDiagramDesc")}
        </p>
      </div>
      <div className="rounded-md border bg-muted/15 px-3 py-2">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-[11px] text-muted-foreground">
          <div className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-primary" />
            <span>{t("flowServerTitle")}</span>
          </div>
          <div className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-[color:var(--color-chart-2)]" />
            <span>{t("flowAgentTitle")}</span>
          </div>
          <div className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-[color:var(--color-chart-1)]" />
            <span>{t("flowWorkerTitle")}</span>
          </div>
          <span className="text-muted-foreground/60">|</span>
          <div className="inline-flex items-center gap-2">
            <span className="w-8 border-t-2 border-primary/70" />
            <span>{t("flowLegendBidirectionalShort")}</span>
          </div>
          <div className="inline-flex items-center gap-2">
            <span className="w-8 border-t-2 border-dashed border-primary/70" />
            <span>{t("flowLegendLocalShort")}</span>
          </div>
        </div>
      </div>
      {isOpen ? <ArchitectureFlow /> : <Skeleton className="h-[360px] w-full" />}
    </div>
  )
}

interface ArchitectureRoleSectionProps {
  t: TranslationFn
  labels: ArchitectureLabels
  roleDetails: ArchitectureRoleDetail[]
}

export function ArchitectureRoleSection({
  t,
  labels,
  roleDetails,
}: ArchitectureRoleSectionProps) {
  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-medium">{t("flowRolesTitle")}</p>
        <p className="text-xs text-muted-foreground">
          {t("flowRolesDesc")}
        </p>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        {roleDetails.map((role) => (
          <div
            key={role.id}
            className="rounded-md border bg-muted/10 p-3"
          >
            <p className="text-sm font-medium">{role.title}</p>
            <div className="mt-2 space-y-1.5 text-xs text-muted-foreground">
              <p>
                <span className="font-medium text-foreground">
                  {labels.location}:
                </span>{" "}
                {role.location}
              </p>
              <p>
                <span className="font-medium text-foreground">
                  {labels.comms}:
                </span>{" "}
                {role.comms}
              </p>
              <div>
                <span className="font-medium text-foreground">
                  {labels.responsibilities}:
                </span>
                <ul className="mt-1 list-disc space-y-0.5 pl-4">
                  {role.responsibilities.map((item, index) => (
                    <li key={index}>{item}</li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

interface ArchitectureStepsSectionProps {
  t: TranslationFn
  steps: string[]
}

export function ArchitectureStepsSection({ t, steps }: ArchitectureStepsSectionProps) {
  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-medium">{t("flowStepsTitle")}</p>
        <p className="text-xs text-muted-foreground">
          {t("flowStepsDesc")}
        </p>
      </div>
      <ol className="space-y-2 text-sm text-muted-foreground">
        {steps.map((step, index) => (
          <li key={step} className="flex gap-2">
            <span className="font-semibold text-foreground">{index + 1}.</span>
            <span>{step}</span>
          </li>
        ))}
      </ol>
    </div>
  )
}

export { ScrollArea }
