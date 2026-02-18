import { IconLoader2 } from "@/components/icons"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { RegistrationTokenResponse } from "@/types/agent.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

interface AgentInstallTokenCardProps {
  t: TranslationFn
  token: RegistrationTokenResponse | null
  hasToken: boolean
  isTokenValid: boolean
  isGenerating: boolean
  formatRelativeTime: (value: string | Date) => string
  onGenerate: () => void
}

export function AgentInstallTokenCard({
  t,
  token,
  hasToken,
  isTokenValid,
  isGenerating,
  formatRelativeTime,
  onGenerate,
}: AgentInstallTokenCardProps) {
  return (
    <div className={cn("rounded-lg border p-3 transition-[background-color,border-color]", isTokenValid ? "bg-background" : "bg-muted/20")}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-2 text-xs">
            <Badge
              variant="outline"
              className={
                !hasToken
                  ? "border-muted-foreground/30 text-muted-foreground"
                  : isTokenValid
                    ? "border-emerald-500/50 text-emerald-700 bg-emerald-500/5 dark:text-emerald-400"
                    : "border-rose-500/50 text-rose-700 bg-rose-500/5 dark:text-rose-400"
              }
            >
              <span
                className={cn(
                  "mr-1.5 inline-block h-1.5 w-1.5 rounded-full",
                  isTokenValid ? "bg-emerald-500" : "bg-muted-foreground/40"
                )}
              />
              {!hasToken
                ? t("install.commandStatusIdle")
                : isTokenValid
                  ? t("install.commandStatusReady")
                  : t("install.commandStatusExpired")}
            </Badge>
            {token?.expiresAt && isTokenValid && (
              <span className="text-xs text-muted-foreground tabular-nums">
                • {t("install.commandExpires", { time: formatRelativeTime(token.expiresAt) })}
              </span>
            )}
          </div>
          {token?.expiresAt && isTokenValid && (
            <span className="text-[11px] text-muted-foreground">
              {t("install.tokenUsage")}
            </span>
          )}
        </div>
        <Button size="sm" onClick={onGenerate} disabled={isGenerating} className="shrink-0">
          {isGenerating && <IconLoader2 className="mr-2 h-4 w-4 animate-spin" />}
          {token ? t("install.regenerateToken") : t("install.generateToken")}
        </Button>
      </div>
    </div>
  )
}


interface AgentInstallCommandPanelProps {
  t: TranslationFn
  tActions: TranslationFn
  tToast: TranslationFn
  token: RegistrationTokenResponse | null
  installCommand: string
  isGenerating: boolean
  canCopyCommand: boolean
  copied: boolean
  onCopy: (value: string) => void
}

export function AgentInstallCommandPanel({
  t,
  tActions,
  tToast,
  token,
  installCommand,
  isGenerating,
  canCopyCommand,
  copied,
  onCopy,
}: AgentInstallCommandPanelProps) {
  return (
    <div className="rounded-lg border bg-background p-3 space-y-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-medium">{t("install.commandTitle")}</p>
          <p className="text-xs text-muted-foreground">{t("install.commandDesc")}</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="h-8 text-xs shrink-0"
          onClick={() => onCopy(installCommand)}
          disabled={!canCopyCommand}
          aria-label={`${tActions("copy")} ${t("install.commandTitle")}`}
        >
          {copied ? tToast("copied") : tActions("copy")}
        </Button>
      </div>
      <div className="rounded-lg border bg-muted/30 p-3">
        {token ? (
          <pre className="max-h-48 overflow-y-auto font-mono text-xs whitespace-pre-wrap break-all">{installCommand}</pre>
        ) : (
          <div className="text-xs text-muted-foreground">
            {isGenerating ? t("install.commandStatusGenerating") : t("install.commandPlaceholder")}
          </div>
        )}
      </div>
    </div>
  )
}

interface AgentInstallCommandTipsProps {
  t: TranslationFn
}

export function AgentInstallCommandTips({ t }: AgentInstallCommandTipsProps) {
  return (
    <div className="rounded-lg border bg-muted/20 p-4 text-xs">
      <div className="space-y-3">
        <div>
          <div className="font-semibold text-foreground mb-2 flex items-center gap-1.5">
            {t("install.stepsTitle")}
          </div>
          <div className="space-y-1.5 text-muted-foreground">
            <div className="flex items-start gap-2">
              <span className="font-medium">1.</span>
              <span>{t("install.step1Desc")}</span>
            </div>
            <div className="flex items-start gap-2">
              <span className="font-medium">2.</span>
              <span>{t("install.step2Desc")}</span>
            </div>
          </div>
        </div>
        <div className="pt-2 border-t border-border/50">
          <div className="font-semibold text-foreground mb-2 flex items-center gap-1.5">
            {t("install.requirementsTitle")}
          </div>
          <div className="space-y-1 text-muted-foreground">
            <div className="flex items-center gap-2">
              <span>•</span>
              <span>{t("install.requirementsDocker")}</span>
            </div>
            <div className="flex items-center gap-2">
              <span>•</span>
              <span>{t("install.requirementsAccess")}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
