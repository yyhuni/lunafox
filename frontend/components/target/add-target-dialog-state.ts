import React from "react"
import { useOrganizations } from "@/hooks/use-organizations"
import { useBatchCreateTargets } from "@/hooks/use-targets"
import {
  compileBusinessListFilter,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  type BusinessListFilterCompilerConfig,
} from "@/components/shared/data-table/business-list-query"
import { TargetValidator } from "@/lib/target-validator"
import type { BatchCreateTargetsRequest } from "@/types/target.types"

const ORGANIZATION_PICKER_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "displayName", operator: "=" },
  facets: {},
}

interface UseAddTargetDialogStateProps {
  onAdd?: () => void
  externalOpen?: boolean
  externalOnOpenChange?: (open: boolean) => void
  prefetchEnabled?: boolean
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

type InvalidTarget = {
  index: number
  lineNumber: number
  originalTarget: string
  error: string
  type?: string
}

export function useAddTargetDialogState({
  onAdd,
  externalOpen,
  externalOnOpenChange,
  prefetchEnabled,
  t,
}: UseAddTargetDialogStateProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const [formData, setFormData] = React.useState({
    targets: "",
    organizationIds: [] as string[],
  })

  const [orgSearchQuery, setOrgSearchQueryState] = React.useState("")
  const [orgPage, setOrgPageState] = React.useState(1)
  const [orgPageSize, setOrgPageSizeState] = React.useState(10)
  const [orgPageTokens, setOrgPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const [invalidTargets, setInvalidTargets] = React.useState<InvalidTarget[]>([])
  const batchCreateTargets = useBatchCreateTargets()

  const resetOrgPaging = React.useCallback(() => {
    setOrgPageTokens({ 1: undefined })
    setOrgPageState(1)
  }, [])

  const setOrgSearchQuery = React.useCallback<React.Dispatch<React.SetStateAction<string>>>((value) => {
    setOrgSearchQueryState((current) => {
      const next = typeof value === "function" ? value(current) : value
      if (next !== current) {
        resetOrgPaging()
      }
      return next
    })
  }, [resetOrgPaging])

  const setOrgPageSize = React.useCallback<React.Dispatch<React.SetStateAction<number>>>((value) => {
    setOrgPageSizeState((current) => {
      const next = typeof value === "function" ? value(current) : value
      if (next !== current) {
        resetOrgPaging()
      }
      return next
    })
  }, [resetOrgPaging])

  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  const shouldEnableOrgsQuery = Boolean(prefetchEnabled || open)
  const orgFilter = compileBusinessListFilter(
    { search: orgSearchQuery, filters: {} },
    ORGANIZATION_PICKER_FILTER_FIELDS
  )
  const {
    data: organizationsData,
    isLoading: isLoadingOrganizations,
    isPlaceholderData: isOrganizationsPlaceholderData,
  } = useOrganizations(
    {
      pageSize: orgPageSize,
      pageToken: orgPageTokens[orgPage],
      pageIndex: orgPage,
      filter: orgFilter,
    },
    { enabled: shouldEnableOrgsQuery }
  )
  const organizationNextPageToken = getCurrentCursorNextPageToken(
    organizationsData?.nextPageToken,
    isOrganizationsPlaceholderData,
  )

  React.useEffect(() => {
    if (organizationNextPageToken) {
      setOrgPageTokens((tokens) => ({ ...tokens, [orgPage + 1]: organizationNextPageToken }))
    }
  }, [orgPage, organizationNextPageToken])

  const organizationPaginationNavigation = React.useMemo(() => getCursorPaginationNavigation({
    currentPage: orgPage,
    pageTokens: orgPageTokens,
    nextPageToken: organizationNextPageToken,
  }), [orgPage, orgPageTokens, organizationNextPageToken])

  const transitionOrgPage = React.useCallback((requestedPage: number) => {
    const transition = getCursorPageTransition({
      currentPage: orgPage,
      pageTokens: orgPageTokens,
      nextPageToken: organizationNextPageToken,
      requestedPage,
    })
    if (!transition.reachable) return

    // Store the authorized token before changing the ordinal so the next query
    // cannot briefly fall back to the first page between renders.
    setOrgPageTokens((tokens) => ({ ...tokens, [requestedPage]: transition.pageToken }))
    setOrgPageState(requestedPage)
  }, [orgPage, orgPageTokens, organizationNextPageToken])

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

  const handleInputChange = React.useCallback((field: "targets", value: string) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }))

