"use client"

import Link from "next/link"
import { useState, useMemo, type FormEvent } from "react"
import { GitBranch, Search, RefreshCw, Settings, Trash2, FolderOpen, Plus } from "@/components/icons"
import { useTranslations, useLocale } from "next-intl"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
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
import {
  useNucleiRepos,
  useCreateNucleiRepo,
  useDeleteNucleiRepo,
  useRefreshNucleiRepo,
  useUpdateNucleiRepo,
  type NucleiRepo,
} from "@/hooks/use-nuclei-repos"
import { cn } from "@/lib/utils"
import { MasterDetailSkeleton } from "@/components/ui/master-detail-skeleton"
import { getDateLocale } from "@/lib/date-utils"

/** Format time display */
function formatDateTime(isoString: string | null, locale: string) {
  if (!isoString) return "-"
  try {
    return new Date(isoString).toLocaleString(getDateLocale(locale))
  } catch {
    return isoString
  }
}

export default function NucleiReposPage() {
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [searchQuery, setSearchQuery] = useState("")
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [newName, setNewName] = useState("")
  const [newRepoUrl, setNewRepoUrl] = useState("")

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingRepo, setEditingRepo] = useState<NucleiRepo | null>(null)
  const [editRepoUrl, setEditRepoUrl] = useState("")
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [repoToDelete, setRepoToDelete] = useState<NucleiRepo | null>(null)

  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const t = useTranslations("pages.nuclei")
  const locale = useLocale()

  // API Hooks
  const { data: repos, isLoading, isError } = useNucleiRepos()
  const createMutation = useCreateNucleiRepo()
  const deleteMutation = useDeleteNucleiRepo()
  const refreshMutation = useRefreshNucleiRepo()
  const updateMutation = useUpdateNucleiRepo()

  // Filter repository list
  const filteredRepos = useMemo(() => {
    if (!repos) return []
    if (!searchQuery.trim()) return repos
    const query = searchQuery.toLowerCase()
    return repos.filter(
      (r) =>
        r.name.toLowerCase().includes(query) ||
        r.repoUrl?.toLowerCase().includes(query)
    )
  }, [repos, searchQuery])

  // Selected repository
  const selectedRepo = useMemo(() => {
    if (!selectedId || !repos) return null
    return repos.find((r) => r.id === selectedId) || null
  }, [selectedId, repos])

  const resetCreateForm = () => {
    setNewName("")
    setNewRepoUrl("")
  }

  const resetEditForm = () => {
    setEditingRepo(null)
    setEditRepoUrl("")
  }

  const handleCreateSubmit = (event: FormEvent) => {
    event.preventDefault()
    const name = newName.trim()
    const repoUrl = newRepoUrl.trim()
    if (!name || !repoUrl) return

    createMutation.mutate(
      { name, repoUrl },
      {
        onSuccess: () => {
          resetCreateForm()
          setCreateDialogOpen(false)
        },
      }
    )
  }

  const handleRefresh = (repoId: number) => {
    refreshMutation.mutate(repoId)
  }

  const handleDelete = (repo: NucleiRepo) => {
    setRepoToDelete(repo)
    setDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (!repoToDelete) return
    deleteMutation.mutate(repoToDelete.id, {
      onSuccess: () => {
        if (selectedId === repoToDelete.id) {
          setSelectedId(null)
        }
        setDeleteDialogOpen(false)
        setRepoToDelete(null)
      },
    })
  }

  const openEditDialog = (repo: NucleiRepo) => {
    setEditingRepo(repo)
    setEditRepoUrl(repo.repoUrl || "")
    setEditDialogOpen(true)
  }

  const handleEditSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (!editingRepo) return
    const repoUrl = editRepoUrl.trim()
    if (!repoUrl) return

    updateMutation.mutate(
      { id: editingRepo.id, repoUrl },
      {
        onSuccess: () => {
          resetEditForm()
          setEditDialogOpen(false)
        },
      }
    )
  }

  // Loading state
  if (isLoading) {
    return <MasterDetailSkeleton title={t("title")} listItemCount={3} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Top: Title + Search + Add button */}
      <div className="flex items-center justify-between gap-4 px-4 py-4 lg:px-6">
        <div className="flex items-center gap-3 shrink-0">
          <span className="px-1.5 py-0.5 text-[10px] font-mono bg-primary text-primary-foreground tracking-wider">
            NCL-01
          </span>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
        </div>
        <div className="flex items-center gap-2 flex-1 max-w-md">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              name="repoSearch"
              autoComplete="off"
              placeholder={t("searchPlaceholder")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-8"
            />
          </div>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("addRepo")}
        </Button>
      </div>

      <Separator />

      {/* Main: Left list + Right details */}
      <div className="flex flex-1 min-h-0">
        {/* Left: Repository list */}
        <div className="w-72 lg:w-80 border-r flex flex-col">
          <div className="px-4 py-3 border-b">
            <h2 className="text-sm font-medium text-muted-foreground">
              {t("listTitle")} ({filteredRepos.length})
            </h2>
          </div>
          <ScrollArea className="flex-1">
            {isLoading ? (
              <div className="p-4 text-sm text-muted-foreground">{t("loading")}</div>
            ) : isError ? (
              <div className="p-4 text-sm text-red-500">{t("loadFailed")}</div>
            ) : filteredRepos.length === 0 ? (
              <div className="p-4 text-sm text-muted-foreground">
                {searchQuery ? t("noMatch") : t("noData")}
              </div>
            ) : (
              <div className="p-2">
                {filteredRepos.map((repo) => (
                  <button type="button"
                    key={repo.id}
                    onClick={() => setSelectedId(repo.id)}
                    className={cn(
                      "w-full text-left rounded-lg px-3 py-2.5 transition-colors",
                      selectedId === repo.id
                        ? "bg-primary/10 text-primary"
                        : "hover:bg-muted"
                    )}
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-sm truncate flex-1">
                        {repo.name}
                      </span>
                      {repo.lastSyncedAt ? (
                        <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 text-xs shrink-0">
                          {t("synced")}
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="text-xs shrink-0">
                          {t("notSynced")}
                        </Badge>
                      )}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5 truncate">
                      {repo.lastSyncedAt
                        ? `${t("syncedAt")} ${formatDateTime(repo.lastSyncedAt, locale)}`
                        : t("notSyncedYet")}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </ScrollArea>
        </div>

        {/* Right: Repository details */}
        <div className="flex-1 flex flex-col min-w-0">
          {selectedRepo ? (
            <>
              {/* Details header */}
              <div className="px-6 py-4 border-b">
                <div className="flex items-start gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 shrink-0">
                    <GitBranch className="h-5 w-5 text-primary" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <h2 className="text-lg font-semibold truncate">
                        {selectedRepo.name}
                      </h2>
                      {selectedRepo.lastSyncedAt ? (
                        <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                          {t("synced")}
                        </Badge>
                      ) : (
                        <Badge variant="outline">{t("notSynced")}</Badge>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {/* Details content */}
              <ScrollArea className="flex-1">
                <div className="p-6 space-y-6">
                  {/* Statistics information */}
                  <div className="rounded-lg border">
                    <div className="grid grid-cols-2 divide-x">
                      <div className="p-4">
                        <div className="text-xs text-muted-foreground">{t("status")}</div>
                        <div className="text-lg font-semibold mt-1">
                          {selectedRepo.lastSyncedAt ? t("synced") : t("notSynced")}
                        </div>
                      </div>
                      <div className="p-4">
                        <div className="text-xs text-muted-foreground">{t("lastSync")}</div>
                        <div className="text-lg font-semibold mt-1">
                          {selectedRepo.lastSyncedAt
                            ? formatDateTime(selectedRepo.lastSyncedAt, locale)
                            : "-"}
                        </div>
                      </div>
                    </div>
                    <Separator />
                    <div className="p-4 space-y-3">
                      <div className="text-sm">
                        <span className="text-muted-foreground">{t("gitUrl")}</span>
                        <div className="font-mono text-xs mt-1 break-all bg-muted p-2 rounded">
                          {selectedRepo.repoUrl}
                        </div>
                      </div>
                      {selectedRepo.localPath && (
                        <div className="text-sm">
                          <span className="text-muted-foreground">{t("localPath")}</span>
                          <div className="font-mono text-xs mt-1 break-all bg-muted p-2 rounded">
                            {selectedRepo.localPath}
                          </div>
                        </div>
                      )}
                      {selectedRepo.commitHash && (
                        <div className="text-sm">
                          <span className="text-muted-foreground">{t("commit")}</span>
                          <div className="font-mono text-xs mt-1 break-all bg-muted p-2 rounded">
                            {selectedRepo.commitHash}
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
                  onClick={() => handleRefresh(selectedRepo.id)}
                  disabled={refreshMutation.isPending}
                >
                  <RefreshCw className={cn("h-4 w-4 mr-1.5", refreshMutation.isPending && "animate-spin")} />
                  {refreshMutation.isPending ? t("syncing") : t("syncRepo")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => openEditDialog(selectedRepo)}
                >
                  <Settings className="h-4 w-4 mr-1.5" />
                  {t("editConfig")}
                </Button>
                <Link href={`/tools/nuclei/${selectedRepo.id}/`}>
                  <Button size="sm">
                    <FolderOpen className="h-4 w-4 mr-1.5" />
                    {t("manageTemplates")}
                  </Button>
                </Link>
                <div className="flex-1" />
                <Button
                  variant="outline"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleDelete(selectedRepo)}
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
                <GitBranch className="h-12 w-12 mx-auto mb-3 opacity-50" />
                <p className="text-sm">{t("selectHint")}</p>
              </div>
            </div>
          )}
        </div>
      </div>

      <Dialog open={createDialogOpen} onOpenChange={(open) => {
        setCreateDialogOpen(open)
        if (!open) {
          resetCreateForm()
        }
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("addDialog.title")}</DialogTitle>
          </DialogHeader>
          <form className="space-y-4" onSubmit={handleCreateSubmit}>
            <div className="space-y-2">
              <Label htmlFor="nuclei-repo-name">{t("addDialog.repoName")}</Label>
              <Input
                id="nuclei-repo-name"
                type="text"
                name="repositoryName"
                autoComplete="off"
                placeholder={t("addDialog.repoNamePlaceholder")}
                value={newName}
                onChange={(event) => setNewName(event.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="nuclei-repo-url">{t("addDialog.gitUrl")}</Label>
              <Input
                id="nuclei-repo-url"
                type="url"
                name="repositoryUrl"
                autoComplete="url"
                inputMode="url"
                placeholder={t("addDialog.gitUrlPlaceholder")}
                value={newRepoUrl}
                onChange={(event) => setNewRepoUrl(event.target.value)}
              />
            </div>

            {/* Currently only public repositories are supported, no authentication method and credential configuration provided here */}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setCreateDialogOpen(false)}
                disabled={createMutation.isPending}
              >
                {t("addDialog.cancel")}
              </Button>
              <Button
                type="submit"
                disabled={!newName.trim() || !newRepoUrl.trim() || createMutation.isPending}
              >
                {createMutation.isPending ? t("addDialog.creating") : t("addDialog.confirm")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog
        open={editDialogOpen}
        onOpenChange={(open) => {
          setEditDialogOpen(open)
          if (!open) {
            resetEditForm()
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("editDialog.title")}</DialogTitle>
          </DialogHeader>
          <form className="space-y-4" onSubmit={handleEditSubmit}>
            <div className="space-y-1 text-sm text-muted-foreground">
              <span className="font-medium">{t("editDialog.repoName")}</span>
              <span>{editingRepo?.name ?? ""}</span>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-nuclei-repo-url">{t("editDialog.gitUrl")}</Label>
              <Input
                id="edit-nuclei-repo-url"
                type="url"
                name="repositoryUrl"
                autoComplete="url"
                inputMode="url"
                placeholder={t("editDialog.gitUrlPlaceholder")}
                value={editRepoUrl}
                onChange={(event) => setEditRepoUrl(event.target.value)}
              />
            </div>

            {/* Editing also no longer supports configuring authentication method/credentials, only allows modifying Git address */}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setEditDialogOpen(false)}
                disabled={updateMutation.isPending}
              >
                {t("editDialog.cancel")}
              </Button>
              <Button
                type="submit"
                disabled={!editRepoUrl.trim() || updateMutation.isPending}
              >
                {updateMutation.isPending ? t("editDialog.saving") : t("editDialog.save")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteNucleiRepoMessage", { name: repoToDelete?.name ?? "" })}
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
