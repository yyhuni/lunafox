"use client"

import * as React from "react"
import { useTranslations } from "next-intl"
import { useQueryClient } from "@tanstack/react-query"

import {
  IconAlertTriangle,
  IconBrandDiscord,
  IconBrandSlack,
  IconEye,
  IconEyeOff,
  IconPlayerPlay,
  semanticIcons,
} from "@/components/icons"
import { PageHeader } from "@/components/common/page-header"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { Spinner } from "@/components/shared/loading/spinner"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  useNotificationDestinations,
  useTestNotificationDestination,
  useUpdateNotificationDestination,
  notificationSettingsKeys,
} from "@/hooks/use-notification-settings"
import { normalizeError } from "@/lib/errors/normalize-error"
import {
  getNotificationProviderIconClass,
  getNotificationProviderIconSurfaceClass,
} from "@/lib/notification-provider-config"
import { useToastMessages } from "@/lib/toast-helpers"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  NOTIFICATION_DESTINATION_PROVIDERS,
  type NotificationDestination,
  type NotificationDestinationListResponse,
  type NotificationDestinationProvider,
  type NotificationDestinationTestResult,
} from "@/types/notification-settings.types"
import type { ExternalNotificationKind } from "@/types/notification.types"

import {
  NOTIFICATION_CHANNEL_ICON_CLASS,
  NOTIFICATION_CHANNELS_STACK_CLASS,
  NOTIFICATION_CREDENTIAL_FIELD_CLASS,
  NOTIFICATION_CREDENTIAL_INPUT_CLASS,
  NOTIFICATION_CREDENTIAL_INPUT_SHELL_CLASS,
  NOTIFICATION_CREDENTIAL_REVEAL_ACTION_CLASS,
  NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS,
  NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS,
  NOTIFICATION_SETTINGS_ACTION_BAR_CLASS,
  NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS,
  NOTIFICATION_SETTINGS_CONTENT_SHELL_CLASS,
  NOTIFICATION_SETTINGS_PAGE_SHELL_CLASS,
  NOTIFICATION_SETTINGS_WORKSPACE_STATE_CLASS,
  NOTIFICATION_SUBSCRIPTION_ITEM_CLASS,
  NOTIFICATION_SUBSCRIPTIONS_LIST_CLASS,
  NOTIFICATION_SUBSCRIPTIONS_SECTION_CLASS,
} from "./notification-settings-layout"
import { NotificationSettingsPageLoadingState } from "./notification-settings-loading-state"

export interface NotificationSettingsPageContentProps {
  pageTitle: string
  pageDescription: string
  onReady?: () => void
  deferInitialSkeleton?: boolean
}

type DestinationDraft = Pick<NotificationDestination, "credential" | "enabled" | "subscriptions">
type DestinationDrafts = Partial<Record<NotificationDestinationProvider, DestinationDraft>>
type DestinationBaselines = Partial<Record<NotificationDestinationProvider, NotificationDestination>>
type ProviderMessages = Partial<Record<NotificationDestinationProvider, string>>

const SAVE_TOAST_ID = "notification-settings-save"

const DESTINATION_KIND_LABEL_KEYS: Record<ExternalNotificationKind, string> = {
  "scan-succeeded": "subscriptions.scanSucceeded",
  "scan-failed": "subscriptions.scanFailed",
  "vulnerability-observed": "subscriptions.vulnerabilityObserved",
  "agent-offline": "subscriptions.agentOffline",
}

function cloneDestinationDraft(destination: NotificationDestination): DestinationDraft {
  return {
    credential: destination.credential,
    enabled: destination.enabled,
    subscriptions: [...destination.subscriptions],
  }
}

function draftsMatch(left: DestinationDraft, right: DestinationDraft): boolean {
  if (left.credential !== right.credential || left.enabled !== right.enabled || left.subscriptions.length !== right.subscriptions.length) {
    return false
  }
  return left.subscriptions.every((subscription, index) => subscription === right.subscriptions[index])
}

