import React from "react"
import {
  URLValidator,
  type ParseResult,
  type TargetType,
  type URLValidationErrorCode,
} from "@/lib/url-validator"
import { useBulkCreateEndpoints } from "@/hooks/use-endpoints"
import { useBulkCreateWebsites } from "@/hooks/use-websites"
import { useBulkCreateDirectories } from "@/hooks/use-directories"
import { countLineNumberedTextareaLines } from "@/components/common/line-numbered-textarea"

export type AssetType = "endpoint" | "website" | "directory"

export type BulkAddUrlLineIssueType = "invalid" | "duplicate" | "mismatch"
export type BulkAddUrlLineIssueTone = "error" | "warning"

export interface BulkAddUrlLineIssue {
  type: BulkAddUrlLineIssueType
  tone: BulkAddUrlLineIssueTone
  lineNumber: number
  url: string
  errorCode?: URLValidationErrorCode
  error?: string
  duplicateOfLine?: number
}

type ValidationResultState = (ParseResult & {
  blockingIssueCount: number
  advisoryIssueCount: number
  lineIssues: BulkAddUrlLineIssue[]
}) | null

type UseBulkAddUrlsDialogStateProps = {
  targetId: number
  assetType: AssetType
  targetName?: string
  targetType?: TargetType
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
}

function hasNonWhitespaceInput(value: string) {
  return /\S/u.test(value)
}

function buildLineIssues(result: ParseResult): BulkAddUrlLineIssue[] {
  const invalidIssues: BulkAddUrlLineIssue[] = result.invalidItems.map((item) => ({
    type: "invalid",
    tone: "error",
    lineNumber: item.lineNumber,
    url: item.url,
    errorCode: item.errorCode,
    error: item.error,
  }))

  const mismatchIssues: BulkAddUrlLineIssue[] = result.mismatchedItems.map((item) => ({
    type: "mismatch",
    tone: "error",
    lineNumber: item.lineNumber,
    url: item.url,
  }))

  const duplicateIssues: BulkAddUrlLineIssue[] = result.duplicateItems.map((item) => ({
    type: "duplicate",
    tone: "warning",
    lineNumber: item.lineNumber,
    url: item.url,
    duplicateOfLine: item.duplicateOfLine,
  }))

  return [...invalidIssues, ...mismatchIssues, ...duplicateIssues]
    .sort((first, second) => first.lineNumber - second.lineNumber)
}

function buildValidationResult(result: ParseResult): NonNullable<ValidationResultState> {
  return {
    ...result,
    blockingIssueCount: result.invalidCount + result.mismatchedCount,
    advisoryIssueCount: result.duplicateCount,
    lineIssues: buildLineIssues(result),
  }
}

export function useBulkAddUrlsDialogState({
  targetId,
  assetType,
  targetName,
  targetType,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  onSuccess,
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

    const parsed = URLValidator.parseLines(value)
    if (parsed.length === 0) {
      setValidationResult(null)
      return
    }

    const result = URLValidator.validateBatch(parsed, targetName, targetType)
    setValidationResult(buildValidationResult(result))
  }, [targetName, targetType])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()

    if (!hasNonWhitespaceInput(inputText)) return
    if (!validationResult || validationResult.validCount === 0) return

    const parsed = URLValidator.parseLines(inputText)
    const result = URLValidator.validateBatch(parsed, targetName, targetType)

    if (result.invalidCount > 0 || result.mismatchedCount > 0 || result.validCount === 0) return

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
  }, [inputText, mutation, onSuccess, setOpen, targetId, targetName, targetType, validationResult])

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

  const lineCount = Math.max(countLineNumberedTextareaLines(inputText), 8)

  const isFormValid =
    hasNonWhitespaceInput(inputText) &&
    validationResult !== null &&
    validationResult.validCount > 0 &&
    validationResult.blockingIssueCount === 0

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
