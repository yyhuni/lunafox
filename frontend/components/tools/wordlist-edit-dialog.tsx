"use client"

import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { useWordlistEditDialogState } from "@/components/tools/wordlist-edit-dialog-state"
import {
  WordlistEditHeader,
  WordlistEditMeta,
  WordlistEditEditor,
  WordlistEditUnsavedNotice,
  WordlistEditFooter,
} from "@/components/tools/wordlist-edit-dialog-sections"
import type { Wordlist } from "@/types/wordlist.types"

interface WordlistEditDialogProps {
  wordlist: Wordlist | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WordlistEditDialog({
  wordlist,
  open,
  onOpenChange,
}: WordlistEditDialogProps) {
  const t = useTranslations("tools.wordlists.editDialog")

  const {
    content,
    hasChanges,
    lineCount,
    isLoading,
    isSaving,
    fileHashShort,
    handleEditorChange,
    handleSave,
    handleClose,
  } = useWordlistEditDialogState({
    wordlist,
    open,
    onOpenChange,
    t,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-6xl max-w-[calc(100%-2rem)] h-[90vh] flex flex-col p-0">
        <div className="flex flex-col h-full">
          <WordlistEditHeader t={t} name={wordlist?.name || ""} />

          <div className="flex-1 overflow-hidden px-6 py-4">
            <div className="flex flex-col h-full gap-2">
              <WordlistEditMeta
                t={t}
                lineCount={lineCount}
                fileHashShort={fileHashShort}
                fileHashFull={wordlist?.fileHash || null}
              />

              <WordlistEditEditor
                t={t}
                isLoading={isLoading}
                content={content}
                onChange={handleEditorChange}
                readOnly={isSaving}
              />

              {hasChanges && <WordlistEditUnsavedNotice t={t} />}
            </div>
          </div>

          <WordlistEditFooter
            t={t}
            isSaving={isSaving}
            hasChanges={hasChanges}
            onCancel={handleClose}
            onSave={handleSave}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
