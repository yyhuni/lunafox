import * as React from "react"
import { IconSearch } from "@/components/icons"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import type { DataTableToolbarDensity } from "@/types/data-table.types"

interface SmartFilterInputFieldProps {
  value: string
  ghostText?: string
  inputRef: React.RefObject<HTMLInputElement | null>
  ghostRef: React.RefObject<HTMLSpanElement | null>
  placeholder?: string
  onChange: (value: string) => void
  onFocus: () => void
  onClick?: () => void
  onBlur: (event: React.FocusEvent<HTMLInputElement>) => void
  onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void
  showIcon?: boolean
  toolbarDensity?: DataTableToolbarDensity
  className?: string
  inputClassName?: string
  ref?: React.Ref<HTMLDivElement>
}

export function SmartFilterInputField({
  value,
  ghostText,
  inputRef,
  ghostRef,
  placeholder,
  onChange,
  onFocus,
  onClick,
  onBlur,
  onKeyDown,
  showIcon = true,
  toolbarDensity = "compact",
  className,
  inputClassName,
  ref,
}: SmartFilterInputFieldProps) {
  const isStandardDensity = toolbarDensity === "standard"

  return (
    <div ref={ref} className={cn("relative flex-1", className)}>
      {showIcon ? (
        <IconSearch
          aria-hidden="true"
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
      ) : null}
      <Input
        ref={inputRef}
        type="search"
        size={isStandardDensity ? "default" : "sm"}
        name="smartQuery"
        autoComplete="off"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onFocus={onFocus}
        onClick={onClick}
        onBlur={onBlur}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        className={cn(
          "w-full font-mono text-sm",
          showIcon && "pl-9",
          inputClassName
        )}
      />
      {ghostText && (
        <div
          className={cn(
            "absolute inset-0 flex items-center overflow-hidden pointer-events-none",
            showIcon ? "pl-9 pr-3" : "px-3"
          )}
          aria-hidden="true"
        >
          <span className="font-mono text-sm">
            <span className="invisible">{value}</span>
            <span ref={ghostRef} className="text-muted-foreground/40">{ghostText}</span>
          </span>
        </div>
      )}
    </div>
  )
}
