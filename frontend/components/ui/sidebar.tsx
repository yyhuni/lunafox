"use client"

import * as React from "react"
import { cva, VariantProps } from "class-variance-authority"
import { ChevronLeft, PanelLeftCloseIcon, PanelLeftIcon } from "@/components/icons"
import { useLinkStatus } from "next/link"
import { useTranslations } from "next-intl"

import { useIsMobile } from "@/hooks/use-mobile"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { PolymorphicSlot } from "@/components/ui/polymorphic"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

const SIDEBAR_WIDTH = "16rem"
const SIDEBAR_WIDTH_MOBILE = "16rem"
const SIDEBAR_WIDTH_ICON = "3rem"
const SIDEBAR_COMPACT_DESKTOP_MEDIA_QUERY =
  "(min-width: 768px) and (max-width: 1279px)"
const SIDEBAR_KEYBOARD_SHORTCUT = "b"
const sidebarNavigationScopeClassName = "group/sidebar-navigation"
// Pending is visual-only; committed data-active state stays intact so router cancellation restores it.
const sidebarNavigationCommittedActiveSuppressionClassName = cn(
  "group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:bg-transparent!",
  "group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:text-sidebar-foreground/65!",
  "group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:font-medium!",
  "group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:after:hidden!",
  "group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:[&>svg:not(.ml-auto)]:text-sidebar-foreground/55!"
)
// Only the menu content moves on hover; the row itself keeps its geometry stable.
// This keeps the affordance visible without making fast pointer scans jump.
const sidebarMenuContentMotionClassName = cn(
  "[&>svg:not(.ml-auto)]:transition-transform [&>svg:not(.ml-auto)]:duration-[var(--motion-duration-fast)] [&>svg:not(.ml-auto)]:ease-[var(--motion-ease-standard)]",
  "[&>span:not([data-sidebar-navigation-pending])]:transition-transform [&>span:not([data-sidebar-navigation-pending])]:duration-[var(--motion-duration-fast)] [&>span:not([data-sidebar-navigation-pending])]:ease-[var(--motion-ease-standard)]",
  "hover:[&>svg:not(.ml-auto)]:translate-x-0.5 hover:[&>span:not([data-sidebar-navigation-pending])]:translate-x-0.5",
  "group-data-[collapsible=icon]:hover:[&>svg:not(.ml-auto)]:translate-x-0 group-data-[collapsible=icon]:hover:[&>span:not([data-sidebar-navigation-pending])]:translate-x-0",
  "motion-reduce:hover:[&>svg:not(.ml-auto)]:translate-x-0 motion-reduce:hover:[&>span:not([data-sidebar-navigation-pending])]:translate-x-0",
  "motion-reduce:[&>svg:not(.ml-auto)]:transition-none motion-reduce:[&>span:not([data-sidebar-navigation-pending])]:transition-none"
)

type SidebarContextProps = {
  state: "expanded" | "collapsed"
  open: boolean
  setOpen: (open: boolean) => void
  openMobile: boolean
  setOpenMobile: (open: boolean) => void
  isMobile: boolean
  toggleSidebar: () => void
}

const SidebarContext = React.createContext<SidebarContextProps | null>(null)

function useSidebar() {
  const context = React.useContext(SidebarContext)
  if (!context) {
    throw new Error("useSidebar must be used within a SidebarProvider.")
  }

  return context
}

const useIsomorphicLayoutEffect =
  typeof window === "undefined" ? React.useEffect : React.useLayoutEffect

