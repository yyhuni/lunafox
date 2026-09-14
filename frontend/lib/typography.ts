/**
 * Shared production typography roles.
 *
 * Keep these roles semantic and compact. Components may add layout classes, but
 * readable UI text should get color, weight, size, and tracking from here.
 */
export const textRole = {
  pageTitle: "text-xl font-semibold text-foreground leading-none tracking-tight",
  pageTitleDisplay: "[font-family:var(--font-display)] text-xl font-semibold text-foreground leading-none tracking-tight",
  pageDescription: "text-xs font-normal text-muted-foreground leading-4 tracking-normal",
  panelTitle: "text-base font-semibold text-foreground leading-5 tracking-normal",
  sectionTitle: "text-[13px] font-semibold text-foreground leading-[18px] tracking-normal",
  compactSectionTitle: "text-xs font-semibold text-foreground leading-4 tracking-normal",
  compactPrimary: "text-xs font-medium text-foreground leading-4 tracking-normal",
  compactCaption: "text-[11px] font-normal text-muted-foreground leading-4 tracking-normal",
  metricValueDisplay: "[font-family:var(--font-display)] text-xl font-medium text-foreground leading-none tracking-tight tabular-nums data-[featured=true]:text-3xl data-[featured=true]:font-semibold",
  body: "text-sm font-normal text-foreground leading-relaxed tracking-normal",
  bodyLarge: "text-base font-normal text-foreground leading-relaxed tracking-normal",
  bodyStrong: "text-[13px] font-medium text-foreground leading-[18px] tracking-normal",
  bodySubtle: "text-xs font-normal text-muted-foreground leading-4 tracking-normal",
  helperText: "text-xs font-normal text-muted-foreground leading-4 tracking-normal",
  caption: "text-xs font-normal text-muted-foreground leading-normal tracking-normal",
  metadataLabel: "text-xs font-normal text-muted-foreground leading-4 tracking-normal",
  metadataValue: "text-xs font-normal text-foreground leading-4 tracking-normal",
  metadataValueStrong: "text-xs font-medium text-foreground leading-4 tracking-normal",
  navLabel: "text-[13px] font-medium text-foreground leading-[18px] tracking-normal",
  tableHeader: "text-[12px] font-medium text-muted-foreground leading-4 tracking-normal",
  tableCellPrimary: "text-[13px] font-medium text-foreground leading-[18px] tracking-normal",
  tableCellSecondary: "text-xs font-normal text-muted-foreground leading-4 tracking-normal",
  badge: "text-[11px] font-medium leading-4 tracking-normal normal-case",
  badgeSubtle: "text-[11px] font-medium text-muted-foreground leading-4 tracking-normal normal-case",
  tab: "text-[13px] font-medium leading-[18px] tracking-normal normal-case",
  monoLabel: "font-mono text-xs font-medium text-muted-foreground leading-normal tracking-normal",
  code: "font-mono text-xs font-normal tracking-normal",
  authHeroEyebrow: "text-base font-normal text-muted-foreground leading-normal tracking-normal",
  authHeroTitle: "text-5xl font-semibold text-foreground leading-none tracking-normal md:text-6xl lg:text-7xl",
  authFormLabel: "text-sm font-normal text-muted-foreground leading-none tracking-normal",
  authFormText: "text-sm font-normal text-muted-foreground leading-normal tracking-normal",
} as const

export type TextRoleName = keyof typeof textRole

export const typographyRoleNames = Object.keys(textRole) as TextRoleName[]
