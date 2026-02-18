import * as React from "react"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface SmartFilterInputFieldProps {
  value: string
  ghostText?: string
  inputRef: React.RefObject<HTMLInputElement | null>
  ghostRef: React.RefObject<HTMLSpanElement | null>
  placeholder?: string
  onChange: (value: string) => void
  onFocus: () => void
  onBlur: (event: React.FocusEvent<HTMLInputElement>) => void
  onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void
  className?: string
}

export function SmartFilterInputField({
  value,
  ghostText,
  inputRef,
  ghostRef,
  placeholder,
  onChange,
  onFocus,
  onBlur,
  onKeyDown,
  className,
}: SmartFilterInputFieldProps) {
  return (
    <div className={cn("relative flex-1", className)}>
      <Input
        ref={inputRef}
        type="search"
        name="smartQuery"
        autoComplete="off"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onFocus={onFocus}
        onBlur={onBlur}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        className="h-8 w-full font-mono text-sm"
      />
      {ghostText && (
        <div
          className="absolute inset-0 flex items-center pointer-events-none overflow-hidden px-3"
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