function SidebarProvider({
  defaultOpen,
  open: openProp,
  onOpenChange: setOpenProp,
  className,
  style,
  children,
  ...props
}: React.ComponentProps<"div"> & {
  defaultOpen?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
}) {
  const isMobile = useIsMobile()
  const [openMobile, setOpenMobile] = React.useState(false)

  // This is the internal state of the sidebar.
  // We use openProp and setOpenProp for control from outside the component.
  const [_open, _setOpen] = React.useState(defaultOpen ?? true)
  const open = openProp ?? _open
  const setOpen = React.useCallback(
    (value: boolean | ((value: boolean) => boolean)) => {
      const openState = typeof value === "function" ? value(open) : value
      if (setOpenProp) {
        setOpenProp(openState)
      } else {
        _setOpen(openState)
      }
    },
    [setOpenProp, open]
  )

  useIsomorphicLayoutEffect(() => {
    if (openProp !== undefined || defaultOpen !== undefined) return

    const mediaQuery = window.matchMedia(SIDEBAR_COMPACT_DESKTOP_MEDIA_QUERY)
    const applyResponsiveDefault = () => {
      _setOpen(!mediaQuery.matches)
    }

    applyResponsiveDefault()
    mediaQuery.addEventListener("change", applyResponsiveDefault)
    return () => mediaQuery.removeEventListener("change", applyResponsiveDefault)
  }, [defaultOpen, openProp])

  // Helper to toggle the sidebar.
  const toggleSidebar = React.useCallback(() => {
    return isMobile ? setOpenMobile((open) => !open) : setOpen((open) => !open)
  }, [isMobile, setOpen, setOpenMobile])

  // Adds a keyboard shortcut to toggle the sidebar.
  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        event.key === SIDEBAR_KEYBOARD_SHORTCUT &&
        (event.metaKey || event.ctrlKey)
      ) {
        event.preventDefault()
        toggleSidebar()
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [toggleSidebar])

  // We add a state so that we can do data-state="expanded" or "collapsed".
  // This makes it easier to style the sidebar with Tailwind classes.
  const state = open ? "expanded" : "collapsed"

  const contextValue = React.useMemo<SidebarContextProps>(
    () => ({
      state,
      open,
      setOpen,
      isMobile,
      openMobile,
      setOpenMobile,
      toggleSidebar,
    }),
    [state, open, setOpen, isMobile, openMobile, setOpenMobile, toggleSidebar]
  )

  return (
    <SidebarContext.Provider value={contextValue}>
      <TooltipProvider delay={0}>
        <div
          data-slot="sidebar-wrapper"
          style={
            {
              "--sidebar-width": SIDEBAR_WIDTH,
              "--sidebar-width-icon": SIDEBAR_WIDTH_ICON,
              "--sidebar-top": "var(--header-height, 0px)",
              "--sidebar-height": "calc(100svh - var(--sidebar-top))",
              ...style,
            } as React.CSSProperties
          }
          className={cn(
            "group/sidebar-wrapper has-data-[variant=inset]:bg-sidebar flex min-h-svh w-full",
            className
          )}
          {...props}
        >
          {children}
        </div>
      </TooltipProvider>
    </SidebarContext.Provider>
  )
}

