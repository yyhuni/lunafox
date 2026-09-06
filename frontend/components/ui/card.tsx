import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const cardVariants = cva(
  "radius-surface bg-card text-card-foreground flex flex-col gap-6 border border-border border-t-2 py-6 shadow-2xs",
  {
    variants: {
      variant: {
        default: "",
        shell: "",
        stat:
          "relative overflow-hidden before:pointer-events-none before:absolute before:top-2 before:left-3 before:font-mono before:text-[10px] before:tracking-[0.05em] before:text-muted-foreground before:opacity-50 before:content-[attr(data-card-index)] after:pointer-events-none after:absolute after:right-0 after:bottom-0 after:size-8 after:opacity-20 after:bg-[linear-gradient(135deg,transparent_50%,var(--border)_50%)]",
        metric:
          "relative overflow-hidden before:pointer-events-none before:absolute before:top-2 before:left-3 before:font-mono before:text-[10px] before:tracking-[0.05em] before:text-muted-foreground before:opacity-50 before:content-[attr(data-card-index)] after:pointer-events-none after:absolute after:right-0 after:bottom-0 after:size-8 after:opacity-20 after:bg-[linear-gradient(135deg,transparent_50%,var(--border)_50%)]",
      },
    },
    defaultVariants: {
      variant: "shell",
    },
  }
)

function Card({
  className,
  variant,
  ...props
}: React.ComponentProps<"div"> & VariantProps<typeof cardVariants>) {
  return (
    <div
      data-slot="card"
      data-card-variant={variant ?? "shell"}
      className={cn(cardVariants({ variant }), className)}
      {...props}
    />
  )
}

type CardSectionDensity = "default" | "compact"

function CardHeader({
  className,
  density = "default",
  ...props
}: React.ComponentProps<"div"> & { density?: CardSectionDensity }) {
  return (
    <div
      data-slot="card-header"
      data-density={density}
      className={cn(
        "@container/card-header grid auto-rows-min grid-rows-[auto_auto] items-start gap-1.5 px-6 has-data-[slot=card-action]:grid-cols-[1fr_auto] [.border-b]:pb-6",
        density === "compact" && "py-2.5 [.border-b]:pb-2.5",
        className
      )}
      {...props}
    />
  )
}

function CardTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-title"
      className={cn("leading-none", textRole.sectionTitle, className)}
      {...props}
    />
  )
}

function CardDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-description"
      className={cn(textRole.bodySubtle, className)}
      {...props}
    />
  )
}

function CardAction({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-action"
      className={cn(
        "col-start-2 row-span-2 row-start-1 self-start justify-self-end",
        className
      )}
      {...props}
    />
  )
}

function CardContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-content"
      className={cn("px-6", className)}
      {...props}
    />
  )
}

function CardFooter({
  className,
  density = "default",
  ...props
}: React.ComponentProps<"div"> & { density?: CardSectionDensity }) {
  return (
    <div
      data-slot="card-footer"
      data-density={density}
      className={cn(
        "flex items-center px-6 [.border-t]:pt-6",
        density === "compact" && "py-2.5 [.border-t]:pt-2.5",
        className
      )}
      {...props}
    />
  )
}

export {
  Card,
  CardHeader,
  CardFooter,
  CardTitle,
  CardAction,
  CardDescription,
  CardContent,
  cardVariants,
}
