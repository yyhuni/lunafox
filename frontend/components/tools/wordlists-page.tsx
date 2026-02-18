"use client"

import { useState, useMemo } from "react"
import dynamic from "next/dynamic"
import { Search, Trash2, Pencil, FileText } from "@/components/icons"
import { useTranslations, useLocale } from "next-intl"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { useWordlists, useDeleteWordlist } from "@/hooks/use-wordlists"
import { WordlistUploadDialog } from "@/components/tools/wordlist-upload-dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { cn } from "@/lib/utils"
import type { Wordlist } from "@/types/wordlist.types"
import { MasterDetailSkeleton } from "@/components/ui/master-detail-skeleton"
import { getDateLocale } from "@/lib/date-utils"

const WordlistEditDialog = dynamic(
  () => import("@/components/tools/wordlist-edit-dialog").then((mod) => mod.WordlistEditDialog),
  { ssr: false }
)

export default function WordlistsPage() {
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [editingWordlist, setEditingWordlist] = useState<Wordlist | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [wordlistToDelete, setWordlistToDelete] = useState<Wordlist | null>(null)

  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tNav = useTranslations("navigation")
  const t = useTranslations("pages.wordlists")
  const locale = useLocale()

  const { data, isLoading } = useWordlists({ page: 1, pageSize: 1000 })
  const deleteMutation = useDeleteWordlist()

  // Filter wordlist list
  const filteredWordlists = useMemo(() => {
    if (!data?.results) return []
    if (!searchQuery.trim()) return data.results
    const query = searchQuery.toLowerCase()
    return data.results.filter(
      (w) =>
        w.name.toLowerCase().includes(query) ||
        w.description?.toLowerCase().includes(query)
    )
  }, [data?.results, searchQuery])

  // Selected wordlist
  const selectedWordlist = useMemo(() => {
    if (!selectedId || !data?.results) return null
    return data.results.find((w) => w.id === selectedId) || null
  }, [selectedId, data?.results])

  const handleEdit = (wordlist: Wordlist) => {
    setEditingWordlist(wordlist)
    setIsEditDialogOpen(true)
  }

  const handleDelete = (wordlist: Wordlist) => {
    setWordlistToDelete(wordlist)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (!wordlistToDelete) return
    deleteMutation.mutate(wordlistToDelete.id, {
      onSuccess: () => {
        if (selectedId === wordlistToDelete.id) {
          setSelectedId(null)
        }
        setDeleteDialogOpen(false)
        setWordlistToDelete(null)
      },
    })
  }

  const formatFileSize = (bytes?: number) => {
    if (bytes === undefined) return "-"
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  // Loading state
  if (isLoading) {
    return <MasterDetailSkeleton title={tNav("wordlists")} listItemCount={5} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Top: Title + Search + Upload button */}
      <div className="flex items-center justify-between gap-4 px-4 py-4 lg:px-6">
        <div className="flex items-center gap-3 shrink-0">
          <span className="px-1.5 py-0.5 text-[10px] font-mono bg-primary text-primary-foreground tracking-wider">
            WDL-01
          </span>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
        </div>
        <div className="flex items-center gap-2 flex-1 max-w-md">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              name="wordlistSearch"
              autoComplete="off"
              placeholder={t("searchPlaceholder")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-8"
            />
          </div>
        </div>
        <WordlistUploadDialog />
      </div>

      <Separator />

      {/* Main: Left list + Right details */}
      <div className="flex flex-1 min-h-0">
        {/* Left: Wordlist list */}
        <div className="w-72 lg:w-80 border-r flex flex-col">
          <div className="px-4 py-3 border-b">
            <h2 className="text-sm font-medium text-muted-foreground">
              {t("listTitle")} ({filteredWordlists.length})
            </h2>
          </div>
          <ScrollArea className="flex-1">
            {isLoading ? (
              <div className="p-4 text-sm text-muted-foreground">{t("loading")}</div>
            ) : filteredWordlists.length === 0 ? (
              <div className="p-4 text-sm text-muted-foreground">
                {searchQuery ? t("noMatch") : t("noData")}
              </div>
            ) : (
              <div className="p-2">
                {filteredWordlists.map((wordlist) => (
                  <button type="button"
                    key={wordlist.id}
                    onClick={() => setSelectedId(wordlist.id)}
                    className={cn(
                      "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                      selectedId === wordlist.id
                        ? "bg-primary/10 text-primary"
                        : "hover:bg-muted"
                    )}
                  >
                    <div className="font-medium text-sm truncate">
                      {wordlist.name}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                      {wordlist.lineCount?.toLocaleString() ?? "-"} {t("lines")} · {formatFileSize(wordlist.fileSize)}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </ScrollArea>
        </div>

        {/* Right: Wordlist details */}
        <div className="flex-1 flex flex-col min-w-0">
          {selectedWordlist ? (
            <>
              {/* Details header */}
              <div className="px-6 py-4 border-b">
                <div className="flex items-start gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 shrink-0">
                    <FileText className="h-5 w-5 text-primary" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <h2 className="text-lg font-semibold truncate">
                      {selectedWordlist.name}
                    </h2>
                    {selectedWordlist.description && (
                      <p className="text-sm text-muted-foreground mt-0.5">
                        {selectedWordlist.description}
                      </p>
                    )}
                  </div>
                </div>
              </div>

              {/* Details content */}
              <ScrollArea className="flex-1">
                <div className="p-6 space-y-6">
                  {/* Basic information */}
                  <div className="rounded-lg border">
                    <div className="grid grid-cols-2 divide-x">
                      <div className="p-4">
                        <div className="text-xs text-muted-foreground">{t("rows")}</div>
                        <div className="text-lg font-semibold mt-1">
                          {selectedWordlist.lineCount?.toLocaleString() ?? "-"}
                        </div>
                      </div>
                      <div className="p-4">
                        <div className="text-xs text-muted-foreground">{t("size")}</div>
                        <div className="text-lg font-semibold mt-1">
                          {formatFileSize(selectedWordlist.fileSize)}
                        </div>
                      </div>
                    </div>
                    <Separator />
                    <div className="p-4 space-y-3">
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">{t("id")}</span>
                        <span className="font-mono">{selectedWordlist.id}</span>
                      </div>
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">{t("updatedAt")}</span>
                        <span>
                          {new Date(selectedWordlist.updatedAt).toLocaleString(getDateLocale(locale))}
                        </span>
                      </div>
                      {selectedWordlist.fileHash && (
                        <div className="text-sm">
                          <span className="text-muted-foreground">{t("hash")}</span>
                          <div className="font-mono text-xs mt-1 break-all bg-muted p-2 rounded">
                            {selectedWordlist.fileHash}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </ScrollArea>

              {/* Action buttons */}
              <div className="px-6 py-4 border-t flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleEdit(selectedWordlist)}
                >
                  <Pencil className="h-4 w-4 mr-1.5" />
                  {t("editContent")}
                </Button>
                <div className="flex-1" />
                <Button
                  variant="outline"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleDelete(selectedWordlist)}
                  disabled={deleteMutation.isPending}
                >
                  <Trash2 className="h-4 w-4 mr-1.5" />
                  {t("delete")}
                </Button>
              </div>
            </>
          ) : (
            // Unselected state
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center text-muted-foreground">
                <FileText className="h-12 w-12 mx-auto mb-3 opacity-50" />
                <p className="text-sm">{t("selectHint")}</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Edit dialog */}
      {isEditDialogOpen ? (
        <WordlistEditDialog
          wordlist={editingWordlist}
          open={isEditDialogOpen}
          onOpenChange={setIsEditDialogOpen}
        />
      ) : null}

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteWordlistMessage", { name: wordlistToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmDelete} 
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? tConfirm("deleting") : tCommon("actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
