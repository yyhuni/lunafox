import type { OverviewLazySectionsProps } from "@/components/overview/overview-lazy-sections"

export const loadOverviewLazySections = () =>
  import("@/components/overview/overview-lazy-sections").then((mod) => mod.OverviewLazySections)

export type { OverviewLazySectionsProps }
