"use client"

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react"
import * as yaml from "js-yaml"
import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { BulkLineValidationInput, type BulkLineValidationIssue } from "@/components/common/bulk-line-validation-input"
import { countLineNumberedTextareaLines } from "@/components/common/line-numbered-textarea"
import { cn } from "@/lib/utils"

import type { ScanConfigEditorState } from "./scan-config-editor-state"

interface ScanConfigEditorLayoutProps {
  state: ScanConfigEditorState
  configuration: string
  onChange: (value: string) => void
  onValidationChange?: (isValid: boolean) => void
  isConfigEdited?: boolean
  disabled?: boolean
  showCapabilities?: boolean
  showLabel?: boolean
  className?: string
}

export function ScanConfigEditorLayout({
  state,
  configuration,
  onChange,
  onValidationChange,
  isConfigEdited = false,
  disabled = false,
  showCapabilities = true,
  showLabel = true,
  className,
}: ScanConfigEditorLayoutProps) {
  const t = useTranslations("scan.initiate")
  const tStages = useTranslations("scan.progress.stages")

  const lineNumbersRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const [yamlError, setYamlError] = useState<{ message: string; line?: number } | null>(null)

  const handleScroll = useCallback((event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }, [])

  const validateYaml = useCallback((content: string) => {
    if (!content.trim()) {
      setYamlError(null)
      onValidationChange?.(true)
      return
    }
    try {
      yaml.load(content)
      setYamlError(null)
      onValidationChange?.(true)
    } catch (err) {
      const e = err as yaml.YAMLException
      setYamlError({ message: e.message, line: e.mark?.line ? e.mark.line + 1 : undefined })
      onValidationChange?.(false)
    }
  }, [onValidationChange])

  useEffect(() => {
    validateYaml(configuration)
  }, [configuration, validateYaml])

  const lineIssues = useMemo<BulkLineValidationIssue[]>(() => {
    if (!yamlError) return []
    return [{
      id: "yaml-error",
      lineNumber: yamlError.line ?? 1,
      tone: "error",
      badgeLabel: t("configError"),
      message: yamlError.message,
    }]
  }, [yamlError, t])

  const validationResult = configuration.trim().length > 0
    ? {
      validCount: yamlError ? 0 : 1,
      blockingIssueCount: yamlError ? 1 : 0,
      advisoryIssueCount: 0,
      lineIssues,
    }
    : null

  return (
    <div className={cn("flex flex-col h-full gap-3", className)}>
      {showCapabilities ? (
        <div className="bg-muted/30 border-b flex gap-2 items-center px-4 py-2 shrink-0">
          {state.capabilityStyles.length > 0 ? (
            <div className="flex flex-wrap gap-1">
              {state.capabilityStyles.map((cap) => (
                <Badge key={cap.key} variant="outline" className={cn("text-xs py-0", cap.color)}>
                  {tStages(cap.key)}
                </Badge>
              ))}
            </div>
          ) : null}
          {isConfigEdited ? (
            <Badge variant="outline" className="ml-auto text-xs">
              {t("configEdited")}
            </Badge>
          ) : null}
        </div>
      ) : null}

      <div className="flex-1 overflow-hidden">
        <BulkLineValidationInput
          id="scan-config"
          name="scan-config"
          label={t("configLabel")}
          labelClassName={showLabel ? undefined : "sr-only"}
          placeholder={t("configPlaceholder")}
          value={configuration}
          lineCount={Math.max(countLineNumberedTextareaLines(configuration), 8)}
          lineNumbersRef={lineNumbersRef}
          textareaRef={textareaRef}
          onValueChange={onChange}
          onScroll={handleScroll}
          fillHeight
          disabled={disabled}
          helper={t("configHelper")}
          example={t("configExample")}
          emptySummary={t("configEmptySummary")}
          validSummary={t("configValidSummary")}
          blockingSummary={t("configBlockingSummary", { count: yamlError ? 1 : 0 })}
          collapseDetails={t("configCollapseDetails")}
          expandDetails={t("configExpandDetails")}
          validationResult={validationResult}
        />
      </div>
    </div>
  )
}
