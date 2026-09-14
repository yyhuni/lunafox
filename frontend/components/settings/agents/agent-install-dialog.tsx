"use client";
import { useAgentInstallDialogState } from "@/components/settings/agents/agent-install-dialog-state";
import { AgentInstallConnectionStatus } from "@/components/settings/agents/agent-install-connection-status";
import { useTranslations } from "next-intl";
import { DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { useFormatRelativeTime } from "@/lib/i18n-format";
import { getStatusToneSurfaceClass, getStatusToneTextClass } from "@/lib/status-config";
import type { RegistrationTokenResponse } from "@/types/agent.types";
import type { AgentInstallConnectionViewState } from "@/hooks/use-agent-install-connection";
import { AgentInstallCommandPanel, AgentInstallCommandTips, AgentInstallTokenCard, } from "@/components/settings/agents/agent-install-dialog-sections";
import { COMPACT_FORM_OVERLAY_INSET_CLASS } from "@/components/shared/layout/page-shell-density";
import { cn } from "@/lib/utils";
type AgentInstallDialogProps = {
    open: boolean;
    token: RegistrationTokenResponse | null;
    connection: AgentInstallConnectionViewState;
    isGenerating: boolean;
    onGenerate: () => void;
};
export function AgentInstallDialog({ open, token, connection, isGenerating, onGenerate, }: AgentInstallDialogProps) {
    const t = useTranslations("settings.agents");
    const tActions = useTranslations("common.actions");
    const tToast = useTranslations("toast");
    const formatRelativeTime = useFormatRelativeTime();
    const { dialogRef, hasToken, isTokenValid, installCommand, canCopyCommand, showLocalOnlyWarning, } = useAgentInstallDialogState({
        open,
        token,
        tToast,
    });
    return (<DialogContent ref={dialogRef} className="flex flex-col gap-0 max-h-[calc(100vh-2rem)] overflow-hidden p-0 sm:max-w-[860px]">
      <DialogHeader className={cn("border-b pr-12 shrink-0", COMPACT_FORM_OVERLAY_INSET_CLASS)}>
        <DialogTitle>{t("install.title")}</DialogTitle>
        <DialogDescription>{t("install.desc")}</DialogDescription>
      </DialogHeader>
      <div className={cn("flex-1 min-h-0 overflow-y-auto", COMPACT_FORM_OVERLAY_INSET_CLASS)}>
        <AgentInstallTokenCard t={t} token={token} hasToken={hasToken} isTokenValid={isTokenValid} isGenerating={isGenerating} formatRelativeTime={formatRelativeTime} onGenerate={onGenerate}/>
        <Separator className="my-3" />
        <AgentInstallCommandPanel t={t} tActions={tActions} tToast={tToast} token={token} installCommand={installCommand} isGenerating={isGenerating} canCopyCommand={canCopyCommand}/>
        {showLocalOnlyWarning && (<div className={`${getStatusToneSurfaceClass("warning")} rounded-lg p-3`}>
            <p className={`text-[11px] ${getStatusToneTextClass("warning")}`}>{t("install.localOnlyWarning")}</p>
          </div>)}
        {open && token && (
          <AgentInstallConnectionStatus connection={connection} />
        )}
        <Separator className="my-3" />
        <AgentInstallCommandTips t={t}/>
      </div>
    </DialogContent>);
}
