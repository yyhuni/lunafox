"use client"

import { VulnAuditDesign } from "@/components/prototypes/vuln-audit-design"
import { BauhausPageHeader } from "@/components/common/bauhaus-page-header"

export default function VulnAuditPage() {
  return (
    <div className="flex h-full flex-col">
      <BauhausPageHeader 
        code="PRT-AUDIT"
        subtitle="Prototypes"
        title="Vulnerability Audit Mode" 
        description="High-efficiency master-detail view for rapid vulnerability triage and verification."
      />
      
      <div className="flex-1 p-4 md:p-6 pt-0 overflow-hidden">
        <VulnAuditDesign />
      </div>
    </div>
  )
}
