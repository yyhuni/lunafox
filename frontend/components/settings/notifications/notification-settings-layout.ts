import { textRole } from "@/lib/typography"
import {
  COMPACT_CONTENT_GUTTER_CLASS,
  COMPACT_FULL_PAGE_SHELL_CLASS,
  COMPACT_PAGE_SCROLL_AREA_CLASS,
  COMPACT_PAGE_SCROLL_AREA_CONTENT_CLASS,
  COMPACT_PAGE_SCROLL_AREA_VIEWPORT_CLASS,
} from "@/components/shared/layout/page-shell-density"

export const NOTIFICATION_SETTINGS_WORKSPACE_HANDOFF_CLASS = "flex h-full min-h-0 flex-1 flex-col"

export const NOTIFICATION_SETTINGS_WORKSPACE_STATE_CLASS = "flex h-full min-h-0 flex-1 flex-col"

export const NOTIFICATION_SETTINGS_PAGE_SHELL_CLASS = COMPACT_FULL_PAGE_SHELL_CLASS

export const NOTIFICATION_SETTINGS_CONTENT_SCROLL_AREA_CLASS = COMPACT_PAGE_SCROLL_AREA_CLASS

export const NOTIFICATION_SETTINGS_CONTENT_SCROLL_VIEWPORT_CLASS =
  COMPACT_PAGE_SCROLL_AREA_VIEWPORT_CLASS

export const NOTIFICATION_SETTINGS_CONTENT_SHELL_CLASS =
  `${COMPACT_PAGE_SCROLL_AREA_CONTENT_CLASS} flex min-h-full flex-col gap-3 ${COMPACT_CONTENT_GUTTER_CLASS}`

export const NOTIFICATION_SETTINGS_CHANNEL_WORKBENCH_CLASS = "flex w-full min-w-0"

export const NOTIFICATION_CHANNELS_STACK_CLASS = "flex w-full min-w-0 flex-col gap-3"

export const NOTIFICATION_CHANNEL_EDITOR_CARD_CLASS =
  "flex min-w-0 shrink-0 flex-col gap-0 overflow-hidden py-0"

export const NOTIFICATION_CHANNEL_DETAIL_HEADER_CLASS = "border-b px-4 pt-3"

export const NOTIFICATION_CHANNEL_DETAIL_HEADER_LAYOUT_CLASS = "flex items-center justify-between gap-3"

export const NOTIFICATION_CHANNEL_DETAIL_TITLE_ROW_CLASS = "flex min-w-0 items-center gap-2"

export const NOTIFICATION_CHANNEL_DETAIL_TITLE_CLASS = textRole.panelTitle

export const NOTIFICATION_CHANNEL_DETAIL_CONTENT_CLASS = "space-y-3 px-4 py-3"

export const NOTIFICATION_SETTINGS_ACTION_BAR_CLASS = "flex shrink-0 items-center justify-end gap-3"

export const NOTIFICATION_CHANNEL_ICON_CLASS = "radius-control flex size-10 shrink-0 items-center justify-center"

export const NOTIFICATION_CHANNEL_TITLE_CLASS = textRole.sectionTitle

export const NOTIFICATION_CREDENTIAL_FIELD_CLASS = "grid gap-2"

export const NOTIFICATION_CREDENTIAL_INPUT_SHELL_CLASS = "relative"

export const NOTIFICATION_CREDENTIAL_INPUT_CLASS = "pr-12"

export const NOTIFICATION_CREDENTIAL_REVEAL_ACTION_CLASS = "absolute right-1 top-1/2 -translate-y-1/2"

export const NOTIFICATION_SUBSCRIPTIONS_SECTION_CLASS = "grid gap-2"

export const NOTIFICATION_SUBSCRIPTIONS_LIST_CLASS = "grid gap-2 sm:grid-cols-2"

export const NOTIFICATION_SUBSCRIPTION_ITEM_CLASS = "flex min-w-0 items-center gap-2"

export const NOTIFICATION_WEBHOOK_LABEL_SKELETON_CLASS = "h-3.5 w-32 max-w-full"

export const NOTIFICATION_WEBHOOK_DESCRIPTION_SKELETON_CLASS = "h-5 w-full max-w-full"
