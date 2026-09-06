"use client"

import * as React from "react"
import { Dialog as SheetPrimitive } from "@base-ui/react/dialog"
import { useTranslations } from "next-intl"

import { semanticIcons } from "@/components/icons"
import { Button } from "@/components/ui/button"

import {
  edgeOverlayBackdropClassName,
  edgeOverlayPanelBaseClassName,
} from "@/lib/ui/overlay-styles"
import { useStagedEdgePanelOpen } from "@/lib/ui/use-staged-edge-panel-open"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

type SheetRootProps = React.ComponentProps<typeof SheetPrimitive.Root>
type SheetEscapeKeyDownHandler = ((event: KeyboardEvent) => void) | undefined

type SheetContextValue = {
  setEscapeKeyDownHandler: (handler: SheetEscapeKeyDownHandler) => () => void
}

const SheetContext = React.createContext<SheetContextValue | null>(null)

function Sheet({ open, onOpenChange, children, ...props }: SheetRootProps) {
  const escapeKeyDownHandlerRef = React.useRef<SheetEscapeKeyDownHandler>(undefined)
  const stagedOpen = useStagedEdgePanelOpen(open)

  const setEscapeKeyDownHandler = React.useCallback(
    (handler: SheetEscapeKeyDownHandler) => {
      escapeKeyDownHandlerRef.current = handler

      return () => {
        if (escapeKeyDownHandlerRef.current === handler) {
          escapeKeyDownHandlerRef.current = undefined
        }
      }
    },
    []
  )

  const handleOpenChange = React.useCallback<NonNullable<SheetRootProps["onOpenChange"]>>(
    (open, eventDetails) => {
      if (!open && eventDetails.reason === "escape-key" && escapeKeyDownHandlerRef.current) {
        escapeKeyDownHandlerRef.current(eventDetails.event)

        if (eventDetails.event.defaultPrevented) {
          eventDetails.cancel()
        }
      }

      onOpenChange?.(open, eventDetails)
    },
    [onOpenChange]
  )

  const contextValue = React.useMemo<SheetContextValue>(
    () => ({ setEscapeKeyDownHandler }),
    [setEscapeKeyDownHandler]
  )

  return (
    <SheetContext.Provider value={contextValue}>
      <SheetPrimitive.Root
        data-slot="sheet"
        open={stagedOpen}
        onOpenChange={handleOpenChange}
        {...props}
      >
        {children}
      </SheetPrimitive.Root>
    </SheetContext.Provider>
  )
}

function SheetTrigger({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Trigger>) {
  return (
    <SheetPrimitive.Trigger
      data-slot="sheet-trigger"
      {...props}
    />
  )
}

function SheetClose({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Close>) {
  return (
    <SheetPrimitive.Close
      data-slot="sheet-close"
      {...props}
    />
  )
}

function SheetPortal({
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Portal>) {
  return <SheetPrimitive.Portal data-slot="sheet-portal" {...props} />
}

function SheetOverlay({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Backdrop>) {
  return (
    <SheetPrimitive.Backdrop
      data-slot="sheet-overlay"
      className={cn(edgeOverlayBackdropClassName, className)}
      {...props}
    />
  )
}

function SheetContent({
  className,
  children,
  onEscapeKeyDown,
  showCloseButton = true,
  side = "right",
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Popup> & {
  onEscapeKeyDown?: (event: KeyboardEvent) => void
  showCloseButton?: boolean
  side?: "top" | "right" | "bottom" | "left"
}) {
  const sheetContext = React.useContext(SheetContext)
  const tActions = useTranslations("common.actions")

  React.useEffect(() => {
    return sheetContext?.setEscapeKeyDownHandler(onEscapeKeyDown)
  }, [onEscapeKeyDown, sheetContext])

  return (
    <SheetPortal>
      <SheetOverlay />
      <SheetPrimitive.Popup
        data-slot="sheet-content"
        data-side={side}
        className={cn(
          edgeOverlayPanelBaseClassName,
          side === "right" &&
            "inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm",
          side === "right" &&
            "data-starting-style:translate-x-full data-ending-style:translate-x-full",
          side === "left" &&
            "inset-y-0 left-0 h-full w-3/4 border-r sm:max-w-sm",
          side === "left" &&
            "data-starting-style:-translate-x-full data-ending-style:-translate-x-full",
          side === "top" &&
            "inset-x-0 top-0 h-auto border-b",
          side === "top" &&
            "data-starting-style:-translate-y-full data-ending-style:-translate-y-full",
          side === "bottom" &&
            "inset-x-0 bottom-0 h-auto border-t",
          side === "bottom" &&
            "data-starting-style:translate-y-full data-ending-style:translate-y-full",
          className
        )}
        {...props}
      >
        {children}
        {showCloseButton && (
          <SheetPrimitive.Close
            data-slot="sheet-close"
            render={<Button type="button" variant="ghost" size="icon-sm" aria-label={tActions("close")} className="absolute top-4 right-4 opacity-70 hover:opacity-100" />}
          >
            <semanticIcons.action.cancel />
          </SheetPrimitive.Close>
        )}
      </SheetPrimitive.Popup>
    </SheetPortal>
  )
}

function SheetHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-header"
      className={cn("flex flex-col gap-1.5 p-4", className)}
      {...props}
    />
  )
}

function SheetFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-footer"
      className={cn("mt-auto flex flex-col gap-2 p-4", className)}
      {...props}
    />
  )
}

function SheetTitle({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Title>) {
  return (
    <SheetPrimitive.Title
      data-slot="sheet-title"
      className={cn(textRole.panelTitle, className)}
      {...props}
    />
  )
}

function SheetDescription({
  className,
  ...props
}: React.ComponentProps<typeof SheetPrimitive.Description>) {
  return (
    <SheetPrimitive.Description
      data-slot="sheet-description"
      className={cn(textRole.bodySubtle, className)}
      {...props}
    />
  )
}

export {
  Sheet,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
}
