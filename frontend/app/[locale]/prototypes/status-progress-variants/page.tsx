import { StatusProgressDemos } from "@/components/prototypes/status-progress-demos"
import { BauhausPageHeader } from "@/components/common/bauhaus-page-header"

export default function StatusProgressVariantsPage() {
  return (
    <div className="flex h-full flex-col">
      <BauhausPageHeader 
        code="PRT-STATUS"
        subtitle="Prototypes"
        title="Status & Progress Variants" 
        description="Exploration of combined status and progress indicators"
        showDescription
      />
      
      <div className="flex-1 overflow-auto p-6">
        <StatusProgressDemos />
      </div>
    </div>
  )
}
