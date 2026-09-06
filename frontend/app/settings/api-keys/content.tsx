"use client"

import { useEffect, useMemo, useState } from "react"
import { useTranslations } from "next-intl"

import { Eye, IconEyeOff, ExternalLink, ShieldCheck } from "@/components/icons"
import { PageHeader } from "@/components/common/page-header"
import { CopyButton } from "@/components/shared/feedback/copy-button"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { SearchInput } from "@/components/shared/search-input"
import { ApiKeysSettingsLoadingState } from "@/components/settings/api-keys/api-keys-settings-loading-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { useApiKeySettings, useUpdateApiKeySettings } from "@/hooks/use-api-key-settings"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import type {
  ApiKeyProviderDefinition,
  ApiKeyProviderUpdate,
  ApiKeySettings,
  ProviderStatus,
} from "@/types/api-key-settings.types"
import {
  API_KEYS_CARD_FOOTER_CLASS,
  API_KEYS_CARD_FOOTER_ACTION_GROUP_CLASS,
  API_KEYS_CARD_HEADER_CLASS,
  API_KEYS_CONTENT_SHELL_CLASS,
  API_KEYS_DETAIL_CONTENT_CLASS,
  API_KEYS_DETAIL_ENABLE_CLASS,
  API_KEYS_DETAIL_HEADER_LAYOUT_CLASS,
  API_KEYS_DETAIL_TITLE_ROW_CLASS,
  API_KEYS_DOCUMENTATION_LINK_ROW_CLASS,
  API_KEYS_FIELD_GROUP_CLASS,
  API_KEYS_MASTER_DETAIL_GRID_CLASS,
  API_KEYS_PAGE_SHELL_CLASS,
  API_KEYS_PASSWORD_COPY_ACTION_CLASS,
  API_KEYS_PASSWORD_INPUT_CLASS,
  API_KEYS_PASSWORD_INPUT_FIELD_SLOT_CLASS,
  API_KEYS_PASSWORD_INPUT_ROW_CLASS,
  API_KEYS_PROVIDER_AVATAR_CLASS,
  API_KEYS_PROVIDER_DETAIL_CARD_CLASS,
  API_KEYS_PROVIDER_LIST_CARD_CLASS,
  API_KEYS_PROVIDER_LIST_CONTENT_CLASS,
  API_KEYS_PROVIDER_LIST_HEADER_CLASS,
  API_KEYS_PROVIDER_ROW_CLASS,
  API_KEYS_PROVIDER_ROW_LABEL_CLASS,
  API_KEYS_PROVIDER_ROW_SELECTED_CLASS,
  API_KEYS_PROVIDER_SEARCH_INPUT_CLASS,
  API_KEYS_PROVIDER_SEARCH_ROW_CLASS,
  API_KEYS_PROVIDER_STATUS_SLOT_CLASS,
  API_KEYS_PROVIDER_SWITCH_SLOT_CLASS,
  API_KEYS_SECURITY_NOTICE_CARD_CLASS,
  API_KEYS_SECURITY_NOTICE_CONTENT_CLASS,
} from "./api-keys-settings-layout"

export interface ApiKeysSettingsPageProps {
  pageTitle: string
  pageDescription: string
  onReady?: () => void
  deferInitialSkeleton?: boolean
}

type ProviderFormState = Record<string, ApiKeyProviderUpdate>

function createDefaultSettings(): ApiKeySettings {
  return { providers: {}, definitions: [] }
}

function toFormState(settings: ApiKeySettings): ProviderFormState {
  return Object.fromEntries(
    settings.definitions.map((definition) => {
      const providerState = settings.providers[definition.key]
      return [
        definition.key,
        {
          enabled: providerState?.enabled ?? false,
          status: providerState?.status ?? "unconfigured",
          values: Object.fromEntries(
            definition.fields.map((field) => [field.name, providerState?.values?.[field.name]?.value ?? ""])
          ),
        },
      ]
    })
  )
}

function providerAbbreviation(displayName: string): string {
  const letters = displayName.replace(/[^a-zA-Z0-9]/g, "").slice(0, 2).toUpperCase()
  return letters || "--"
}

function fieldInputType(fieldName: string, secret: boolean): "text" | "password" {
  if (secret) return "password"
  if (fieldName.toLowerCase().includes("email")) return "text"
  return "text"
}

function getFieldLabel(fieldName: string, t: ReturnType<typeof useTranslations>): string {
  const known: Record<string, string> = {
    apiKey: t("fields.apiKey"),
    email: t("fields.email"),
    pat: t("fields.pat"),
    orgId: t("fields.orgId"),
    token: t("fields.token"),
    username: t("fields.username"),
    password: t("fields.password"),
    appId: t("fields.appId"),
    appSecret: t("fields.appSecret"),
    host: t("fields.host"),
    baseUrl: t("fields.baseUrl"),
    blobrKey: t("fields.blobrKey"),
  }
  return known[fieldName] ?? fieldName
}

