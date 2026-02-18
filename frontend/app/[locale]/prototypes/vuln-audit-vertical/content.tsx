"use client"

import { VulnAuditVertical } from "@/components/prototypes/vuln-audit-vertical"
import { PageHeader } from "@/components/common/page-header"

export default function VulnAuditVerticalPage() {
  return (
    <div className="flex h-full flex-col">
      <PageHeader 
        code="PRT-AUDIT-V"
        title="Vulnerability Audit - Vertical Split" 
        description="Classic security tool layout: full-width table on top, detail view on bottom. Inspired by Burp Suite."
      />
      
      <div className="flex-1 min-h-0 p-4 md:p-6 pt-0 overflow-hidden">
        <VulnAuditVertical />
      </div>
    </div>
  )
}
