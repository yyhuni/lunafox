"use client"

import { WorkersPageDesign } from "@/components/prototypes/workers-page-design"
import { BauhausPageHeader } from "@/components/common/bauhaus-page-header"

export default function WorkersPageDesignPage() {
  return (
    <div className="flex h-full flex-col">
      <BauhausPageHeader 
        code="PRT-WORKERS"
        subtitle="Prototypes"
        title="Workers Page Layout" 
        description="Full page layout design for Worker Node management using Adaptive Console cards."
      />
      
      <div className="flex-1 overflow-hidden">
        <WorkersPageDesign />
      </div>
    </div>
  )
}