function Sidebar({
  side = "left",
  variant = "sidebar",
  collapsible = "offcanvas",
  className,
  children,
  ...props
}: React.ComponentProps<"div"> & {
  side?: "left" | "right"
  variant?: "sidebar" | "floating" | "inset"
  collapsible?: "offcanvas" | "icon" | "none"
}) {
  const { isMobile, state, openMobile, setOpenMobile } = useSidebar()
  const t = useTranslations("common.ui")

  if (collapsible === "none") {
    return (
      <div
        data-slot="sidebar"
        className={cn(
          "bg-sidebar text-sidebar-foreground flex h-full w-(--sidebar-width) flex-col",
          sidebarNavigationScopeClassName,
          className
        )}
        {...props}
      >
        {children}
      </div>
    )
  }

  if (isMobile) {
    return (
      <Sheet open={openMobile} onOpenChange={setOpenMobile} {...props}>
        <SheetContent
          data-sidebar="sidebar"
          data-slot="sidebar"
          data-mobile="true"
          className={cn(
            "[&>button]:hidden bg-sidebar p-0 text-sidebar-foreground w-(--sidebar-width)",
            sidebarNavigationScopeClassName
          )}
          style={
            {
              "--sidebar-width": SIDEBAR_WIDTH_MOBILE,
            } as React.CSSProperties
          }
          side={side}
        >
          <SheetHeader className="sr-only">
            <SheetTitle>{t("sidebarTitle")}</SheetTitle>
            <SheetDescription>{t("sidebarDescription")}</SheetDescription>
          </SheetHeader>
          <div className="flex flex-col h-full w-full">{children}</div>
        </SheetContent>
      </Sheet>
    )
  }

  return (
    <div
      className={cn(
        "group hidden md:block peer text-sidebar-foreground",
        sidebarNavigationScopeClassName
      )}
      data-state={state}
      data-collapsible={state === "collapsed" ? collapsible : ""}
      data-variant={variant}
      data-side={side}
      data-slot="sidebar"
    >
      {/* This is what handles the sidebar gap on desktop */}
      <div
        data-slot="sidebar-gap"
        className={cn(
          "relative h-(--sidebar-height) w-(--sidebar-width) bg-transparent transition-[width] duration-200 ease-[cubic-bezier(0.25,1,0.5,1)] motion-reduce:transition-none",
          "group-data-[collapsible=offcanvas]:w-0",
          "group-data-[side=right]:rotate-180",
          variant === "floating" || variant === "inset"
            ? "group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4)))]"
            : "group-data-[collapsible=icon]:w-(--sidebar-width-icon)"
        )}
      />
      <div
        data-slot="sidebar-container"
        className={cn(
          "fixed top-(--sidebar-top) z-10 hidden h-(--sidebar-height) w-(--sidebar-width) transition-[width,transform] duration-200 ease-[cubic-bezier(0.25,1,0.5,1)] motion-reduce:transition-none md:flex",
          side === "left"
            ? "left-0 group-data-[collapsible=offcanvas]:-translate-x-full"
            : "right-0 group-data-[collapsible=offcanvas]:translate-x-full",
          // Adjust the padding for floating and inset variants.
          variant === "floating" || variant === "inset"
            ? "p-2 group-data-[collapsible=icon]:w-[calc(var(--sidebar-width-icon)+(--spacing(4))+2px)]"
            : "group-data-[collapsible=icon]:w-(--sidebar-width-icon) group-data-[side=left]:border-r group-data-[side=right]:border-l",
          className
        )}
        {...props}
      >
        <div
          data-sidebar="sidebar"
          data-slot="sidebar-inner"
          className="bg-sidebar border-r border-sidebar-border flex flex-col group-data-[side=right]:border-l group-data-[side=right]:border-r-0 group-data-[variant=floating]:border group-data-[variant=floating]:border-sidebar-border group-data-[variant=floating]:shadow-sm h-full w-full"
        >
          {children}
        </div>
      </div>
    </div>
  )
}

function SidebarTrigger({
  className,
  onClick,
  ...props
}: React.ComponentProps<typeof Button>) {
  const { toggleSidebar, state, isMobile, openMobile } = useSidebar()
  const t = useTranslations("common.ui")
  const isSidebarOpen = isMobile ? openMobile : state === "expanded"
  const TriggerIcon = isSidebarOpen ? PanelLeftCloseIcon : PanelLeftIcon

  return (
    <Button
      data-sidebar="trigger"
      data-slot="sidebar-trigger"
      variant="ghost"
      size="icon"
      className={cn("size-7", className)}
      onClick={(event) => {
        onClick?.(event)
        toggleSidebar()
      }}
      {...props}
    >
      <TriggerIcon />
      <span className="sr-only">{t("toggleSidebar")}</span>
    </Button>
  )
}

