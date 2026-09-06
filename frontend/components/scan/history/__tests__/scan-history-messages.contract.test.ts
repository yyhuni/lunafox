import { describe, expect, it } from "vitest"
import zhMessages from "@/messages/zh.json"
import enMessages from "@/messages/en.json"

describe("scan history messages contract", () => {
  it("keeps the full-page row action label concise", () => {
    expect(zhMessages.scan.history.actions.scanDetail).toBe("详情")
    expect(enMessages.scan.history.actions.scanDetail).toBe("Details")
  })

  it("labels the scan history execution column as engines, not workflow", () => {
    expect(zhMessages.columns.scanHistory.executedEngines).toBe("执行引擎")
    expect(enMessages.columns.scanHistory.executedEngines).toBe("Executed Engines")
    expect(zhMessages.columns.scanHistory).not.toHaveProperty("workflowName")
    expect(enMessages.columns.scanHistory).not.toHaveProperty("workflowName")
  })

  it("localizes the fixed trigger provenance vocabulary", () => {
    expect(zhMessages.columns.scanHistory.triggerType).toBe("来源")
    expect(enMessages.columns.scanHistory.triggerType).toBe("Trigger")
    expect(zhMessages.scan.history.triggerType).toEqual({ manual: "手动触发", scheduled: "定时触发", ai: "AI 触发" })
    expect(enMessages.scan.history.triggerType).toEqual({ manual: "Manual trigger", scheduled: "Scheduled trigger", ai: "AI trigger" })
  })

  it("discloses the fixed scan-history retention policy without a setting", () => {
    expect(zhMessages.scan.history.retention.summary).toContain("{duration}")
    expect(enMessages.scan.history.retention.summary).toContain("{duration}")
    expect(zhMessages.scan.history.retention.automaticCleanupDescription).toContain("扫描结束后")
    expect(zhMessages.scan.history.retention.automaticCleanupDescription).toContain("条件满足时自动清理")
    expect(enMessages.scan.history.retention.automaticCleanupDescription).toContain("after a scan ends")
    expect(enMessages.scan.history.retention.inactiveCleanupDescription).toContain("inactive")
  })

  it("localizes every runtime task status delegated to common status", () => {
    expect(zhMessages.common.status.blocked).toBe("已阻塞")
    expect(enMessages.common.status.blocked).toBe("Blocked")
  })
})
