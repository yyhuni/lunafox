import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ScheduledScanScheduleStep } from "@/components/scan/scheduled/scheduled-scan-dialog-sections"

vi.mock("@/components/scan/scheduled/scheduled-scan-time-zone-field", () => ({
  ScheduledScanTimeZoneField: ({ value }: { value: string }) => <span>{value}</span>,
}))

const t = (key: string, values?: Record<string, string | number | Date>) => {
  if (key === "form.scheduleRule") return `${values?.rule} in ${values?.timeZone}`
  return key
}

describe("ScheduledScanScheduleStep", () => {
  it("renders the create rule sentence with the selected IANA time zone", () => {
    render(
      <ScheduledScanScheduleStep
        t={t}
        timeZone="Asia/Shanghai"
        setTimeZone={vi.fn()}
        cronExpression="0 19 * * *"
        setCronExpression={vi.fn()}
        cronPresets={[]}
        getCronDescription={() => "At 7:00 PM"}
        getNextExecutions={() => ["1/1/2026, 7:00:00 PM"]}
      />
    )

    expect(screen.getByText("At 7:00 PM in Asia/Shanghai")).toBeVisible()
  })
})
