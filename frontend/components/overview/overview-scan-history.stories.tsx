import { OverviewSingleSection } from "./overview-section-layouts"
import { OverviewScanHistory } from "./overview-scan-history"

const meta = {
  title: "Overview/Scan History",
  component: OverviewScanHistory,
  parameters: {
    layout: "fullscreen",
  },
  render: () => (
    <div className="min-h-screen bg-background py-8">
      <OverviewSingleSection>
        <OverviewScanHistory />
      </OverviewSingleSection>
    </div>
  ),
}

export default meta

export const Happy = {}

export const Stress = {
  parameters: {
    mockScenario: "stress",
  },
}

export const Edge = {
  parameters: {
    mockScenario: "edge",
  },
}
