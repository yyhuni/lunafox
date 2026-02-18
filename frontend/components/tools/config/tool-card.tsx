"use client"

import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { IconBrandGithub, IconRefresh, IconEdit, IconTrash } from "@/components/icons"
import type { Tool } from "@/types/tool.types"
import { CategoryNameMap } from "@/types/tool.types"
import Link from "next/link"
import { useTranslations } from "next-intl"

interface ToolCardProps {
  tool: Tool
  onCheckUpdate?: (toolId: number) => void | Promise<void>
  onEdit?: (tool: Tool) => void  // Edit tool callback
  onDelete?: (toolId: number) => void  // Delete tool callback
  isChecking?: boolean  // Whether checking for updates
}

/**
 * Tool card component
 * Display information for a single scan tool
 */
export function ToolCard({ tool, onCheckUpdate, onEdit, onDelete, isChecking = false }: ToolCardProps) {
  const t = useTranslations("tools.config")
  // Generate capitalized displayName from name
  const displayName = tool.name.charAt(0).toUpperCase() + tool.name.slice(1)
  
  return (
    <Card className="flex flex-col h-full hover:shadow-lg transition-shadow">
      <CardHeader className=" space-y-2">
        {/* Tool name */}
        <CardTitle 
          className="text-center text-2xl font-bold truncate px-2" 
          title={displayName}
        >
          {displayName}
        </CardTitle>

        {/* GitHub/repository link */}
        <div className="flex items-center justify-center">
          {tool.repoUrl && (
            <Link 
              href={tool.repoUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 text-sm text-primary hover:underline"
            >
              <IconBrandGithub className="h-4 w-4" />
              <span>{t("repository")}</span>
            </Link>
          )}
        </div>

        {/* Category tags (centered, max 3) */}
        <div className="flex items-center justify-center pt-1">
          {tool.categoryNames && tool.categoryNames.length > 0 ? (
            <div 
              className="flex flex-wrap gap-1 justify-center max-w-full"
              title={tool.categoryNames.map(c => CategoryNameMap[c] || c).join(', ')}
            >
              {tool.categoryNames.slice(0, 3).map((categoryName: string) => (
                <Badge key={categoryName} variant="secondary" className="text-xs whitespace-nowrap">
                  {CategoryNameMap[categoryName] || categoryName}
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
              {t("uncategorized")}
            </Badge>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 flex flex-col">
        {/* Current installed version */}
        <div className="mb-2">
          <div className="text-xs text-muted-foreground text-center mb-1">
            {t("currentVersion")}
          </div>
          <div className="text-center font-semibold text-base">
            {tool.version || 'N/A'}
          </div>
        </div>

        {/* Tool description */}
        <CardDescription 
          className="flex-1 text-center line-clamp-3 text-sm leading-snug"
          title={tool.description || t("noDescription")}
        >
          {tool.description || t("noDescription")}
        </CardDescription>
      </CardContent>

      <CardFooter className="flex gap-2">
        <Button
          variant="default"
          className="flex-1"
          onClick={() => onCheckUpdate?.(tool.id)}
          disabled={isChecking}
        >
          <IconRefresh className={isChecking ? "animate-spin h-4 w-4" : "h-4 w-4"} />
          {isChecking ? t("checking") : t("checkUpdate")}
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => onEdit?.(tool)}
        >
          <IconEdit className="h-4 w-4" />
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => onDelete?.(tool.id)}
        >
          <IconTrash className="h-4 w-4" />
        </Button>
      </CardFooter>
    </Card>
  )
}