function SidebarRail({ className, ...props }: React.ComponentProps<"button">) {
  const { toggleSidebar, state } = useSidebar()
  const t = useTranslations("common.ui")
  const isCollapsed = state === "collapsed"

  return (
    <button
      type="button"
      data-sidebar="rail"
      data-slot="sidebar-rail"
      aria-label={t("toggleSidebar")}
      tabIndex={-1}
      onClick={toggleSidebar}
      title={t("toggleSidebar")}
      className={cn(
        // Base positioning: fixed on right edge of sidebar, vertically centered
        // group/rail creates a localized hover zone just for the edge
        "absolute inset-y-0 z-20 hidden -right-3 w-6 sm:flex items-center justify-center group/rail",
        // Cursor based on direction
        "cursor-pointer",
        className
      )}
      {...props}
    >
      {/* The visible handle pill - turned into a small circle */}
      <span
        className={cn(
          "flex items-center justify-center",
          "size-6 rounded-full",
          "bg-sidebar border border-sidebar-border shadow-sm",
          // Only becomes visible when user hovers the specific edge zone, NOT the whole sidebar
          "opacity-0 group-hover/rail:opacity-100",
          "transition-[opacity,background-color,color,box-shadow,transform] duration-200 ease-in-out",
          "group-hover/rail:bg-sidebar-accent group-hover/rail:text-sidebar-accent-foreground group-hover/rail:shadow-md hover:scale-110",
          "text-sidebar-foreground/60",
        )}
      >
        <ChevronLeft
          className={cn(
            "size-3.5 transition-transform duration-200",
            isCollapsed && "rotate-180",
          )}
        />
      </span>
    </button>
  )
}

function SidebarInset({ className, id, tabIndex, ...props }: React.ComponentProps<"main">) {
  return (
    <main
      id={id ?? "main-content"}
      data-slot="sidebar-inset"
      tabIndex={tabIndex ?? -1}
      className={cn(
        "bg-transparent relative flex w-full flex-1 flex-col",
        "md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ml-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:shadow-sm md:peer-data-[variant=inset]:peer-data-[state=collapsed]:ml-2",
        className
      )}
      {...props}
    />
  )
}

function SidebarInput({
  className,
  ...props
}: React.ComponentProps<typeof Input>) {
  return (
    <Input
      data-slot="sidebar-input"
      data-sidebar="input"
      name={props.name ?? "sidebar-input"}
      autoComplete="off"
      className={cn("bg-background h-8 w-full shadow-none", className)}
      {...props}
    />
  )
}

function SidebarHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sidebar-header"
      data-sidebar="header"
      className={cn("border-b border-sidebar-border bg-sidebar flex flex-col gap-2 p-2", className)}
      {...props}
    />
  )
}

function SidebarFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sidebar-footer"
      data-sidebar="footer"
      className={cn("flex flex-col gap-2 p-2", className)}
      {...props}
    />
  )
}

function SidebarSeparator({
  className,
  ...props
}: React.ComponentProps<typeof Separator>) {
  return (
    <Separator
      data-slot="sidebar-separator"
      data-sidebar="separator"
      className={cn("bg-sidebar-border mx-2 w-auto", className)}
      {...props}
    />
  )
}

function SidebarContent({ className, ...props }: React.ComponentProps<"div">) {
  const { children, ...rootProps } = props

  // Overlay scroll chrome keeps scrollbar visibility from changing menu row width.
  return (
    <ScrollArea
      data-slot="sidebar-content"
      data-sidebar="content"
      type="always"
      className={cn("min-h-0 flex-1", className)}
      contentClassName="flex min-h-full !min-w-0 w-full flex-col gap-0"
      {...rootProps}
    >
      {children}
    </ScrollArea>
  )
}

function SidebarGroup({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sidebar-group"
      data-sidebar="group"
      className={cn("relative flex w-full min-w-0 flex-col py-1", className)}
      {...props}
    />
  )
}

