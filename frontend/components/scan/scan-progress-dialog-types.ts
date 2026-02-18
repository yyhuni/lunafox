import type { ScanStage, StageStatus } from "@/types/scan.types"

/**
 * Scan stage details
 */
export interface StageDetail {
  stage: ScanStage
  status: StageStatus
  duration?: string
  detail?: string
  resultCount?: number
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
  engineNames: string[]
  status: string
  progress: number
  currentStage?: ScanStage
  startedAt?: string
  errorMessage?: string
  stages: StageDetail[]
}
