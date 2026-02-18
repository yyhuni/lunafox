"use client"

import {
  TargetSettingsContent,
  TargetSettingsDialogs,
  TargetSettingsErrorState,
  TargetSettingsLoadingState,
} from "./target-settings-sections"
import { useTargetSettingsState } from "./target-settings-state"

interface TargetSettingsProps {
  targetId: number
}

/**
 * Target settings component
 * Contains blacklist configuration and scheduled scans
 */
export function TargetSettings({ targetId }: TargetSettingsProps) {
  const state = useTargetSettingsState({ targetId })

  if (state.isLoading) {
    return <TargetSettingsLoadingState />
  }

  if (state.error) {
    return <TargetSettingsErrorState t={state.t} />
  }

  return (
    <>
      <TargetSettingsContent state={state} />
      <TargetSettingsDialogs state={state} />
    </>
  )
}
