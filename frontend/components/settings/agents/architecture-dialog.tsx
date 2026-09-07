"use client";
import { Dialog, DialogContent, DialogTrigger, } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { IconInfoCircle } from "@/components/icons";
import { useArchitectureDialogState } from "@/components/settings/agents/architecture-dialog-state";
import { ArchitectureCommandCenter, ArchitectureDialogHeader, } from "@/components/settings/agents/architecture-dialog-sections";
export function ArchitectureDialog({ trigger }: {
    trigger?: React.ReactElement;
}) {
    const { t, open, setOpen, } = useArchitectureDialogState();
    return (<Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={trigger || (<Button variant="outline" size="sm">
            <IconInfoCircle className="h-4 mr-2 w-4"/>
            {t("viewArchitecture")}
          </Button>)}></DialogTrigger>
      <DialogContent className="flex max-w-screen-2xl flex-col gap-0 overflow-hidden p-0" style={{
            width: "min(1184px, calc(100vw - 48px))",
            height: "min(720px, calc(100vh - 48px))",
        }}>
        <div className="flex items-center justify-between gap-6 border-b pl-6 pr-14 py-5">
          <ArchitectureDialogHeader t={t}/>
        </div>
        <ArchitectureCommandCenter isOpen={open} t={t}/>
      </DialogContent>
    </Dialog>);
}
