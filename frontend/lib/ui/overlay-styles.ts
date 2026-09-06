export const floatingContentMotionClassName =
  "data-[open]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[open]:fade-in-0 data-[ending-style]:zoom-out-95 data-[starting-style]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"

export const floatingSurfaceClassName =
  "z-50 radius-overlay border bg-popover p-4 text-popover-foreground shadow-md outline-none"

// Shell triggers sit inside header or sidebar padding, so their popup offsets
// must include that inset to preserve a visible gap from the shell boundary.
export const shellOverlaySideOffsets = {
  header: 8,
  sidebarDesktop: 16,
} as const

const overlayBackdropBaseClassName =
  "fixed inset-0 z-50 bg-(--overlay-backdrop-background) transition-opacity duration-150 data-[open]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[open]:fade-in-0 data-[ending-style]:opacity-0 data-[starting-style]:opacity-0"

export const overlayBackdropClassName =
  `${overlayBackdropBaseClassName} supports-backdrop-filter:backdrop-blur-xs`

const edgeOverlayBackdropBaseClassName =
  "fixed inset-0 z-50 bg-(--overlay-backdrop-background) transition-opacity duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)] data-ending-style:pointer-events-none data-ending-style:opacity-0 data-starting-style:opacity-0"

export const edgeOverlayBackdropClassName =
  `${edgeOverlayBackdropBaseClassName} supports-backdrop-filter:backdrop-blur-xs`

const drawerOverlayBackdropBaseClassName =
  "fixed inset-0 z-50 min-h-dvh bg-(--overlay-backdrop-background) [--drawer-overlay-min-opacity:0] opacity-[max(var(--drawer-overlay-min-opacity),calc(1-var(--drawer-swipe-progress)))] transition-opacity duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)] select-none data-ending-style:pointer-events-none data-ending-style:opacity-0 data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)] data-snap-points:[--drawer-overlay-min-opacity:0.5] data-starting-style:opacity-0 data-swiping:duration-0 supports-[-webkit-touch-callout:none]:absolute"

export const drawerOverlayBackdropClassName =
  `${drawerOverlayBackdropBaseClassName} supports-backdrop-filter:backdrop-blur-xs`

export const drawerPopupClassName =
  "pointer-events-auto fixed z-50 flex min-w-0 max-w-full flex-col gap-4 border bg-card text-sm text-card-foreground shadow-lg outline-none select-none [--drawer-inset:0px] transition-transform duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)] will-change-transform data-ending-style:duration-[calc(var(--drawer-swipe-strength)*400ms)] data-starting-style:transform-(--closed-transform) data-ending-style:transform-(--closed-transform) data-swiping:duration-0 data-ending-style:data-swiping:duration-[calc(var(--drawer-swipe-strength)*400ms)]"

export const drawerPanelMotionClassName =
  "transform-gpu will-change-transform"

export const centeredOverlayPanelClassName =
  "fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 radius-overlay border bg-card p-6 text-card-foreground shadow-lg duration-200 data-[open]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[open]:fade-in-0 data-[ending-style]:zoom-out-95 data-[starting-style]:zoom-in-95"

export const scrollableFormDialogContentClassName =
  "max-h-[90vh] overflow-y-auto sm:max-w-[650px]"

export const formDrawerContentClassName =
  `${drawerPanelMotionClassName} min-w-0 w-full max-w-full gap-0 p-0 sm:max-w-[650px]`

export const scanWorkbenchDrawerContentClassName =
  `${drawerPanelMotionClassName} flex min-w-0 w-full max-w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl lg:max-w-3xl xl:max-w-4xl`

export const editorDialogPanelClassName =
  "flex h-[90vh] flex-col p-0"

export const narrowFormDialogContentClassName =
  "sm:max-w-md"

export const edgeOverlayPanelBaseClassName =
  "fixed z-50 flex min-w-0 max-w-full flex-col gap-4 border border-border bg-card text-card-foreground shadow-lg transition-transform duration-[var(--motion-duration-shell)] ease-[var(--motion-ease-standard)] will-change-transform"

export const detailDrawerContentClassName =
  `${drawerPanelMotionClassName} min-w-0 w-full max-w-full p-0 sm:max-w-5xl`

export const workbenchLogDrawerContentClassName =
  `${drawerPanelMotionClassName} min-w-0 w-full max-w-full gap-0 p-0 sm:max-w-4xl`

export const compactFeedbackDrawerContentClassName =
  `${drawerPanelMotionClassName} flex min-w-0 w-full max-w-full flex-col gap-0 p-0 sm:max-w-[420px]`

export const tooltipSurfaceClassName =
  "z-50 inline-flex w-fit max-w-xs origin-(--transform-origin) items-center gap-1.5 radius-overlay bg-foreground px-3 py-1.5 text-xs text-background has-data-[slot=kbd]:pr-1.5 **:data-[slot=kbd]:relative **:data-[slot=kbd]:isolate **:data-[slot=kbd]:z-50 **:data-[slot=kbd]:rounded-sm"

export const tooltipArrowClassName =
  "z-50 size-2.5 translate-y-[calc(-50%-2px)] rotate-45 rounded-[2px] bg-foreground fill-foreground data-[side=bottom]:top-1 data-[side=inline-end]:top-1/2! data-[side=inline-end]:-left-1 data-[side=inline-end]:-translate-y-1/2 data-[side=inline-start]:top-1/2! data-[side=inline-start]:-right-1 data-[side=inline-start]:-translate-y-1/2 data-[side=left]:top-1/2! data-[side=left]:-right-1 data-[side=left]:-translate-y-1/2 data-[side=right]:top-1/2! data-[side=right]:-left-1 data-[side=right]:-translate-y-1/2 data-[side=top]:-bottom-2.5"
