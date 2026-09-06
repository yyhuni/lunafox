import { VulnerabilitiesVerticalView } from "./vulnerabilities-vertical-view"

const meta = {
  title: "Vulnerabilities/Workbench",
  component: VulnerabilitiesVerticalView,
  parameters: {
    layout: "fullscreen",
  },
  render: () => (
    <div className="min-h-screen bg-background px-4 py-8 lg:px-6">
      <VulnerabilitiesVerticalView />
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
