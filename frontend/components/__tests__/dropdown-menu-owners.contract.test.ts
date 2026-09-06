import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/shared/dropdown-menu-owners.tsx"), "utf8")

describe("dropdown-menu owners contract", () => {
  it("exports shared non-table dropdown owners", () => {
    expect(source).toContain("export function HeaderIconActionMenu")
    expect(source).toContain("export function SidebarUserMenu")
    expect(source).toContain("export function SidebarUserMenuSkeleton")
  })

  it("keeps compact icon-trigger menus on the shared 32px action geometry", () => {
    expect(source).toContain('buttonSize = "icon-sm"')
    expect(source).toContain('buttonVariant = "ghost"')
    expect(source).toContain('width="content-fit"')
  })

  it("keeps shell-bound menu placement in the shared owners", () => {
    expect(source).toContain('from "@/lib/ui/overlay-styles"')
    expect(source).toContain("sideOffset = shellOverlaySideOffsets.header")
    expect(source).toContain("const resolvedSideOffset")
    expect(source).toContain("shellOverlaySideOffsets.sidebarDesktop")
    expect(source).toContain('sideOffset={resolvedSideOffset}')
    expect(source).toContain("isMobile ? undefined")
  })

  it("owns the shared sidebar account trigger and label shell", () => {
    expect(source).toContain("SidebarMenuButton")
    expect(source).toContain('size="lg"')
    expect(source).toContain("DropdownMenuLabel")
    expect(source).toContain("data-slot=\"sidebar-user-name\"")
    expect(source).toContain("data-slot=\"sidebar-user-subline\"")
    expect(source).toContain("data-[popup-open]:bg-sidebar-accent")
    expect(source).toContain("group/sidebar-user")
    expect(source).toContain("text-sidebar-foreground")
    expect(source).toContain("text-sidebar-foreground/75")
    expect(source).toContain('"relative z-10 h-8 w-8 rounded-lg"')
    expect(source).toContain('className="relative z-10 grid flex-1 text-left leading-tight"')
    expect(source).toContain('className="relative z-10 ml-auto size-4')
    expect(source).toContain("group-hover/sidebar-user:text-sidebar-accent-foreground")
    expect(source).toContain("group-data-[popup-open]/sidebar-user:text-sidebar-accent-foreground")
    expect(source).toContain("data-[popup-open]:text-sidebar-accent-foreground")
    expect(source).not.toContain("group-hover/sidebar-user:text-foreground")
    expect(source).not.toContain("group-data-[popup-open]/sidebar-user:text-foreground")
    expect(source).not.toContain("data-[popup-open]:text-foreground")
    expect(source).not.toContain("data-[popup-open]:before:translate-x-0")
    expect(source).not.toContain("transition-colors group-hover/sidebar-user")
  })

  it("derives the sidebar account skeleton from the real account trigger owner", () => {
    expect(source).toContain("export function SidebarUserMenuSkeleton")
    expect(source).toContain('<SidebarMenuButton size="lg"')
    expect(source).toContain("<SidebarUserSummary userName={<span className=\"loading-skeleton")
    expect(source).toContain("userSubline={<span className=\"loading-skeleton")
    expect(source).toContain("avatarFallback={<span className=\"loading-skeleton")
    expect(source).toContain('<IconDotsVertical className="relative z-10 ml-auto size-4')
    expect(source).not.toContain('from "@/components/ui/skeleton"')
  })
})
