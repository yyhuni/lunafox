"use client";
import { useState, type ReactElement } from "react";
import { Dialog, DialogContent, DialogTrigger } from "@/components/ui/dialog";
import { useAboutDialogState } from "@/components/about-dialog-state";
import { AboutDialogHeader, AboutDialogBranding, AboutDialogVersionInfo, AboutDialogLinks, AboutDialogFooter, Separator, } from "@/components/about-dialog-sections";
interface AboutDialogProps {
    children: ReactElement;
}
export function AboutDialog({ children }: AboutDialogProps) {
    const [open, setOpen] = useState(false);
    const { t, isChecking, updateResult, checkError, currentVersion, latestVersion, hasUpdate, releaseUrl, handleCheckUpdate, } = useAboutDialogState({ enabled: open });
    return (<Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={children}></DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <AboutDialogHeader t={t}/>

        <div className="space-y-6">
          <AboutDialogBranding t={t}/>

          <AboutDialogVersionInfo t={t} currentVersion={currentVersion} latestVersion={latestVersion} hasUpdate={hasUpdate} checkError={checkError} isChecking={isChecking} releaseUrl={releaseUrl} showLatest={!!updateResult} onCheckUpdate={handleCheckUpdate}/>

          <Separator />

          <AboutDialogLinks t={t}/>

          <AboutDialogFooter t={t}/>
        </div>
      </DialogContent>
    </Dialog>);
}
