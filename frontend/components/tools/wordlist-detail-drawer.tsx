"use client"

import { useCallback, useEffect, useState } from "react"
import { useTranslations } from "next-intl"

import { Trash2 } from "@/components/icons"
import {
  DetailDrawer,
  DetailDrawerTabs,
  DetailDrawerTabsContent,
  DetailDrawerTabsList,
  DetailDrawerTabsTrigger,
} from "@/components/shared/detail-drawer"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { Wordlist } from "@/types/wordlist.types"

import {
  WordlistEditEditor,
  WordlistEditFooter,
  WordlistEditMeta,
  WordlistEditMetadata,
  WordlistEditUnsavedNotice,
} from "./wordlist-edit-dialog-sections"
import { useWordlistEditDialogState } from "./wordlist-edit-dialog-state"
import { formatWordlistFileSize, formatWordlistUpdatedAt } from "./wordlist-formatters"

type WordlistDetailDrawerTab = "details" | "edit"

interface WordlistDetailDrawerProps {
  wordlist: Wordlist | null
  locale: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onDelete: (wordlist: Wordlist) => void
  canEditContent: (wordlist: Wordlist) => boolean
}

export function WordlistDetailDrawer({
  wordlist,
  locale,
  open,
  onOpenChange,
  onDelete,
  canEditContent,
}: WordlistDetailDrawerProps) {
  const tCommon = useTranslations("common")
  const t = useTranslations("pages.wordlists")
  const tEdit = useTranslations("tools.wordlists.editDialog")
  const [activeTab, setActiveTab] = useState<WordlistDetailDrawerTab>("details")
  const canEditSelectedContent = wordlist ? canEditContent(wordlist) : false
  const editState = useWordlistEditDialogState({
    wordlist,
    open,
    canEditContent: canEditSelectedContent,
    contentEnabled: activeTab === "edit",
    onOpenChange,
    t: tEdit,
  })
  const { handleClose } = editState

  useEffect(() => {
    if (open) setActiveTab("details")
  }, [open, wordlist?.id])

  const handleDrawerOpenChange = useCallback((nextOpen: boolean) => {
    if (nextOpen) {
      onOpenChange(true)
      return
    }

    handleClose()
  }, [handleClose, onOpenChange])

  return (
    <DetailDrawer
      open={open}
      onOpenChange={handleDrawerOpenChange}
      title={wordlist?.fileName ?? ""}
      description={wordlist?.description}
    >
      {wordlist ? (
        <DetailDrawerTabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as WordlistDetailDrawerTab)}
        >
          <DetailDrawerTabsList className="px-6" aria-label={tCommon("actions.details")}>
            <DetailDrawerTabsTrigger value="details">
              {tCommon("actions.details")}
            </DetailDrawerTabsTrigger>
            <DetailDrawerTabsTrigger value="edit">
              {tCommon("actions.edit")}
            </DetailDrawerTabsTrigger>
          </DetailDrawerTabsList>

          <DetailDrawerTabsContent value="details" keepMounted className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
              <div className="space-y-6">
                <div className="space-y-2">
                  <span className={textRole.metadataLabel}>{t("tags")}</span>
                  <div className="flex min-w-0 flex-wrap gap-1.5">
                    {wordlist.tags.length === 0 ? (
                      <span className={textRole.bodySubtle}>{t("tagEmpty")}</span>
                    ) : (
                      wordlist.tags.map((tag) => <Badge key={tag} variant="secondary">{tag}</Badge>)
                    )}
                  </div>
                </div>

                <dl className="grid grid-cols-1 gap-x-6 gap-y-5 border-y border-border/70 py-5 sm:grid-cols-2">
                  <WordlistDetailField label={t("rows")} value={wordlist.lineCount?.toLocaleString() ?? "-"} />
                  <WordlistDetailField label={t("size")} value={formatWordlistFileSize(wordlist.fileSize)} />
                  <WordlistDetailField label={t("updatedAt")} value={formatWordlistUpdatedAt(wordlist.updatedAt, locale)} />
                  <WordlistDetailField label={t("id")} value={String(wordlist.id)} />
                </dl>

                <div className="space-y-2">
                  <span className={textRole.metadataLabel}>{t("hash")}</span>
                  <div className="flex min-w-0 items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 font-mono text-xs text-muted-foreground">
                    <span className="min-w-0 flex-1 truncate">{wordlist.fileHash || "-"}</span>
                    {wordlist.fileHash ? (
                      <CopyButton
                        value={wordlist.fileHash}
                        copyLabel={tCommon("actions.copy")}
                        copiedLabel={t("hashCopied")}
                        size="icon-sm"
                        toastId={`wordlist-hash-${wordlist.id}`}
                      />
                    ) : null}
                  </div>
                </div>
              </div>
            </div>

            <div className="flex flex-wrap items-center justify-end gap-2 border-t px-6 py-4">
              <Button
                variant="outline"
                className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                onClick={() => onDelete(wordlist)}
              >
                <Trash2 aria-hidden="true" className="size-4" />
                {t("delete")}
              </Button>
            </div>
          </DetailDrawerTabsContent>

          <DetailDrawerTabsContent value="edit" className="min-h-0 flex-1 overflow-hidden">
            <div className="flex h-full min-h-0 flex-col">
              <div className="min-h-0 flex-1 overflow-hidden px-6 py-4">
                <div className="flex h-full min-h-0 flex-col gap-2">
                  <WordlistEditMetadata
                    t={tEdit}
                    description={editState.description}
                    tags={editState.tags}
                    onDescriptionChange={editState.setDescription}
                    onTagsChange={editState.setTags}
                  />

                  {canEditSelectedContent ? (
                    <>
                      <WordlistEditMeta
                        t={tEdit}
                        lineCount={editState.lineCount}
                        fileHashShort={editState.fileHashShort}
                        fileHashFull={wordlist.fileHash || null}
                      />
                      <WordlistEditEditor
                        t={tEdit}
                        isLoading={editState.isLoading}
                        content={editState.content}
                        onChange={editState.handleEditorChange}
                        readOnly={editState.isSaving}
                      />
                      {editState.hasChanges ? <WordlistEditUnsavedNotice t={tEdit} /> : null}
                    </>
                  ) : (
                    <div className="rounded-md border bg-muted/30 px-4 py-3 text-muted-foreground">
                      {tEdit("oversizedMetadataOnly")}
                    </div>
                  )}
                </div>
              </div>

              <WordlistEditFooter
                t={tEdit}
                isSaving={editState.isSaving}
                isSavingMetadata={editState.isSavingMetadata}
                hasChanges={editState.hasChanges}
                hasMetadataChanges={editState.hasMetadataChanges}
                onSaveDialog={editState.handleSaveDialog}
                canEditContent={canEditSelectedContent}
              />
            </div>
          </DetailDrawerTabsContent>
        </DetailDrawerTabs>
      ) : null}
    </DetailDrawer>
  )
}

function WordlistDetailField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 space-y-1">
      <dt className={textRole.metadataLabel}>{label}</dt>
      <dd className={cn("break-words tabular-nums", textRole.metadataValueStrong)}>{value}</dd>
    </div>
  )
}
