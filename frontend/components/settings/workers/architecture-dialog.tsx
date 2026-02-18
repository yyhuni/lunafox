"use client"

import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"
import { useArchitectureDialogState } from "@/components/settings/workers/architecture-dialog-state"
import {
  ArchitectureDialogTrigger,
  ArchitectureDialogHeader,
  ArchitectureFlowSection,
  ArchitectureRoleSection,
  ArchitectureStepsSection,
  ScrollArea,
  Separator,
} from "@/components/settings/workers/architecture-dialog-sections"

export function ArchitectureDialog() {
  const {
    t,
    open,
    setOpen,
    labels,
    roleDetails,
    flowSteps,
  } = useArchitectureDialogState()

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <ArchitectureDialogTrigger t={t} />
      </DialogTrigger>
      <DialogContent className="max-w-5xl max-h-[90vh]">
        <ArchitectureDialogHeader t={t} />
        <ScrollArea className="max-h-[calc(90vh-120px)] pr-4">
          <div className="space-y-6">
            <ArchitectureFlowSection t={t} isOpen={open} />

            <Separator />

            <ArchitectureRoleSection
              t={t}
              labels={labels}
              roleDetails={roleDetails}
            />

            <Separator />

            <ArchitectureStepsSection
              t={t}
              steps={flowSteps}
            />
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
