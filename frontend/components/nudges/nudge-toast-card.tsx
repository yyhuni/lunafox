"use client"

import * as React from "react"
import type { VariantProps } from "class-variance-authority"

import { IconX } from "@/components/icons"
import { Button, buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

type ButtonVariant = VariantProps<typeof buttonVariants>["variant"]
type ButtonSize = VariantProps<typeof buttonVariants>["size"]

export interface NudgeToastAction {
  label: string
  icon?: React.ReactNode
  onClick?: () => void
  href?: string
  target?: string
  rel?: string
  className?: string
  buttonVariant?: ButtonVariant
  buttonSize?: ButtonSize
}

export interface NudgeToastCardProps {
  title: string
  description: string

  icon: React.ReactNode
  iconHref?: string
  iconOnClick?: () => void
  iconClassName?: string

  primaryAction: NudgeToastAction
  secondaryAction?: NudgeToastAction

  onDismiss: () => void
  closeAriaLabel?: string
  className?: string
}

function ActionButton({
  action,
  fallbackVariant,
  fallbackSize = "sm",
  className,
}: {
  action: NudgeToastAction
  fallbackVariant: ButtonVariant
  fallbackSize?: ButtonSize
  className?: string
}) {
  const variant = action.buttonVariant ?? fallbackVariant
  const size = action.buttonSize ?? fallbackSize
  const content = (
    <>
      {action.icon}
      {action.label}
    </>
  )

  if (action.href) {
    const target = action.target ?? "_blank"
    const rel = action.rel ?? "noopener noreferrer"
    return (
      <Button
        variant={variant}
        size={size}
        className={cn(className, action.className)}
        asChild
      >
        <a href={action.href} target={target} rel={rel} onClick={action.onClick}>
          {content}
        </a>
      </Button>
    )
  }

  return (
    <Button
      variant={variant}
      size={size}
      className={cn(className, action.className)}
      onClick={action.onClick}
      type="button"
    >
      {content}
    </Button>
  )
}

export function NudgeToastCard({
  title,
  description,
  icon,
  iconHref,
  iconOnClick,
  iconClassName,
  primaryAction,
  secondaryAction,
  onDismiss,
  closeAriaLabel = "Close",
  className,
}: NudgeToastCardProps) {
  const iconBaseClassName =
    "flex size-12 shrink-0 cursor-pointer items-center justify-center rounded-2xl border bg-background text-3xl shadow-sm transition-transform hover:scale-105 active:scale-95"

  return (
    <div
      className={cn(
        "relative flex w-full max-w-sm flex-col gap-4 rounded-xl border bg-background p-4 shadow-xl",
        "data-[state=open]:animate-in data-[state=closed]:animate-out",
        "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
        "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
        "data-[side=bottom]:slide-in-from-top-2 data-[side=top]:slide-in-from-bottom-2",
        "sm:w-[380px]",
        className
      )}
    >
      {/* Close Button */}
      <button
        type="button"
        onClick={onDismiss}
        aria-label={closeAriaLabel}
        className="absolute right-2 top-2 rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
      >
        <IconX className="size-4" />
      </button>

      <div className="flex gap-4">
        {iconHref ? (
          <a
            href={iconHref}
            target="_blank"
            rel="noopener noreferrer"
            onClick={iconOnClick}
            className={cn(iconBaseClassName, iconClassName)}
          >
            {icon}
          </a>
        ) : iconOnClick ? (
          <button
            type="button"
            onClick={iconOnClick}
            className={cn(iconBaseClassName, iconClassName)}
          >
            {icon}
          </button>
        ) : (
          <div className={cn(iconBaseClassName, "cursor-default", iconClassName)}>
            {icon}
          </div>
        )}

        <div className="flex flex-col gap-1.5">
          <h3 className="font-semibold leading-none tracking-tight">{title}</h3>
          <p className="text-sm leading-relaxed text-muted-foreground">
            {description}
          </p>
        </div>
      </div>

      <div className="flex w-full items-center gap-2">
        {secondaryAction && (
          <ActionButton
            action={secondaryAction}
            fallbackVariant="outline"
            className="flex-1 text-muted-foreground"
          />
        )}
        <ActionButton
          action={primaryAction}
          fallbackVariant="default"
          className="flex-1 gap-2"
        />
      </div>
    </div>
  )
}
