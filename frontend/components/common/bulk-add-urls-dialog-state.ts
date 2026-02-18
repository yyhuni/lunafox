import React from "react"
import { URLValidator, type TargetType } from "@/lib/url-validator"
import { useBulkCreateEndpoints } from "@/hooks/use-endpoints"
import { useBulkCreateWebsites } from "@/hooks/use-websites"
import { useBulkCreateDirectories } from "@/hooks/use-directories"

export type AssetType = "endpoint" | "website" | "directory"

type ValidationResultState = {
  validCount: number
  invalidCount: number
  duplicateCount: number
  mismatchedCount: number
  firstError?: { index: number; url: string; error: string }
  firstMismatch?: { index: number; url: string }
} | null

type UseBulkAddUrlsDialogStateProps = {
  targetId: number
  assetType: AssetType
  targetName?: string
  targetType?: TargetType
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
  tBulkAdd: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useBulkAddUrlsDialogState({
  targetId,
  assetType,
  targetName,
  targetType,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  onSuccess,
  tBulkAdd,
}: UseBulkAddUrlsDialogStateProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const [inputText, setInputText] = React.useState("")
  const [validationResult, setValidationResult] = React.useState<ValidationResultState>(null)

  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  const bulkCreateEndpoints = useBulkCreateEndpoints()
  const bulkCreateWebsites = useBulkCreateWebsites()
  const bulkCreateDirectories = useBulkCreateDirectories()

  const mutation = React.useMemo(() => {
    switch (assetType) {
      case "endpoint":
        return bulkCreateEndpoints
      case "website":
        return bulkCreateWebsites
      case "directory":
        return bulkCreateDirectories
    }
  }, [assetType, bulkCreateDirectories, bulkCreateEndpoints, bulkCreateWebsites])

  if (!mutation) {
    throw new Error(`Unsupported asset type: ${assetType}`)
  }

  const handleInputChange = React.useCallback((value: string) => {
    setInputText(value)

    const parsed = URLValidator.parse(value)
    if (parsed.length === 0) {
      setValidationResult(null)
      return
    }

    const result = URLValidator.validateBatch(parsed, targetName, targetType)
    setValidationResult({
      validCount: result.validCount,
      invalidCount: result.invalidCount,
      duplicateCount: result.duplicateCount,
      mismatchedCount: result.mismatchedCount,
      firstError: result.invalidItems[0]
        ? {
            index: result.invalidItems[0].index,
            url: result.invalidItems[0].url,
            error: result.invalidItems[0].error || tBulkAdd("formatInvalid"),
          }
        : undefined,
      firstMismatch: result.mismatchedItems[0]
        ? {
            index: result.mismatchedItems[0].index,
            url: result.mismatchedItems[0].url,
          }
        : undefined,
    })
  }, [tBulkAdd, targetName, targetType])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()

    if (!inputText.trim()) return
    if (!validationResult || validationResult.validCount === 0) return

    const parsed = URLValidator.parse(inputText)
    const result = URLValidator.validateBatch(parsed)

    mutation.mutate(
      { targetId, urls: result.urls },
      {
        onSuccess: () => {
          setInputText("")
          setValidationResult(null)
          setOpen(false)
          onSuccess?.()
        },
      }
    )
  }, [inputText, mutation, onSuccess, setOpen, targetId, validationResult])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!mutation.isPending) {
      setOpen(nextOpen)
      if (!nextOpen) {
        setInputText("")
        setValidationResult(null)
      }
    }
  }, [mutation.isPending, setOpen])

  const handleTextareaScroll = React.useCallback((event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }, [])

  const lineCount = Math.max(inputText.split("\n").length, 8)

  const hasMismatchError = validationResult !== null
    && validationResult.mismatchedCount > 0
    && targetType !== "cidr"

  const isFormValid =
    inputText.trim().length > 0 &&
    validationResult !== null &&
    validationResult.validCount > 0 &&
    validationResult.invalidCount === 0 &&
    !hasMismatchError

  return {
    open,
    handleOpenChange,
    inputText,
    setInputText,
    validationResult,
    lineNumbersRef,
    textareaRef,
    mutation,
    handleInputChange,
    handleSubmit,
    handleTextareaScroll,
    lineCount,
    isFormValid,
  }
}
