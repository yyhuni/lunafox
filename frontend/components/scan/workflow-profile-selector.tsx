"use client"

import React from "react"
import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { adaptWorkflowProfile, serializeWorkflowProfileDraft } from "@/lib/workflow-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { ScanWorkflow, ScanWorkflowProfile } from "@/types/scan-workflow.types"

interface WorkflowProfileSelectorProps {
  workflows: ScanWorkflow[]
  selectedWorkflowNames: string[]
  onWorkflowNamesChange: (workflowNames: string[]) => void
  onConfigurationChange: (config: string) => void
  loadWorkflowProfile: (workflowName: string) => Promise<ScanWorkflowProfile>
  disabled?: boolean
  className?: string
  showHeader?: boolean
  contentClassName?: string
}

export function WorkflowProfileSelector({
  workflows,
  selectedWorkflowNames,
  onWorkflowNamesChange,
  onConfigurationChange,
  loadWorkflowProfile,
  disabled = false,
  className,
  showHeader = true,
  contentClassName,
}: WorkflowProfileSelectorProps) {
  const t = useTranslations("scan.initiate")
  const [isLoadingProfile, setIsLoadingProfile] = React.useState(false)
  const selectedWorkflowName = selectedWorkflowNames[0] ?? ""

  const selectWorkflow = React.useCallback(async (workflowName: string) => {
    const workflow = workflows.find((item) => item.name === workflowName)
    if (!workflow || !workflow.isExecutable) return

    setIsLoadingProfile(true)
    try {
      const profile = await loadWorkflowProfile(workflowName)
      const draft = adaptWorkflowProfile(profile, workflow)
      onWorkflowNamesChange([workflowName])
      onConfigurationChange(serializeWorkflowProfileDraft(draft))
    } catch {
      // A malformed or stale Profile blocks initialization; no Workflow or
      // Engine defaults are synthesized as a recovery path.
      onWorkflowNamesChange([])
      onConfigurationChange("")
    } finally {
      setIsLoadingProfile(false)
    }
  }, [loadWorkflowProfile, onConfigurationChange, onWorkflowNamesChange, workflows])

  return (
    <div className={cn("flex flex-col", className)}>
      <div className={cn("p-6", contentClassName)}>
        {showHeader ? (
          <div className="mb-4 space-y-1">
            <p className={textRole.bodyStrong}>{t("workflowListTitle")}</p>
            <p className={textRole.helperText}>{t("workflowListHint")}</p>
          </div>
        ) : null}
        <RadioGroup value={selectedWorkflowName} onValueChange={(value) => void selectWorkflow(value)} disabled={disabled || isLoadingProfile} className="grid gap-2">
          {workflows.map((workflow) => {
            const itemId = `scheduled-workflow-${workflow.name}`
            const unavailable = !workflow.isExecutable
            return (
              <label key={workflow.name} htmlFor={itemId} className={cn("radius-control flex min-w-0 items-center gap-3 border px-3 py-2.5", workflow.name === selectedWorkflowName ? "border-primary/60 bg-primary/10" : "border-border", unavailable && "cursor-not-allowed opacity-60")}>
                <RadioGroupItem id={itemId} value={workflow.name} disabled={unavailable} />
                <span className={cn("min-w-0 flex-1 truncate", textRole.navLabel)}>{workflow.displayName || workflow.name}</span>
                {unavailable ? <Badge variant="outline">{t("workflowUnavailable")}</Badge> : null}
              </label>
            )
          })}
        </RadioGroup>
      </div>
    </div>
  )
}
