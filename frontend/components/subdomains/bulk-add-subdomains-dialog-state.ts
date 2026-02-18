import React from "react"
import { SubdomainValidator } from "@/lib/subdomain-validator"
import { useBulkCreateSubdomains } from "@/hooks/use-subdomains"

type ValidationResultState = {
  validCount: number
  invalidCount: number
  duplicateCount: number
  firstError?: { index: number; subdomain: string; error: string }
} | null

type UseBulkAddSubdomainsDialogStateProps = {
  targetId: number
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSuccess?: () => void
  t: (key: string, params?: Record<string, string | number | Date>) => string
}

export function useBulkAddSubdomainsDialogState({
  targetId,
  open: externalOpen,
  onOpenChange: externalOnOpenChange,
  onSuccess,
  t,
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

    const parsed = SubdomainValidator.parse(value)
    if (parsed.length === 0) {
      setValidationResult(null)
      return
    }

    const result = SubdomainValidator.validateBatch(parsed)
    setValidationResult({
      validCount: result.validCount,
      invalidCount: result.invalidCount,
      duplicateCount: result.duplicateCount,
      firstError: result.invalidItems[0]
        ? {
            index: result.invalidItems[0].index,
            subdomain: result.invalidItems[0].subdomain,
            error: result.invalidItems[0].error || t("formatInvalid"),
          }
        : undefined,
    })
  }, [t])

  const handleSubmit = React.useCallback((event: React.FormEvent) => {
    event.preventDefault()

    if (!inputText.trim()) return
    if (!validationResult || validationResult.validCount === 0) return

    const parsed = SubdomainValidator.parse(inputText)
    const result = SubdomainValidator.validateBatch(parsed)

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

  const lineCount = Math.max(inputText.split("\n").length, 8)

  const isFormValid =
    inputText.trim().length > 0 &&
    validationResult !== null &&
    validationResult.validCount > 0

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
