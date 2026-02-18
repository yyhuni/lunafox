"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { ToolCard } from "@/components/tools/config/tool-card"
import { AddToolDialog } from "@/components/tools/config/add-tool-dialog"
import { useTools, useDeleteTool } from "@/hooks/use-tools"
import type { Tool } from "@/types/tool.types"
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
import { LoadingSpinner } from "@/components/loading-spinner"
import { CardGridSkeleton } from "@/components/ui/card-grid-skeleton"

/**
 * Open source tools list component
 * Display and manage open source scan tools
 */
export function OpensourceToolsList() {
  const [checkingToolId, setCheckingToolId] = useState<number | null>(null)
  const [editingTool, setEditingTool] = useState<Tool | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [toolToDelete, setToolToDelete] = useState<Tool | null>(null)
  
  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tConfig = useTranslations("tools.config")
  
  // Get tool list (only open source tools)
  const { data, isLoading, error } = useTools({
    page: 1,
    pageSize: 100,
  })

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

  // Loading state
  if (isLoading) {
    return <CardGridSkeleton cards={4} />
  }

  // Error state
  if (error) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <p className="text-destructive">{tCommon("status.error")}: {error.message}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Tool list */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
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
        <div className="text-center py-12">
          <p className="text-muted-foreground">{tConfig("noTools")}</p>
        </div>
      )}

      {/* Edit tool dialog */}
      <AddToolDialog 
        tool={editingTool || undefined}
        open={isEditDialogOpen}
        onOpenChange={handleEditDialogClose}
      />

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
            <AlertDialogCancel disabled={deleteTool.isPending}>{tCommon("actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction 
              onClick={confirmDelete} 
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteTool.isPending}
            >
              {deleteTool.isPending ? (
                <>
                  <LoadingSpinner/>
                  {tConfirm("deleting")}
                </>
              ) : (
                tCommon("actions.delete")
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
