"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import { semanticIcons } from "@/components/icons"
import { ConfirmDialog } from "@/components/shared/feedback/confirm-dialog"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { useGenerateMcpKey, useMcpKeyStatus } from "@/hooks/use-mcp-key"
import { textRole } from "@/lib/typography"
import { shellOverlaySideOffsets } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import type { McpKeyGeneration } from "@/types/mcp-key.types"

const MCP_PATH = "/mcp"
const mcpClientOptions = ["vscode", "cursor", "claudeCode", "codex"] as const

type McpClient = (typeof mcpClientOptions)[number]

function escapeTomlString(value: string) {
  return value.replaceAll("\\", "\\\\").replaceAll('"', '\\"')
}

export function getMcpClientConfiguration(client: McpClient, endpoint: string, key: string) {
  const authorization = `Bearer ${key}`

  switch (client) {
    case "vscode":
      return JSON.stringify({
        servers: {
          lunafox: {
            type: "http",
            url: endpoint,
            headers: { Authorization: authorization },
          },
        },
      }, null, 2)
    case "cursor":
      return JSON.stringify({
        mcpServers: {
          lunafox: {
            type: "streamable-http",
            url: endpoint,
            headers: { Authorization: authorization },
          },
        },
      }, null, 2)
    case "claudeCode":
      return JSON.stringify({
        mcpServers: {
          lunafox: {
            type: "http",
            url: endpoint,
            headers: { Authorization: authorization },
          },
        },
      }, null, 2)
    case "codex":
      return [
        "[mcp_servers.lunafox]",
        `url = "${escapeTomlString(endpoint)}"`,
        `http_headers = { Authorization = "${escapeTomlString(authorization)}" }`,
      ].join("\n")
  }
}