function SidebarGroupLabel({
  className,
  render,
  ...props
}: React.ComponentProps<"div"> & { render?: React.ReactElement }) {
  return (
    <PolymorphicSlot
      defaultTagName="div"
      render={render}
      data-slot="sidebar-group-label"
      data-sidebar="group-label"
      className={cn(
        "pointer-events-none ring-sidebar-ring flex h-8 shrink-0 items-center overflow-hidden whitespace-nowrap px-3 outline-hidden transition-[opacity] duration-100 ease-[cubic-bezier(0.25,1,0.5,1)] motion-reduce:transition-none focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0",
        textRole.helperText,
        "group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0 group-data-[collapsible=icon]:transition-none",
        className
      )}
      {...props}
    />
  )
}

function SidebarGroupAction({
  className,
  render,
  ...props
}: React.ComponentProps<"button"> & { render?: React.ReactElement }) {
  return (
    <PolymorphicSlot
      defaultTagName="button"
      render={render}
      data-slot="sidebar-group-action"
      data-sidebar="group-action"
      className={cn(
        "text-sidebar-foreground ring-sidebar-ring hover:bg-sidebar-accent hover:text-sidebar-accent-foreground absolute top-3.5 right-3 flex aspect-square w-5 items-center justify-center rounded-md p-0 outline-hidden transition-transform focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0",
        // Increases the hit area of the button on mobile.
        "after:absolute after:-inset-2 md:after:hidden",
        "group-data-[collapsible=icon]:hidden",
        className
      )}
      {...props}
    />
  )
}

function SidebarGroupContent({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sidebar-group-content"
      data-sidebar="group-content"
      className={cn("w-full px-2 text-sm", className)}
      {...props}
    />
  )
}

function SidebarMenu({ className, ...props }: React.ComponentProps<"ul">) {
  return (
    <ul
      data-slot="sidebar-menu"
      data-sidebar="menu"
      className={cn("flex w-full min-w-0 flex-col gap-0.5", className)}
      {...props}
    />
  )
}

function SidebarMenuItem({ className, ...props }: React.ComponentProps<"li">) {
  return (
    <li
      data-slot="sidebar-menu-item"
      data-sidebar="menu-item"
      className={cn("group/menu-item relative m-0 p-0", className)}
      {...props}
    />
  )
}

function SidebarNavigationPendingIndicator({
  className,
}: {
  className?: string
}) {
  const { pending } = useLinkStatus()

  if (!pending) return null

  return (
    <span
      data-sidebar-navigation-pending="true"
      data-loading-layer="compact-pending"
      data-loading-intent="navigation"
      data-loading-phase="loading"
      aria-hidden="true"
      className={cn("pointer-events-none absolute! inset-0 z-0!", className)}
    />
  )
}

