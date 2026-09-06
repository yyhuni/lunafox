"use client"

import { useState } from "react"
import dynamic from "next/dynamic"
import { useTranslations, useLocale } from "next-intl"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { IconEdit, IconTrash, semanticIcons } from "@/components/icons"
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
import { CategoryNameMap, type Tool } from "@/types/tool.types"
import { useTools, useDeleteTool } from "@/hooks/use-tools"
import {
  deferredInteractionUnmountDelayMs,
  useDeferredInteractionMount,
} from "@/hooks/use-deferred-interaction-mount"
import { getDateLocale } from "@/lib/date-utils"
import { normalizeError } from "@/lib/errors/normalize-error"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const AddCustomToolDialog = dynamic(
  () =>
    import("@/components/tools/config/add-custom-tool-dialog").then((mod) => ({
      default: mod.AddCustomToolDialog,
    })),
  { ssr: false, loading: () => null }
)

/**
 * Custom tools list component
 * Display and manage custom scan scripts and tools
 */
export function CustomToolsList() {
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
  const tColumns = useTranslations("columns")
  const locale = useLocale()
  
  // Get tool list (only custom tools)
  const { data, isLoading, error, refetch } = useTools({
    page: 1,
    pageSize: 100,
  })
  const isInitialLoading = isLoading && !data

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
      owner="custom-tools-list-content"
      isLoading={isInitialLoading}
      skeleton={<CardGridSkeleton cards={4} />}
      className="flex flex-col gap-4"
    >
      {/* Tool list */}
      <div className="gap-6 grid grid-cols-1 lg:grid-cols-3 md:grid-cols-2 xl:grid-cols-4">
        {customTools.map((tool: Tool) => (
          <Card key={tool.id} className="flex flex-col h-full hover:shadow-lg transition-shadow">
            <CardHeader>
              <CardTitle className={cn(textRole.panelTitle, "truncate")} title={tool.name}>{tool.name}</CardTitle>
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
                  <Badge variant="outline" className="text-muted-foreground text-xs">
                    {tConfig("uncategorized")}
                  </Badge>
                )}
              </div>
            </CardHeader>
            <CardContent className="flex-1">
              <div className="space-y-4">
                {/* Tool directory */}
                <div className="bg-muted p-3 rounded-md">
                  <div className="flex gap-2 items-center mb-1 text-muted-foreground text-sm">
                    <semanticIcons.concept.directory className="h-4 w-4" />
                    <span>{tConfig("directory")}</span>
                  </div>
                  <code 
                    className="break-all font-mono line-clamp-2 text-sm" 
                    title={tool.directory}
                  >
                    {tool.directory}
                  </code>
                </div>

                {/* Last updated time */}
                <div className="text-muted-foreground text-sm">
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
        <div className="py-12 text-center">
          <p className="text-muted-foreground">{tConfig("noCustomTools")}</p>
        </div>
      )}

     

      {/* Edit tool dialog */}
      {shouldMountEditDialog ? (
        <AddCustomToolDialog
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
              {tConfirm("deleteCustomToolMessage", { name: toolToDelete?.name ?? "" })}
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
