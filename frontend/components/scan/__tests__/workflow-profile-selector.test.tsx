import { fireEvent, render, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { WorkflowProfileSelector } from "@/components/scan/workflow-profile-selector"
import type { ScanWorkflow } from "@/types/scan-workflow.types"

vi.mock("next-intl", () => ({ useTranslations: () => (key: string) => key }))

describe("WorkflowProfileSelector", () => {
  it("只加载选中工作流的单例 Profile", async () => {
    const onWorkflowNamesChange = vi.fn()
    const onConfigurationChange = vi.fn()
    const loadWorkflowProfile = vi.fn().mockResolvedValue({
      name: "scanWorkflows/full/profile",
      scanWorkflow: "scanWorkflows/full",
      configuration: { steps: { step: { enabled: false, engineConfig: { recon: { enabled: true, timeout: 60 } } } } },
    })
    render(<WorkflowProfileSelector workflows={[createWorkflow("full")]} selectedWorkflowNames={[]} onWorkflowNamesChange={onWorkflowNamesChange} onConfigurationChange={onConfigurationChange} loadWorkflowProfile={loadWorkflowProfile} />)

    fireEvent.click(document.getElementById("scheduled-workflow-full")!)
    await waitFor(() => expect(onWorkflowNamesChange).toHaveBeenCalledWith(["full"]))
    expect(loadWorkflowProfile).toHaveBeenCalledWith("full")
    expect(onConfigurationChange).toHaveBeenCalledWith(expect.stringContaining("step"))
  })
})

function createWorkflow(name: string): ScanWorkflow {
  const step = { stageId: "stage", stepId: "step", engineId: "engine.lunafox.subdomain_discovery" }
  const workflowStep = { ...step, profileDefaultEnabled: false }
  return { name, displayName: name, description: "", stages: [{ stageId: "stage", steps: [workflowStep] }], steps: [workflowStep], isBuiltin: false, isExecutable: true, etag: "etag", createTime: "", updateTime: "" }
}
