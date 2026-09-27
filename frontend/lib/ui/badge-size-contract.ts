export const badgeStructuralSizeClassNames = {
  tag: "badge-size-tag",
  compact: "badge-size-compact",
  micro: "badge-size-micro",
} as const

export type BadgeStructuralSize = keyof typeof badgeStructuralSizeClassNames
