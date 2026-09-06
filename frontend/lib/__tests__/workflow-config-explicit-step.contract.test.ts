import { describe, expect, it } from "vitest"

import {
  adaptWorkflowProfile,
  parseWorkflowConfigurationDraftStrict,
  parseWorkflowConfigurationStrict,
  serializeCanonicalWorkflowConfiguration,
  WorkflowConfigurationDraftError,
} from "@/lib/workflow-config"
import type { ScanWorkflow } from "@/types/scan-workflow.types"
import type { ScanWorkflowWithEngines } from "@/types/engine-config.types"

const workflow: ScanWorkflow = {
  name: "scanWorkflows/default",
  displayName: "Default",
  description: "",
  stages: [{
    stageId: "stage",
    steps: [
      { stageId: "stage", stepId: "discover", engineId: "engine.discover", profileDefaultEnabled: true },
      { stageId: "stage", stepId: "ports", engineId: "engine.ports", profileDefaultEnabled: false },
    ],
  }],
  steps: [
    { stageId: "stage", stepId: "discover", engineId: "engine.discover", profileDefaultEnabled: true },
    { stageId: "stage", stepId: "ports", engineId: "engine.ports", profileDefaultEnabled: false },
  ],
  isBuiltin: true,
  isExecutable: true,
  etag: "etag",
  createTime: "",
  updateTime: "",
}

const workflowWithEngines: ScanWorkflowWithEngines = {
  scanWorkflowId: workflow.name,
  displayName: workflow.displayName,
  description: workflow.description,
  stages: [{
    stageId: "stage",
    steps: workflow.steps.map((step) => ({
      stepId: step.stepId,
      engineId: step.engineId,
      profileDefaultEnabled: step.profileDefaultEnabled,
      engine: {
        manifestVersion: "engine.v5" as const,
        engineId: step.engineId,
        publisher: "test",
        displayName: step.stepId,
        description: "",
        execution: {
          engineApiMajor: 2,
          supportedTargetTypes: ["domain"],
          configSections: [{
            id: "scan",
            defaultEnabled: true,
            name: "Scan",
            params: [{ key: "timeout", type: "integer" as const, default: 60 }],
          }],
        },
      },
    })),
  }],
}

