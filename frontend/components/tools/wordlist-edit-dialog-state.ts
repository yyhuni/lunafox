import React from "react"
import { useUpdateWordlistMetadata, useWordlistContent, useUpdateWordlistContent } from "@/hooks/use-wordlists"
import { countLineNumberedTextareaLines } from "@/components/common/line-numbered-textarea"
import type { Wordlist } from "@/types/wordlist.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

type UseWordlistEditDialogStateProps = {
  wordlist: Wordlist | null
  open: boolean
  canEditContent: boolean
  contentEnabled?: boolean
  onOpenChange: (open: boolean) => void
  t: TranslationFn
}

export function useWordlistEditDialogState({
  wordlist,
  open,
  canEditContent,
  contentEnabled = false,
  onOpenChange,
  t,
}: UseWordlistEditDialogStateProps) {
  const [content, setContent] = React.useState("")
  const [hasChanges, setHasChanges] = React.useState(false)
  const [description, setDescription] = React.useState("")
  const [tags, setTags] = React.useState<string[]>([])
  const [hasMetadataChanges, setHasMetadataChanges] = React.useState(false)
  const loadedContentWordlistIdRef = React.useRef<number | null>(null)

  const { data: originalContent, isLoading } = useWordlistContent(
    contentEnabled && open && wordlist && canEditContent ? wordlist.id : null
  )
  const updateMutation = useUpdateWordlistContent()
  const updateMetadataMutation = useUpdateWordlistMetadata()

  React.useEffect(() => {
    if (wordlist && open) {
      setDescription(wordlist.description ?? "")
      setTags(wordlist.tags ?? [])
      setHasMetadataChanges(false)
    }
  }, [open, wordlist])

  React.useEffect(() => {
    if (
      originalContent !== undefined
      && contentEnabled
      && open
      && wordlist
      && loadedContentWordlistIdRef.current !== wordlist.id
    ) {
      setContent(originalContent)
      setHasChanges(false)
      // The edit panel unmounts outside the active tab; do not replace a local
      // draft with cached content when the operator returns to it.
      loadedContentWordlistIdRef.current = wordlist.id
    }
  }, [contentEnabled, originalContent, open, wordlist])

  React.useEffect(() => {
    if (!open) {
      setContent("")
      setHasChanges(false)
      loadedContentWordlistIdRef.current = null
      setDescription("")
      setTags([])
      setHasMetadataChanges(false)
    }
  }, [open])

  React.useEffect(() => {
    if (!wordlist || !open) return
    const nextTags = tags.join("\u0000")
    const originalTags = (wordlist.tags ?? []).join("\u0000")
    setHasMetadataChanges(
      description !== (wordlist.description ?? "") ||
      nextTags !== originalTags
    )
  }, [description, open, tags, wordlist])

  const handleEditorChange = React.useCallback((value: string) => {
    setContent(value)
    setHasChanges(value !== originalContent)
  }, [originalContent])

  const handleSaveDialog = React.useCallback(() => {
    if (!wordlist) return

    let saveCompletionCount = 0
    const expectedSaveCount = (hasMetadataChanges ? 1 : 0) + (canEditContent && hasChanges ? 1 : 0)
    const closeWhenComplete = () => {
      saveCompletionCount += 1
      if (saveCompletionCount === expectedSaveCount) {
        onOpenChange(false)
      }
    }

    if (hasMetadataChanges) {
      updateMetadataMutation.mutate(
        {
          id: wordlist.id,
          name: wordlist.name,
          description,
          tags,
          updateMask: "description,tags",
        },
        {
          onSuccess: () => {
            setHasMetadataChanges(false)
            closeWhenComplete()
          },
        }
      )
    }

    if (canEditContent && hasChanges) {
      updateMutation.mutate(
        { id: wordlist.id, content },
        {
          onSuccess: () => {
            setHasChanges(false)
            closeWhenComplete()
          },
        }
      )
    }
  }, [canEditContent, content, description, hasChanges, hasMetadataChanges, onOpenChange, tags, updateMetadataMutation, updateMutation, wordlist])

  const handleClose = React.useCallback(() => {
    if ((canEditContent && hasChanges) || hasMetadataChanges) {
      const confirmed = window.confirm(t("confirmClose"))
      if (!confirmed) return
    }
    onOpenChange(false)
  }, [canEditContent, hasChanges, hasMetadataChanges, onOpenChange, t])

  const lineCount = React.useMemo(() => countLineNumberedTextareaLines(content), [content])
  const isSaving = updateMutation.isPending
  const isSavingMetadata = updateMetadataMutation.isPending
  const fileHashShort = wordlist?.fileHash ? `${wordlist.fileHash.slice(0, 12)}...` : null

  return {
    content,
    hasChanges,
    description,
    tags,
    hasMetadataChanges,
    lineCount,
    isLoading,
    isSaving,
    isSavingMetadata,
    fileHashShort,
    setDescription,
    setTags,
    handleSaveDialog,
    handleEditorChange,
    handleClose,
  }
}
