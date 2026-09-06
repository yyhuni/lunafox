"use client"

import * as React from "react"
import { Button as ButtonPrimitive } from "@base-ui/react/button"

import { Minus, Plus } from "@/components/icons"
import { cn } from "@/lib/utils"

interface NumberStepperInputProps
  extends Omit<React.ComponentProps<"input">, "onChange" | "size" | "type" | "value"> {
  value: number
  min?: number
  max?: number
  step?: number
  onChange: (value: number) => void
}

function clampNumber(value: number, min?: number, max?: number) {
  let nextValue = value
  if (min !== undefined) nextValue = Math.max(nextValue, min)
  if (max !== undefined) nextValue = Math.min(nextValue, max)
  return nextValue
}

export function NumberStepperInput({
  className,
  value,
  min,
  max,
  step = 1,
  disabled,
  onChange,
  ...props
}: NumberStepperInputProps) {
  const changeBy = React.useCallback(
    (delta: number) => {
      onChange(clampNumber(value + delta, min, max))
    },
    [max, min, onChange, value]
  )

  const canDecrement = !disabled && (min === undefined || value > min)
  const canIncrement = !disabled && (max === undefined || value < max)

  return (
    <div
      data-slot="number-stepper-input"
      data-disabled={disabled ? "" : undefined}
      className={cn(
        "radius-control border-input focus-within:border-ring focus-within:ring-ring/50 bg-background dark:bg-input/30 relative inline-flex h-8 w-full min-w-0 items-center overflow-hidden border text-base whitespace-nowrap transition-colors outline-none focus-within:ring-[3px] data-[disabled]:pointer-events-none data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 md:text-sm",
        className
      )}
    >
      <ButtonPrimitive
        type="button"
        aria-label="减少"
        disabled={!canDecrement}
        className="border-input bg-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground -ms-px flex aspect-square h-[inherit] items-center justify-center border text-sm transition-colors disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50"
        onClick={() => changeBy(-step)}
      >
        <Minus className="size-4" />
        <span className="sr-only">减少</span>
      </ButtonPrimitive>
      <input
        {...props}
        type="number"
        value={Number.isFinite(value) ? String(value) : ""}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        className="selection:bg-primary selection:text-primary-foreground min-w-0 grow bg-transparent px-2.5 py-1 text-center tabular-nums outline-none [appearance:textfield] disabled:cursor-not-allowed [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
        onChange={(event) => {
          const nextValue = event.target.value === "" ? 0 : Number(event.target.value)
          if (Number.isNaN(nextValue)) return
          onChange(clampNumber(nextValue, min, max))
        }}
      />
      <ButtonPrimitive
        type="button"
        aria-label="增加"
        disabled={!canIncrement}
        className="border-input bg-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground -me-px flex aspect-square h-[inherit] items-center justify-center border text-sm transition-colors disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50"
        onClick={() => changeBy(step)}
      >
        <Plus className="size-4" />
        <span className="sr-only">增加</span>
      </ButtonPrimitive>
    </div>
  )
}
