export const buttonStructuralSizeClassNames = {
  default: "h-9",
  sm: "h-8",
  lg: "h-10",
  "action-card": "h-full min-h-16 w-full",
  content: "h-auto",
  "chip-icon": "size-4",
  "icon-sm": "size-8",
  icon: "size-9",
  "icon-lg": "size-10",
} as const

export type ButtonStructuralSize = keyof typeof buttonStructuralSizeClassNames

export const buttonIconOnlySizes = new Set<ButtonStructuralSize>(["chip-icon", "icon-sm", "icon", "icon-lg"])
