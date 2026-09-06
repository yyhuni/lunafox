import { describe, expect, it } from "vitest"
import { readFileSync } from "node:fs"
import path from "node:path"

const source = readFileSync(path.resolve(process.cwd(), "components/notifications/notification-drawer-sections.tsx"), "utf8")

describe("notification-drawer-sections contract", () => {
  it("preserves current source markers", () => {
    expect(source).toContain("export function NotificationDrawerLayout")
    expect(source).toContain("className")
    expect(source).toContain("from \"react\"")
  })

  it("uses the shared compact icon action size for the header trigger", () => {
    expect(source).toContain('size="icon-sm"')
    expect(source).toContain('<semanticIcons.concept.notification className="h-4 w-4"')
    expect(source).not.toContain('<Bell className="h-4 w-4"')
    expect(source).not.toContain('<Bell className="h-5 w-5"')
  })

  it("uses the shared compact feedback drawer layout without overriding edge-panel motion", () => {
    expect(source).toContain("compactFeedbackDrawerContentClassName")
    expect(source).not.toContain("sideMotion")
    expect(source).not.toContain('className="flex flex-col gap-0 p-0 sm:max-w-[440px] w-full"')
  })

  it("uses the reviewed close-control opt-out for the notification drawer", () => {
    expect(source).toContain("<SheetContent showCloseButton={false}")
    expect(source).not.toContain("SheetClose")
    expect(source).toContain("EdgePanelHeader")
    expect(source).toContain('variant="compact"')
    expect(source).not.toContain("leading={<semanticIcons.concept.notification")
    expect(source).toContain('className={cn("flex min-w-0 items-center gap-2", textRole.sectionTitle)}')
    expect(source).toContain('<semanticIcons.concept.notification aria-hidden="true" className="size-4 shrink-0" />')
  })

  it("routes the closed notification priority through shared status tones", () => {
    expect(source).toContain("function notificationPriorityTone")
    expect(source).toContain("getStatusToneBadgeClass")
    expect(source).not.toContain("getStatusToneTextClass")
    expect(source).toContain('case "normal"')
    expect(source).toContain('case "high"')
    expect(source).toContain('case "critical"')
    expect(source).not.toContain('case "low"')
    expect(source).not.toContain("text-gray-500")
    expect(source).not.toContain("text-[var(--error)]")
    expect(source).not.toContain("text-[var(--warning)]")
  })

  it("keeps notification metadata compact without a redundant category icon", () => {
    expect(source).not.toContain("NotificationCategoryIcon")
    expect(source).toContain('"h-4 shrink-0 px-1.5 py-0", getStatusToneBadgeClass(priorityTone)')
  })

  it("reserves the filter border in every state so selecting a filter does not move neighboring controls", () => {
    expect(source).toContain('"radius-pill w-full justify-center border border-transparent whitespace-nowrap sm:w-auto sm:shrink-0"')
  })

  it("keeps acknowledgement bulk-only and aligns the unread marker with metadata", () => {
    expect(source).toContain("getStatusToneBgClass")
    expect(source).toContain('"absolute inset-y-0 left-0 w-1", getStatusToneBgClass(priorityTone)')
    expect(source).toContain('className={cn("radius-round size-1.5", getStatusToneBgClass(priorityTone))}')
    expect(source).toContain("state.handleMarkAll")
    expect(source).not.toContain("onMarkRead")
    expect(source).not.toContain("isMarkingRead")
    expect(source).not.toContain("state.handleMarkRead")
    expect(source).not.toContain("state.isMarkingRead")
  })

  it("keeps cards presentation-only without task navigation or payload parsing", () => {
    const cardSource = source.slice(source.indexOf("function NotificationCard"), source.indexOf("export function NotificationDrawerLayout"))
    expect(cardSource).toContain("<article")
    expect(cardSource).toContain("notification.title")
    expect(cardSource).toContain("notification.message")
    expect(cardSource).not.toContain("onClick")
    expect(cardSource).not.toContain("router.push")
    expect(cardSource).not.toContain("nucleiPocSyncTasks")
    expect(cardSource).not.toContain("payload")
  })

  it("derives notification loading rows from the real card owner", () => {
    expect(source).toContain("function NotificationCard")
    expect(source).toContain("loading = false")
    expect(source).toContain("<NotificationCard key={index} loading />")
    expect(source).toContain("NotificationCard requires durable notification display data unless loading is true.")
    expect(source).not.toContain("function NotificationSkeleton")
    expect(source).not.toContain("Notification skeleton screen")
  })

  it("keeps sticky group headings on a solid drawer surface", () => {
    expect(source).toContain('"sticky top-0 z-10 mb-2 bg-card px-1 py-1 text-muted-foreground", textRole.compactSectionTitle')
    expect(source).not.toContain("backdrop-blur bg-background/90")
  })

  it("only shows time group headings when multiple groups are visible", () => {
    expect(source).toContain("const notificationGroups =")
    expect(source).toContain("const showGroupHeadings = notificationGroups.length > 1")
    expect(source).toContain("showGroupHeadings ? (")
    expect(source).toContain("<h3 className={cn(")
  })
})
