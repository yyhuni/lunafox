"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import { IconCode, semanticIcons } from "@/components/icons"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useGenerateMcpKey, useMcpKeyStatus } from "@/hooks/use-mcp-key"
import { compactSurfaceClassNames } from "@/lib/ui/compact-surface-contract"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type { McpKeyGeneration } from "@/types/mcp-key.types"
import { getMcpClientConfiguration, mcpClientFiles, mcpClients, type McpClient } from "./mcp-client-configuration"

const KeyIcon = semanticIcons.concept.apiKey
const SuccessIcon = semanticIcons.status.success
const RefreshIcon = semanticIcons.action.refresh

export function McpConnectionSettings({ client, onClientChange }: { client: McpClient; onClientChange: (client: McpClient) => void }) {
  const t = useTranslations("mcpIntegration")
  const [endpoint, setEndpoint] = React.useState("")
  const [revealedKey, setRevealedKey] = React.useState<string | null>(null)
  const [confirmationOpen, setConfirmationOpen] = React.useState(false)
  const [wasRegenerated, setWasRegenerated] = React.useState(false)
  const rotatingRef = React.useRef(false)
  const statusQuery = useMcpKeyStatus(true)
  const handleGenerated = React.useCallback((generation: McpKeyGeneration) => {
    setRevealedKey(generation.key)
    setWasRegenerated(rotatingRef.current)
    rotatingRef.current = false
    setConfirmationOpen(false)
  }, [])
  const generateMcpKey = useGenerateMcpKey(handleGenerated)

  React.useEffect(() => {
    setEndpoint(`${window.location.origin}/mcp`)
  }, [])

  // The page owns the one-time secret. The hook caches only non-secret status,
  // so unmounting this route also discards both the key and generated configuration.
  const configuration = revealedKey && endpoint ? getMcpClientConfiguration(client, endpoint, revealedKey) : ""
  const configured = statusQuery.data?.configured === true
  const generate = async (rotation: boolean) => {
    rotatingRef.current = rotation
    try { await generateMcpKey.mutateAsync() } catch { rotatingRef.current = false }
  }
  const clear = () => {
    setRevealedKey(null)
    setWasRegenerated(false)
    generateMcpKey.reset()
  }

  return (
    <section className="min-w-0" aria-labelledby="mcp-setup-title">
      <Card variant="compact">
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <h2 id="mcp-setup-title" className={textRole.panelTitle}>{t("page.setupTitle")}</h2>
            <p className={textRole.helperText}>{t("page.setupDescription")}</p>
          </div>
          <Tabs value={client} onValueChange={(value) => onClientChange(value as McpClient)}>
            <TabsList variant="content" size="sm" aria-label={t("configuration.label")}>
              {mcpClients.map((option) => (
                <TabsTrigger key={option} value={option} variant="content" size="sm" className="flex-1 sm:flex-none">
                  {t(`configuration.clients.${option}`)}
                </TabsTrigger>
              ))}
            </TabsList>
            <TabsContent value={client} className="grid min-w-0 gap-6 pt-4 lg:grid-cols-5">
              <div className="min-w-0 space-y-5 lg:col-span-2">
                <div className="space-y-2">
                  <Label htmlFor="mcp-endpoint">{t("endpoint.label")}</Label>
                  <div className="relative">
                    <Input id="mcp-endpoint" size="sm" value={endpoint} readOnly className="pr-10 font-mono" />
                    <CopyButton value={endpoint} disabled={!endpoint} copyLabel={t("endpoint.copy")} copiedLabel={t("endpoint.copied")} copyFailedLabel={t("endpoint.copyFailed")} className="absolute right-1 top-1/2 -translate-y-1/2" />
                  </div>
                </div>
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <Label id="mcp-key-label" htmlFor={revealedKey || statusQuery.isPending ? "mcp-key" : undefined}>{t("key.label")}</Label>
                    {revealedKey || configured ? <span className={cn("inline-flex items-center gap-1 text-success", textRole.caption)}><SuccessIcon className="size-3.5" aria-hidden="true" />{t(revealedKey ? "key.revealed" : "key.configured")}</span> : null}
                  </div>
                  {statusQuery.isError ? (
                    <div className={cn(compactSurfaceClassNames.info, "flex flex-wrap items-center justify-between gap-3 border-destructive/30")} role="alert">
                      <p className={textRole.helperText}>{t("key.statusError")}</p>
                      <Button variant="outline" size="sm" onClick={() => void statusQuery.refetch()}>{t("key.retry")}</Button>
                    </div>
                  ) : revealedKey ? (
                    <>
                      <div className="relative">
                        <Input id="mcp-key" size="sm" value={revealedKey} readOnly className="pr-10 font-mono" />
                        <CopyButton value={revealedKey} copyLabel={t("key.copy")} copiedLabel={t("key.copied")} copyFailedLabel={t("key.copyFailed")} className="absolute right-1 top-1/2 -translate-y-1/2" />
                      </div>
                      <p className={textRole.helperText}>{t(wasRegenerated ? "key.rotatedDescription" : "key.revealedDescription")}</p>
                      <Button variant="ghost" size="sm" onClick={clear}>{t("page.hideKey")}</Button>
                    </>
                  ) : statusQuery.isPending ? (
                    <Input id="mcp-key" size="sm" value={t("key.loading")} readOnly disabled />
                  ) : (
                    <div className={cn(compactSurfaceClassNames.mutedInfo, "space-y-3")} role="group" aria-labelledby="mcp-key-label">
                      <p className={textRole.helperText}>{t(configured ? "key.lostDescription" : "key.emptyDescription")}</p>
                      <Button size="sm" variant={configured ? "outline" : "default"} loading={generateMcpKey.isPending} loadingLabel={t("key.generating")} onClick={() => configured ? setConfirmationOpen(true) : void generate(false)}>
                        {configured ? <RefreshIcon className="size-4" /> : <KeyIcon className="size-4" />}{t(configured ? "key.regenerate" : "key.generate")}
                      </Button>
                    </div>
                  )}
                  {generateMcpKey.isError ? <p className={cn(textRole.helperText, "text-destructive")} role="alert">{t("key.generateError")}</p> : null}
                </div>
              </div>
              <div className="min-w-0 space-y-2 lg:col-span-3">
                <h3 className={textRole.sectionTitle}>{t("page.configurationTitle")}</h3>
                <div className="radius-surface min-w-0 overflow-hidden border bg-muted/20">
                  <div className="flex min-h-10 items-center justify-between gap-2 border-b px-3">
                    <span className={cn("inline-flex min-w-0 items-center gap-2", textRole.monoLabel)}><IconCode className="size-3.5 shrink-0" aria-hidden="true" /><span className="truncate">{mcpClientFiles[client]}</span></span>
                    <CopyButton value={configuration} disabled={!configuration} copyLabel={t("configuration.copy")} copiedLabel={t("configuration.copied")} copyFailedLabel={t("configuration.copyFailed")} />
                  </div>
                  {configuration ? (
                    <pre className={cn("max-h-96 min-h-48 overflow-auto p-4 leading-5", textRole.code)} tabIndex={0} aria-label={t("configuration.label")}><code>{configuration}</code></pre>
                  ) : (
                    <div className="flex min-h-48 items-center justify-center p-4"><p className={cn("max-w-xs text-center", textRole.helperText)}>{t("page.configurationEmpty")}</p></div>
                  )}
                </div>
                <p className={textRole.helperText}>{t(`page.instructions.${client}`)}</p>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
      <ConfirmDialog open={confirmationOpen} onOpenChange={setConfirmationOpen} title={t("key.confirmation.title")} description={t("key.confirmation.description")} onConfirm={() => void generate(true)} loading={generateMcpKey.isPending} variant="destructive" confirmText={t("key.confirmation.confirm")} cancelText={t("key.confirmation.cancel")} processingText={t("key.confirmation.processing")} />
    </section>
  )
}