    if (field === "targets") {
      const lines = TargetValidator.parseLines(value)

      if (lines.length === 0) {
        setInvalidTargets([])
        return
      }

      const results = TargetValidator.validateTargetBatch(lines)
      const invalid = results
        .filter((r) => !r.isValid)
        .map((r) => ({
          index: r.index,
          lineNumber: r.lineNumber,
          originalTarget: r.originalTarget,
          error: r.error || t("invalidFormat"),
          type: r.type,
        }))
      setInvalidTargets(invalid)
    }
  }, [t])

  const targetCount = React.useMemo(() =>
    formData.targets
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0).length,
    [formData.targets]
  )

  const resetForm = React.useCallback(() => {
    setFormData({
      targets: "",
      organizationIds: [],
    })
    setInvalidTargets([])
    setOrgSearchQuery("")
    setOrgPageState(1)
    setOrgPageSize(10)
    setOrgPageTokens({ 1: undefined })
  }, [setOrgPageSize, setOrgSearchQuery])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()

    if (!formData.targets.trim()) return
    if (invalidTargets.length > 0) return

    const targetList = formData.targets
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0)
      .map((name) => ({ name }))

    if (targetList.length === 0) return

    const payload: BatchCreateTargetsRequest = {
      targets: targetList,
    }

    if (formData.organizationIds.length > 0) {
      payload.organizationIds = formData.organizationIds.map((id) => Number.parseInt(id, 10))
    }

    batchCreateTargets.mutate(payload, {
      onSuccess: () => {
        resetForm()
        setOpen(false)
        onAdd?.()
      },
    })
  }, [batchCreateTargets, formData.organizationIds, formData.targets, invalidTargets.length, onAdd, resetForm, setOpen])

  const handleOpenChange = React.useCallback((newOpen: boolean) => {
    if (batchCreateTargets.isPending) return
    setOpen(newOpen)
    if (!newOpen) {
      resetForm()
    }
  }, [batchCreateTargets.isPending, resetForm, setOpen])

  const isFormValid = formData.targets.trim().length > 0 && invalidTargets.length === 0

  const handleTextareaScroll = React.useCallback((event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }, [])

  const filteredOrganizations = React.useMemo(() => {
    if (!organizationsData?.organizations) return []
    return organizationsData.organizations
  }, [organizationsData?.organizations])

  const handleToggleOrganization = React.useCallback((orgId: string) => {
    setFormData((prev) => {
      const isSelected = prev.organizationIds.includes(orgId)
      return {
        ...prev,
        organizationIds: isSelected
          ? prev.organizationIds.filter((id) => id !== orgId)
          : [...prev.organizationIds, orgId],
      }
    })
  }, [])

  const handleClearOrganizations = React.useCallback(() => {
    setFormData((prev) => ({
      ...prev,
      organizationIds: [],
    }))
  }, [])

  return {
    open,
    handleOpenChange,
    formData,
    handleInputChange,
    handleSubmit,
    targetCount,
    invalidTargets,
    isFormValid,
    lineNumbersRef,
    textareaRef,
    handleTextareaScroll,
    isLoadingOrganizations,
    filteredOrganizations,
    orgSearchQuery,
    setOrgSearchQuery,
    orgPageSize,
    setOrgPageSize,
    organizationTotalCount: organizationsData?.totalSize ?? organizationsData?.total ?? 0,
    organizationPaginationNavigation,
    onFirstOrgPage,
    onPreviousOrgPage,
    onNextOrgPage,
    handleToggleOrganization,
    handleClearOrganizations,
    batchCreateTargets,
  }
}
