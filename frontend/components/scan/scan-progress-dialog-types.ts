import type { ScanStage, StageStatus } from "@/types/scan.types"

/**
 * Scan stage details
 */
export interface StageDetail {
  stage: ScanStage
  engineId: string
  status: StageStatus
  skipReason?: string
  duration?: string
  detail?: string
  resultCount?: number
}

/**
 * Actionable guidance for a known execution failure. The guidance replaces the
 * raw failure message as the primary explanation while the raw message stays
 * available as secondary detail.
 */
export interface ScanProgressFailureGuidance {
  kind: "no_enabled_templates"
  targetHref: "/tools/nuclei/"
}

/**
 * Scan progress data
 */
export interface ScanProgressData {
  id: number
  target?: {
    id: number
    name: string
    type: string
  }
  scanWorkflow: string
  plannedEngineIds: string[]
  status: string
  progress: number
  currentStage?: ScanStage
  startedAt?: string
  errorMessage?: string
  failureKind?: string
  failureGuidance?: ScanProgressFailureGuidance | null
  stages: StageDetail[]
}
