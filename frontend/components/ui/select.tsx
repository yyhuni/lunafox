"use client"

import * as React from "react"
import {
  Select as SelectPrimitive,
  type SelectRootChangeEventDetails,
} from "@base-ui/react/select"
import { CheckIcon, ChevronDownIcon, ChevronUpIcon } from "@/components/icons"

import { cn } from "@/lib/utils"

type SelectPortalContainerState = {
  contentContainer: HTMLElement | null
  itemAlignedContainer: HTMLElement | null
  setContainers: React.Dispatch<React.SetStateAction<SelectPortalContainers>>
}

type SelectPortalContainers = {
  contentContainer: HTMLElement | null
  itemAlignedContainer: HTMLElement | null
}

const SelectPortalContainerContext = React.createContext<SelectPortalContainerState | null>(null)

type SelectRootProps<TValue extends string | null = string> = Omit<
  React.ComponentProps<typeof SelectPrimitive.Root>,
  "defaultValue" | "multiple" | "onValueChange" | "value"
> & {
  defaultValue?: TValue
  multiple?: false
  onValueChange?: (value: TValue, eventDetails: SelectRootChangeEventDetails) => void
  value?: TValue
}

function Select<TValue extends string | null = string>({
  modal = false,
  onValueChange,
  ...props
}: SelectRootProps<TValue>) {
  const [portalContainers, setPortalContainers] = React.useState<SelectPortalContainers>({
    contentContainer: null,
    itemAlignedContainer: null,
  })
  const portalContainerState = React.useMemo(
    () => ({ ...portalContainers, setContainers: setPortalContainers }),
    [portalContainers]
  )

  return (
    <SelectPortalContainerContext.Provider value={portalContainerState}>
      <SelectPrimitive.Root
        data-slot="select"
        modal={modal}
        onValueChange={
          onValueChange
            ? (value, eventDetails) => onValueChange(value as TValue, eventDetails)
            : undefined
        }
        {...props}
      />
    </SelectPortalContainerContext.Provider>
  )
}

