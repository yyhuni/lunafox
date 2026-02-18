"use client"

import { useTranslations } from "next-intl"

import { Badge } from "@/components/ui/badge"
import { YamlEditor } from "@/components/ui/yaml-editor"
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
  className,
}: ScanConfigEditorLayoutProps) {
  const t = useTranslations("scan.initiate")
  const tStages = useTranslations("scan.progress.stages")

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {showCapabilities ? (
        <div className="px-4 py-2 border-b bg-muted/30 flex items-center gap-2 shrink-0">
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
        <YamlEditor
          value={configuration}
          onChange={onChange}
          disabled={disabled}
          onValidationChange={onValidationChange}
        />
      </div>
    </div>
  )
}
