"use client"

import { useState } from "react"
import { useTranslations, useLocale } from "next-intl"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { IconEdit, IconTrash, IconFolder } from "@/components/icons"
import { AddCustomToolDialog } from "@/components/tools/config/add-custom-tool-dialog"
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
import { CategoryNameMap, type Tool } from "@/types/tool.types"
import { useTools, useDeleteTool } from "@/hooks/use-tools"
import { getDateLocale } from "@/lib/date-utils"

/**
 * Custom tools list component
 * Display and manage custom scan scripts and tools
 */
export function CustomToolsList() {
  const [editingTool, setEditingTool] = useState<Tool | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [toolToDelete, setToolToDelete] = useState<Tool | null>(null)
  
  // Internationalization
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tConfig = useTranslations("tools.config")
  const tColumns = useTranslations("columns")
  const locale = useLocale()
  
  // Get tool list (only custom tools)
  const { data, isLoading, error } = useTools({
    page: 1,
    pageSize: 100,
  })

  // Filter out custom tools
  const customTools = (data?.tools || []).filter((tool: Tool) => tool.type === 'custom')
  
  // Delete tool mutation
  const deleteTool = useDeleteTool()

  const handleEditTool = (tool: Tool) => {
    setEditingTool(tool)
    setIsEditDialogOpen(true)
  }

  const handleEditDialogClose = (open: boolean) => {
    setIsEditDialogOpen(open)
    if (!open) {
      setEditingTool(null)
    }
  }

  const handleDeleteTool = (toolId: number) => {
    const tool = customTools.find((t: Tool) => t.id === toolId)
    if (!tool) return
    setToolToDelete(tool)
  }

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
        {customTools.map((tool: Tool) => (
          <Card key={tool.id} className="flex flex-col h-full hover:shadow-lg transition-shadow">
            <CardHeader>
              <CardTitle className="text-lg truncate" title={tool.name}>{tool.name}</CardTitle>
              <CardDescription className="line-clamp-2" title={tool.description || tCommon("status.noData")}>
                {tool.description || tCommon("status.noData")}
              </CardDescription>
              
              {/* Category tags */}
              <div className="flex flex-wrap gap-1 mt-2">
                {tool.categoryNames && tool.categoryNames.length > 0 ? (
                  <div 
                    className="flex flex-wrap gap-1"
                    title={tool.categoryNames.map(c => CategoryNameMap[c] || c).join(', ')}
                  >
                    {tool.categoryNames.slice(0, 3).map((category: string) => (
                      <Badge key={category} variant="secondary" className="text-xs whitespace-nowrap">
                        {CategoryNameMap[category] || category}
                      </Badge>
                    ))}
                    {tool.categoryNames.length > 3 && (
                      <Badge variant="secondary" className="text-xs">
                        +{tool.categoryNames.length - 3}
                      </Badge>
                    )}
                  </div>
                ) : (
                  <Badge variant="outline" className="text-xs text-muted-foreground">
                    {tConfig("uncategorized")}
                  </Badge>
                )}
              </div>
            </CardHeader>
            <CardContent className="flex-1">
              <div className="space-y-4">
                {/* Tool directory */}
                <div className="bg-muted rounded-md p-3">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                    <IconFolder className="h-4 w-4" />
                    <span>{tConfig("directory")}</span>
                  </div>
                  <code 
                    className="text-sm font-mono break-all line-clamp-2" 
                    title={tool.directory}
                  >
                    {tool.directory}
                  </code>
                </div>

                {/* Last updated time */}
                <div className="text-sm text-muted-foreground">
                  {tColumns("common.updatedAt")}: {new Date(tool.updatedAt).toLocaleDateString(getDateLocale(locale))}
                </div>
              </div>
            </CardContent>
            <CardFooter className="flex gap-2 pt-0">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => handleEditTool(tool)}
              >
                <IconEdit className="h-4 w-4" />
                {tCommon("actions.edit")}
              </Button>
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => handleDeleteTool(tool.id)}
              >
                <IconTrash className="h-4 w-4" />
                {tCommon("actions.delete")}
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>

      {/* Empty state */}
      {customTools.length === 0 && (
        <div className="text-center py-12">
          <p className="text-muted-foreground">{tConfig("noCustomTools")}</p>
        </div>
      )}

     

      {/* Edit tool dialog */}
      <AddCustomToolDialog 
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
              {tConfirm("deleteCustomToolMessage", { name: toolToDelete?.name ?? "" })}
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
