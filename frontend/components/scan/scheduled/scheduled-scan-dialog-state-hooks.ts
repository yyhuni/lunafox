import React from "react"
import { useTargets } from "@/hooks/use-targets"
import { useOrganizations } from "@/hooks/use-organizations"
import type { Organization } from "@/types/organization.types"
import type { Target } from "@/types/target.types"

type UseScheduledScanSearchProps = {
  open: boolean
}

export function useScheduledScanSearch({ open }: UseScheduledScanSearchProps) {
  const [orgSearchInput, setOrgSearchInput] = React.useState("")
  const [targetSearchInput, setTargetSearchInput] = React.useState("")
  const [orgSearch, setOrgSearch] = React.useState("")
  const [targetSearch, setTargetSearch] = React.useState("")

  const handleOrgSearch = React.useCallback(
    () => setOrgSearch(orgSearchInput),
    [orgSearchInput]
  )
  const handleTargetSearch = React.useCallback(
    () => setTargetSearch(targetSearchInput),
    [targetSearchInput]
  )

  const { data: organizationsData, isFetching: isOrgFetching } = useOrganizations({
    pageSize: 20,
    filter: orgSearch || undefined,
  }, { enabled: open })

  const { data: targetsData, isFetching: isTargetFetching } = useTargets({
    pageSize: 20,
    filter: targetSearch || undefined,
  }, { enabled: open })

  const organizations: Organization[] = organizationsData?.organizations || []
  const targets: Target[] = targetsData?.targets || []

  return {
    orgSearchInput,
    setOrgSearchInput,
    targetSearchInput,
    setTargetSearchInput,
    handleOrgSearch,
    handleTargetSearch,
    isOrgFetching,
    isTargetFetching,
    organizations,
    targets,
  }
}

export function useScheduledScanConfigState() {
  const [configuration, setConfiguration] = React.useState("")
  const [isConfigEdited, setIsConfigEdited] = React.useState(false)
  const [isYamlValid, setIsYamlValid] = React.useState(true)
  const [showOverwriteConfirm, setShowOverwriteConfirm] = React.useState(false)
  const [pendingConfigChange, setPendingConfigChange] = React.useState<string | null>(null)

  const handlePresetConfigChange = React.useCallback((value: string) => {
    if (isConfigEdited && configuration !== value) {
      setPendingConfigChange(value)
      setShowOverwriteConfirm(true)
    } else {
      setConfiguration(value)
      setIsConfigEdited(false)
    }
  }, [configuration, isConfigEdited])

  const handleManualConfigChange = React.useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const handleOverwriteConfirm = React.useCallback(() => {
    if (pendingConfigChange !== null) {
      setConfiguration(pendingConfigChange)
      setIsConfigEdited(false)
    }
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
  }, [pendingConfigChange])

  const handleOverwriteCancel = React.useCallback(() => {
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
  }, [])

  const handleYamlValidationChange = React.useCallback((isValid: boolean) => {
    setIsYamlValid(isValid)
  }, [])

  const resetConfigState = React.useCallback(() => {
    setConfiguration("")
    setIsConfigEdited(false)
    setIsYamlValid(true)
    setShowOverwriteConfirm(false)
    setPendingConfigChange(null)
  }, [])

  return {
    configuration,
    setConfiguration,
    isConfigEdited,
    isYamlValid,
    showOverwriteConfirm,
    pendingConfigChange,
    setShowOverwriteConfirm,
    setPendingConfigChange,
    handlePresetConfigChange,
    handleManualConfigChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    resetConfigState,
  }
}
