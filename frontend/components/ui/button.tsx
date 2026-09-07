import * as React from "react"
import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"

import { Loader2Icon } from "@/components/icons"
import { PolymorphicSlot } from "@/components/ui/polymorphic"
import { textRole } from "@/lib/typography"
import { buttonStructuralSizeClassNames } from "@/lib/ui/button-size-contract"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "radius-control inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap text-sm font-medium outline-none transition-[color,background-color,border-color,box-shadow,opacity] disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40",
  {
    variants: {
      variant: {
        default:
          "border border-transparent bg-interaction-accent text-interaction-accent-foreground hover:bg-interaction-accent/90",
        primary:
          "border border-transparent bg-interaction-accent text-interaction-accent-foreground hover:bg-interaction-accent/90",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/60",
        outline:
          "border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50",
        surface:
          "border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:border-input",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost:
          "hover:bg-primary/10 hover:text-primary dark:hover:bg-primary/20",
        quiet:
          "hover:bg-primary/10 hover:text-primary dark:hover:bg-primary/20",
        link: "text-primary underline-offset-4 hover:text-interaction-accent hover:underline",
      },
      size: {
        default: `${buttonStructuralSizeClassNames.default} px-4 py-2 has-[>svg]:px-3`,
        sm: `${buttonStructuralSizeClassNames.sm} gap-1.5 px-3 has-[>svg]:px-2.5`,
        lg: `${buttonStructuralSizeClassNames.lg} px-6 has-[>svg]:px-4`,
        "action-card": `${buttonStructuralSizeClassNames["action-card"]} justify-start whitespace-normal px-4 py-3 text-left`,
        content: buttonStructuralSizeClassNames.content,
        "chip-icon": buttonStructuralSizeClassNames["chip-icon"],
        icon: buttonStructuralSizeClassNames.icon,
        "icon-sm": buttonStructuralSizeClassNames["icon-sm"],
        "icon-lg": buttonStructuralSizeClassNames["icon-lg"],
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

const buttonLayoutClassNames = {
  default: "",
  fullWidth: "w-full",
  between: "w-full justify-between",
  tableHeader: `-ml-2 w-full justify-start gap-1 overflow-hidden hover:bg-transparent dark:hover:bg-transparent has-[>svg]:px-2 ${textRole.tableHeader}`,
  tableHeaderInline: `-ml-2 cursor-pointer justify-start gap-1 data-[popup-open]:bg-primary/10 data-[popup-open]:text-primary data-[popup-open]:[&_svg]:text-primary has-[>svg]:px-2 ${textRole.tableHeader}`,
  textCellLink: `justify-start bg-transparent p-0 text-left shadow-none underline-offset-2 hover:bg-transparent hover:text-primary hover:underline ${textRole.tableCellPrimary}`,
  actionTile: "min-w-0 flex-col items-center gap-3 border-2 text-center",
} as const

type ButtonLayout = keyof typeof buttonLayoutClassNames

function Button({
  className,
  variant,
  size,
  layout = "default",
  selected = false,
  loading = false,
  loadingLabel,
  loadingIndicator,
  disabled,
  children,
  render,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    render?: React.ReactElement
    layout?: ButtonLayout
    selected?: boolean
    loading?: boolean
    loadingLabel?: React.ReactNode
  loadingIndicator?: React.ReactNode
  }) {
  const isDisabled = disabled || loading
  const content = loading ? (
    <>
      {loadingIndicator ?? (
        <Loader2Icon
          aria-hidden="true"
          data-slot="button-loading-indicator"
          className="size-4 animate-spin"
        />
      )}
      <span>{loadingLabel ?? children}</span>
    </>
  ) : (
    children
  )

  const sharedProps = {
    "data-slot": "button",
    "data-size": size ?? "default",
    "data-layout": layout,
    "data-selected": selected ? "" : undefined,
    "data-loading": loading ? "" : undefined,
    "aria-busy": loading || props["aria-busy"] ? true : undefined,
    className: cn(
      buttonVariants({ variant, size }),
      buttonLayoutClassNames[layout],
      selected && layout === "actionTile" && "border-primary bg-primary/5",
      !selected && layout === "actionTile" && "border-muted hover:border-muted-foreground/50",
      className
    ),
    ...props,
  }

  if (render) {
    return (
      <PolymorphicSlot
        defaultTagName="button"
        render={render}
        disabled={disabled}
        {...sharedProps}
      >
        {content}
      </PolymorphicSlot>
    )
  }

  return (
    <ButtonPrimitive
      disabled={isDisabled}
      {...sharedProps}
    >
      {content}
    </ButtonPrimitive>
  )
}

export { Button, buttonVariants }
