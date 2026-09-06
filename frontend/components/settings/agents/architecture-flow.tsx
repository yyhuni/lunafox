"use client"

import { useTranslations } from "next-intl"

import { textRole } from "@/lib/typography"

import { ArchitectureFlowShell } from "./architecture-flow-layout"
import { ArchitectureFlowCanvas } from "./architecture-flow-sections"
import { useArchitectureFlowState } from "./architecture-flow-state"

interface ArchitectureFlowProps {
  fillAvailableSpace?: boolean
  showDiagramNote?: boolean
}

export function ArchitectureFlow({
  fillAvailableSpace = false,
  showDiagramNote = false,
}: ArchitectureFlowProps) {
  const t = useTranslations("pages.agents")
  const flow = useArchitectureFlowState(t)

  return (
    <ArchitectureFlowShell fillAvailableSpace={fillAvailableSpace}>
      <ArchitectureFlowCanvas
        fillAvailableSpace={fillAvailableSpace}
        flow={flow}
        overlay={showDiagramNote ? (
          <p className={`${textRole.helperText} text-center`}>{t("flowDiagramNote")}</p>
        ) : undefined}
      />
    </ArchitectureFlowShell>
  )
}