function hasRequiredCredentials(provider: ApiKeyProviderDefinition, formData: ProviderFormState): boolean {
  const config = formData[provider.key]
  if (!config) return false
  return provider.fields.every((field) => !field.required || config.values[field.name]?.trim().length > 0)
}

function getProviderStatus(
  provider: ApiKeyProviderDefinition,
  formData: ProviderFormState,
  dirtyProviderKeys: Set<string>
): ProviderStatus {
  if (dirtyProviderKeys.has(provider.key)) return "pending"
  const config = formData[provider.key]
  if (!config) return "unconfigured"
  if (config.status === "requiresReconfiguration" || config.status === "unsupported") return config.status
  if (!hasRequiredCredentials(provider, formData)) return "unconfigured"
  return config.enabled ? "enabled" : "disabled"
}

function providerMatchesSearch(provider: ApiKeyProviderDefinition, query: string): boolean {
  const normalizedQuery = query.trim().toLowerCase()
  if (!normalizedQuery) return true
  return [provider.displayName, provider.sourceName, provider.key].some((value) =>
    value.toLowerCase().includes(normalizedQuery)
  )
}

function PasswordInput({
  value,
  onChange,
  placeholder,
  id,
  name,
  hideLabel,
  showLabel,
  copyLabel,
  copiedLabel,
  copyFailedLabel,
}: {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  id: string
  name: string
  hideLabel: string
  showLabel: string
  copyLabel: string
  copiedLabel: string
  copyFailedLabel: string
}) {
  const [show, setShow] = useState(false)

  return (
    <div className={API_KEYS_PASSWORD_INPUT_ROW_CLASS}>
      <div className={API_KEYS_PASSWORD_INPUT_FIELD_SLOT_CLASS}>
        <Input
          id={id}
          className={API_KEYS_PASSWORD_INPUT_CLASS}
          type={show ? "text" : "password"}
          name={name}
          autoComplete={show ? "off" : "current-password"}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
        />
        <CopyButton
          value={value}
          copyLabel={copyLabel}
          copiedLabel={copiedLabel}
          copyFailedLabel={copyFailedLabel}
          disabled={!value}
          size="icon-sm"
          className={API_KEYS_PASSWORD_COPY_ACTION_CLASS}
        />
      </div>
      <Button
        type="button"
        variant="outline"
        size="icon"
        aria-label={show ? hideLabel : showLabel}
        onClick={() => setShow((current) => !current)}
      >
        {show ? <IconEyeOff /> : <Eye />}
      </Button>
    </div>
  )
}

