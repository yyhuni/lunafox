export const WEBSITE_RELATION_DETAIL_SECTIONS = [
  "overview",
  "ip-addresses",
  "urls",
  "directories",
  "vulnerabilities",
] as const

export type WebsiteRelationDetailSection = typeof WEBSITE_RELATION_DETAIL_SECTIONS[number]

export function resolveWebsiteRelationDetailSection(section: string | undefined): WebsiteRelationDetailSection {
  return WEBSITE_RELATION_DETAIL_SECTIONS.includes(section as WebsiteRelationDetailSection)
    ? section as WebsiteRelationDetailSection
    : "overview"
}
