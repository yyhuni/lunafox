import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/nav-user.tsx"), "utf8")

describe("nav-user contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function NavUser")
    expect(source).toContain("export function NavUserSkeleton")
    expect(source).toContain("from \"react\"")
    expect(source).toContain("ChangePasswordDialog")
  })

  it("uses the shared sidebar user dropdown owner instead of page-local menu markup", () => {
    expect(source).toContain('from \'@/components/shared/dropdown-menu-owners\'')
    expect(source).toContain('from "@/components/brand/lunafox-mark"')
    expect(source).toContain("SidebarUserMenu")
    expect(source).toContain("SidebarUserMenuSkeleton")
    expect(source).toContain("avatarFallback={<LunaFoxMark")
    expect(source).toContain("auth?.user?.email || user.email")
    expect(source).toContain("displaySubline")
    expect(source).not.toContain("DropdownMenuTrigger")
    expect(source).not.toContain("DropdownMenuContent")
  })

  it("keeps the warmup footer skeleton on the shared sidebar user menu owner", () => {
    expect(source).toContain("<SidebarUserMenuSkeleton />")
    expect(source).toContain("<SidebarMenu>")
    expect(source).toContain("<SidebarMenuItem>")
  })
})
