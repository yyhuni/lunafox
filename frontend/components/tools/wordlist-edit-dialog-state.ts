import React from "react"
import { useWordlistContent, useUpdateWordlistContent } from "@/hooks/use-wordlists"
import type { Wordlist } from "@/types/wordlist.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseWordlistEditDialogStateProps = {
  wordlist: Wordlist | null
  open: boolean
  onOpenChange: (open: boolean) => void
  t: TranslationFn
}

export function useWordlistEditDialogState({
  wordlist,
  open,
  onOpenChange,
  t,
}: UseWordlistEditDialogStateProps) {
  const [content, setContent] = React.useState("")
  const [hasChanges, setHasChanges] = React.useState(false)

  const { data: originalContent, isLoading } = useWordlistContent(
    open && wordlist ? wordlist.id : null
  )
  const updateMutation = useUpdateWordlistContent()

  React.useEffect(() => {
    if (originalContent !== undefined && open) {
      setContent(originalContent)
      setHasChanges(false)
    }
  }, [originalContent, open])

  React.useEffect(() => {
    if (!open) {
      setContent("")
      setHasChanges(false)
    }
  }, [open])

  const handleEditorChange = React.useCallback((value: string) => {
    setContent(value)
    setHasChanges(value !== originalContent)
  }, [originalContent])

  const handleSave = React.useCallback(() => {
    if (!wordlist) return

    updateMutation.mutate(
      { id: wordlist.id, content },
      {
        onSuccess: () => {
          setHasChanges(false)
          onOpenChange(false)
        },
      }
    )
  }, [content, onOpenChange, updateMutation, wordlist])

  const handleClose = React.useCallback(() => {
    if (hasChanges) {
      const confirmed = window.confirm(t("confirmClose"))
      if (!confirmed) return
    }
    onOpenChange(false)
  }, [hasChanges, onOpenChange, t])

  const lineCount = React.useMemo(() => content.split("\n").length, [content])
  const isSaving = updateMutation.isPending
  const fileHashShort = wordlist?.fileHash ? `${wordlist.fileHash.slice(0, 12)}...` : null

  return {
    content,
    hasChanges,
    lineCount,
    isLoading,
    isSaving,
    fileHashShort,
    handleEditorChange,
    handleSave,
    handleClose,
  }
}
