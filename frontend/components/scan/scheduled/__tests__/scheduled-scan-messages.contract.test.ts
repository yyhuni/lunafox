import { describe, expect, it } from "vitest"

import enMessages from "@/messages/en.json"
import zhMessages from "@/messages/zh.json"

describe("scheduled scan execution messages", () => {
  it("describes handoff aggregates without claiming final Scan status", () => {
    expect(zhMessages.columns.scheduledScan.handoffResults).toBe("调度结果")
    expect(zhMessages.columns.scheduledScan.trigger).toBe("触发")
    expect(zhMessages.columns.scheduledScan.success).toBe("成功")
    expect(zhMessages.columns.scheduledScan.failure).toBe("失败")
    expect(zhMessages.columns.scheduledScan.lastRun).toBe("上次触发时间")
    expect(enMessages.columns.scheduledScan.handoffResults).toBe("Handoff Results")
    expect(enMessages.columns.scheduledScan.trigger).toBe("Trigger")
    expect(enMessages.columns.scheduledScan.success).toBe("Success")
    expect(enMessages.columns.scheduledScan.failure).toBe("Failed")
    expect(enMessages.columns.scheduledScan.lastRun).toBe("Last Trigger")
  })

  it("does not expose a user-configurable time-zone field", () => {
    for (const messages of [zhMessages, enMessages]) {
      expect(messages.scan.scheduled.form).not.toHaveProperty("timeZone")
    }
  })

	it("uses the next-24-hour task count as compact timeline context and supplies failure copy", () => {
		expect(zhMessages.scan.scheduled.workbench.timeline.next24HoursCount).toContain("未来 24 小时")
		expect(zhMessages.scan.scheduled.workbench.timeline.next24HoursCount).toContain("{count}")
		expect(enMessages.scan.scheduled.workbench.timeline.next24HoursCount).toContain("Next 24 hours")
		expect(enMessages.scan.scheduled.workbench.timeline.next24HoursCount).toContain("{count}")
		for (const messages of [zhMessages, enMessages]) {
			expect(messages.scan.scheduled.workbench.overview.loadFailed).toBeTruthy()
			expect(messages.scan.scheduled.workbench.overview.loadFailedDescription).toBeTruthy()
			expect(messages.scan.scheduled.workbench.overview.retry).toBeTruthy()
			expect(messages.scan.scheduled.workbench.overview.stale.description).toBeTruthy()
			expect(messages.scan.scheduled.workbench.overview.stale.retry).toBeTruthy()
		}
	})
})
