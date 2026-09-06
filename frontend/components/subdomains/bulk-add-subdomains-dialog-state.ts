import React from "react"
import {
  SubdomainValidator,
  type ParseResult,
  type SubdomainValidationErrorCode,
} from "@/lib/subdomain-validator"
import { useBulkCreateSubdomains } from "@/hooks/use-subdomains"
import { countLineNumberedTextareaLines } from "@/components/common/line-numbered-textarea"

export type BulkAddSubdomainLineIssueType = "invalid" | "duplicate"
export type BulkAddSubdomainLineIssueTone = "error" | "warning"

export interface BulkAddSubdomainLineIssue {
  type: BulkAddSubdomainLineIssueType
  tone: BulkAddSubdomainLineIssueTone
  lineNumber: number
  subdomain: string
  errorCode?: SubdomainValidationErrorCode
  error?: string
  duplicateOfLine?: number
}

type ValidationResultState = (ParseResult & {
  blockingIssueCount: number
  advisoryIssueCount: number
  lineIssues: BulkAddSubdomainLineIssue[]
}) | null

type UseBulkAddSubdomainsDialogStateProps = {
  targetId: number
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
}

function buildLineIssues(result: ParseResult): BulkAddSubdomainLineIssue[] {
  const invalidIssues: BulkAddSubdomainLineIssue[] = result.invalidItems.map((item) => ({
    type: "invalid",
    tone: "error",
    lineNumber: item.lineNumber,
    subdomain: item.subdomain,
    errorCode: item.errorCode,
    error: item.error,
  }))

  const duplicateIssues: BulkAddSubdomainLineIssue[] = result.duplicateItems.map((item) => ({
    type: "duplicate",
    tone: "warning",
    lineNumber: item.lineNumber,
    subdomain: item.subdomain,
    duplicateOfLine: item.duplicateOfLine,
  }))

  return [...invalidIssues, ...duplicateIssues]
    .sort((first, second) => first.lineNumber - second.lineNumber)
}

function buildValidationResult(result: ParseResult): NonNullable<ValidationResultState> {
  return {
    ...result,
    blockingIssueCount: result.invalidCount,
    advisoryIssueCount: result.duplicateCount,
    lineIssues: buildLineIssues(result),
  }
}

export function useBulkAddSubdomainsDialogState({
  targetId,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  onSuccess,
}: UseBulkAddSubdomainsDialogStateProps) {
  const [internalOpen, setInternalOpen] = React.useState(false)
  const open = externalOpen !== undefined ? externalOpen : internalOpen
  const setOpen = externalOnOpenChange || setInternalOpen

  const [inputText, setInputText] = React.useState("")
  const [validationResult, setValidationResult] = React.useState<ValidationResultState>(null)

  const lineNumbersRef = React.useRef<HTMLDivElement | null>(null)
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null)

  const bulkCreateSubdomains = useBulkCreateSubdomains()

  const handleInputChange = React.useCallback((value: string) => {
    setInputText(value)

    const parsed = SubdomainValidator.parseLines(value)
    if (parsed.length === 0) {
      setValidationResult(null)
      return
    }

    const result = SubdomainValidator.validateBatch(parsed)
    setValidationResult(buildValidationResult(result))
  }, [])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()

    if (!inputText.trim()) return
    if (!validationResult || validationResult.validCount === 0) return

    const parsed = SubdomainValidator.parseLines(inputText)
    const result = SubdomainValidator.validateBatch(parsed)

    if (result.invalidCount > 0 || result.validCount === 0) return

    bulkCreateSubdomains.mutate(
      { targetId, subdomains: result.subdomains },
      {
        onSuccess: () => {
          setInputText("")
          setValidationResult(null)
          setOpen(false)
          onSuccess?.()
        },
      }
    )
  }, [bulkCreateSubdomains, inputText, onSuccess, setOpen, targetId, validationResult])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!bulkCreateSubdomains.isPending) {
      setOpen(nextOpen)
      if (!nextOpen) {
        setInputText("")
        setValidationResult(null)
      }
    }
  }, [bulkCreateSubdomains.isPending, setOpen])

  const handleTextareaScroll = React.useCallback((event: React.UIEvent<HTMLTextAreaElement>) => {
    if (lineNumbersRef.current) {
      lineNumbersRef.current.scrollTop = event.currentTarget.scrollTop
    }
  }, [])

  const lineCount = Math.max(countLineNumberedTextareaLines(inputText), 8)

  const isFormValid =
    inputText.trim().length > 0 &&
    validationResult !== null &&
    validationResult.validCount > 0 &&
    validationResult.blockingIssueCount === 0

  return {
    open,
    handleOpenChange,
    inputText,
    validationResult,
    lineNumbersRef,
    textareaRef,
    bulkCreateSubdomains,
    handleInputChange,
    handleSubmit,
    handleTextareaScroll,
    lineCount,
    isFormValid,
  }
}
