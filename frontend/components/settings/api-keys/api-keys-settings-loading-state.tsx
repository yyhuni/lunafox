import { ShieldCheck } from "@/components/icons"
import { PageHeader } from "@/components/common/page-header"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
} from "@/components/shared/loading/loading-owner"
import { SearchInput } from "@/components/shared/search-input"
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
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  API_KEYS_CARD_FOOTER_ACTION_GROUP_CLASS,
  API_KEYS_CARD_FOOTER_CLASS,
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
  API_KEYS_PROVIDER_ROW_CONTENT_CLASS,
  API_KEYS_PROVIDER_ROW_SELECTED_CLASS,
  API_KEYS_PROVIDER_SEARCH_INPUT_CLASS,
  API_KEYS_PROVIDER_SEARCH_ROW_CLASS,
  API_KEYS_PROVIDER_STATUS_SLOT_CLASS,
  API_KEYS_PROVIDER_SWITCH_SLOT_CLASS,
  API_KEYS_SECURITY_NOTICE_CARD_CLASS,
  API_KEYS_SECURITY_NOTICE_CONTENT_CLASS,
} from "@/app/settings/api-keys/api-keys-settings-layout"

export interface ApiKeysSettingsLoadingStateProps {
  owner?: string
  pageTitle: string
  pageDescription: string
  enableLabel: string
  emailLabel: string
  apiKeyLabel: string
  searchPlaceholder: string
  getApiKeyLabel: string
  saveLabel: string
  securityNoticeLabel: string
  securityNoticeDescription: string
}

const API_KEYS_LOADING_PROVIDER_ROWS = [
  { selected: true, nameWidthClassName: "w-20" },
  { selected: false, nameWidthClassName: "w-24" },
  { selected: false, nameWidthClassName: "w-16" },
  { selected: false, nameWidthClassName: "w-20" },
  { selected: false, nameWidthClassName: "w-28" },
  { selected: false, nameWidthClassName: "w-24" },
] as const

