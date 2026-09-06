"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"
import { ApiKeysSettingsLoadingState } from "@/components/settings/api-keys/api-keys-settings-loading-state"
import {
  API_KEYS_WORKSPACE_HANDOFF_CLASS,
  API_KEYS_WORKSPACE_STATE_CLASS,
} from "./api-keys-settings-layout"
import type { ApiKeysSettingsPageProps } from "./content"

const ApiKeysSettingsPageContent = dynamic<ApiKeysSettingsPageProps>(
  () => import("./content"),
  {
    ssr: false,
    loading: () => null,
  }
)

export function ApiKeysSettingsWorkspace() {
  const t = useTranslations("pages.apiKeys")
  const pageTitle = t("title")
  const pageDescription = t("description")

  return (
    <HiddenReadinessRouteBoundary
      owner="api-keys-page-route"
      layer="workspace"
      intent="data"
      skeleton={(
        <ApiKeysSettingsLoadingState
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
      )}
      className={API_KEYS_WORKSPACE_HANDOFF_CLASS}
      skeletonClassName={API_KEYS_WORKSPACE_STATE_CLASS}
      contentClassName={API_KEYS_WORKSPACE_STATE_CLASS}
    >
      {({ onReady, deferInitialSkeleton }) => (
        <ApiKeysSettingsPageContent
          pageTitle={pageTitle}
          pageDescription={pageDescription}
          onReady={onReady}
          deferInitialSkeleton={deferInitialSkeleton}
        />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
