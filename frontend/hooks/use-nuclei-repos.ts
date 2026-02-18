/**
 * Nuclei template warehouse related Hooks
 */

import { useQuery } from "@tanstack/react-query"
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { nucleiRepoApi } from "../services/nuclei-repo.service"
import type { NucleiTemplateTreeNode, NucleiTemplateContent } from "@/types/nuclei.types"

export const nucleiRepoKeys = {
  repos: ["nuclei-repos"] as const,
  repo: (repoId: number | null) => ["nuclei-repos", repoId] as const,
  tree: (repoId: number | null) => ["nuclei-repo-tree", repoId] as const,
  content: (repoId: number | null, path: string | null) =>
    ["nuclei-repo-content", repoId, path] as const,
}

// ==================== Warehouse CRUD ====================

export interface NucleiRepo {
  id: number
  name: string
  repoUrl: string
  localPath: string
  commitHash: string | null
  lastSyncedAt: string | null
  createdAt: string
  updatedAt: string
}

/** Get the warehouse list */
export function useNucleiRepos() {
  return useQuery<NucleiRepo[]>({
    queryKey: nucleiRepoKeys.repos,
    queryFn: nucleiRepoApi.listRepos,
  })
}

/** Get details of a single warehouse */
export function useNucleiRepo(repoId: number | null) {
  return useQuery<NucleiRepo>({
    queryKey: nucleiRepoKeys.repo(repoId),
    queryFn: () => nucleiRepoApi.getRepo(repoId!),
    enabled: !!repoId,
  })
}

/** Create warehouse */
export function useCreateNucleiRepo() {
  return useResourceMutation({
    mutationFn: nucleiRepoApi.createRepo,
    invalidate: [{ queryKey: nucleiRepoKeys.repos }],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiRepo.create.success')
    },
    errorFallbackKey: 'toast.nucleiRepo.create.error',
  })
}

/** Update warehouse */
export function useUpdateNucleiRepo() {
  return useResourceMutation({
    mutationFn: (data: {
      id: number
      repoUrl?: string
    }) => nucleiRepoApi.updateRepo(data.id, { repoUrl: data.repoUrl }),
    invalidate: [
      { queryKey: nucleiRepoKeys.repos },
      ({ variables }) => ({ queryKey: nucleiRepoKeys.repo(variables.id) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiRepo.update.success')
    },
    errorFallbackKey: 'toast.nucleiRepo.update.error',
  })
}

/** Delete warehouse */
export function useDeleteNucleiRepo() {
  return useResourceMutation({
    mutationFn: nucleiRepoApi.deleteRepo,
    invalidate: [{ queryKey: nucleiRepoKeys.repos }],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiRepo.delete.success')
    },
    errorFallbackKey: 'toast.nucleiRepo.delete.error',
  })
}

// ==================== Git synchronization ====================

/** Refresh the warehouse (Git clone/pull) */
export function useRefreshNucleiRepo() {
  return useResourceMutation({
    mutationFn: nucleiRepoApi.refreshRepo,
    invalidate: [
      { queryKey: nucleiRepoKeys.repos },
      ({ variables }) => ({ queryKey: nucleiRepoKeys.repo(variables) }),
      ({ variables }) => ({ queryKey: nucleiRepoKeys.tree(variables) }),
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.nucleiRepo.sync.success')
    },
    errorFallbackKey: 'toast.nucleiRepo.sync.error',
  })
}

// ==================== Template read only ====================

/** Get the warehouse template directory tree */
export function useNucleiRepoTree(repoId: number | null) {
  return useQuery({
    queryKey: nucleiRepoKeys.tree(repoId),
    queryFn: async () => {
      const res = await nucleiRepoApi.getTemplateTree(repoId!)
      return (res.roots ?? []) as NucleiTemplateTreeNode[]
    },
    enabled: !!repoId,
  })
}

/** Get template file content */
export function useNucleiRepoContent(repoId: number | null, path: string | null) {
  return useQuery<NucleiTemplateContent>({
    queryKey: nucleiRepoKeys.content(repoId, path),
    queryFn: () => nucleiRepoApi.getTemplateContent(repoId!, path!),
    enabled: !!repoId && !!path,
  })
}