function NotificationProviderIcon({ provider }: { provider: NotificationDestinationProvider }) {
  const Icon = provider === "discord"
    ? IconBrandDiscord
    : provider === "wecom"
      ? IconBrandSlack
      : semanticIcons.concept.notification

  return (
    <span className={cn(NOTIFICATION_CHANNEL_ICON_CLASS, getNotificationProviderIconSurfaceClass(provider))}>
      <Icon aria-hidden="true" className={cn("size-5", getNotificationProviderIconClass(provider))} />
    </span>
  )
}

function NotificationDestinationEditor({
  provider,
  draft,
  baseline,
  supportedKinds,
  isSavePending,
  isTestPending,
  errorMessage,
  testResult,
  onDraftChange,
  onTest,
}: {
  provider: NotificationDestinationProvider
  draft: DestinationDraft
  baseline: NotificationDestination
  supportedKinds: readonly ExternalNotificationKind[]
  isSavePending: boolean
  isTestPending: boolean
  errorMessage?: string
  testResult?: NotificationDestinationTestResult
  onDraftChange: (next: Partial<DestinationDraft>) => void
  onTest: () => void
}) {
  const t = useTranslations("settings.notifications")
  const [showCredential, setShowCredential] = React.useState(false)
  const credentialId = React.useId()
  const providerTitle = t(`${provider}.title`)
  const isLocked = isSavePending || isTestPending

  const toggleSubscription = (kind: ExternalNotificationKind, checked: boolean) => {
    onDraftChange({
      subscriptions: checked
        ? [...new Set([...draft.subscriptions, kind])]
        : draft.subscriptions.filter((item) => item !== kind),
    })
  }

  return (
    <Collapsible open={draft.enabled} className="flex min-w-0 flex-1 flex-col">
      <CardHeader className={NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS}>
        <div className={NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS}>
          <div className="min-w-0">
            <div className={NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS}>
              <NotificationProviderIcon provider={provider} />
              <CardTitle className={NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS}>{providerTitle}</CardTitle>
              <Badge variant={draft.enabled ? "success" : "outline"}>
                {draft.enabled ? t("status.enabled") : t("status.disabled")}
              </Badge>
            </div>
            <CardDescription>{t(`${provider}.description`)}</CardDescription>
          </div>
          <Switch
            aria-label={t("destination.enabledLabel", { provider: providerTitle })}
            checked={draft.enabled}
            disabled={isLocked}
            onCheckedChange={(enabled) => onDraftChange({ enabled })}
          />
        </div>
      </CardHeader>
      <CollapsibleContent data-testid={`notification-channel-details-${provider}`} className="min-h-0 flex-1">
        <CardContent className={NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS}>
          {baseline.requiresWebhookUpdate ? (
            <Alert className="border-warning/20 bg-warning/10 text-foreground">
              <IconAlertTriangle aria-hidden="true" className="text-warning" />
              <AlertDescription>{t("remediation.required")}</AlertDescription>
            </Alert>
          ) : null}

          <div className={NOTIFICATION_CREDENTIAL_FIELD_CLASS}>
            <Label htmlFor={credentialId}>{t("destination.credentialLabel")}</Label>
            <div className={NOTIFICATION_CREDENTIAL_INPUT_SHELL_CLASS}>
              <Input
                id={credentialId}
                className={NOTIFICATION_CREDENTIAL_INPUT_CLASS}
                type={showCredential ? "text" : "password"}
                autoComplete="off"
                inputMode="url"
                placeholder={t(`${provider}.credentialPlaceholder`)}
                value={draft.credential}
                disabled={isLocked}
                onChange={(event) => onDraftChange({ credential: event.target.value })}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className={NOTIFICATION_CREDENTIAL_REVEAL_ACTION_CLASS}
                aria-label={showCredential ? t("destination.hideCredential") : t("destination.showCredential")}
                disabled={isLocked}
                onClick={() => setShowCredential((current) => !current)}
              >
                {showCredential ? <IconEyeOff className="size-4" /> : <IconEye className="size-4" />}
              </Button>
            </div>
            <p className={textRole.helperText}>{t("destination.credentialHelp")}</p>
          </div>

          <div className={NOTIFICATION_SUBSCRIPTIONS_SECTION_CLASS}>
            <Label>{t("subscriptions.title")}</Label>
            <p className={textRole.helperText}>{t("subscriptions.description")}</p>
            <div className={NOTIFICATION_SUBSCRIPTIONS_LIST_CLASS}>
              {supportedKinds.map((kind) => {
                const checkboxId = `${credentialId}-${kind}`
                return (
                  <div key={kind} className={NOTIFICATION_SUBSCRIPTION_ITEM_CLASS}>
                    <Checkbox
                      id={checkboxId}
                      checked={draft.subscriptions.includes(kind)}
                      disabled={isLocked}
                      onCheckedChange={(checked) => toggleSubscription(kind, checked === true)}
                    />
                    <Label htmlFor={checkboxId} className={cn("cursor-pointer", textRole.bodySubtle)}>
                      {t(DESTINATION_KIND_LABEL_KEYS[kind])}
                    </Label>
                  </div>
                )
              })}
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Button type="button" variant="outline" disabled={isLocked} onClick={onTest}>
              {isTestPending ? <Spinner className="size-4" /> : <IconPlayerPlay className="size-4" />}
              {t("test.action")}
            </Button>
            {testResult ? <p role={testResult === "delivered" ? "status" : "alert"} className={textRole.helperText}>{t(`test.result.${testResult}`)}</p> : null}
          </div>
          {errorMessage ? <p role="alert" className={cn("text-destructive", textRole.helperText)}>{errorMessage}</p> : null}
        </CardContent>
      </CollapsibleContent>
    </Collapsible>
  )
}