describe("explicit Workflow Step configuration boundaries", () => {
  it("preserves explicit Server Profile Step enablement and complete engineConfig", () => {
    const draft = adaptWorkflowProfile({
      name: "scanWorkflows/default/profile",
      scanWorkflow: "scanWorkflows/default",
      configuration: {
        steps: {
          discover: { enabled: true, engineConfig: { scope: { enabled: true } } },
          ports: { enabled: false, engineConfig: { scan: { enabled: true } } },
        },
      },
    }, workflow)
    expect(draft.steps.discover.engineConfig).toEqual({ scope: { enabled: true } })
    expect(draft.steps.discover.enabled).toBe(true)
    expect(draft.steps.ports.enabled).toBe(false)
  })

  it("keeps the editable draft envelope strict without applying request enablement rules", () => {
    expect(parseWorkflowConfigurationDraftStrict({
      steps: {
        discover: { enabled: false, engineConfig: { scan: { enabled: true, timeout: 60 } } },
        ports: { enabled: false },
      },
    })).toEqual({
      steps: {
        discover: { enabled: false, engineConfig: { scan: { enabled: true, timeout: 60 } } },
        ports: { enabled: false },
      },
    })
    expect(() => parseWorkflowConfigurationDraftStrict({
      steps: { discover: { enabled: false, engineConfig: {} } },
      profileDefault: true,
    })).toThrow(WorkflowConfigurationDraftError)
  })

  it("emits disabled branches without retained engineConfig and rejects all-disabled requests", () => {
    expect(() => serializeCanonicalWorkflowConfiguration({
      steps: {
        discover: { enabled: false, engineConfig: { scope: {} } },
        ports: { enabled: true, engineConfig: { scan: { enabled: true } } },
      },
    }, workflow)).toThrow(WorkflowConfigurationDraftError)

    expect(serializeCanonicalWorkflowConfiguration({
      steps: {
        discover: { enabled: false },
        ports: { enabled: true, engineConfig: { scan: { enabled: true } } },
      },
    }, workflow)).toEqual({
      steps: {
        discover: { enabled: false },
        ports: { enabled: true, engineConfig: { scan: { enabled: true } } },
      },
    })
  })

  it("blocks missing, unknown, malformed, and parent-mismatched Profile data", () => {
    expect(() => adaptWorkflowProfile({
      name: "scanWorkflows/default/profile",
      scanWorkflow: "scanWorkflows/other",
      configuration: { steps: {} },
    }, workflow)).toThrow(WorkflowConfigurationDraftError)
    expect(() => adaptWorkflowProfile({
      name: "scanWorkflows/default/profile",
      scanWorkflow: "scanWorkflows/default",
      configuration: { steps: { discover: { enabled: false, engineConfig: {} } } },
    }, workflow)).toThrow(WorkflowConfigurationDraftError)
  })

  it("requires complete Profile Engine defaults when the current catalog schema is available", () => {
    expect(() => adaptWorkflowProfile({
      name: "scanWorkflows/default/profile",
      scanWorkflow: "scanWorkflows/default",
      configuration: {
        steps: {
          discover: { enabled: false, engineConfig: { scan: { enabled: true } } },
          ports: { enabled: false, engineConfig: { scan: { enabled: true, timeout: 60 } } },
        },
      },
    }, workflow, workflowWithEngines)).toThrow(WorkflowConfigurationDraftError)

    expect(adaptWorkflowProfile({
      name: "scanWorkflows/default/profile",
      scanWorkflow: "scanWorkflows/default",
      configuration: {
        steps: {
          discover: { enabled: false, engineConfig: { scan: { enabled: true, timeout: 60 } } },
          ports: { enabled: false, engineConfig: { scan: { enabled: true, timeout: 60 } } },
        },
      },
    }, workflow, workflowWithEngines).steps.discover.engineConfig).toEqual({
      scan: { enabled: true, timeout: 60 },
    })
  })

  it("rejects malformed canonical service payloads before the network", () => {
    expect(() => parseWorkflowConfigurationStrict({
      steps: { discover: { engineConfig: {} } },
    })).toThrow(WorkflowConfigurationDraftError)
    expect(() => parseWorkflowConfigurationStrict({
      steps: { discover: { enabled: false } },
    })).toThrow(WorkflowConfigurationDraftError)
    expect(() => parseWorkflowConfigurationStrict({
      steps: { discover: { enabled: true, engineConfig: {} } },
    })).toThrow(WorkflowConfigurationDraftError)
    expect(() => parseWorkflowConfigurationStrict("steps:\n  discover: [")).toThrow(WorkflowConfigurationDraftError)
  })

  it("serializes enabled Steps only when the enriched Engine schema is complete", () => {
    expect(() => serializeCanonicalWorkflowConfiguration({
      steps: {
        discover: { enabled: true, engineConfig: { scan: { enabled: true } } },
        ports: { enabled: false },
      },
    }, workflow, workflowWithEngines)).toThrow(WorkflowConfigurationDraftError)

    expect(serializeCanonicalWorkflowConfiguration({
      steps: {
        discover: { enabled: true, engineConfig: { scan: { enabled: true, timeout: 60 } } },
        ports: { enabled: false },
      },
    }, workflow, workflowWithEngines)).toEqual({
      steps: {
        discover: { enabled: true, engineConfig: { scan: { enabled: true, timeout: 60 } } },
        ports: { enabled: false },
      },
    })
  })

  it("rejects a disabled required Engine config section before request submission", () => {
    const workflowWithRequiredSection: ScanWorkflowWithEngines = {
      ...workflowWithEngines,
      stages: [{
        ...workflowWithEngines.stages[0],
        steps: workflowWithEngines.stages[0].steps.map((step) => ({
          ...step,
          engine: {
            ...step.engine,
            execution: {
              ...step.engine.execution,
              configSections: [{
                ...step.engine.execution.configSections[0],
                requiredEnabled: step.stepId === "discover",
              }],
            },
          },
        })),
      }],
    }

    expect(() => serializeCanonicalWorkflowConfiguration({
      steps: {
        discover: { enabled: true, engineConfig: { scan: { enabled: false } } },
        ports: { enabled: false },
      },
    }, workflow, workflowWithRequiredSection)).toThrow(WorkflowConfigurationDraftError)
  })

  it("enforces enum string-array cardinality before request submission", () => {
	const workflowWithScope = {
	  ...workflowWithEngines,
	  stages: workflowWithEngines.stages.map((stage) => ({
		...stage,
		steps: stage.steps.map((step) => ({
		  ...step,
		  engine: {
			...step.engine,
			execution: {
			  ...step.engine.execution,
			  configSections: [{
				...step.engine.execution.configSections[0],
				params: [{ key: "scan-targets", type: "stringArray" as const, default: ["website"], enum: ["website", "endpoint"], minItems: 1, maxItems: 2 }],
			  }],
			},
		  },
		})),
	  })),
	} satisfies ScanWorkflowWithEngines
	const config = (targets: string[]) => ({
	  steps: {
		discover: { enabled: true, engineConfig: { scan: { enabled: true, "scan-targets": targets } } },
		ports: { enabled: false },
	  },
	})
	for (const targets of [[], ["website", "website"], ["api"]]) {
	  expect(() => serializeCanonicalWorkflowConfiguration(config(targets), workflow, workflowWithScope)).toThrow(WorkflowConfigurationDraftError)
	}
	expect(serializeCanonicalWorkflowConfiguration(config(["website", "endpoint"]), workflow, workflowWithScope)).toEqual(config(["website", "endpoint"]))
  })
})
