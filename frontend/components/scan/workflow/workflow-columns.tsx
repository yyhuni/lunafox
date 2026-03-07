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
import { normalizeWorkflowConfiguration } from "@/lib/workflow-config"
import type { ScanWorkflow } from "@/types/workflow.types"

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

function parseWorkflowFeatures(workflow: ScanWorkflow) {
  const config = normalizeWorkflowConfiguration(workflow.configuration)
  return {
    subdomain_discovery: !!config.subdomain_discovery,
    port_scan: !!config.port_scan,
    site_scan: !!config.site_scan,
    directory_scan: !!config.directory_scan,
    url_fetch: !!config.url_fetch || !!config.fetch_url,
    osint: !!config.osint,
    vulnerability_scan: !!config.vulnerability_scan,
    waf_detection: !!config.waf_detection,
    screenshot: !!config.screenshot,
  }
}

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
        <Button variant="ghost" className="flex h-8 w-8 p-0 data-[state=open]:bg-muted">
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
        <DropdownMenuItem onClick={onDelete} className="text-destructive focus:text-destructive">
          <Trash2 />
          {t.actions.delete}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function createWorkflowColumns({ handleEdit, handleDelete, t }: CreateColumnsProps): ColumnDef<ScanWorkflow>[] {
  return [
    {
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title={t.columns.workflowName} />,
      cell: ({ row }) => <span className="font-medium">{row.original.title || row.original.name}</span>,
      enableSorting: true,
      enableHiding: false,
    },
    {
      id: "subdomain_discovery",
      header: t.columns.subdomainDiscovery,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).subdomain_discovery} />,
      size: 80,
    },
    {
      id: "port_scan",
      header: t.columns.portScan,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).port_scan} />,
      size: 80,
    },
    {
      id: "site_scan",
      header: t.columns.siteScan,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).site_scan} />,
      size: 80,
    },
    {
      id: "directory_scan",
      header: t.columns.directoryScan,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).directory_scan} />,
      size: 80,
    },
    {
      id: "url_fetch",
      header: t.columns.urlFetch,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).url_fetch} />,
      size: 80,
    },
    {
      id: "osint",
      header: t.columns.osint,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).osint} />,
      size: 80,
    },
    {
      id: "vulnerability_scan",
      header: t.columns.vulnerabilityScan,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).vulnerability_scan} />,
      size: 80,
    },
    {
      id: "waf_detection",
      header: t.columns.wafDetection,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).waf_detection} />,
      size: 80,
    },
    {
      id: "screenshot",
      header: t.columns.screenshot,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).screenshot} />,
      size: 80,
    },
    {
      id: "actions",
      header: () => <span className="sr-only">{t.actions.openMenu}</span>,
      cell: ({ row }) => (
        <div className="flex justify-end">
          <Tooltip>
            <TooltipTrigger asChild>
              <div>
                <WorkflowRowActions
                  onEdit={() => handleEdit(row.original)}
                  onDelete={() => handleDelete(row.original)}
                  t={t}
                />
              </div>
            </TooltipTrigger>
            <TooltipContent>{t.tooltips.editWorkflow}</TooltipContent>
          </Tooltip>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
  ]
}