export default function NotificationSettingsPageContent({
  pageTitle,
  pageDescription,
  onReady,
  deferInitialSkeleton = false,
}: NotificationSettingsPageContentProps) {
  const [isRouteBoundaryEntry] = React.useState(deferInitialSkeleton)
  const [destinationBaselines, setDestinationBaselines] = React.useState<DestinationBaselines>({})
  const [destinationDrafts, setDestinationDrafts] = React.useState<DestinationDrafts>({})
  const [providerErrors, setProviderErrors] = React.useState<ProviderMessages>({})
  const [testResults, setTestResults] = React.useState<Partial<Record<NotificationDestinationProvider, NotificationDestinationTestResult>>>({})
  const [savePending, setSavePending] = React.useState(false)
  const [testingProviders, setTestingProviders] = React.useState<NotificationDestinationProvider[]>([])
  const destinationsQuery = useNotificationDestinations()
  const updateDestination = useUpdateNotificationDestination()
  const testDestination = useTestNotificationDestination()
  const toast = useToastMessages()
  const t = useTranslations("settings.notifications")
  const queryClient = useQueryClient()
  const isInitialLoading = destinationsQuery.isLoading && !destinationsQuery.data
  const destinationMap = new Map(
    destinationsQuery.data?.results.map((destination) => [destination.provider, destination])
  )
  const hasAllDestinationProviders = NOTIFICATION_DESTINATION_PROVIDERS.every((provider) => destinationMap.has(provider))
  const destinationContractError = destinationsQuery.data && !hasAllDestinationProviders
    ? new Error("Notification destinations response omitted a required provider.")
    : null
  const settingsError = destinationsQuery.isError
    ? destinationsQuery.error
    : destinationContractError

  React.useEffect(() => {
    if (!isInitialLoading) {
      onReady?.()
    }
  }, [isInitialLoading, onReady])

  React.useEffect(() => {
    const destinations = destinationsQuery.data?.results
    if (!destinations) return

    setDestinationBaselines((current) => {
      const next = { ...current }
      let changed = false
      for (const destination of destinations) {
        if (!next[destination.provider]) {
          next[destination.provider] = destination
          changed = true
        }
      }
      return changed ? next : current
    })
    setDestinationDrafts((current) => {
      const next = { ...current }
      let changed = false
      for (const destination of destinations) {
        if (!next[destination.provider]) {
          next[destination.provider] = cloneDestinationDraft(destination)
          changed = true
        }
      }
      return changed ? next : current
    })
  }, [destinationsQuery.data])

  const dirtyProviders = NOTIFICATION_DESTINATION_PROVIDERS.filter((provider) => {
    const baseline = destinationBaselines[provider]
    const draft = destinationDrafts[provider]
    return baseline !== undefined && draft !== undefined && !draftsMatch(draft, cloneDestinationDraft(baseline))
  })
  const hasDirtyDrafts = dirtyProviders.length > 0
  const hasPendingTests = testingProviders.length > 0

  React.useEffect(() => {
    if (!hasDirtyDrafts) return

    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ""
    }
    window.addEventListener("beforeunload", onBeforeUnload)
    return () => window.removeEventListener("beforeunload", onBeforeUnload)
  }, [hasDirtyDrafts])

  const updateDestinationDraft = (
    provider: NotificationDestinationProvider,
    next: Partial<DestinationDraft>,
  ) => {
    const baseline = destinationBaselines[provider] ?? destinationMap.get(provider)
    if (!baseline) return

    setProviderErrors((current) => ({ ...current, [provider]: undefined }))
    setTestResults((current) => ({ ...current, [provider]: undefined }))
    setDestinationDrafts((current) => ({
      ...current,
      [provider]: {
        ...(current[provider] ?? cloneDestinationDraft(baseline)),
        ...next,
      },
    }))
  }

  const validateDraft = (draft: DestinationDraft): string | null => {
    if (draft.enabled && !draft.credential.trim()) return t("validation.credentialRequired")
    if (draft.enabled && draft.subscriptions.length === 0) return t("validation.subscriptionRequired")
    return null
  }

  const handleSave = async () => {
    if (savePending || hasPendingTests || dirtyProviders.length === 0) return

    const validationErrors: ProviderMessages = {}
    for (const provider of dirtyProviders) {
      const draft = destinationDrafts[provider]
      if (!draft) continue
      const validationError = validateDraft(draft)
      if (validationError) validationErrors[provider] = validationError
    }
    if (Object.keys(validationErrors).length > 0) {
      setProviderErrors((current) => ({ ...current, ...validationErrors }))
      return
    }

    const submitted = dirtyProviders.flatMap((provider) => {
      const draft = destinationDrafts[provider]
      return draft ? [{ provider, draft: { ...draft, credential: provider === "feishu" ? draft.credential : draft.credential.trim(), subscriptions: [...draft.subscriptions] } }] : []
    })
    if (submitted.length === 0) return

    setSavePending(true)
    setProviderErrors({})
    toast.loading("toast.notification.destinationBatch.loading", undefined, SAVE_TOAST_ID)
    const results = await Promise.allSettled(
      submitted.map(({ provider, draft }) => updateDestination.mutateAsync({ provider, data: draft }))
    )
    const successful = results.flatMap((result, index) => result.status === "fulfilled"
      ? [{ provider: submitted[index].provider, destination: result.value }]
      : [])
    const failed = results.flatMap((result, index) => result.status === "rejected"
      ? [submitted[index].provider]
      : [])

    if (successful.length > 0) {
      setDestinationBaselines((current) => ({
        ...current,
        ...Object.fromEntries(successful.map(({ provider, destination }) => [provider, destination])),
      }))
      setDestinationDrafts((current) => ({
        ...current,
        ...Object.fromEntries(successful.map(({ provider, destination }) => [provider, cloneDestinationDraft(destination)])),
      }))
      queryClient.setQueryData<NotificationDestinationListResponse>(
        notificationSettingsKeys.destinations(),
        (current) => current
          ? {
              ...current,
              results: current.results.map((destination) => successful.find(({ provider }) => provider === destination.provider)?.destination ?? destination),
            }
          : current,
      )
    }
    if (failed.length > 0) {
      setProviderErrors((current) => ({
        ...current,
        ...Object.fromEntries(failed.map((provider) => [provider, t("destination.saveFailed")])),
      }))
    }

    if (failed.length === 0) {
      toast.success("toast.notification.destinationBatch.success", undefined, SAVE_TOAST_ID)
    } else if (successful.length > 0) {
      toast.warning("toast.notification.destinationBatch.partial", undefined, SAVE_TOAST_ID)
    } else {
      toast.error("toast.notification.destinationBatch.error", undefined, SAVE_TOAST_ID)
    }
    setSavePending(false)
  }

  const handleTest = async (provider: NotificationDestinationProvider) => {
    if (savePending || testingProviders.includes(provider)) return
    const draft = destinationDrafts[provider]
    if (!draft) return

    setTestingProviders((current) => [...current, provider])
    setProviderErrors((current) => ({ ...current, [provider]: undefined }))
    setTestResults((current) => ({ ...current, [provider]: undefined }))
    try {
      const response = await testDestination.mutateAsync({
        provider,
        data: { credential: provider === "feishu" ? draft.credential : draft.credential.trim() },
      })
      setTestResults((current) => ({ ...current, [provider]: response.result }))
    } catch {
      setTestResults((current) => ({ ...current, [provider]: "internal_failure" }))
    } finally {
      setTestingProviders((current) => current.filter((currentProvider) => currentProvider !== provider))
    }
  }

  const retrySettings = () => {
    void destinationsQuery.refetch()
  }

  if (isInitialLoading && deferInitialSkeleton) {
    return null
  }

  const pageContent = (
    <div className={NOTIFICATION_SETTINGS_PAGE_SHELL_CLASS}>
      <header {...getLoadingStructureSlotAttributes("notification-settings-header")}>
        <PageHeader code="NTF-01" title={pageTitle} description={pageDescription} />
      </header>

      <div className={NOTIFICATION_SETTINGS_CONTENT_SHELL_CLASS}>
        {settingsError ? (
          <AppErrorState
            variant="section"
            error={normalizeError(settingsError, { notFoundKind: "unexpected-error" })}
            onRetry={retrySettings}
          />
        ) : destinationsQuery.data ? (
          <div
            {...getLoadingStructureSlotAttributes("notification-settings-channel-workbench")}
            className={NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS}
          >
            <div className={NOTIFICATION_CHANNELS_STACK_CLASS} data-testid="notification-channel-editors">
              {NOTIFICATION_DESTINATION_PROVIDERS.map((provider) => {
                const destination = destinationBaselines[provider] ?? destinationMap.get(provider)
                if (!destination) return null
                const draft = destinationDrafts[provider] ?? cloneDestinationDraft(destination)

                return (
                  <Card key={provider} className={NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS} data-testid={`notification-channel-editor-${provider}`}>
                    <NotificationDestinationEditor
                      provider={provider}
                      draft={draft}
                      baseline={destination}
                      supportedKinds={destinationsQuery.data.supportedKinds}
                      isSavePending={savePending}
                      isTestPending={testingProviders.includes(provider)}
                      errorMessage={providerErrors[provider]}
                      testResult={testResults[provider]}
                      onDraftChange={(next) => updateDestinationDraft(provider, next)}
                      onTest={() => void handleTest(provider)}
                    />
                  </Card>
                )
              })}
              <div className={NOTIFICATION_SETTINGS_ACTION_BAR_CLASS}>
                <Button type="button" disabled={!hasDirtyDrafts || savePending || hasPendingTests} onClick={() => void handleSave()}>
                  {savePending ? <Spinner className="size-4" /> : null}
                  {t("destination.saveAll")}
                </Button>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  )

  if (isRouteBoundaryEntry) {
    return pageContent
  }

  return (
    <ContentHandoff
      owner="notification-settings-page-content"
      isLoading={isInitialLoading}
      skeleton={(
        <NotificationSettingsPageLoadingState
          pageTitle={pageTitle}
          pageDescription={pageDescription}
          destinations={destinationsQuery.data?.results}
          supportedKindCount={destinationsQuery.data?.supportedKinds.length}
        />
      )}
      className={NOTIFICATION_SETTINGS_WORKSPACE_STATE_CLASS}
    >
      {pageContent}
    </ContentHandoff>
  )
}