export default function ApiKeysSettingsPage({
  pageTitle,
  pageDescription,
  onReady,
  deferInitialSkeleton = false,
}: ApiKeysSettingsPageProps) {
  const t = useTranslations("pages.apiKeys")
  const { data: settings = createDefaultSettings(), isLoading } = useApiKeySettings()
  const updateMutation = useUpdateApiKeySettings()

  const [formData, setFormData] = useState<ProviderFormState>({})
  const [selectedProviderKey, setSelectedProviderKey] = useState<string>("")
  const [providerSearchQuery, setProviderSearchQuery] = useState("")
  const [dirtyProviderKeys, setDirtyProviderKeys] = useState<Set<string>>(new Set())
  const providers = settings.definitions
  const filteredProviders = useMemo(
    () => providers.filter((provider) => providerMatchesSearch(provider, providerSearchQuery)),
    [providers, providerSearchQuery]
  )

  const statusBadge: Record<ProviderStatus, { label: string; variant: "success" | "outline" | "warning" | "destructive" }> = {
    configured: { label: t("status.enabled"), variant: "success" },
    enabled: { label: t("status.enabled"), variant: "success" },
    unconfigured: { label: t("status.unconfigured"), variant: "outline" },
    disabled: { label: t("status.disabled"), variant: "outline" },
    pending: { label: t("status.pending"), variant: "warning" },
    requiresReconfiguration: { label: t("status.requiresReconfiguration"), variant: "warning" },
    unsupported: { label: t("status.unsupported"), variant: "destructive" },
  }

  useEffect(() => {
    if (!settings.definitions.length) return
    setFormData(toFormState(settings))
    setDirtyProviderKeys(new Set())
    setSelectedProviderKey((current) => current && settings.definitions.some((provider) => provider.key === current) ? current : settings.definitions[0].key)
  }, [settings])

  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.key === selectedProviderKey) ?? providers[0],
    [providers, selectedProviderKey]
  )
  const selectedConfig = selectedProvider ? formData[selectedProvider.key] : undefined
  const selectedStatus = selectedProvider ? getProviderStatus(selectedProvider, formData, dirtyProviderKeys) : "unconfigured"
  const hasChanges = dirtyProviderKeys.size > 0

  const updateProvider = (
    providerKey: string,
    field: string,
    value: string | boolean
  ) => {
    setFormData((previous) => {
      const current = previous[providerKey] ?? { enabled: false, values: {} }
      const updated = field === "enabled"
        ? { ...current, enabled: Boolean(value) }
        : { ...current, values: { ...current.values, [field]: String(value) } }

      return {
        ...previous,
        [providerKey]: updated,
      }
    })
    setDirtyProviderKeys((previous) => new Set(previous).add(providerKey))
  }

  const handleSave = () => {
    updateMutation.mutate({
      providers: Object.fromEntries([...dirtyProviderKeys].map((providerKey) => [providerKey, formData[providerKey]])),
    })
    setDirtyProviderKeys(new Set())
  }

  useEffect(() => {
    if (isLoading) return
    onReady?.()
  }, [isLoading, onReady])

  if (isLoading && !deferInitialSkeleton) {
    return (
      <ApiKeysSettingsLoadingState
        owner="api-keys-page"
        pageTitle={pageTitle}
        pageDescription={pageDescription}
        enableLabel={t("enableLabel")}
        emailLabel={t("fields.email")}
        apiKeyLabel={t("fields.apiKey")}
        searchPlaceholder={t("searchPlaceholder")}
        getApiKeyLabel={t("getApiKey")}
        saveLabel={t("save")}
        securityNoticeLabel={t("securityNotice.label")}
        securityNoticeDescription={t("securityNotice.description")}
      />
    )
  }
  if (isLoading) return null

  return (
    <div className={API_KEYS_PAGE_SHELL_CLASS}>
      <div {...getLoadingStructureSlotAttributes("api-keys-header")}>
        <PageHeader code="API-01" title={pageTitle} description={pageDescription} />
      </div>

      <div className={API_KEYS_CONTENT_SHELL_CLASS}>
        <div className={API_KEYS_MASTER_DETAIL_GRID_CLASS}>
          <Card
            {...getLoadingStructureSlotAttributes("api-keys-provider-list")}
            data-testid="api-key-provider-list"
            className={API_KEYS_PROVIDER_LIST_CARD_CLASS}
          >
            <CardHeader className={API_KEYS_PROVIDER_LIST_HEADER_CLASS} density="compact">
              <div className={API_KEYS_PROVIDER_SEARCH_ROW_CLASS}>
                <SearchInput
                  className={API_KEYS_PROVIDER_SEARCH_INPUT_CLASS}
                  value={providerSearchQuery}
                  onChange={(event) => setProviderSearchQuery(event.target.value)}
                  placeholder={t("searchPlaceholder")}
                  aria-label={t("searchPlaceholder")}
                />
              </div>
            </CardHeader>
            <CardContent className={API_KEYS_PROVIDER_LIST_CONTENT_CLASS}>
              <div role="radiogroup" aria-label={t("providerGroupAriaLabel")}>
                {filteredProviders.map((provider) => {
                  const status = getProviderStatus(provider, formData, dirtyProviderKeys)
                  const statusMeta = statusBadge[status]
                  const selected = provider.key === selectedProvider?.key

                  return (
                    <div
                      key={provider.key}
                      data-source-name={provider.sourceName}
                      className={cn(
                        API_KEYS_PROVIDER_ROW_CLASS,
                        selected && API_KEYS_PROVIDER_ROW_SELECTED_CLASS
                      )}
                    >
                      <input
                        id={`api-key-provider-${provider.key}`}
                        className="sr-only"
                        type="radio"
                        name="api-key-provider"
                        value={provider.key}
                        checked={selected}
                        onChange={() => setSelectedProviderKey(provider.key)}
                      />
                      <label
                        htmlFor={`api-key-provider-${provider.key}`}
                        className={API_KEYS_PROVIDER_ROW_LABEL_CLASS}
                      >
                        <span className={cn(
                          API_KEYS_PROVIDER_AVATAR_CLASS,
                          textRole.metadataValueStrong
                        )}>
                          {providerAbbreviation(provider.displayName)}
                        </span>
                        <span className={cn("min-w-0 flex-1 truncate", textRole.bodyStrong)}>
                          {provider.displayName}
                        </span>
                        <span className={API_KEYS_PROVIDER_STATUS_SLOT_CLASS}>
                          <Badge className="shrink-0" variant={statusMeta.variant}>{statusMeta.label}</Badge>
                        </span>
                      </label>
                      <div className={API_KEYS_PROVIDER_SWITCH_SLOT_CLASS}>
                        <Switch
                          aria-label={t("providerToggleAriaLabel", { name: provider.displayName })}
                          checked={formData[provider.key]?.enabled ?? false}
                          onCheckedChange={(checked) => updateProvider(provider.key, "enabled", checked)}
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </CardContent>
          </Card>

          {selectedProvider && selectedConfig ? (
            <Card
              {...getLoadingStructureSlotAttributes("api-keys-provider-detail")}
              data-testid="api-key-provider-detail"
              className={API_KEYS_PROVIDER_DETAIL_CARD_CLASS}
            >
              <CardHeader className={API_KEYS_CARD_HEADER_CLASS} density="compact">
                <div className={API_KEYS_DETAIL_HEADER_LAYOUT_CLASS}>
                  <div>
                    <div className={API_KEYS_DETAIL_TITLE_ROW_CLASS}>
                      <CardTitle className={textRole.panelTitle}>{selectedProvider.displayName}</CardTitle>
                      <Badge variant={statusBadge[selectedStatus].variant}>{statusBadge[selectedStatus].label}</Badge>
                    </div>
                  </div>
                  <div className={API_KEYS_DETAIL_ENABLE_CLASS}>
                    <span className={textRole.bodyStrong}>{t("enableLabel")}</span>
                    <Switch
                      aria-label={t("providerToggleAriaLabel", { name: selectedProvider.displayName })}
                      checked={selectedConfig.enabled}
                      onCheckedChange={(checked) => updateProvider(selectedProvider.key, "enabled", checked)}
                    />
                  </div>
                </div>
              </CardHeader>

              <CardContent className={API_KEYS_DETAIL_CONTENT_CLASS}>
                {selectedProvider.fields.map((field) => {
                  const value = selectedConfig.values[field.name] ?? ""
                  const id = `${selectedProvider.key}-${field.name}`

                  return (
                    <div key={field.name} className={API_KEYS_FIELD_GROUP_CLASS}>
                      <Label htmlFor={id}>
                        {getFieldLabel(field.name, t)}
                        {field.required ? <span aria-hidden="true" className="ml-1 text-destructive">*</span> : null}
                      </Label>
                      {fieldInputType(field.name, field.secret) === "password" ? (
                        <PasswordInput
                          id={id}
                          name={id}
                          value={value}
                          onChange={(nextValue) => updateProvider(selectedProvider.key, field.name, nextValue)}
                          placeholder={t("fieldPlaceholder", { field: getFieldLabel(field.name, t) })}
                          hideLabel={t("password.hide")}
                          showLabel={t("password.show")}
                          copyLabel={t("password.copy")}
                          copiedLabel={t("password.copied")}
                          copyFailedLabel={t("password.copyFailed")}
                        />
                      ) : (
                        <Input
                          id={id}
                          name={id}
                          type="text"
                          autoComplete="off"
                          value={value}
                          onChange={(event) => updateProvider(selectedProvider.key, field.name, event.target.value)}
                          placeholder={t("fieldPlaceholder", { field: getFieldLabel(field.name, t) })}
                        />
                      )}
                    </div>
                  )
                })}

                {selectedProvider.docsUrl ? (
                  <div className={API_KEYS_DOCUMENTATION_LINK_ROW_CLASS}>
                    <a
                      href={selectedProvider.docsUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={cn("inline-flex items-center gap-1 text-primary underline-offset-4 hover:underline", textRole.helperText)}
                    >
                      {t("getApiKey")}
                      <ExternalLink className="size-3" />
                    </a>
                  </div>
                ) : null}
              </CardContent>

              <CardFooter className={API_KEYS_CARD_FOOTER_CLASS} density="compact">
                <div className={API_KEYS_CARD_FOOTER_ACTION_GROUP_CLASS}>
                  <Button
                    type="button"
                    onClick={handleSave}
                    disabled={updateMutation.isPending || !hasChanges}
                  >
                    {updateMutation.isPending ? t("saving") : t("save")}
                  </Button>
                </div>
              </CardFooter>
            </Card>
          ) : null}
        </div>

        <Card
          {...getLoadingStructureSlotAttributes("api-keys-notice")}
          className={API_KEYS_SECURITY_NOTICE_CARD_CLASS}
        >
          <CardContent className={API_KEYS_SECURITY_NOTICE_CONTENT_CLASS}>
            <ShieldCheck className="mt-0.5 size-4 shrink-0" />
            <p className={textRole.bodySubtle}>
              <span className={textRole.bodyStrong}>{t("securityNotice.label")}</span>
              {t("securityNotice.description")}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