export function McpAccessPopover() {
  const t = useTranslations("mcpIntegration")
  const [open, setOpen] = React.useState(false)
  const [endpoint, setEndpoint] = React.useState(MCP_PATH)
  const [client, setClient] = React.useState<McpClient>("vscode")
  const [revealedKey, setRevealedKey] = React.useState<string | null>(null)
  const [regenerationConfirmationOpen, setRegenerationConfirmationOpen] = React.useState(false)
  const [wasRegenerated, setWasRegenerated] = React.useState(false)
  const generationWasRotationRef = React.useRef(false)
  const regenerationConfirmationOpenRef = React.useRef(false)
  const McpIcon = semanticIcons.concept.mcp
  const KeyIcon = semanticIcons.concept.apiKey
  const SuccessIcon = semanticIcons.status.success
  const RefreshIcon = semanticIcons.action.refresh
  const statusQuery = useMcpKeyStatus(open)

  const setRegenerationConfirmation = React.useCallback((nextOpen: boolean) => {
    regenerationConfirmationOpenRef.current = nextOpen
    setRegenerationConfirmationOpen(nextOpen)
  }, [])

  const handleGenerated = React.useCallback((generation: McpKeyGeneration) => {
    setRevealedKey(generation.key)
    setWasRegenerated(generationWasRotationRef.current)
    generationWasRotationRef.current = false
    setRegenerationConfirmation(false)
  }, [setRegenerationConfirmation])
  const generateMcpKey = useGenerateMcpKey(handleGenerated)

  const clearOneTimeKey = React.useCallback(() => {
    setRevealedKey(null)
    setWasRegenerated(false)
    generationWasRotationRef.current = false
    regenerationConfirmationOpenRef.current = false
    generateMcpKey.reset()
  }, [generateMcpKey])

  const handleOpenChange = React.useCallback((nextOpen: boolean) => {
    if (!nextOpen && regenerationConfirmationOpenRef.current) {
      return
    }
    setOpen(nextOpen)
    if (!nextOpen) {
      setRegenerationConfirmation(false)
      clearOneTimeKey()
    }
  }, [clearOneTimeKey, setRegenerationConfirmation])

  const requestGeneration = React.useCallback(async (rotation: boolean) => {
    generationWasRotationRef.current = rotation
    try {
      await generateMcpKey.mutateAsync()
    } catch {
      generationWasRotationRef.current = false
    }
  }, [generateMcpKey])

  const handleGenerate = React.useCallback(() => {
    void requestGeneration(false)
  }, [requestGeneration])

  const handleConfirmRegeneration = React.useCallback(() => {
    void requestGeneration(true)
  }, [requestGeneration])

  const clientConfiguration = React.useMemo(
    () => revealedKey ? getMcpClientConfiguration(client, endpoint, revealedKey) : "",
    [client, endpoint, revealedKey]
  )

  React.useEffect(() => {
    setEndpoint(`${window.location.origin}${MCP_PATH}`)
  }, [])

  const configured = statusQuery.data?.configured ?? false
  const statusUnavailable = statusQuery.isError
  const generationPending = generateMcpKey.isPending

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger render={<Button type="button" variant="ghost" size="icon-sm" aria-label={t("open")} />}>
        <McpIcon className="size-4" />
        <span className="sr-only">{t("open")}</span>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={shellOverlaySideOffsets.header} collisionPadding={12} className="w-80 p-0 sm:w-[28rem]">
        <div className="space-y-4 p-4">
          <div className="flex items-center gap-3">
            <div className="radius-control flex size-9 shrink-0 items-center justify-center bg-primary/10 text-primary">
              <McpIcon className="size-4" />
            </div>
            <div className="min-w-0">
              <p className={textRole.sectionTitle}>{t("title")}</p>
              <p className={cn("mt-0.5", textRole.helperText)}>{t("description")}</p>
            </div>
          </div>

          <Separator />

          <div className="space-y-2">
            <Label htmlFor="mcp-popover-url">{t("endpoint.label")}</Label>
            <div className="relative">
              <Input id="mcp-popover-url" value={endpoint} readOnly className="pr-10" />
              <CopyButton
                value={endpoint}
                copyLabel={t("endpoint.copy")}
                copiedLabel={t("endpoint.copied")}
                copyFailedLabel={t("endpoint.copyFailed")}
                className="absolute right-1 top-1/2 -translate-y-1/2"
              />
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <Label htmlFor="mcp-popover-key">{t("key.label")}</Label>
              {revealedKey ? (
                <span className={cn("inline-flex items-center gap-1 text-success", textRole.helperText)} aria-live="polite">
                  <SuccessIcon className="size-3.5" />
                  {t("key.revealed")}
                </span>
              ) : configured ? (
                <span className={cn("inline-flex items-center gap-1 text-success", textRole.helperText)}>
                  <SuccessIcon className="size-3.5" />
                  {t("key.configured")}
                </span>
              ) : null}
            </div>

            {statusUnavailable ? (
              <div className="flex items-center justify-between gap-3 border border-destructive/30 bg-destructive/10 p-2">
                <p className={textRole.helperText}>{t("key.statusError")}</p>
                <Button type="button" variant="outline" size="sm" onClick={() => void statusQuery.refetch()}>
                  {t("key.retry")}
                </Button>
              </div>
            ) : revealedKey ? (
              <>
                <div className="relative">
                  <Input id="mcp-popover-key" value={revealedKey} readOnly className="pr-10 font-mono" />
                  <CopyButton
                    value={revealedKey}
                    copyLabel={t("key.copy")}
                    copiedLabel={t("key.copied")}
                    copyFailedLabel={t("key.copyFailed")}
                    className="absolute right-1 top-1/2 -translate-y-1/2"
                  />
                </div>
                <p className={textRole.helperText}>{wasRegenerated ? t("key.rotatedDescription") : t("key.revealedDescription")}</p>
              </>
            ) : statusQuery.isPending ? (
              <Input id="mcp-popover-key" value={t("key.loading")} readOnly disabled />
            ) : (
              <div className="flex items-center justify-between gap-3">
                <p className={textRole.helperText}>{configured ? t("key.lostDescription") : t("key.emptyDescription")}</p>
                {configured ? (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    aria-label={t("key.regenerate")}
                    disabled={generationPending}
                    onClick={() => setRegenerationConfirmation(true)}
                  >
                    <RefreshIcon className="size-4" />
                    {t("key.regenerate")}
                  </Button>
                ) : (
                  <Button type="button" size="sm" disabled={generationPending} onClick={handleGenerate}>
                    <KeyIcon className="size-4" />
                    {generationPending ? t("key.generating") : t("key.generate")}
                  </Button>
                )}
              </div>
            )}

            {generateMcpKey.isError ? <p className={cn(textRole.helperText, "text-destructive")} role="alert">{t("key.generateError")}</p> : null}
          </div>

          {revealedKey ? (
            <div className="space-y-2">
              <Label htmlFor="mcp-popover-client">{t("configuration.label")}</Label>
              <Select value={client} onValueChange={(value) => setClient(value as McpClient)}>
                <SelectTrigger id="mcp-popover-client" size="sm" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {mcpClientOptions.map((option) => (
                    <SelectItem key={option} value={option}>
                      {t(`configuration.clients.${option}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <div className="relative">
                <CopyButton
                  value={clientConfiguration}
                  copyLabel={t("configuration.copy")}
                  copiedLabel={t("configuration.copied")}
                  copyFailedLabel={t("configuration.copyFailed")}
                  className="absolute right-1 top-1 z-10"
                />
                <pre className="radius-control max-h-48 overflow-auto border bg-muted/30 p-3 pr-11 font-mono text-xs leading-5 whitespace-pre-wrap break-all">
                  <code>{clientConfiguration}</code>
                </pre>
              </div>
              <p className={textRole.helperText}>{t("configuration.description")}</p>
            </div>
          ) : null}
        </div>

        <ConfirmDialog
          open={regenerationConfirmationOpen}
          onOpenChange={setRegenerationConfirmation}
          title={t("key.confirmation.title")}
          description={t("key.confirmation.description")}
          onConfirm={handleConfirmRegeneration}
          loading={generationPending}
          variant="destructive"
          confirmText={t("key.confirmation.confirm")}
          cancelText={t("key.confirmation.cancel")}
          processingText={t("key.confirmation.processing")}
        />
      </PopoverContent>
    </Popover>
  )
}
