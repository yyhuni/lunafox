import { CustomToolsList } from "./custom-tools-list"
import { OpensourceToolsList } from "./opensource-tools-list"

function ToolsConfigListsStory() {
  return (
    <div className="min-h-screen bg-background">
      <div className="mx-auto flex max-w-7xl flex-col gap-12 px-4 py-8 lg:px-6">
        <section className="space-y-6">
          <div className="space-y-1">
            <h2 className="text-xl font-semibold tracking-tight">开源工具</h2>
          </div>
          <OpensourceToolsList />
        </section>

        <section className="space-y-6">
          <div className="space-y-1">
            <h2 className="text-xl font-semibold tracking-tight">自定义工具</h2>
          </div>
          <CustomToolsList />
        </section>
      </div>
    </div>
  )
}

const meta = {
  title: "Tools/Config Lists",
  parameters: {
    layout: "fullscreen",
  },
  render: () => <ToolsConfigListsStory />,
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
