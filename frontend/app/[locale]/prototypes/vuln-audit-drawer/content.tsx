"use client"

import { VulnAuditDrawer } from "@/components/prototypes/vuln-audit-drawer"
import { PageHeader } from "@/components/common/page-header"

export default function VulnAuditDrawerPage() {
  return (
    <div className="flex h-full flex-col">
      <PageHeader 
        code="PRT-AUDIT-D"
        title="Vulnerability Audit - Drawer View" 
        description="Modern SaaS layout: full-screen table with slide-over detail drawer. Maximizes table visibility."
      />
      
      <div className="flex-1 p-4 md:p-6 pt-0 overflow-hidden">
        <VulnAuditDrawer />
      </div>
    </div>
  )
}
