"use client"

import {
  Dialog,
  DialogContent,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { IconInfoCircle } from "@/components/icons"
import { useArchitectureDialogState } from "@/components/settings/workers/architecture-dialog-state"
import {
  ArchitectureDialogHeader,
  ArchitectureFlowSection,
  ArchitectureRoleSection,
  ArchitectureStepsSection,
  ScrollArea,
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
        <Button variant="outline" size="sm">
          <IconInfoCircle className="h-4 w-4 mr-2" />
          {t("viewArchitecture")}
        </Button>
      </DialogTrigger>
      <DialogContent className="w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] h-[calc(100vh-1rem)] sm:w-[calc(100vw-2rem)] sm:max-w-[1280px] sm:h-[calc(100vh-2rem)] p-0 gap-0 flex flex-col overflow-hidden">
        <div className="border-b px-4 py-4 sm:px-6">
          <ArchitectureDialogHeader t={t} />
        </div>
        <Tabs defaultValue="diagram" className="flex-1 min-h-0 gap-0">
          <div className="border-b px-4 pt-2 sm:px-6">
            <TabsList variant="underline" size="sm" className="h-9">
              <TabsTrigger value="diagram" variant="underline" size="sm">
                {t("flowTabDiagram")}
              </TabsTrigger>
              <TabsTrigger value="roles" variant="underline" size="sm">
                {t("flowTabRoles")}
              </TabsTrigger>
              <TabsTrigger value="steps" variant="underline" size="sm">
                {t("flowTabSteps")}
              </TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="diagram" className="flex-1 min-h-0 mt-0">
            <ScrollArea className="h-full px-4 py-3 sm:px-6">
              <ArchitectureFlowSection t={t} isOpen={open} />
            </ScrollArea>
          </TabsContent>

          <TabsContent value="roles" className="flex-1 min-h-0 mt-0">
            <ScrollArea className="h-full px-4 py-4 sm:px-6">
              <ArchitectureRoleSection
                t={t}
                labels={labels}
                roleDetails={roleDetails}
              />
            </ScrollArea>
          </TabsContent>

          <TabsContent value="steps" className="flex-1 min-h-0 mt-0">
            <ScrollArea className="h-full px-4 py-4 sm:px-6">
              <ArchitectureStepsSection
                t={t}
                steps={flowSteps}
              />
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}
