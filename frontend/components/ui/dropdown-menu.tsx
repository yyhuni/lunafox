"use client"

import * as React from "react"
import { Menu as DropdownMenuPrimitive } from "@base-ui/react/menu"
import { CheckIcon, ChevronRightIcon, CircleIcon } from "@/components/icons"

import {
  floatingContentMotionClassName,
  floatingSurfaceClassName,
} from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"

type MenuPositionProps = Pick<
  React.ComponentProps<typeof DropdownMenuPrimitive.Positioner>,
  "align" | "side" | "sideOffset" | "alignOffset" | "collisionPadding" | "collisionBoundary" | "collisionAvoidance" | "sticky" | "positionMethod"
>

type DropdownMenuContentProps = React.ComponentProps<typeof DropdownMenuPrimitive.Popup> & MenuPositionProps & {
  width?: "default" | "content-fit" | "compact"
  container?: React.ComponentProps<typeof DropdownMenuPrimitive.Portal>["container"]
}

type DropdownMenuCheckboxItemProps = Omit<
  React.ComponentProps<typeof DropdownMenuPrimitive.CheckboxItem>,
  "closeOnClick"
>

type DropdownMenuRadioItemProps = Omit<
  React.ComponentProps<typeof DropdownMenuPrimitive.RadioItem>,
  "closeOnClick"
>

