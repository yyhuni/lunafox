import React from "react"
import { useTargets } from "@/hooks/use-targets"
import { useOrganizations } from "@/hooks/use-organizations"
import {
  compileBusinessListFilter,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  type BusinessListFilterCompilerConfig,
} from "@/components/shared/data-table/business-list-query"
import type { Organization } from "@/types/organization.types"
import type { Target } from "@/types/target.types"

const ORGANIZATION_PICKER_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "displayName", operator: "=" },
  facets: {},
}

type UseScheduledScanSearchProps = {
  open: boolean
}

export function useScheduledScanSearch({ open }: UseScheduledScanSearchProps) {
  const [orgSearchInput, setOrgSearchInputValue] = React.useState("")
  const [targetSearchInput, setTargetSearchInput] = React.useState("")
  const [orgSearch, setOrgSearch] = React.useState("")
  const [targetSearch, setTargetSearch] = React.useState("")
  const [orgPage, setOrgPageState] = React.useState(1)
  const [orgPageSize, setOrgPageSizeState] = React.useState(8)
  const [orgPageTokens, setOrgPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [targetPage, setTargetPageState] = React.useState(1)
  const [targetPageSize, setTargetPageSizeState] = React.useState(10)
  const [targetPageTokens, setTargetPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const resetOrgPaging = React.useCallback(() => {
    setOrgPageTokens({ 1: undefined })
    setOrgPageState(1)
  }, [])

  const resetTargetPaging = React.useCallback(() => {
    setTargetPageTokens({ 1: undefined })
    setTargetPageState(1)
  }, [])

  const setOrgPageSize = React.useCallback<React.Dispatch<React.SetStateAction<number>>>((value) => {
    setOrgPageSizeState((current) => {
      const next = typeof value === "function" ? value(current) : value
      if (next !== current) {
        resetOrgPaging()
      }
      return next
    })
  }, [resetOrgPaging])

  const setOrgSearchInput = React.useCallback((value: string) => {
    setOrgSearchInputValue(value)
    setOrgSearch(value)
    resetOrgPaging()
  }, [resetOrgPaging])

  const setTargetPageSize = React.useCallback<React.Dispatch<React.SetStateAction<number>>>((value) => {
    setTargetPageSizeState((current) => {
      const next = typeof value === "function" ? value(current) : value
      if (next !== current) resetTargetPaging()
      return next
    })
  }, [resetTargetPaging])

  const setTargetSearchInputValue = React.useCallback((value: string) => {
    setTargetSearchInput(value)
    setTargetSearch(value)
    resetTargetPaging()
  }, [resetTargetPaging])

  const handleOrgSearch = React.useCallback(
    () => setOrgSearch(orgSearchInput),
    [orgSearchInput]
  )

  const orgFilter = compileBusinessListFilter(
    { search: orgSearch, filters: {} },
    ORGANIZATION_PICKER_FILTER_FIELDS
  )

  const {
    data: organizationsData,
    isFetching: isOrgFetching,
    isPlaceholderData: isOrganizationsPlaceholderData,
  } = useOrganizations({
    pageSize: orgPageSize,
    pageToken: orgPageTokens[orgPage],
    pageIndex: orgPage,
    filter: orgFilter,
  }, { enabled: open })

  const organizationNextPageToken = getCurrentCursorNextPageToken(
    organizationsData?.nextPageToken,
    isOrganizationsPlaceholderData,
  )

  React.useEffect(() => {
    if (organizationNextPageToken) {
      setOrgPageTokens((tokens) => ({ ...tokens, [orgPage + 1]: organizationNextPageToken }))
    }
  }, [orgPage, organizationNextPageToken])

  const {
    data: targetsData,
    isFetching: isTargetFetching,
    isPlaceholderData: isTargetsPlaceholderData,
  } = useTargets({
    pageSize: targetPageSize,
    pageToken: targetPageTokens[targetPage],
    filter: targetSearch || undefined,
  }, { enabled: open })

  const targetNextPageToken = getCurrentCursorNextPageToken(
    targetsData?.nextPageToken,
    isTargetsPlaceholderData,
  )

  React.useEffect(() => {
    if (targetNextPageToken) {
      setTargetPageTokens((tokens) => ({ ...tokens, [targetPage + 1]: targetNextPageToken }))
    }
  }, [targetNextPageToken, targetPage])

  const organizationPaginationNavigation = React.useMemo(() => getCursorPaginationNavigation({
    currentPage: orgPage,
    pageTokens: orgPageTokens,
    nextPageToken: organizationNextPageToken,
  }), [orgPage, orgPageTokens, organizationNextPageToken])

  const targetPaginationNavigation = React.useMemo(() => getCursorPaginationNavigation({
    currentPage: targetPage,
    pageTokens: targetPageTokens,
    nextPageToken: targetNextPageToken,
  }), [targetNextPageToken, targetPage, targetPageTokens])

  const transitionOrgPage = React.useCallback((requestedPage: number) => {
    const transition = getCursorPageTransition({
      currentPage: orgPage,
      pageTokens: orgPageTokens,
      nextPageToken: organizationNextPageToken,
      requestedPage,
    })
    if (!transition.reachable) return
    setOrgPageTokens((tokens) => ({ ...tokens, [requestedPage]: transition.pageToken }))
    setOrgPageState(requestedPage)
  }, [orgPage, orgPageTokens, organizationNextPageToken])

  const transitionTargetPage = React.useCallback((requestedPage: number) => {
    const transition = getCursorPageTransition({
      currentPage: targetPage,
      pageTokens: targetPageTokens,
      nextPageToken: targetNextPageToken,
      requestedPage,
    })
    if (!transition.reachable) return
    setTargetPageTokens((tokens) => ({ ...tokens, [requestedPage]: transition.pageToken }))
    setTargetPageState(requestedPage)
  }, [targetNextPageToken, targetPage, targetPageTokens])

  const onPreviousOrgPage = React.useCallback(() => {
    transitionOrgPage(orgPage - 1)
  }, [orgPage, transitionOrgPage])

  const onFirstOrgPage = React.useCallback(() => {
    if (orgPage === 1) return
    setOrgPageTokens({ 1: undefined })
    setOrgPageState(1)
  }, [orgPage])

  const onNextOrgPage = React.useCallback(() => {
    transitionOrgPage(orgPage + 1)
  }, [orgPage, transitionOrgPage])

  const onPreviousTargetPage = React.useCallback(() => {
    transitionTargetPage(targetPage - 1)
  }, [targetPage, transitionTargetPage])

  const onFirstTargetPage = React.useCallback(() => {
    if (targetPage === 1) return
    setTargetPageTokens({ 1: undefined })
    setTargetPageState(1)
  }, [targetPage])

  const onNextTargetPage = React.useCallback(() => {
    transitionTargetPage(targetPage + 1)
  }, [targetPage, transitionTargetPage])

  const organizations: Organization[] = organizationsData?.organizations || []
  const targets: Target[] = targetsData?.targets || []

  return {
    orgSearchInput,
    setOrgSearchInput,
    targetSearchInput,
    setTargetSearchInput: setTargetSearchInputValue,
    handleOrgSearch,
    orgPage,
    orgPageSize,
    setOrgPageSize,
    organizationTotalCount: organizationsData?.totalSize ?? organizationsData?.total ?? 0,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    targetPage,
    targetPageSize,
    setTargetPageSize,
    targetTotalCount: targetsData?.totalSize ?? targetsData?.total ?? 0,
    targetPaginationNavigation,
    onFirstTargetPage,
    onPreviousTargetPage,
    onNextTargetPage,
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
      setIsYamlValid(true)
    }
  }, [configuration, isConfigEdited])

  const applyProfileConfiguration = React.useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(false)
    setIsYamlValid(true)
  }, [])

  const handleConfigSync = React.useCallback((value: string) => {
    setConfiguration(value)
  }, [])

  const handleManualConfigChange = React.useCallback((value: string) => {
    setConfiguration(value)
    setIsConfigEdited(true)
  }, [])

  const handleOverwriteConfirm = React.useCallback(() => {
    if (pendingConfigChange !== null) {
      setConfiguration(pendingConfigChange)
      setIsConfigEdited(false)
      setIsYamlValid(true)
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
    applyProfileConfiguration,
    handleConfigSync,
    handleManualConfigChange,
    handleOverwriteConfirm,
    handleOverwriteCancel,
    handleYamlValidationChange,
    resetConfigState,
  }
}
