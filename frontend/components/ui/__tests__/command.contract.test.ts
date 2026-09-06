import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/ui/command.tsx"), "utf8")

describe("command contract", () => {
  it("uses a project-owned command implementation instead of cmdk", () => {
    expect(source).not.toContain('from "cmdk"')
    expect(source).toContain("CommandContext")
    expect(source).toContain("data-command-item")
  })

  it("keeps the command dialog shell on the shared Dialog wrapper", () => {
    expect(source).not.toContain("@radix-ui/react-dialog")
    expect(source).toContain("React.ComponentProps<typeof Dialog>")
  })

  it("uses shared control radius for selectable item backgrounds", () => {
    expect(source).toContain("radius-control")
    expect(source).not.toContain("items-center rounded-sm")
  })

  it("keeps popover command search visually quiet like the shadcn tasks filter", () => {
    expect(source).toContain('className="flex h-10 items-center gap-2 border-b px-3"')
    expect(source).toContain('className="flex size-4 shrink-0 items-center justify-center"')
    expect(source).toContain('<IconSearch className="size-4 opacity-50" />')
    expect(source).toContain('"flex h-10 w-full min-w-0 bg-transparent outline-none placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"')
    expect(source).not.toContain("rounded-md bg-transparent py-3 outline-none placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring")
    expect(source).not.toContain("focus-visible:ring-2 focus-visible:ring-ring/50 data-[active=true]")
  })
})
