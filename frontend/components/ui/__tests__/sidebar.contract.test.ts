import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/sidebar.tsx"), "utf8")

describe("sidebar contract", () => {
  it("keeps sidebar polymorphic composition project-owned instead of using Radix Slot", () => {
    expect(source).toContain('from "@/components/ui/polymorphic"')
    expect(source).not.toContain("@radix-ui/react-slot")
  })

  it("keeps menu rows stable while allowing bounded content hover motion", () => {
    const start = source.indexOf("const sidebarMenuButtonVariants")
    const end = source.indexOf("function SidebarMenuButton")
    const menuVariantsSource = source.slice(start, end)

    expect(source).toContain("const sidebarMenuContentMotionClassName = cn(")
    expect(menuVariantsSource).toContain("sidebarMenuContentMotionClassName")
    expect(source).toContain("hover:[&>svg:not(.ml-auto)]:translate-x-0.5")
    expect(source).toContain("hover:[&>span:not([data-sidebar-navigation-pending])]:translate-x-0.5")
    expect(source).toContain("duration-[var(--motion-duration-fast)]")
    expect(source).toContain("ease-[var(--motion-ease-standard)]")
    expect(source).toContain("group-data-[collapsible=icon]:hover:[&>svg:not(.ml-auto)]:translate-x-0")
    expect(source).toContain("motion-reduce:hover:[&>svg:not(.ml-auto)]:translate-x-0")
    expect(source).toContain("motion-reduce:[&>svg:not(.ml-auto)]:transition-none")
    expect(menuVariantsSource).toContain("hover:bg-sidebar-accent")
    expect(menuVariantsSource).toContain("has-data-[sidebar-navigation-pending=true]:bg-sidebar-accent/70")
    expect(menuVariantsSource).toContain("data-[active=true]:bg-sidebar-accent")
    expect(menuVariantsSource).toContain("data-[active=true]:after:right-4")
    expect(menuVariantsSource).toContain("data-[active=true]:after:translate-x-px")
    expect(source).toContain("data-[active=true]:after:size-1.5")
    expect(source).toContain("data-[active=true]:after:rounded-full")
    expect(source).not.toContain("data-[active=true]:after:w-[3px]")
    expect(source).toContain("group-data-[collapsible=icon]:after:hidden")
    expect(source).toContain("group-data-[collapsible=icon]:data-[active=true]:after:hidden")
    expect(source).toContain("has-[>svg.ml-auto]:data-[active=true]:after:hidden")
    expect(menuVariantsSource).not.toContain("before:translate")
    expect(menuVariantsSource).not.toContain("transition-")
    expect(menuVariantsSource).not.toContain("hover:translate-x-")
    expect(menuVariantsSource).not.toContain("[&>span]:translate-x-")
    expect(menuVariantsSource).not.toContain("[&>svg]:translate-x-")
  })

  it("owns an immediate disclosure state without a menu motion variant", () => {
    const start = source.indexOf("const sidebarMenuButtonVariants")
    const end = source.indexOf("function SidebarMenuButton")
    const menuVariantsSource = source.slice(start, end)

    expect(menuVariantsSource).toContain("data-[panel-open]:[&>svg.ml-auto]:rotate-90")
    expect(source).not.toContain("motion: {")
    expect(source).not.toContain("motion = \"item\"")
    expect(menuVariantsSource).not.toContain("transition-transform")
    expect(source).not.toContain("group-data-[state=open]/collapsible:[&>svg.ml-auto]:rotate-90")
  })

  it("owns the primary sidebar navigation geometry, typography, and active marker", () => {
    expect(source).toContain("min-h-8")
    expect(source).toContain("px-2.5 py-1.5")
    expect(source).toContain("gap-3")
    expect(source).toContain("rounded-lg")
    expect(source).toContain("[&>svg]:size-4")
    expect(source).toContain("textRole.navLabel")
    expect(source).toContain("[&>span]:leading-5")
    expect(source).toContain("hover:bg-sidebar-accent")
    expect(source).toContain("data-[active=true]:bg-sidebar-accent")
    expect(source).toContain("text-sidebar-foreground/65")
    expect(source).toContain("[&>svg:not(.ml-auto)]:text-sidebar-foreground/55")
    expect(source).toContain("hover:text-sidebar-accent-foreground")
    expect(source).toContain("hover:[&>svg:not(.ml-auto)]:text-sidebar-accent-foreground")
    expect(source).toContain("data-[active=true]:text-sidebar-accent-foreground")
    expect(source).toContain("data-[active=true]:[&>svg:not(.ml-auto)]:text-sidebar-primary")
    expect(source).toContain("data-[active=true]:font-semibold")
    expect(source).toContain("data-[active=true]:after:bg-sidebar-primary")
    expect(source).toContain("group-data-[collapsible=icon]:size-8!")
    expect(source).toContain("group-data-[collapsible=icon]:data-[active=true]:after:hidden")
    expect(source).not.toContain("data-[active=true]:font-bold")
    expect(source).not.toContain("data-[active=true]:after:bg-[var(--highlight)]")
    expect(source).not.toContain("tracking-[0.2em] uppercase")
  })

  it("owns a filled active row for secondary sidebar items while preserving the group rail", () => {
    const start = source.indexOf("function SidebarMenuSubButton")
    const end = source.indexOf("export {")
    const subButtonSource = source.slice(start, end)

    expect(subButtonSource).toContain("min-h-8")
    expect(subButtonSource).toContain("py-1.5")
    expect(subButtonSource).toContain("pr-6 pl-3.5")
    expect(subButtonSource).not.toContain("px-6")
    expect(subButtonSource).not.toContain("pl-1.5")
    expect(subButtonSource).not.toContain("h-10")
    expect(subButtonSource).not.toContain("h-14")
    expect(subButtonSource).not.toContain("h-17")
    expect(subButtonSource).not.toContain("h-9")
    expect(subButtonSource).not.toContain("pl-5.5")
    expect(subButtonSource).toContain("rounded-lg")
    expect(subButtonSource).not.toContain("radius-control")
    expect(subButtonSource).toContain("hover:bg-sidebar-accent")
    expect(subButtonSource).toContain("data-[active=true]:bg-sidebar-accent")
    expect(subButtonSource).toContain("data-[active=true]:text-sidebar-accent-foreground")
    expect(subButtonSource).toContain("data-[active=true]:after:right-2")
    expect(subButtonSource).toContain("data-[active=true]:after:size-1.5")
    expect(subButtonSource).toContain("data-[active=true]:after:rounded-full")
    expect(subButtonSource).toContain("data-[active=true]:after:bg-sidebar-primary")
    expect(subButtonSource).toContain("sidebarMenuContentMotionClassName")
    expect(subButtonSource).not.toContain("overflow-visible")
    expect(subButtonSource).not.toContain("hover:translate-x-")
    expect(subButtonSource).not.toContain("data-[active=true]:[&>span:last-child]:translate-x-1")
  })

  it("owns shared sidebar section label geometry and collapsed behavior", () => {
    const start = source.indexOf("function SidebarGroupLabel")
    const end = source.indexOf("function SidebarGroupAction")
    const groupLabelSource = source.slice(start, end)

    expect(groupLabelSource).toContain('data-sidebar="group-label"')
    expect(groupLabelSource).toContain("pointer-events-none")
    expect(groupLabelSource).toContain("flex h-8 shrink-0 items-center overflow-hidden whitespace-nowrap px-3")
    expect(source).toContain("textRole.helperText")
    expect(groupLabelSource).toContain("transition-[opacity]")
    expect(groupLabelSource).toContain("duration-100")
    expect(groupLabelSource).toContain("motion-reduce:transition-none")
    expect(groupLabelSource).toContain("group-data-[collapsible=icon]:-mt-8")
    expect(groupLabelSource).toContain("group-data-[collapsible=icon]:opacity-0")
    expect(groupLabelSource).toContain("group-data-[collapsible=icon]:transition-none")
    expect(groupLabelSource).not.toContain("transition-opacity duration-200")
    expect(source).not.toContain("tracking-[0.2em] uppercase")
  })

  it("reuses the rounded menu button as the link-local pending highlight", () => {
    const pendingIndicatorSource = source.slice(
      source.indexOf("function SidebarNavigationPendingIndicator"),
      source.indexOf("const sidebarMenuButtonVariants")
    )

    expect(source).toContain("function SidebarNavigationPendingIndicator")
    expect(source).toContain("useLinkStatus")
    expect(pendingIndicatorSource).toContain("if (!pending) return null")
    expect(source).toContain('data-sidebar-navigation-pending="true"')
    expect(source).toContain('aria-hidden="true"')
    expect(source).toContain("pointer-events-none absolute! inset-0 z-0!")
    expect(source).toContain("[&>span]:truncate")
    expect(source).toContain("has-data-[sidebar-navigation-pending=true]:bg-sidebar-accent/70")
    expect(source).toContain("overflow-hidden rounded-lg")
    expect(pendingIndicatorSource).not.toContain("Spinner")
    expect(pendingIndicatorSource).not.toContain("animate-spin")
    expect(source).not.toContain('from "@/components/shared/loading/spinner"')
    expect(source).not.toContain("has-data-[sidebar-navigation-pending=true]:after:bg-sidebar-primary")
  })

  it("makes the pending destination the only visual selection without changing committed active state", () => {
    expect(source).toContain('const sidebarNavigationScopeClassName = "group/sidebar-navigation"')
    expect(source).toContain("const sidebarNavigationCommittedActiveSuppressionClassName = cn(")
    expect(source).toContain("group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:bg-transparent!")
    expect(source).toContain("group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:font-medium!")
    expect(source).toContain("group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:after:hidden!")
    expect(source).not.toContain("group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:font-normal!")
    expect(source).not.toContain("group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:before:")
    expect(source).not.toContain("group-has-data-[sidebar-navigation-pending=true]/sidebar-navigation:data-[active=true]:[&>span")
    expect(source.match(/sidebarNavigationCommittedActiveSuppressionClassName/g)).toHaveLength(4)
    expect(source).not.toContain("setOptimisticPathname")
    expect(source).not.toContain("setTimeout(() =>")
  })

  it("switches menu selection backgrounds without transitions", () => {
    const buttonStart = source.indexOf("const sidebarMenuButtonVariants")
    const buttonEnd = source.indexOf("function SidebarMenuButton")
    const subButtonStart = source.indexOf("function SidebarMenuSubButton")
    const subButtonEnd = source.indexOf("export {")

    expect(source.slice(buttonStart, buttonEnd)).not.toContain("transition-")
    expect(source.slice(subButtonStart, subButtonEnd)).not.toContain("transition-")
  })

  it("uses explicit sidebar-open and sidebar-close icons for the shell trigger", () => {
    const start = source.indexOf("function SidebarTrigger")
    const end = source.indexOf("function SidebarRail")
    const triggerSource = source.slice(start, end)

    expect(triggerSource).toContain("isSidebarOpen")
    expect(triggerSource).toContain("PanelLeftCloseIcon")
    expect(triggerSource).toContain("PanelLeftIcon")
    expect(triggerSource).not.toContain("<IconMenu")
  })

  it("keeps default sidebar framing in the component tree", () => {
    expect(source).toContain('const SIDEBAR_WIDTH = "16rem"')
    expect(source).toContain('const SIDEBAR_WIDTH_MOBILE = "16rem"')
    expect(source).toContain('const SIDEBAR_WIDTH_ICON = "3rem"')
    expect(source).toContain('"(min-width: 768px) and (max-width: 1279px)"')
    expect(source).toContain('"--sidebar-top": "var(--header-height, 0px)"')
    expect(source).toContain('"--sidebar-height": "calc(100svh - var(--sidebar-top))"')
    expect(source).toContain("top-(--sidebar-top)")
    expect(source).toContain("h-(--sidebar-height)")
    expect(source).toContain("border-b border-sidebar-border bg-sidebar")
    expect(source).toContain('from "@/lib/typography"')
    expect(source).toContain("textRole.helperText")
    expect(source).toContain("textRole.navLabel")
    expect(source).not.toContain("tracking-[0.2em] uppercase")
  })

  it("keeps the responsive band as the unpersisted desktop default", () => {
    expect(source).not.toContain("function readSidebarStateCookie()")
    expect(source).not.toContain("SIDEBAR_COOKIE_NAME")
    expect(source).not.toContain("responsiveDefaultRef")
    expect(source).toContain(
      "if (openProp !== undefined || defaultOpen !== undefined) return"
    )
    expect(source).toContain("mediaQuery.addEventListener(\"change\", applyResponsiveDefault)")
    expect(source).toContain("_setOpen(!mediaQuery.matches)")
  })

  it("uses the shared overlay scroll area without changing menu width", () => {
    const start = source.indexOf("function SidebarContent")
    const end = source.indexOf("function SidebarGroup", start)
    const contentSource = source.slice(start, end)

    expect(source).toContain('from "@/components/ui/scroll-area"')
    expect(contentSource).toContain("<ScrollArea")
    expect(contentSource).toContain('type="always"')
    expect(contentSource).toContain('contentClassName="flex min-h-full !min-w-0 w-full flex-col gap-0"')
    expect(contentSource).not.toContain("overflow-auto")
  })

  it("owns restrained shell collapse motion with reduced-motion support", () => {
    const start = source.indexOf("data-slot=\"sidebar-gap\"")
    const end = source.indexOf("function SidebarTrigger")
    const shellSource = source.slice(start, end)

    expect(shellSource).toContain("transition-[width]")
    expect(shellSource).toContain("transition-[width,transform]")
    expect(shellSource).toContain("duration-200")
    expect(shellSource).toContain("ease-[cubic-bezier(0.25,1,0.5,1)]")
    expect(shellSource).toContain("motion-reduce:transition-none")
    expect(shellSource).not.toContain("scale-")
    expect(shellSource).not.toContain("ease-bounce")
    expect(shellSource).not.toContain("ease-elastic")
    expect(source).not.toContain("transition-[width,height,padding,color]")
    expect(source).not.toContain("transition-[margin,opacity]")
    expect(source).not.toContain("transition-all")
  })

  it("keeps sidebar menu buttons without tooltips out of sidebar context updates", () => {
    const start = source.indexOf("function SidebarMenuButton")
    const end = source.indexOf("function SidebarMenuAction")
    const buttonSource = source.slice(start, end)
    const noTooltipBranchIndex = buttonSource.indexOf("if (!tooltip)")
    const contextReadIndex = buttonSource.indexOf("useSidebar()")

    expect(noTooltipBranchIndex).toBeGreaterThan(-1)
    expect(contextReadIndex).toBeGreaterThan(-1)
    expect(noTooltipBranchIndex).toBeLessThan(contextReadIndex)
  })

  it("uses sidebar theme tokens for active surfaces and markers", () => {
    expect(source).toContain("hover:bg-sidebar-accent")
    expect(source).toContain("data-[active=true]:bg-sidebar-accent")
    expect(source).toContain("data-[active=true]:after:bg-sidebar-primary")
    expect(source).not.toContain("data-[active=true]:after:bg-highlight")
    expect(source).not.toContain("data-[active=true]:after:bg-[var(--highlight)]")
  })

  it("keeps the dense menu padding as an explicit tokenized visual correction instead of an arbitrary literal", () => {
    expect(source).toContain("px-2.5 py-1.5")
    expect(source).not.toContain("p-[10px]")
  })

  it("clamps navigation label line-height inside sidebar menu primitives so the 32px shell density renders as specified", () => {
    expect(source).toContain("min-h-8")
    expect(source).toContain("h-8")
    expect(source).toContain("[&>span]:leading-5")
    expect(source).not.toContain("[&>span]:leading-normal")
  })
})
