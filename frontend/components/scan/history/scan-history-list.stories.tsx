import { ScanHistoryList } from "./scan-history-list"

const meta = {
  title: "Scans/History List",
  component: ScanHistoryList,
  parameters: {
    layout: "fullscreen",
  },
  render: () => (
    <div className="min-h-screen bg-background px-4 py-8 lg:px-6">
      <ScanHistoryList pageSize={8} />
    </div>
  ),
}

export default meta

export const Happy = {}

export const Empty = {
  parameters: {
    mockScenario: "empty",
  },
}

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
