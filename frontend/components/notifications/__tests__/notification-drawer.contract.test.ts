import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/notifications/notification-drawer.tsx"), "utf8")

describe("notification-drawer contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function NotificationDrawer")
    expect(source).toContain("from \"./notification-drawer-sections\"")
  })
})
