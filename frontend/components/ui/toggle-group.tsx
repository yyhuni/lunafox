"use client"

import * as React from "react"
import { Toggle as TogglePrimitive } from "@base-ui/react/toggle"
import { ToggleGroup as ToggleGroupPrimitive } from "@base-ui/react/toggle-group"
import { type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"
import { toggleVariants } from "@/components/ui/toggle"

const ToggleGroupContext = React.createContext<
  VariantProps<typeof toggleVariants>
>({
  size: "default",
  variant: "default",
})

type ToggleGroupType = "single" | "multiple"

type ToggleGroupValue<TType extends ToggleGroupType | undefined> =
  TType extends "multiple" ? string[] : string

type BaseToggleGroupProps = Omit<
  ToggleGroupPrimitive.Props,
  "defaultValue" | "multiple" | "onValueChange" | "value"
>

type ToggleGroupProps<TType extends ToggleGroupType | undefined = "single"> =
  BaseToggleGroupProps &
    VariantProps<typeof toggleVariants> & {
      type?: TType
      value?: string | string[]
      defaultValue?: string | string[]
      onValueChange?: (value: string | string[]) => void
    }

function toBaseGroupValue(value: string | string[] | undefined): string[] | undefined {
  if (value === undefined) return undefined
  return Array.isArray(value) ? value : [value]
}

function fromBaseGroupValue<TType extends ToggleGroupType | undefined>(
  type: TType,
  value: string[]
): ToggleGroupValue<TType> {
  return (type === "multiple" ? value : (value[0] ?? "")) as ToggleGroupValue<TType>
}

function ToggleGroup<TType extends ToggleGroupType | undefined = "single">({
  className,
  variant,
  size,
  children,
  type,
  value,
  defaultValue,
  onValueChange,
  ...props
}: ToggleGroupProps<TType>) {
  return (
    <ToggleGroupPrimitive
      data-slot="toggle-group"
      data-variant={variant}
      data-size={size}
      multiple={type === "multiple"}
      value={toBaseGroupValue(value)}
      defaultValue={toBaseGroupValue(defaultValue)}
      onValueChange={(nextValue) => {
        onValueChange?.(fromBaseGroupValue(type, nextValue))
      }}
      className={cn(
        "group/toggle-group radius-surface flex w-fit items-center data-[variant=outline]:shadow-xs",
        className
      )}
      {...props}
    >
      <ToggleGroupContext.Provider value={{ variant, size }}>
        {children}
      </ToggleGroupContext.Provider>
    </ToggleGroupPrimitive>
  )
}

type ToggleGroupItemProps = React.ComponentProps<typeof TogglePrimitive> &
  VariantProps<typeof toggleVariants>

function ToggleGroupItem({
  className,
  children,
  variant,
  size,
  ...props
}: ToggleGroupItemProps) {
  const context = React.useContext(ToggleGroupContext)

  return (
    <TogglePrimitive
      data-slot="toggle-group-item"
      data-variant={context.variant || variant}
      data-size={context.size || size}
      className={cn(
        toggleVariants({
          variant: context.variant || variant,
          size: context.size || size,
        }),
        "min-w-0 flex-1 shrink-0 shadow-none first:radius-control-left last:radius-control-right focus:z-10 focus-visible:z-10 data-[variant=outline]:border-l-0 data-[variant=outline]:first:border-l",
        className
      )}
      {...props}
    >
      {children}
    </TogglePrimitive>
  )
}

export { ToggleGroup, ToggleGroupItem }
