import { OverviewStatCards } from "./overview-stat-cards"

const meta = {
  title: "Overview/Stat Cards",
  component: OverviewStatCards,
  parameters: {
    layout: "padded",
  },
  render: () => (
    <div className="min-h-screen bg-background py-8">
      <OverviewStatCards />
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

export const Empty = {
  parameters: {
    mockScenario: "empty",
  },
}

export const Edge = {
  parameters: {
    mockScenario: "edge",
  },
}