function SelectGroup({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Group>) {
  return <SelectPrimitive.Group data-slot="select-group" {...props} />
}

function SelectValue({
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Value>) {
  return <SelectPrimitive.Value data-slot="select-value" {...props} />
}

const SelectTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<typeof SelectPrimitive.Trigger> & {
    size?: "sm" | "default"
  }
>(function SelectTrigger({
  className,
  size = "default",
  children,
  ...props
}, ref) {
  const portalContainer = React.useContext(SelectPortalContainerContext)
  const setPortalContainers = portalContainer?.setContainers
  const handleRef = React.useCallback(
    (node: HTMLButtonElement | null) => {
      if (typeof ref === "function") {
        ref(node)
      } else if (ref) {
        ref.current = node
      }

      const contentContainer =
        node?.closest<HTMLElement>(
          "[data-slot='drawer-content'], [data-slot='dialog-content'], [data-slot='sheet-content']"
        ) ?? null
      // Item-aligned Select uses fixed positioning; keep it in the overlay event layer,
      // but outside transformed/flex panel content that can shift layout.
      const itemAlignedContainer =
        node?.closest<HTMLElement>(
          "[data-slot='drawer-viewport'], [data-slot='dialog-portal'], [data-slot='sheet-portal']"
        ) ?? null

      setPortalContainers?.((current) => {
        if (
          current.contentContainer === contentContainer &&
          current.itemAlignedContainer === itemAlignedContainer
        ) {
          return current
        }

        return { contentContainer, itemAlignedContainer }
      })
    },
    [ref, setPortalContainers]
  )

  return (
    <SelectPrimitive.Trigger
      ref={handleRef}
      data-slot="select-trigger"
      data-size={size}
      className={cn(
        "radius-control border-input data-[placeholder]:text-muted-foreground [&_svg:not([class*='text-'])]:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive bg-background dark:bg-input/30 hover:bg-accent/50 dark:hover:bg-input/50 flex w-fit items-center justify-between gap-2 border px-3 py-2 text-sm whitespace-nowrap shadow-xs transition-[color,background-color,border-color,box-shadow] outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 data-[size=default]:h-9 data-[size=sm]:h-8 *:data-[slot=select-value]:line-clamp-1 *:data-[slot=select-value]:flex *:data-[slot=select-value]:items-center *:data-[slot=select-value]:gap-2 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon>
        <ChevronDownIcon className="opacity-50 size-4" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  )
})

function SelectContent({
  className,
  children,
  position = "item-aligned",
  width = "default",
  align = "start",
  sideOffset = 4,
  alignItemWithTrigger = false,
  listClassName,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Popup> &
  Pick<
    React.ComponentProps<typeof SelectPrimitive.Positioner>,
    | "align"
    | "alignOffset"
    | "anchor"
    | "collisionBoundary"
    | "collisionPadding"
    | "positionMethod"
    | "side"
    | "sideOffset"
    | "sticky"
    | "alignItemWithTrigger"
  > & {
    position?: "popper" | "item-aligned"
    listClassName?: string
    width?: "default" | "content-fit"
}) {
  const portalContainer = React.useContext(SelectPortalContainerContext)
  const shouldAlignItemWithTrigger = position === "item-aligned" || alignItemWithTrigger
  const contentPortalContainer = shouldAlignItemWithTrigger
    ? portalContainer?.itemAlignedContainer ?? undefined
    : portalContainer?.contentContainer ?? undefined
  const {
    alignOffset,
    anchor,
    collisionBoundary,
    collisionPadding,
    positionMethod,
    side,
    sticky,
    ...popupProps
  } = props

  return (
    <SelectPrimitive.Portal container={contentPortalContainer}>
      <SelectPrimitive.Positioner
        data-slot="select-positioner"
        // Base UI positions this wrapper, so the stacking level must live here.
        className="z-50"
        align={align}
        alignOffset={alignOffset}
        alignItemWithTrigger={shouldAlignItemWithTrigger}
        anchor={anchor}
        collisionBoundary={collisionBoundary}
        collisionPadding={collisionPadding}
        positionMethod={positionMethod}
        side={side}
        sideOffset={shouldAlignItemWithTrigger ? 0 : sideOffset}
        sticky={sticky}
      >
        <SelectPrimitive.Popup
          data-slot="select-content"
          className={cn(
            "radius-overlay bg-popover text-popover-foreground data-[open]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[open]:fade-in-0 data-[ending-style]:zoom-out-95 data-[starting-style]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 relative z-50 max-h-[var(--available-height)] origin-[var(--transform-origin)] overflow-x-hidden overflow-y-auto border shadow-md",
            width === "default" && "min-w-[max(8rem,var(--anchor-width))]",
            width === "content-fit" && "w-max min-w-[var(--anchor-width)]",
            className
          )}
          {...popupProps}
        >
          <SelectScrollUpButton />
          <SelectPrimitive.List
            data-slot="select-list"
            className={cn(
              "p-1 scroll-my-1",
              !shouldAlignItemWithTrigger && "min-h-[var(--anchor-height)] min-w-full",
              listClassName
            )}
          >
            {children}
          </SelectPrimitive.List>
          <SelectScrollDownButton />
        </SelectPrimitive.Popup>
      </SelectPrimitive.Positioner>
    </SelectPrimitive.Portal>
  )
}

function SelectLabel({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.GroupLabel>) {
  return (
    <SelectPrimitive.GroupLabel
      data-slot="select-label"
      className={cn("text-muted-foreground px-2 py-1.5 text-xs", className)}
      {...props}
    />
  )
}

function SelectItem({
  className,
  children,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Item>) {
  return (
    <SelectPrimitive.Item
      data-slot="select-item"
      className={cn(
        "radius-control data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground [&_svg:not([class*='text-'])]:text-muted-foreground relative flex w-full cursor-pointer items-center gap-1.5 py-1.5 pr-8 pl-2 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4 *:[span]:last:flex *:[span]:last:items-center *:[span]:last:gap-1.5",
        className
      )}
      {...props}
    >
      <span className="absolute flex items-center justify-center right-2 size-3.5">
        <SelectPrimitive.ItemIndicator>
          <CheckIcon className="size-4" />
        </SelectPrimitive.ItemIndicator>
      </span>
      <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
    </SelectPrimitive.Item>
  )
}

function SelectSeparator({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Separator>) {
  return (
    <SelectPrimitive.Separator
      data-slot="select-separator"
      className={cn("bg-border pointer-events-none -mx-1 my-1 h-px", className)}
      {...props}
    />
  )
}

function SelectScrollUpButton({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.ScrollUpArrow>) {
  return (
    <SelectPrimitive.ScrollUpArrow
      data-slot="select-scroll-up-button"
      className={cn(
        "flex cursor-default items-center justify-center py-1",
        className
      )}
      {...props}
    >
      <ChevronUpIcon className="size-4" />
    </SelectPrimitive.ScrollUpArrow>
  )
}

function SelectScrollDownButton({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.ScrollDownArrow>) {
  return (
    <SelectPrimitive.ScrollDownArrow
      data-slot="select-scroll-down-button"
      className={cn(
        "flex cursor-default items-center justify-center py-1",
        className
      )}
      {...props}
    >
      <ChevronDownIcon className="size-4" />
    </SelectPrimitive.ScrollDownArrow>
  )
}

export {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
}