function DropdownMenu({
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Root>) {
  return <DropdownMenuPrimitive.Root data-slot="dropdown-menu" {...props} />
}

function DropdownMenuPortal({
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Portal>) {
  return (
    <DropdownMenuPrimitive.Portal data-slot="dropdown-menu-portal" {...props} />
  )
}

function DropdownMenuTrigger({
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Trigger>) {
  return (
    <DropdownMenuPrimitive.Trigger
      data-slot="dropdown-menu-trigger"
      {...props}
    />
  )
}

function DropdownMenuContent({
  className,
  sideOffset = 4,
  align,
  side,
  alignOffset,
  collisionPadding,
  collisionBoundary,
  collisionAvoidance,
  sticky,
  positionMethod,
  container,
  width = "default",
  ...props
}: DropdownMenuContentProps) {
  return (
    <DropdownMenuPrimitive.Portal container={container}>
      <DropdownMenuPrimitive.Positioner
        data-slot="dropdown-menu-positioner"
        // Base UI positions this wrapper, so it must establish the menu's stack above a Sheet panel.
        className="z-50"
        align={align}
        side={side}
        sideOffset={sideOffset}
        alignOffset={alignOffset}
        collisionPadding={collisionPadding}
        collisionBoundary={collisionBoundary}
        collisionAvoidance={collisionAvoidance}
        sticky={sticky}
        positionMethod={positionMethod}
      >
        <DropdownMenuPrimitive.Popup
          data-slot="dropdown-menu-content"
          className={cn(
            floatingSurfaceClassName,
            floatingContentMotionClassName,
            "max-h-(--available-height) origin-(--transform-origin) overflow-x-hidden overflow-y-auto p-1",
            width === "default" && "min-w-[8rem]",
            width === "content-fit" && "w-max min-w-(--anchor-width)",
            width === "compact" && "w-max min-w-[5rem]",
            className
          )}
          {...props}
        />
      </DropdownMenuPrimitive.Positioner>
    </DropdownMenuPrimitive.Portal>
  )
}

function DropdownMenuGroup({
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Group>) {
  return (
    <DropdownMenuPrimitive.Group data-slot="dropdown-menu-group" {...props} />
  )
}

function DropdownMenuItem({
  className,
  inset,
  variant = "default",
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Item> & {
  inset?: boolean
  variant?: "default" | "destructive"
}) {
  return (
    <DropdownMenuPrimitive.Item
      data-slot="dropdown-menu-item"
      data-inset={inset}
      data-variant={variant}
      className={cn(
        "radius-control focus:bg-accent focus:text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground data-[variant=destructive]:text-destructive data-[variant=destructive]:focus:bg-destructive/10 data-[variant=destructive]:data-[highlighted]:bg-destructive/10 dark:data-[variant=destructive]:focus:bg-destructive/20 dark:data-[variant=destructive]:data-[highlighted]:bg-destructive/20 data-[variant=destructive]:focus:text-destructive data-[variant=destructive]:data-[highlighted]:text-destructive data-[variant=destructive]:*:[svg]:!text-destructive [&_svg:not([class*='text-'])]:text-muted-foreground relative flex cursor-pointer items-center gap-1.5 px-2 py-1.5 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[inset]:pl-8 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      {...props}
    />
  )
}

function DropdownMenuCheckboxItem({
  className,
  children,
  checked,
  ...props
}: DropdownMenuCheckboxItemProps) {
  return (
    <DropdownMenuPrimitive.CheckboxItem
      data-slot="dropdown-menu-checkbox-item"
      className={cn(
        "radius-control focus:bg-accent focus:text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground relative flex cursor-pointer items-center gap-1.5 py-1.5 pr-2 pl-8 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      checked={checked}
      {...props}
      closeOnClick={true}
    >
      <span className="absolute flex items-center justify-center left-2 pointer-events-none size-3.5">
        <DropdownMenuPrimitive.CheckboxItemIndicator>
          <CheckIcon className="size-4" />
        </DropdownMenuPrimitive.CheckboxItemIndicator>
      </span>
      {children}
    </DropdownMenuPrimitive.CheckboxItem>
  )
}

function DropdownMenuRadioGroup({
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.RadioGroup>) {
  return (
    <DropdownMenuPrimitive.RadioGroup
      data-slot="dropdown-menu-radio-group"
      {...props}
    />
  )
}

function DropdownMenuRadioItem({
  className,
  children,
  ...props
}: DropdownMenuRadioItemProps) {
  return (
    <DropdownMenuPrimitive.RadioItem
      data-slot="dropdown-menu-radio-item"
      className={cn(
        "radius-control focus:bg-accent focus:text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground relative flex cursor-pointer items-center gap-1.5 py-1.5 pr-2 pl-8 text-sm outline-hidden select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      {...props}
      closeOnClick={true}
    >
      <span className="absolute flex items-center justify-center left-2 pointer-events-none size-3.5">
        <DropdownMenuPrimitive.RadioItemIndicator>
          <CircleIcon className="fill-current size-2" />
        </DropdownMenuPrimitive.RadioItemIndicator>
      </span>
      {children}
    </DropdownMenuPrimitive.RadioItem>
  )
}

function DropdownMenuLabel({
  className,
  inset,
  ...props
}: React.ComponentProps<"div"> & {
  inset?: boolean
}) {
  return (
    <div
      data-slot="dropdown-menu-label"
      data-inset={inset}
      className={cn(
        "px-2 py-1.5 text-sm font-medium data-[inset]:pl-8",
        className
      )}
      {...props}
    />
  )
}

function DropdownMenuSeparator({
  className,
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Separator>) {
  return (
    <DropdownMenuPrimitive.Separator
      data-slot="dropdown-menu-separator"
      className={cn("bg-border -mx-1 my-1 h-px", className)}
      {...props}
    />
  )
}

function DropdownMenuShortcut({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="dropdown-menu-shortcut"
      className={cn(
        "text-muted-foreground ml-auto text-xs tracking-widest",
        className
      )}
      {...props}
    />
  )
}

function DropdownMenuSub({
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.SubmenuRoot>) {
  return <DropdownMenuPrimitive.SubmenuRoot data-slot="dropdown-menu-sub" {...props} />
}

function DropdownMenuSubTrigger({
  className,
  inset,
  children,
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.SubmenuTrigger> & {
  inset?: boolean
}) {
  return (
    <DropdownMenuPrimitive.SubmenuTrigger
      data-slot="dropdown-menu-sub-trigger"
      data-inset={inset}
      className={cn(
        "radius-control focus:bg-accent focus:text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground data-[open]:bg-accent data-[open]:text-accent-foreground flex cursor-default items-center px-2 py-1.5 text-sm outline-hidden select-none data-[inset]:pl-8",
        className
      )}
      {...props}
    >
      {children}
      <ChevronRightIcon className="ml-auto size-4" />
    </DropdownMenuPrimitive.SubmenuTrigger>
  )
}

function DropdownMenuSubContent({
  className,
  sideOffset = 4,
  align,
  side,
  alignOffset,
  collisionPadding,
  collisionBoundary,
  collisionAvoidance,
  sticky,
  positionMethod,
  ...props
}: React.ComponentProps<typeof DropdownMenuPrimitive.Popup> & MenuPositionProps) {
  return (
    <DropdownMenuPrimitive.Positioner
      data-slot="dropdown-menu-sub-positioner"
      align={align}
      side={side}
      sideOffset={sideOffset}
      alignOffset={alignOffset}
      collisionPadding={collisionPadding}
      collisionBoundary={collisionBoundary}
      collisionAvoidance={collisionAvoidance}
      sticky={sticky}
      positionMethod={positionMethod}
    >
      <DropdownMenuPrimitive.Popup
        data-slot="dropdown-menu-sub-content"
        className={cn(
          floatingSurfaceClassName,
          floatingContentMotionClassName,
          "min-w-[8rem] origin-(--transform-origin) overflow-hidden p-1 shadow-lg",
          className
        )}
        {...props}
      />
    </DropdownMenuPrimitive.Positioner>
  )
}

export {
  DropdownMenu,
  DropdownMenuPortal,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
}
