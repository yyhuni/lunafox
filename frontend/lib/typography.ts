/**
 * Shared production typography roles.
 *
 * Keep these roles semantic and compact. Components may add layout classes, but
 * readable UI text should get color, weight, size, and tracking from here.
 */
export const textRole = {
  pageTitle: "text-2xl font-semibold text-foreground leading-none tracking-tight",
  pageTitleDisplay: "[font-family:var(--font-display)] text-2xl font-semibold text-foreground leading-none tracking-tight",
  pageDescription: "text-sm font-normal text-muted-foreground leading-relaxed tracking-normal",
  panelTitle: "text-lg font-semibold text-foreground leading-snug tracking-normal",
  sectionTitle: "text-sm font-semibold text-foreground leading-5 tracking-normal",
  compactSectionTitle: "text-xs font-semibold text-foreground leading-4 tracking-normal",
  compactPrimary: "text-xs font-medium text-foreground leading-4 tracking-normal",
  compactCaption: "text-[11px] font-normal text-muted-foreground leading-4 tracking-normal",
  metricValueDisplay: "[font-family:var(--font-display)] text-xl font-medium text-foreground leading-none tracking-tight tabular-nums data-[featured=true]:text-3xl data-[featured=true]:font-semibold",
  body: "text-sm font-normal text-foreground leading-relaxed tracking-normal",
  bodyLarge: "text-base font-normal text-foreground leading-relaxed tracking-normal",
  bodyStrong: "text-sm font-medium text-foreground leading-normal tracking-normal",
  bodySubtle: "text-sm font-normal text-muted-foreground leading-relaxed tracking-normal",
  helperText: "text-xs font-normal text-muted-foreground leading-4 tracking-normal",
  caption: "text-xs font-normal text-muted-foreground leading-normal tracking-normal",
  metadataLabel: "text-sm font-normal text-muted-foreground leading-5 tracking-normal",
  metadataValue: "text-sm font-normal text-foreground leading-5 tracking-normal",
  metadataValueStrong: "text-sm font-medium text-foreground leading-5 tracking-normal",
  navLabel: "text-sm font-medium text-foreground leading-normal tracking-normal",
  tableHeader: "text-[12px] font-medium text-muted-foreground leading-4 tracking-normal",
  tableCellPrimary: "text-sm font-medium text-foreground leading-5 tracking-normal",
  tableCellSecondary: "text-sm font-normal text-muted-foreground leading-5 tracking-normal",
  badge: "text-[11px] font-medium leading-4 tracking-normal normal-case",
  badgeSubtle: "text-[11px] font-medium text-muted-foreground leading-4 tracking-normal normal-case",
  tab: "text-sm font-medium leading-5 tracking-normal normal-case",
  monoLabel: "font-mono text-xs font-medium text-muted-foreground leading-normal tracking-normal",
  code: "font-mono text-xs font-normal tracking-normal",
  authHeroEyebrow: "text-base font-normal text-muted-foreground leading-normal tracking-normal",
  authHeroTitle: "text-5xl font-semibold text-foreground leading-none tracking-normal md:text-6xl lg:text-7xl",
  authFormLabel: "text-sm font-normal text-muted-foreground leading-none tracking-normal",
  authFormText: "text-sm font-normal text-muted-foreground leading-normal tracking-normal",
} as const

export type TextRoleName = keyof typeof textRole

export const typographyRoleNames = Object.keys(textRole) as TextRoleName[]
