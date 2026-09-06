"use client"

import { useState } from "react"
import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { ToolCard } from "@/components/tools/config/tool-card"
import { useTools, useDeleteTool } from "@/hooks/use-tools"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"
import type { Tool } from "@/types/tool.types"
import {
  AlertDialog,
  AlertDialogClose, AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Spinner } from "@/components/shared/loading/spinner"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { CardGridSkeleton } from "@/components/shared/loading/card-grid-skeleton"
import { normalizeError } from "@/lib/errors/normalize-error"

const AddToolDialog = dynamic(
  () =>
    import("@/components/tools/config/add-tool-dialog").then((mod) => ({
      default: mod.AddToolDialog,
    })),
  { ssr: false, loading: () => null }
)

/**
 * Open source tools list component
 * Display and manage open source scan tools
 */
export function OpensourceToolsList() {
  const [checkingToolId, setCheckingToolId] = useState<number | null>(null)
  const [editingTool, setEditingTool] = useState<Tool | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [toolToDelete, setToolToDelete] = useState<Tool | null>(null)
  const shouldMountEditDialog = useDeferredInteractionMount(isEditDialogOpen, {
    unmountDelayMs: deferredInteractionUnmountDelayMs,
  })
  
  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tConfig = useTranslations("tools.config")
  
  // Get tool list (only open source tools)
  const { data, isLoading, error, refetch } = useTools({
    page: 1,
    pageSize: 100,
  })
  const isInitialLoading = isLoading && !data

  // Filter out open source tools
  const tools = (data?.tools || []).filter((tool: Tool) => tool.type === 'opensource')
  
  // Delete tool mutation
  const deleteTool = useDeleteTool()

  // Handle check update
  const handleCheckUpdate = async (toolId: number) => {
    try {
      setCheckingToolId(toolId)
      
      // TODO: Call backend API to check update
      // Simulate async operation
      await new Promise(resolve => setTimeout(resolve, 2000))
      
    } catch {
    } finally {
      setCheckingToolId(null)
    }
  }

  // Handle edit tool
  const handleEditTool = (tool: Tool) => {
    setEditingTool(tool)
    setIsEditDialogOpen(true)
  }

  // Edit dialog close callback
  const handleEditDialogClose = (open: boolean) => {
    setIsEditDialogOpen(open)
    if (!open) {
      setEditingTool(null)
    }
  }

  // Handle delete tool
  const handleDeleteTool = (toolId: number) => {
    const tool = tools.find((t: Tool) => t.id === toolId)
    if (!tool) return
    setToolToDelete(tool)
  }

  // Confirm delete tool
  const confirmDelete = async () => {
    if (!toolToDelete) return
    
    try {
      await deleteTool.mutateAsync(toolToDelete.id)
      // Close dialog after successful deletion
      setToolToDelete(null)
    } catch {
      // Error already handled in hook
    }
  }

  // Error state
  if (error) {
    return (
      <AppErrorState
        error={normalizeError(error, { notFoundKind: "unexpected-error" })}
        onRetry={refetch}
        variant="section"
        className="min-h-96"
      />
    )
  }

  return (
    <ContentHandoff
      owner="opensource-tools-list-content"
      isLoading={isInitialLoading}
      skeleton={<CardGridSkeleton cards={4} />}
      className="flex flex-col gap-4"
    >
      {/* Tool list */}
      <div className="gap-6 grid grid-cols-1 lg:grid-cols-3 md:grid-cols-2 xl:grid-cols-4">
        {tools.map((tool: Tool) => (
          <ToolCard 
            key={tool.id} 
            tool={tool}
            onCheckUpdate={handleCheckUpdate}
            onEdit={handleEditTool}
            onDelete={handleDeleteTool}
            isChecking={checkingToolId === tool.id}
          />
        ))}
      </div>
      
      {/* Empty state */}
      {tools.length === 0 && (
        <div className="py-12 text-center">
          <p className="text-muted-foreground">{tConfig("noTools")}</p>
        </div>
      )}

      {/* Edit tool dialog */}
      {shouldMountEditDialog ? (
        <AddToolDialog
          tool={editingTool || undefined}
          open={isEditDialogOpen}
          onOpenChange={handleEditDialogClose}
        />
      ) : null}

      {/* Delete confirmation dialog */}
      <AlertDialog open={!!toolToDelete} onOpenChange={(open) => !open && setToolToDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tConfirm("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tConfirm("deleteToolMessage", { name: toolToDelete?.name ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose disabled={deleteTool.isPending} variant="outline">{tCommon("actions.cancel")}</AlertDialogClose>
            <AlertDialogClose 
              onClick={confirmDelete} 
              className="bg-destructive hover:bg-destructive/90 text-destructive-foreground"
              disabled={deleteTool.isPending}
            >
              {deleteTool.isPending ? (
                <>
                  <Spinner />
                  {tConfirm("deleting")}
                </>
              ) : (
                tCommon("actions.delete")
              )}
            </AlertDialogClose>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </ContentHandoff>
  )
}