export function ApiKeysSettingsLoadingState({
  owner,
  pageTitle,
  pageDescription,
  enableLabel,
  emailLabel,
  apiKeyLabel,
  searchPlaceholder,
  getApiKeyLabel,
  saveLabel,
  securityNoticeLabel,
  securityNoticeDescription,
}: ApiKeysSettingsLoadingStateProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("ApiKeysSettingsLoadingState requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer: "route", intent: "route" }) : {})}
      data-slot="api-keys-settings-loading-state"
      className={API_KEYS_PAGE_SHELL_CLASS}
    >
      <div {...getLoadingStructureSlotAttributes("api-keys-header")}>
        <PageHeader code="API-01" title={pageTitle} description={pageDescription} />
      </div>

      <div className={API_KEYS_CONTENT_SHELL_CLASS}>
        <div className={API_KEYS_MASTER_DETAIL_GRID_CLASS}>
          <Card
            {...getLoadingStructureSlotAttributes("api-keys-provider-list")}
            className={API_KEYS_PROVIDER_LIST_CARD_CLASS}
          >
            <CardHeader className={API_KEYS_PROVIDER_LIST_HEADER_CLASS} density="compact">
              <div className={API_KEYS_PROVIDER_SEARCH_ROW_CLASS}>
                <SearchInput
                  className={API_KEYS_PROVIDER_SEARCH_INPUT_CLASS}
                  value=""
                  disabled
                  placeholder={searchPlaceholder}
                  aria-label={searchPlaceholder}
                />
              </div>
            </CardHeader>
            <CardContent className={API_KEYS_PROVIDER_LIST_CONTENT_CLASS}>
              <div aria-hidden="true">
                {API_KEYS_LOADING_PROVIDER_ROWS.map((provider, index) => (
                  <div
                    key={index}
                    className={cn(
                      API_KEYS_PROVIDER_ROW_CLASS,
                      provider.selected && API_KEYS_PROVIDER_ROW_SELECTED_CLASS
                    )}
                  >
                    <div className={API_KEYS_PROVIDER_ROW_CONTENT_CLASS}>
                      <span
                        className={cn(
                          API_KEYS_PROVIDER_AVATAR_CLASS,
                          textRole.metadataValueStrong
                        )}
                      >
                        <Skeleton className="h-4 w-5" />
                      </span>
                      <span className={cn("min-w-0 flex-1 truncate", textRole.bodyStrong)}>
                        <Skeleton className={cn("h-5", provider.nameWidthClassName)} />
                      </span>
                      <span className={API_KEYS_PROVIDER_STATUS_SLOT_CLASS}>
                        <Skeleton className="h-6 w-12 rounded-full" />
                      </span>
                    </div>
                    <div className={API_KEYS_PROVIDER_SWITCH_SLOT_CLASS}>
                      <Switch checked={provider.selected} disabled aria-hidden="true" />
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card
            {...getLoadingStructureSlotAttributes("api-keys-provider-detail")}
            className={API_KEYS_PROVIDER_DETAIL_CARD_CLASS}
          >
            <CardHeader className={API_KEYS_CARD_HEADER_CLASS} density="compact">
              <div className={API_KEYS_DETAIL_HEADER_LAYOUT_CLASS}>
                <div>
                  <div className={API_KEYS_DETAIL_TITLE_ROW_CLASS}>
                    <CardTitle className={textRole.panelTitle}>
                      <Skeleton className="h-5 w-24" />
                    </CardTitle>
                    <Badge variant="outline">
                      <span className="inline-flex">
                        <Skeleton className="h-4 w-12 rounded-full" />
                      </span>
                    </Badge>
                  </div>
                </div>
                <div className={API_KEYS_DETAIL_ENABLE_CLASS}>
                  <span className={textRole.bodyStrong}>{enableLabel}</span>
                  <Switch checked disabled aria-hidden="true" />
                </div>
              </div>
            </CardHeader>

            <CardContent className={API_KEYS_DETAIL_CONTENT_CLASS}>
              <div className={API_KEYS_FIELD_GROUP_CLASS}>
                <Label>
                  {emailLabel}
                  <span aria-hidden="true" className="ml-1 text-destructive">*</span>
                </Label>
                <Input disabled aria-hidden="true" />
              </div>

              <div className={API_KEYS_FIELD_GROUP_CLASS}>
                <Label>
                  {apiKeyLabel}
                  <span aria-hidden="true" className="ml-1 text-destructive">*</span>
                </Label>
                <div className={API_KEYS_PASSWORD_INPUT_ROW_CLASS}>
                  <div className={API_KEYS_PASSWORD_INPUT_FIELD_SLOT_CLASS}>
                    <Input className={API_KEYS_PASSWORD_INPUT_CLASS} disabled aria-hidden="true" />
                    <ActionSkeleton size="icon-sm" className={API_KEYS_PASSWORD_COPY_ACTION_CLASS} />
                  </div>
                  <ActionSkeleton size="icon" />
                </div>
              </div>

              <div className={API_KEYS_DOCUMENTATION_LINK_ROW_CLASS}>
                <span className={cn("inline-flex items-center gap-1", textRole.helperText)}>
                  {getApiKeyLabel}
                </span>
              </div>
            </CardContent>

            <CardFooter className={API_KEYS_CARD_FOOTER_CLASS} density="compact">
              <div className={API_KEYS_CARD_FOOTER_ACTION_GROUP_CLASS}>
                <Button type="button" disabled>
                  {saveLabel}
                </Button>
              </div>
            </CardFooter>
          </Card>
        </div>

        <Card
          {...getLoadingStructureSlotAttributes("api-keys-notice")}
          className={API_KEYS_SECURITY_NOTICE_CARD_CLASS}
        >
          <CardContent className={API_KEYS_SECURITY_NOTICE_CONTENT_CLASS}>
            <ShieldCheck className="mt-0.5 size-4 shrink-0" />
            <p className={textRole.bodySubtle}>
              <span className={textRole.bodyStrong}>{securityNoticeLabel}</span>
              {securityNoticeDescription}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
