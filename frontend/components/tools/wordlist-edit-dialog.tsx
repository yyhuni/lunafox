"use client"

import { useTranslations } from "next-intl"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { editorDialogPanelClassName } from "@/lib/ui/overlay-styles"
import { compactSurfaceClassNames } from "@/lib/ui/compact-surface-contract"
import { cn } from "@/lib/utils"
import { useWordlistEditDialogState } from "@/components/tools/wordlist-edit-dialog-state"
import {
  WordlistEditHeader,
  WordlistEditMetadata,
  WordlistEditMeta,
  WordlistEditEditor,
  WordlistEditFooter,
} from "@/components/tools/wordlist-edit-dialog-sections"
import type { Wordlist } from "@/types/wordlist.types"

interface WordlistEditDialogProps {
  wordlist: Wordlist | null
  open: boolean
  canEditContent: boolean
  onOpenChange: (open: boolean) => void
}

export function WordlistEditDialog({
  wordlist,
  open,
  canEditContent,
  onOpenChange,
}: WordlistEditDialogProps) {
  const t = useTranslations("tools.wordlists.editDialog")

  const {
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
  } = useWordlistEditDialogState({
    wordlist,
    open,
    canEditContent,
    contentEnabled: true,
    onOpenChange,
    t,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn(editorDialogPanelClassName, "sm:max-w-6xl")}>
        <div className="flex flex-col h-full">
          <WordlistEditHeader t={t} name={wordlist?.fileName || ""} />

          <div className="flex-1 overflow-hidden px-4 py-3">
            <div className="flex flex-col gap-2 h-full">
              <WordlistEditMetadata
                density="compact"
                t={t}
                description={description}
                tags={tags}
                onDescriptionChange={setDescription}
                onTagsChange={setTags}
              />

              {canEditContent ? (
                <>
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
                </>
              ) : (
                <div className={cn(compactSurfaceClassNames.mutedInfo, "text-muted-foreground", "text-xs leading-4")}>
                  {t("oversizedMetadataOnly")}
                </div>
              )}
            </div>
          </div>

          <WordlistEditFooter
            density="compact"
            t={t}
            isSaving={isSaving}
            isSavingMetadata={isSavingMetadata}
            hasChanges={hasChanges}
            hasMetadataChanges={hasMetadataChanges}
            onSaveDialog={handleSaveDialog}
            canEditContent={canEditContent}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
