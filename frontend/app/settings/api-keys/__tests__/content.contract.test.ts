import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "app/settings/api-keys/content.tsx"), "utf8")
const layoutSource = readFileSync(path.resolve(process.cwd(), "app/settings/api-keys/api-keys-settings-layout.ts"), "utf8")
const loadingStateSource = readFileSync(
  path.resolve(process.cwd(), "components/settings/api-keys/api-keys-settings-loading-state.tsx"),
  "utf8"
)
const resolvedPageSource = source.slice(source.indexOf("export default function ApiKeysSettingsPage"))

describe("content contract", () => {
  it("keeps the API keys page scoped to subfinder data-source credentials", () => {
    expect(source).toContain("export default function ApiKeysSettingsPage")
    expect(source).toContain('useTranslations("pages.apiKeys")')
    expect(source).toContain("pageTitle: string")
    expect(source).toContain("pageDescription: string")
    expect(source).toContain("<PageHeader code=\"API-01\" title={pageTitle} description={pageDescription} />")
    expect(source).not.toContain('const pageTitle = t("title")')
    expect(source).not.toContain('const pageDescription = t("description")')
    expect(source).toContain("useApiKeySettings")
    expect(source).not.toContain("系统开放 API")
  })

  it("uses the reference master-detail structure without replacing the app shell", () => {
    expect(source).toContain("api-key-provider-list")
    expect(source).toContain("api-key-provider-detail")
    expect(source).toContain("selectedProviderKey")
    expect(source).toContain("ApiKeysSettingsLoadingState")
    expect(source).toContain('from "@/components/settings/api-keys/api-keys-settings-loading-state"')
    expect(source).not.toContain("export function ApiKeysSettingsLoadingState")
    expect(source).not.toContain("api-keys-settings-skeleton")
    expect(source).toContain("API_KEYS_MASTER_DETAIL_GRID_CLASS")
    expect(source).toContain("<PageHeader")
    expect(source).toContain("code=\"API-01\"")
    expect(source).not.toContain('Skeleton className="h-full min-h-96')
  })

  it("derives the resolved page and loading state geometry from the same layout contract", () => {
    const sharedLayoutConstants = [
      "API_KEYS_PAGE_SHELL_CLASS",
      "API_KEYS_CONTENT_SHELL_CLASS",
      "API_KEYS_MASTER_DETAIL_GRID_CLASS",
      "API_KEYS_PROVIDER_LIST_CARD_CLASS",
      "API_KEYS_PROVIDER_DETAIL_CARD_CLASS",
      "API_KEYS_CARD_HEADER_CLASS",
      "API_KEYS_PROVIDER_ROW_CLASS",
      "API_KEYS_PROVIDER_ROW_SELECTED_CLASS",
      "API_KEYS_PROVIDER_AVATAR_CLASS",
      "API_KEYS_PROVIDER_STATUS_SLOT_CLASS",
      "API_KEYS_PROVIDER_SWITCH_SLOT_CLASS",
      "API_KEYS_DETAIL_HEADER_LAYOUT_CLASS",
      "API_KEYS_DETAIL_ENABLE_CLASS",
      "API_KEYS_DETAIL_CONTENT_CLASS",
      "API_KEYS_CARD_FOOTER_CLASS",
      "API_KEYS_CARD_FOOTER_ACTION_GROUP_CLASS",
      "API_KEYS_FIELD_GROUP_CLASS",
      "API_KEYS_PASSWORD_INPUT_ROW_CLASS",
      "API_KEYS_PASSWORD_INPUT_FIELD_SLOT_CLASS",
      "API_KEYS_PASSWORD_INPUT_CLASS",
      "API_KEYS_PASSWORD_COPY_ACTION_CLASS",
      "API_KEYS_DOCUMENTATION_LINK_ROW_CLASS",
      "API_KEYS_SECURITY_NOTICE_CARD_CLASS",
      "API_KEYS_SECURITY_NOTICE_CONTENT_CLASS",
    ]

    expect(source).toContain('from "./api-keys-settings-layout"')
    expect(loadingStateSource).toContain('from "@/app/settings/api-keys/api-keys-settings-layout"')
    for (const constant of sharedLayoutConstants) {
      expect(layoutSource).toContain(`export const ${constant}`)
      expect(source).toContain(constant)
      expect(loadingStateSource).toContain(constant)
    }
    expect(layoutSource).toContain("export const API_KEYS_PROVIDER_ROW_LABEL_CLASS")
    expect(layoutSource).toContain("export const API_KEYS_PROVIDER_ROW_CONTENT_CLASS")
    expect(source).toContain("API_KEYS_PROVIDER_ROW_LABEL_CLASS")
    expect(loadingStateSource).toContain("API_KEYS_PROVIDER_ROW_CONTENT_CLASS")

    expect(source).not.toContain('className="flex min-h-0 flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6"')
    expect(source).not.toContain('className="grid min-h-0 flex-1 items-stretch gap-4 lg:grid-cols-5"')
    expect(source).not.toContain('className="flex gap-2"')
    expect(source).not.toContain('className="relative flex-1"')
    expect(source).not.toContain('className="flex flex-wrap items-center gap-2"')
  })

  it("keeps the route fallback and data loading on the same page-specific loading owner", () => {
    expect(source).toContain("export interface ApiKeysSettingsPageProps")
    expect(loadingStateSource).toContain("export function ApiKeysSettingsLoadingState")
    expect(source).toContain("onReady?: () => void")
    expect(source).toContain("deferInitialSkeleton?: boolean")
    expect(source).toContain("onReady,")
    expect(source).toContain("deferInitialSkeleton = false")
    expect(source).toContain("onReady?.()")
    expect(source).toContain("if (isLoading && !deferInitialSkeleton)")
    expect(source).toContain("if (isLoading) return null")
    expect(source).toContain('owner="api-keys-page"')
    expect(source).toContain('pageTitle={pageTitle}')
    expect(source).toContain('pageDescription={pageDescription}')
    expect(source).not.toContain("MasterDetailSkeleton")
    expect(source).not.toContain("ApiKeysSettingsSkeleton")
  })

  it("keeps the visual API key loading state owner optional", () => {
    expect(loadingStateSource).toContain("owner?: string")
    expect(loadingStateSource).toContain("if (owner !== undefined && !owner.trim())")
    expect(loadingStateSource).toContain('owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {}')
    expect(source).toContain('owner="api-keys-page"')
  })

  it("stretches the settings workspace to the available viewport height", () => {
    expect(source).toContain("API_KEYS_PAGE_SHELL_CLASS")
    expect(source).toContain("API_KEYS_CONTENT_SHELL_CLASS")
    expect(source).toContain("API_KEYS_MASTER_DETAIL_GRID_CLASS")
    expect(source).toContain("API_KEYS_PROVIDER_LIST_CARD_CLASS")
    expect(source).toContain("API_KEYS_PROVIDER_DETAIL_CARD_CLASS")
  })

  it("keeps provider row status badges right-aligned before the enable switch", () => {
    expect(source).toContain("<span className={API_KEYS_PROVIDER_STATUS_SLOT_CLASS}>")
    expect(source).toContain("<Badge className=\"shrink-0\"")
    expect(source).toContain("<div className={API_KEYS_PROVIDER_SWITCH_SLOT_CLASS}>")
  })

  it("filters providers locally from the provider list header", () => {
    expect(source).toContain("providerSearchQuery")
    expect(source).toContain("filteredProviders")
    expect(source).toContain("SearchInput")
    expect(source).toContain("API_KEYS_PROVIDER_SEARCH_INPUT_CLASS")
    expect(layoutSource).toContain("dark:bg-card")
    expect(source).toContain("API_KEYS_PROVIDER_SEARCH_ROW_CLASS")
    expect(source).toContain('placeholder={t("searchPlaceholder")}')
    expect(source).toContain("providerMatchesSearch")
    expect(source).toContain("provider.displayName")
    expect(source).toContain("provider.sourceName")
    expect(source).toContain("provider.key")
    expect(source).toContain("filteredProviders.map")
    expect(source).not.toContain("{filteredProviders.length} / {providers.length}")
    expect(source).not.toContain("textRole.monoLabel")
    expect(source).not.toContain('<CardTitle>{t("listTitle")}</CardTitle>')
    expect(loadingStateSource).not.toContain("listTitle: string")
  })

  it("keeps the provider search toolbar compact without shrinking the detail header", () => {
    expect(resolvedPageSource.match(/<CardHeader className=\{API_KEYS_PROVIDER_LIST_HEADER_CLASS\} density="compact">/g) ?? []).toHaveLength(1)
    expect(resolvedPageSource.match(/<CardHeader className=\{API_KEYS_CARD_HEADER_CLASS\} density="compact">/g) ?? []).toHaveLength(1)
    expect(loadingStateSource.match(/<CardHeader className=\{API_KEYS_PROVIDER_LIST_HEADER_CLASS\} density="compact">/g) ?? []).toHaveLength(1)
    expect(loadingStateSource.match(/<CardHeader className=\{API_KEYS_CARD_HEADER_CLASS\} density="compact">/g) ?? []).toHaveLength(1)
    expect(source).not.toContain('<CardHeader className="border-b py-3">')
    expect(source).not.toContain('<CardHeader className="border-b py-4">')
    expect(source).not.toContain('<CardHeader className="border-b py-5">')
  })

  it("keeps the provider detail focused on credentials without optional advanced controls", () => {
    expect(source).not.toContain("高级设置")
    expect(source).not.toContain("请求超时时间")
    expect(source).not.toContain("并发请求数")
    expect(source).not.toContain("handleResetSelectedProvider")
    expect(source).not.toContain(">重置<")
    expect(source).not.toContain("configuredState")
    expect(source).not.toContain("unconfiguredState")
    expect(source).not.toContain("fieldHelper")
  })

  it("marks required credential fields with a compact required indicator", () => {
    expect(source).toContain("field.required ?")
    expect(source).toContain("aria-hidden=\"true\"")
    expect(source).toContain("text-destructive")
  })

  it("keeps footer actions on a consistent Button size", () => {
    const footerStart = resolvedPageSource.indexOf("<CardFooter")
    const footerEnd = resolvedPageSource.indexOf("</CardFooter>", footerStart)
    const footerSource = resolvedPageSource.slice(footerStart, footerEnd)

    expect(footerSource).toContain('{updateMutation.isPending ? t("saving") : t("save")}')
    expect(footerSource).toContain('className={API_KEYS_CARD_FOOTER_CLASS}')
    expect(footerSource).not.toContain('t("testConnection")')
    expect(source).not.toContain('t("testConnectionHint")')
    expect(footerSource).not.toMatch(/<Button[\s\S]*?size="sm"/)
  })

  it("keeps API key input action buttons aligned with the Input height", () => {
    const passwordInputStart = source.indexOf("function PasswordInput")
    const passwordInputEnd = source.indexOf("export default function", passwordInputStart)
    const passwordInputSource = source.slice(passwordInputStart, passwordInputEnd)
    const iconButtonSizes = passwordInputSource.match(/size="icon"/g) ?? []

    expect(passwordInputSource).toContain("<Input")
    expect(passwordInputSource).toContain("className={API_KEYS_PASSWORD_INPUT_CLASS}")
    expect(passwordInputSource).toContain("aria-label={show ? hideLabel : showLabel}")
    expect(passwordInputSource).toContain("<CopyButton")
    expect(passwordInputSource).toContain("copyLabel={copyLabel}")
    expect(passwordInputSource).toContain("className={API_KEYS_PASSWORD_COPY_ACTION_CLASS}")
    expect(passwordInputSource).toContain('size="icon-sm"')
    expect(iconButtonSizes).toHaveLength(1)
  })

  it("keeps the API key loading credential controls on shared structural owners", () => {
    expect(loadingStateSource).toContain("ActionSkeleton")
    expect(loadingStateSource).toContain('<ActionSkeleton size="icon-sm" className={API_KEYS_PASSWORD_COPY_ACTION_CLASS} />')
    expect(loadingStateSource).toContain('<ActionSkeleton size="icon" />')
    expect(loadingStateSource).toContain("className={API_KEYS_PASSWORD_INPUT_CLASS}")
    expect(loadingStateSource).not.toContain('Skeleton className="h-9 w-full rounded-md"')
    expect(loadingStateSource).not.toContain('Skeleton className="absolute right-1 top-1/2 size-8 -translate-y-1/2 rounded-md"')
    expect(loadingStateSource).not.toContain('Skeleton className="size-9 rounded-md"')
  })

  it("keeps visual styling under the UI foundation instead of provider rainbow colors", () => {
    expect(source).toContain("@/components/ui/button")
    expect(source).toContain("@/components/ui/input")
    expect(source).toContain("@/components/ui/switch")
    expect(source).toContain("@/components/ui/badge")
    expect(source).toContain("@/components/shared/feedback/copy-button")
    expect(source).toContain("@/lib/typography")
    expect(loadingStateSource).toContain("@/components/ui/button")
    expect(loadingStateSource).toContain("@/components/ui/input")
    expect(loadingStateSource).toContain("@/components/ui/switch")
    expect(source + loadingStateSource).not.toMatch(/text-(blue|orange|red|purple|green|cyan|indigo|teal)-500/)
    expect(source + loadingStateSource).not.toMatch(/bg-(blue|orange|red|purple|green|cyan|indigo|teal)-500\/10/)
    expect(source).not.toContain("<button")
  })

  it("uses registry-backed provider definitions instead of the old fixed list", () => {
    for (const provider of [
      "settings.definitions",
      "provider.displayName",
      "data-source-name={provider.sourceName}",
      "provider.fields",
    ]) {
      expect(source).toContain(provider)
    }
    expect(source).not.toContain("PROVIDER_META")
    expect(source).not.toContain("DEFAULT_SETTINGS")
    expect(source).not.toContain("Hunter")
    expect(source).not.toContain("apiId")
    expect(source).not.toContain("apiSecret")
  })
})
