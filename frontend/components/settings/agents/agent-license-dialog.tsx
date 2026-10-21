"use client"

import { useEffect, useRef, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import { semanticIcons } from "@/components/icons"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useActivateAgentLicense } from "@/hooks/use-agent-license"
import { getStatusToneSurfaceClass, getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { narrowFormDialogContentClassName } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import type { AgentLicense } from "@/types/agent-license.types"
import { formatAgentLicenseExpiry } from "./agent-license-date"

export function AgentLicenseDialog({ license, onClose }: { license: AgentLicense; onClose: () => void }) {
  const t = useTranslations("settings.agents.activation")
  const locale = useLocale()
  const [key, setKey] = useState("")
  const [success, setSuccess] = useState<AgentLicense | null>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const returnRef = useRef<HTMLButtonElement>(null)
  const activation = useActivateAgentLicense()
  const errorKey = activation.errorReason === "LICENSE_KEY_EXPIRED" ? "expiredError"
    : activation.errorReason === "LICENSE_KEY_NOT_YET_VALID" ? "notYetValidError"
    : activation.errorReason === "LICENSE_KEY_INVALID" || activation.errorReason === "LICENSE_KEY_REQUIRED" ? "invalidError" : "requestError"

  useEffect(() => {
    if (activation.isError) inputRef.current?.focus()
  }, [activation.isError])
  useEffect(() => { if (success) returnRef.current?.focus() }, [success])

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (activation.isPending || !key.trim()) return
    try { setSuccess(await activation.mutateAsync(key)) } catch {
      // The hook retains the failed command; keep the Key available for correction.
    }
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open && !activation.isPending) onClose() }}>
      <DialogContent initialFocus={inputRef} className={cn(narrowFormDialogContentClassName, "max-h-dvh overflow-y-auto")}>
        <DialogHeader className="pr-8 text-left">
          <DialogTitle>{t(success ? "successTitle" : license.state === "inactive" ? "title" : "renewTitle")}</DialogTitle>
          <DialogDescription>{t(success ? "successDescription" : "description")}</DialogDescription>
        </DialogHeader>
        {success ? (
          <div className="space-y-5 py-2">
            <div role="status" className="flex items-center gap-3">
              <span className={cn("radius-round flex size-10 shrink-0 items-center justify-center", getStatusToneSurfaceClass("success"))}>
                <semanticIcons.status.success aria-hidden="true" className={cn("size-5", getStatusToneTextClass("success"))} />
              </span>
              <div>
                <p className={textRole.sectionTitle}>{t("successLimit", { limit: success.agentLimit })}</p>
                <p className={cn("mt-1", textRole.helperText)}>{t("validity", { months: success.months!, date: formatAgentLicenseExpiry(success.expiresAt!, locale) })}</p>
              </div>
            </div>
            <dl className={cn("grid grid-cols-2 gap-3 border-y py-4", textRole.metadataLabel)}>
              <dt>{t("quotaLabel")}</dt><dd className={cn("text-right", textRole.metadataValueStrong)}>{success.agentLimit}</dd>
              <dt>{t("licenseIdLabel")}</dt><dd className={cn("break-all text-right", textRole.code)}>{success.licenseId}</dd>
              <dt>{t("taskConfigLabel")}</dt><dd className={cn("text-right", textRole.metadataValue)}>{t("taskConfigUnchanged")}</dd>
            </dl>
            <DialogFooter><Button ref={returnRef} onClick={onClose}>{t("return")}</Button></DialogFooter>
          </div>
        ) : (
          <form onSubmit={submit} className="space-y-4 pt-2" aria-busy={activation.isPending}>
            <div className="space-y-2">
              <Label htmlFor="agent-license-key">{t("keyLabel")}</Label>
              <Textarea ref={inputRef} id="agent-license-key" name="activationKey" value={key}
                onChange={(event) => { setKey(event.target.value); activation.reset() }}
                placeholder={t("placeholder")} className={cn("min-h-24 resize-none", textRole.code)}
                spellCheck={false} autoCapitalize="none" disabled={activation.isPending} required
                aria-invalid={activation.isError} aria-describedby={activation.isError ? "agent-license-error" : "agent-license-help"} />
              <p id="agent-license-help" className={textRole.helperText}>{t("help")}</p>
              {activation.isError && <p id="agent-license-error" role="alert" className={cn(textRole.helperText, getStatusToneTextClass("error"))}>{t(errorKey)}</p>}
            </div>
            <div className="radius-control space-y-1 border bg-muted/20 p-3">
              <p className={textRole.bodyStrong}>{t("currentQuota", { limit: license.agentLimit })}</p>
              <p className={textRole.helperText}>{t("keyDeterminesQuota")}</p>
            </div>
            <Collapsible>
              <CollapsibleTrigger render={<Button type="button" variant="link" size="sm" />}>
                <semanticIcons.navigation.support aria-hidden="true" />{t("getKey")}
              </CollapsibleTrigger>
              <CollapsibleContent><div className={cn("space-y-2 pb-1 pt-2", textRole.bodySubtle)}>
                <p className={textRole.bodyStrong}>{t("followTitle")}</p><p>{t("followDescription")}</p>
              </div></CollapsibleContent>
            </Collapsible>
            {activation.isPending && <Alert role="status"><semanticIcons.concept.apiKey aria-hidden="true" /><AlertDescription>{t("pending")}</AlertDescription></Alert>}
            <DialogFooter className="pt-1">
              <Button type="button" variant="outline" disabled={activation.isPending} onClick={onClose}>{t("cancel")}</Button>
              <Button type="submit" disabled={!key.trim() || activation.isPending} loading={activation.isPending} loadingLabel={t("activating")}>{t("activate")}</Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
