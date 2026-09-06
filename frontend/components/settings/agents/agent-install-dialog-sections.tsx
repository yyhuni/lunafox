import { Badge } from "@/components/ui/badge"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { Button } from "@/components/ui/button"
import { getStatusToneBadgeClass, getStatusToneBgClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { RegistrationTokenResponse } from "@/types/agent.types"

type TranslationFn = (key: string, params?: Record<string, string | number | Date>) => string

function getTokenStatusTone(hasToken: boolean, isTokenValid: boolean) {
  if (!hasToken) return "muted"
  return isTokenValid ? "success" : "error"
}

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
  const tone = getTokenStatusTone(hasToken, isTokenValid)

  return (
    <div className="py-1">
      <div className="flex flex-wrap gap-3 items-center justify-between">
        <div className="flex flex-col gap-1.5">
          <div className="flex flex-wrap gap-2 items-center text-xs">
            <Badge
              variant="outline"
              className={getStatusToneBadgeClass(tone)}
            >
              <span
                className={cn(
                  "mr-1.5 inline-block h-1.5 w-1.5 rounded-full",
                  getStatusToneBgClass(tone)
                )}
              />
              {!hasToken
                ? t("install.commandStatusIdle")
                : isTokenValid
                  ? t("install.commandStatusReady")
                  : t("install.commandStatusExpired")}
            </Badge>
            {token?.expiresAt && isTokenValid && (
              <>
                <span className="tabular-nums text-muted-foreground text-xs">
                  • {t("install.commandExpires", { time: formatRelativeTime(token.expiresAt) })}
                </span>
                <span className="text-muted-foreground text-xs">• {t("install.tokenUsage")}</span>
              </>
            )}
          </div>
        </div>
        <Button
          size="sm"
          onClick={onGenerate}
          disabled={isGenerating}
          loading={isGenerating}
          loadingLabel={token ? t("install.regenerateToken") : t("install.generateToken")}
          className="shrink-0"
        >
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
}

export function AgentInstallCommandPanel({
  t,
  tActions,
  tToast,
  token,
  installCommand,
  isGenerating,
  canCopyCommand,
}: AgentInstallCommandPanelProps) {
  return (
    <div className="py-1 space-y-3">
      <p className={textRole.bodyStrong}>{t("install.commandTitle")}</p>
      <div className="bg-muted/30 border p-3 relative rounded-lg">
        <CopyButton
          value={installCommand}
          copyLabel={`${tActions("copy")} ${t("install.commandTitle")}`}
          copiedLabel={tToast("copied")}
          copyFailedLabel={tToast("copyFailed")}
          toastId="install-command-copy"
          disabled={!canCopyCommand}
          className="absolute right-0 top-0 z-10"
        />
        {token ? (
          <pre className="break-all font-mono max-h-48 overflow-y-auto pr-12 text-xs whitespace-pre-wrap">{installCommand}</pre>
        ) : (
          <div className="pr-12 text-muted-foreground text-xs">
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
    <div className={cn("py-1", textRole.helperText)}>
      <div className="space-y-3">
        <div>
          <div className={cn("flex gap-1.5 items-center mb-2", textRole.sectionTitle)}>
            {t("install.stepsTitle")}
          </div>
          <div className="space-y-1.5 text-muted-foreground">
            <div className="flex gap-2 items-start">
              <span className="font-medium">1.</span>
              <span>{t("install.step1Desc")}</span>
            </div>
            <div className="flex gap-2 items-start">
              <span className="font-medium">2.</span>
              <span>{t("install.step2Desc")}</span>
            </div>
          </div>
        </div>
        <div className="border-border/50 border-t pt-2">
          <div className={cn("flex gap-1.5 items-center mb-2", textRole.sectionTitle)}>
            {t("install.requirementsTitle")}
          </div>
          <div className="space-y-1 text-muted-foreground">
            <div className="flex gap-2 items-center">
              <span>•</span>
              <span>{t("install.requirementsDocker")}</span>
            </div>
            <div className="flex gap-2 items-center">
              <span>•</span>
              <span>{t("install.requirementsAccess")}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
