"use client"

import React from "react"
import { ColumnDef } from "@tanstack/react-table"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import {
  Check,
  Edit,
  X as XIcon,
} from "@/components/icons"
import { DataTableColumnHeader } from "@/components/shared/data-table/column-header"
import { DenseRowActionMenu } from "@/components/shared/data-table/menu-owners"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import type { ScanWorkflow } from "@/types/scan-workflow.types"

export interface WorkflowTranslations {
  columns: {
    workflowName: string
    subdomainDiscovery: string
    portScan: string
    websiteDiscovery: string
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
  const engineIds = new Set(workflow.steps.map((step) => step.engineId))
  return {
    subdomain_discovery: engineIds.has("engine.lunafox.subdomain_discovery"),
    port_scan: engineIds.has("engine.lunafox.port_scan"),
    website_discovery: engineIds.has("engine.lunafox.website_discovery"),
    directory_scan: engineIds.has("engine.lunafox.directory_scan"),
    url_fetch: engineIds.has("engine.lunafox.url_fetch"),
    osint: engineIds.has("engine.lunafox.osint"),
    vulnerability_scan: engineIds.has("engine.lunafox.vulnerability_scan"),
    waf_detection: engineIds.has("engine.lunafox.waf_detection"),
    screenshot: engineIds.has("engine.lunafox.screenshot"),
  }
}

function FeatureStatus({ enabled }: { enabled?: boolean }) {
  if (enabled) {
    return (
      <div className="flex justify-center">
        <Check className={`h-5 w-5 ${getStatusToneTextClass("success")}`} />
      </div>
    )
  }
  return (
    <div className="flex justify-center">
      <XIcon className="h-5 text-destructive w-5" />
    </div>
  )
}

interface CreateColumnsProps {
  handleEdit: (workflow: ScanWorkflow) => void
  t: WorkflowTranslations
}

function WorkflowRowActions({
  onEdit,
  t,
}: {
  onEdit: () => void
  t: WorkflowTranslations
}) {
  return (
    <DenseRowActionMenu ariaLabel={t.actions.openMenu}>
      <DropdownMenuItem onClick={onEdit}>
        <Edit />
        {t.actions.editWorkflow}
      </DropdownMenuItem>
    </DenseRowActionMenu>
  )
}

export function createWorkflowColumns({ handleEdit, t }: CreateColumnsProps): ColumnDef<ScanWorkflow>[] {
  return [
    {
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title={t.columns.workflowName} />,
      cell: ({ row }) => <span className={textRole.tableCellPrimary}>{row.original.displayName || row.original.name}</span>,
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
      id: "website_discovery",
      header: t.columns.websiteDiscovery,
      cell: ({ row }) => <FeatureStatus enabled={parseWorkflowFeatures(row.original).website_discovery} />,
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
          <WorkflowRowActions
            onEdit={() => handleEdit(row.original)}
            t={t}
          />
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
  ]
}
