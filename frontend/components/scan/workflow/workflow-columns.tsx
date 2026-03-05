"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  MoreHorizontal,
  Trash2,
  Check,
  Edit,
  X as XIcon,
} from "@/components/icons"
import { DataTableColumnHeader } from "@/components/ui/data-table/column-header"
import * as yaml from "js-yaml"
import type { ScanWorkflow } from "@/types/workflow.types"

// Translation type definitions (workflow template management page)
export interface WorkflowTranslations {
  columns: {
    workflowName: string
    subdomainDiscovery: string
    portScan: string
    siteScan: string
    directoryScan: string
    urlFetch: string
    osint: string
    vulnerabilityScan: string
    wafDetection: string
    screenshot: string
  }
  actions: {
    editWorkflow: string
    delete: string
    openMenu: string
  }
  tooltips: {
    editWorkflow: string
  }
}

/**
 * Parse workflow YAML configuration and detect if features are enabled
 */
function parseWorkflowFeatures(workflow: ScanWorkflow) {
  if (workflow.configuration) {
    try {
      const config = yaml.load(workflow.configuration) as Record<string, unknown> | null
      return {
        subdomain_discovery: !!config?.subdomain_discovery,
        port_scan: !!config?.port_scan,
        site_scan: !!config?.site_scan,
        directory_scan: !!config?.directory_scan,
        url_fetch: !!config?.url_fetch || !!config?.fetch_url,
        osint: !!config?.osint,
        vulnerability_scan: !!config?.vulnerability_scan,
        waf_detection: !!config?.waf_detection,
        screenshot: !!config?.screenshot,
      }
    } catch (error) {
      void error
    }
  }
  
  return {
    subdomain_discovery: false,
    port_scan: false,
    site_scan: false,
    directory_scan: false,
    url_fetch: false,
    osint: false,
    vulnerability_scan: false,
    waf_detection: false,
    screenshot: false,
  }
}

/**
 * Feature support status component
 */
function FeatureStatus({ enabled }: { enabled?: boolean }) {
  if (enabled) {
    return (
      <div className="flex justify-center">
        <Check className="h-5 w-5 text-chart-4" />
      </div>
    )
  }
  return (
    <div className="flex justify-center">
      <XIcon className="h-5 w-5 text-destructive" />
    </div>
  )
}

interface CreateColumnsProps {
  handleEdit: (workflow: ScanWorkflow) => void
  handleDelete: (workflow: ScanWorkflow) => void
  t: WorkflowTranslations
}

/**
 * Workflow template row actions component
 */
function WorkflowRowActions({
  onEdit,
  onDelete,
  t,
}: {
  onEdit: () => void
  onDelete: () => void
  t: WorkflowTranslations
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
        >
          <MoreHorizontal />
          <span className="sr-only">{t.actions.openMenu}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>
          <Edit />
          {t.actions.editWorkflow}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={onDelete}
          className="text-destructive focus:text-destructive"
        >
          <Trash2 />
          {t.actions.delete}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

/**
 * Create workflow template table column definitions
 */
export const createWorkflowColumns = ({
  handleEdit,
  handleDelete,
  t,
}: CreateColumnsProps): ColumnDef<ScanWorkflow>[] => [
  {
    accessorKey: "name",
    size: 200,
    minSize: 150,
    maxSize: 350,
    meta: { title: t.columns.workflowName },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.workflowName} />
    ),
    cell: ({ row }) => {
      const name = row.getValue("name") as string
      return (
        <Tooltip>
          <TooltipTrigger asChild>
            <button type="button"
              onClick={() => handleEdit(row.original)}
              className="max-w-[300px] truncate font-medium text-left hover:text-primary hover:underline underline-offset-2 cursor-pointer transition-colors"
            >
              {name}
            </button>
          </TooltipTrigger>
          <TooltipContent>{t.tooltips.editWorkflow}</TooltipContent>
        </Tooltip>
      )
    },
  },
  {
    id: "subdomain_discovery",
    meta: { title: t.columns.subdomainDiscovery },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.subdomainDiscovery} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.subdomain_discovery} />
    },
    enableSorting: false,
  },
  {
    id: "port_scan",
    meta: { title: t.columns.portScan },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.portScan} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.port_scan} />
    },
    enableSorting: false,
  },
  {
    id: "site_scan",
    meta: { title: t.columns.siteScan },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.siteScan} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.site_scan} />
    },
    enableSorting: false,
  },
  {
    id: "directory_scan",
    meta: { title: t.columns.directoryScan },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.directoryScan} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.directory_scan} />
    },
    enableSorting: false,
  },
  {
    id: "url_fetch",
    meta: { title: t.columns.urlFetch },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.urlFetch} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.url_fetch} />
    },
    enableSorting: false,
  },
  {
    id: "osint",
    meta: { title: t.columns.osint },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.osint} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.osint} />
    },
    enableSorting: false,
  },
  {
    id: "vulnerability_scan",
    meta: { title: t.columns.vulnerabilityScan },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.vulnerabilityScan} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.vulnerability_scan} />
    },
    enableSorting: false,
  },
  {
    id: "waf_detection",
    meta: { title: t.columns.wafDetection },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.wafDetection} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.waf_detection} />
    },
    enableSorting: false,
  },
  {
    id: "screenshot",
    meta: { title: t.columns.screenshot },
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t.columns.screenshot} />
    ),
    size: 80,
    minSize: 60,
    maxSize: 100,
    enableResizing: false,
    cell: ({ row }) => {
      const features = parseWorkflowFeatures(row.original)
      return <FeatureStatus enabled={features.screenshot} />
    },
    enableSorting: false,
  },
  {
    id: "actions",
    size: 60,
    minSize: 60,
    maxSize: 60,
    enableResizing: false,
    cell: ({ row }) => (
      <WorkflowRowActions
        onEdit={() => handleEdit(row.original)}
        onDelete={() => handleDelete(row.original)}
        t={t}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
]