const sidebarMenuButtonVariants = cva(
  cn(
    // Keep the active marker centered on the disclosure-chevron axis without changing layout.
    "peer/menu-button ring-sidebar-ring relative flex h-auto min-h-8 w-full items-center gap-3 overflow-hidden rounded-lg bg-transparent px-2.5 py-1.5 text-left outline-hidden focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 group-has-data-[sidebar=menu-action]/menu-item:pr-8 [&>span]:truncate [&>span]:leading-5 [&>svg]:size-4 [&>svg]:shrink-0 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:[&>svg:not(.ml-auto)]:text-sidebar-accent-foreground has-data-[sidebar-navigation-pending=true]:bg-sidebar-accent/70 has-data-[sidebar-navigation-pending=true]:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground data-[active=true]:font-semibold data-[active=true]:after:absolute data-[active=true]:after:right-4 data-[active=true]:after:translate-x-px data-[active=true]:after:top-1/2 data-[active=true]:after:z-20 data-[active=true]:after:block data-[active=true]:after:size-1.5 data-[active=true]:after:-translate-y-1/2 data-[active=true]:after:rounded-full data-[active=true]:after:bg-sidebar-primary data-[active=true]:after:content-[''] data-[active=true]:[&>svg:not(.ml-auto)]:text-sidebar-primary has-[>svg.ml-auto]:data-[active=true]:after:hidden data-[sidebar-has-badge=true]:data-[active=true]:after:hidden data-[panel-open]:[&>svg.ml-auto]:rotate-90 group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-2! group-data-[collapsible=icon]:after:hidden group-data-[collapsible=icon]:data-[active=true]:after:hidden",
    sidebarMenuContentMotionClassName,
    sidebarNavigationCommittedActiveSuppressionClassName,
    textRole.navLabel,
    "text-sidebar-foreground/65 [&>svg:not(.ml-auto)]:text-sidebar-foreground/55"
  ),
  {
    variants: {
      variant: {
        default: "",
        outline:
          "border border-border bg-background shadow-xs hover:border-border",
      },
      size: {
        default: "",
        sm: "min-h-8 text-xs",
        lg: "min-h-12 text-sm group-data-[collapsible=icon]:p-0!",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function SidebarMenuButton({
  render,
  isActive = false,
  variant = "default",
  size = "default",
  tooltip,
  className,
  ...props
}: React.ComponentProps<"button"> & {
  render?: React.ReactElement
  isActive?: boolean
  tooltip?: string | React.ComponentProps<typeof TooltipContent>
} & VariantProps<typeof sidebarMenuButtonVariants>) {
  const button = (
    <PolymorphicSlot
      defaultTagName="button"
      render={render}
      data-slot="sidebar-menu-button"
      data-sidebar="menu-button"
      data-size={size}
      data-active={isActive}
      className={cn(sidebarMenuButtonVariants({ variant, size }), className)}
      {...props}
    />
  )

  if (!tooltip) {
    return button
  }

  return <SidebarMenuButtonTooltip button={button} tooltip={tooltip} />
}

function SidebarMenuButtonTooltip({
  button,
  tooltip,
}: {
  button: React.ReactElement
  tooltip: string | React.ComponentProps<typeof TooltipContent>
}) {
  const { isMobile, state } = useSidebar()

  if (typeof tooltip === "string") {
    tooltip = {
      children: tooltip,
    }
  }

  return (
    <Tooltip>
      <TooltipTrigger render={button} />
      <TooltipContent
        side="right"
        align="center"
        hidden={state !== "collapsed" || isMobile}
        {...tooltip}
      />
    </Tooltip>
  )
}

function SidebarMenuAction({
  className,
  render,
  showOnHover = false,
  ...props
}: React.ComponentProps<"button"> & {
  render?: React.ReactElement
  showOnHover?: boolean
}) {
  return (
    <PolymorphicSlot
      defaultTagName="button"
      render={render}
      data-slot="sidebar-menu-action"
      data-sidebar="menu-action"
      className={cn(
        "text-sidebar-foreground ring-sidebar-ring hover:bg-sidebar-accent hover:text-sidebar-accent-foreground peer-hover/menu-button:text-sidebar-accent-foreground absolute top-1.5 right-1 flex aspect-square w-5 items-center justify-center rounded-md p-0 outline-hidden transition-transform focus-visible:ring-2 [&>svg]:size-4 [&>svg]:shrink-0",
        // Increases the hit area of the button on mobile.
        "after:absolute after:-inset-2 md:after:hidden",
        "peer-data-[size=sm]/menu-button:top-1",
        "peer-data-[size=default]/menu-button:top-1.5",
        "peer-data-[size=lg]/menu-button:top-2.5",
        "group-data-[collapsible=icon]:hidden",
        showOnHover &&
        "peer-data-[active=true]/menu-button:text-sidebar-accent-foreground group-focus-within/menu-item:opacity-100 group-hover/menu-item:opacity-100 data-[popup-open]:opacity-100 md:opacity-0",
        className
      )}
      {...props}
    />
  )
}

function SidebarMenuBadge({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sidebar-menu-badge"
      data-sidebar="menu-badge"
      className={cn(
        "text-sidebar-foreground pointer-events-none absolute right-1 flex h-5 min-w-5 items-center justify-center rounded-md px-1 tabular-nums select-none",
        textRole.badge,
        "peer-hover/menu-button:text-sidebar-accent-foreground peer-data-[active=true]/menu-button:text-sidebar-accent-foreground",
        "peer-data-[size=sm]/menu-button:top-1",
        "peer-data-[size=default]/menu-button:top-1.5",
        "peer-data-[size=lg]/menu-button:top-2.5",
        "group-data-[collapsible=icon]:hidden",
        className
      )}
      {...props}
    />
  )
}

function SidebarMenuSkeleton({
  className,
  showIcon = false,
  ...props
}: React.ComponentProps<"div"> & {
  showIcon?: boolean
}) {
  // Random width between 50 to 90%.
  const width = React.useMemo(() => {
    return `${Math.floor(Math.random() * 40) + 50}%`
  }, [])

  return (
    <div
      data-slot="sidebar-menu-skeleton"
      data-sidebar="menu-skeleton"
      className={cn("flex h-8 items-center gap-2 rounded-md px-2", className)}
      {...props}
    >
      {showIcon && (
        <Skeleton
          className="rounded-sm size-4"
          data-sidebar="menu-skeleton-icon"
        />
      )}
      <Skeleton
        className="flex-1 h-4 max-w-(--skeleton-width)"
        data-sidebar="menu-skeleton-text"
        style={
          {
            "--skeleton-width": width,
          } as React.CSSProperties
        }
      />
    </div>
  )
}

function SidebarMenuSub({ className, ...props }: React.ComponentProps<"ul">) {
  return (
    <ul
      data-slot="sidebar-menu-sub"
      data-sidebar="menu-sub"
      className={cn(
        "flex min-w-0 flex-col gap-2 px-0 py-1",
        "group-data-[collapsible=icon]:hidden",
        className
      )}
      {...props}
    />
  )
}

function SidebarMenuSubItem({
  className,
  ...props
}: React.ComponentProps<"li">) {
  return (
    <li
      data-slot="sidebar-menu-sub-item"
      data-sidebar="menu-sub-item"
      className={cn("group/menu-sub-item relative", className)}
      {...props}
    />
  )
}

function SidebarMenuSubButton({
  render,
  size = "md",
  isActive = false,
  className,
  ...props
}: React.ComponentProps<"a"> & {
  render?: React.ReactElement
  size?: "sm" | "md"
  isActive?: boolean
}) {
  return (
    <PolymorphicSlot
      defaultTagName="a"
      render={render}
      data-slot="sidebar-menu-sub-button"
      data-sidebar="menu-sub-button"
      data-size={size}
      data-active={isActive}
      className={cn(
        "ring-sidebar-ring relative flex min-h-8 min-w-0 items-center gap-2 rounded-lg bg-transparent py-1.5 pr-6 pl-3.5 outline-hidden focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&>span]:truncate [&>span]:leading-5 [&>svg]:size-4 [&>svg]:shrink-0 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground has-data-[sidebar-navigation-pending=true]:bg-sidebar-accent/70 has-data-[sidebar-navigation-pending=true]:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground data-[active=true]:font-medium data-[active=true]:after:absolute data-[active=true]:after:right-2 data-[active=true]:after:top-1/2 data-[active=true]:after:size-1.5 data-[active=true]:after:-translate-y-1/2 data-[active=true]:after:rounded-full data-[active=true]:after:bg-sidebar-primary data-[active=true]:after:content-['']",
        sidebarMenuContentMotionClassName,
        sidebarNavigationCommittedActiveSuppressionClassName,
        size === "sm" ? textRole.helperText : textRole.navLabel,
        "text-sidebar-foreground/65",
        "group-data-[collapsible=icon]:hidden",
        className
      )}
      {...props}
    />
  )
}

export {
  sidebarNavigationCommittedActiveSuppressionClassName,
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInput,
  SidebarInset,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarNavigationPendingIndicator,
  SidebarMenuSkeleton,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
  useSidebar,
}
