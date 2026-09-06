"use client";
import Link from "next/link";
import { IconRefresh, IconExternalLink, IconBrandGithub, IconMessageReport, IconBook, IconFileText, IconCheck, IconArrowUp, IconHeart, } from "@/components/icons";
import { LunaFoxMark } from "@/components/brand/lunafox-mark";
import { DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { textRole } from "@/lib/typography";
type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string;
interface AboutDialogHeaderProps {
    t: TranslationFn;
}
export function AboutDialogHeader({ t }: AboutDialogHeaderProps) {
    return (<DialogHeader>
      <DialogTitle>{t("title")}</DialogTitle>
    </DialogHeader>);
}
interface AboutDialogBrandingProps {
    t: TranslationFn;
}
export function AboutDialogBranding({ t }: AboutDialogBrandingProps) {
    return (<div className="flex flex-col items-center py-4">
      <div className="flex h-16 items-center justify-center mb-3 w-16">
        <LunaFoxMark className="h-12 w-12"/>
      </div>
      <h2 className={textRole.panelTitle}>{t("productName")}</h2>
      <p className={textRole.bodySubtle}>{t("description")}</p>
    </div>);
}
interface AboutDialogVersionInfoProps {
    t: TranslationFn;
    currentVersion: string;
    latestVersion?: string;
    hasUpdate?: boolean;
    checkError: string | null;
    isChecking: boolean;
    releaseUrl?: string;
    showLatest: boolean;
    onCheckUpdate: () => void;
}
export function AboutDialogVersionInfo({ t, currentVersion, latestVersion, hasUpdate, checkError, isChecking, releaseUrl, showLatest, onCheckUpdate, }: AboutDialogVersionInfoProps) {
    return (<div className="border p-4 rounded-lg space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-muted-foreground text-sm">{t("currentVersion")}</span>
        <span className="font-mono text-sm">{currentVersion}</span>
      </div>

      {showLatest && (<div className="flex items-center justify-between">
          <span className="text-muted-foreground text-sm">{t("latestVersion")}</span>
          <div className="flex gap-2 items-center">
            <span className="font-mono text-sm">{latestVersion}</span>
            {hasUpdate ? (<Badge variant="default" className="gap-1">
                <IconArrowUp className="h-3 w-3"/>
                {t("updateAvailable")}
              </Badge>) : (<Badge variant="secondary" className="gap-1">
                <IconCheck className="h-3 w-3"/>
                {t("upToDate")}
              </Badge>)}
          </div>
        </div>)}

      {checkError && (<p className="text-destructive text-sm">{checkError}</p>)}

      <div className="flex gap-2">
        <Button variant="outline" size="sm" className="flex-1" onClick={onCheckUpdate} disabled={isChecking} loading={isChecking} loadingLabel={t("checking")}>
          <IconRefresh className="h-4 w-4 mr-2"/>
          {t("checkUpdate")}
        </Button>

        {hasUpdate && releaseUrl && (<Button variant="default" size="sm" className="flex-1" render={<a href={releaseUrl} target="_blank" rel="noopener noreferrer"/>}>
              <IconExternalLink className="h-4 mr-2 w-4"/>
              {t("viewRelease")}
            </Button>)}
      </div>

      {hasUpdate && (<div className="bg-muted p-3 rounded-md text-muted-foreground text-sm">
          <p>{t("updateHint")}</p>
          <code className="bg-background block font-mono mt-1 px-2 py-1 rounded text-xs">
            sudo ./update.sh
          </code>
        </div>)}
    </div>);
}
interface AboutDialogLinksProps {
    t: TranslationFn;
}
export function AboutDialogLinks({ t }: AboutDialogLinksProps) {
    return (<div className="gap-2 grid grid-cols-2">
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/xingrin" target="_blank" rel="noopener noreferrer"/>}>
          <IconBrandGithub className="h-4 mr-2 w-4"/>
          GitHub
        </Button>
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/xingrin/releases" target="_blank" rel="noopener noreferrer"/>}>
          <IconFileText className="h-4 mr-2 w-4"/>
          {t("changelog")}
        </Button>
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/xingrin/issues" target="_blank" rel="noopener noreferrer"/>}>
          <IconMessageReport className="h-4 mr-2 w-4"/>
          {t("feedback")}
        </Button>
      <Button variant="ghost" size="sm" className="justify-start" render={<a href="https://github.com/yyhuni/xingrin#readme" target="_blank" rel="noopener noreferrer"/>}>
          <IconBook className="h-4 mr-2 w-4"/>
          {t("docs")}
        </Button>
      <Button variant="outline" size="sm" className="col-span-2 justify-center" render={<Link href="/settings/support/"/>}>
          <IconHeart className="h-4 mr-2 w-4"/>
          {t("supportAuthor")}
        </Button>
    </div>);
}
interface AboutDialogFooterProps {
    t: TranslationFn;
}
export function AboutDialogFooter({ t }: AboutDialogFooterProps) {
    return (<p className="text-center text-muted-foreground text-xs">
      © 2026 {t("productName")} · GPL-3.0
    </p>);
}
export { Separator };
