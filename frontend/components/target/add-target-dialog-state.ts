import React from "react"
import { useOrganizations } from "@/hooks/use-organizations"
import { useBatchCreateTargets } from "@/hooks/use-targets"
import { TargetValidator } from "@/lib/target-validator"
import type { BatchCreateTargetsRequest } from "@/types/target.types"

interface UseAddTargetDialogStateProps {
  onAdd?: () => void
  externalOpen?: boolean
  externalOnOpenChange?: (open: boolean) => void
  prefetchEnabled?: boolean
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

type InvalidTarget = {
  index: number
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

  const [orgPickerOpen, setOrgPickerOpen] = React.useState(false)
  const [formData, setFormData] = React.useState({
    targets: "",
    organizationId: "",
  })

  const [orgSearchQuery, setOrgSearchQuery] = React.useState("")
  const [orgPage, setOrgPage] = React.useState(1)
  const [orgPageSize, setOrgPageSize] = React.useState(20)
  const pageSizeOptions = React.useMemo(() => [20, 50, 200, 500, 1000], [])

  const [invalidTargets, setInvalidTargets] = React.useState<InvalidTarget[]>([])
  const [selectedOrgName, setSelectedOrgName] = React.useState("")

  const batchCreateTargets = useBatchCreateTargets()

  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  const shouldEnableOrgsQuery = Boolean(prefetchEnabled || orgPickerOpen)
  const { data: organizationsData, isLoading: isLoadingOrganizations } = useOrganizations(
    {
      page: orgPage,
      pageSize: orgPageSize,
    },
    { enabled: shouldEnableOrgsQuery }
  )

  const handleInputChange = React.useCallback((field: keyof typeof formData, value: string) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }))

    if (field === "targets") {
      const lines = value
        .split("\n")
        .map((s) => s.trim())
        .filter((s) => s.length > 0)

      if (lines.length === 0) {
        setInvalidTargets([])
        return
      }

      const results = TargetValidator.validateTargetBatch(lines)
      const invalid = results
        .filter((r) => !r.isValid)
        .map((r) => ({
          index: r.index,
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
      organizationId: "",
    })
    setInvalidTargets([])
    setSelectedOrgName("")
    setOrgSearchQuery("")
    setOrgPage(1)
    setOrgPageSize(20)
  }, [])

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

    if (formData.organizationId) {
      payload.organizationId = Number.parseInt(formData.organizationId, 10)
    }

    batchCreateTargets.mutate(payload, {
      onSuccess: () => {
        resetForm()
        setOpen(false)
        onAdd?.()
      },
    })
  }, [batchCreateTargets, formData.organizationId, formData.targets, invalidTargets.length, onAdd, resetForm, setOpen])

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

  const selectedOrganization = React.useMemo(() => (
    organizationsData?.organizations.find((org) => org.id.toString() === formData.organizationId)
  ), [formData.organizationId, organizationsData?.organizations])

  React.useEffect(() => {
    if (selectedOrganization) {
      setSelectedOrgName(selectedOrganization.name)
    }
  }, [selectedOrganization])

  const filteredOrganizations = React.useMemo(() => {
    if (!organizationsData?.organizations) return []
    if (!orgSearchQuery) return organizationsData.organizations
    return organizationsData.organizations.filter((org) =>
      org.name.toLowerCase().includes(orgSearchQuery.toLowerCase())
    )
  }, [organizationsData?.organizations, orgSearchQuery])

  const handleSelectOrganization = React.useCallback((orgId: string, orgName: string) => {
    handleInputChange("organizationId", orgId)
    setSelectedOrgName(orgName)
    setOrgPickerOpen(false)
    setOrgSearchQuery("")
    setOrgPage(1)
    setOrgPageSize(20)
  }, [handleInputChange])

  const handleOrgPickerOpenChange = React.useCallback((nextOpen: boolean) => {
    setOrgPickerOpen(nextOpen)
    if (!nextOpen) {
      setOrgSearchQuery("")
      setOrgPage(1)
      setOrgPageSize(20)
    }
  }, [])

  return {
    open,
    handleOpenChange,
    orgPickerOpen,
    setOrgPickerOpen,
    handleOrgPickerOpenChange,
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
    organizationsData,
    filteredOrganizations,
    selectedOrgName,
    orgSearchQuery,
    setOrgSearchQuery,
    orgPage,
    setOrgPage,
    orgPageSize,
    setOrgPageSize,
    pageSizeOptions,
    handleSelectOrganization,
    batchCreateTargets,
  }
}
