import type { ScanWorkflow, ScanWorkflowProfile } from '@/types/scan-workflow.types'
import { getMockEngineCatalogDetail } from '@/mock/data/engine-catalog'

const defaultScanWorkflowStages: ScanWorkflow["stages"] = [
    {
      stageId: 'discovery',
      steps: [
        {
          stageId: 'discovery',
          stepId: 'subdomain_discovery',
          engineId: 'engine.lunafox.subdomain_discovery',
          profileDefaultEnabled: true,
        },
      ],
    },
    {
      stageId: 'ports',
      steps: [
        {
          stageId: 'ports',
          stepId: 'port_scan',
          engineId: 'engine.lunafox.port_scan',
          profileDefaultEnabled: true,
        },
      ],
    },
    {
      stageId: 'websites',
      steps: [
        {
          stageId: 'websites',
          stepId: 'website_discovery',
          engineId: 'engine.lunafox.website_discovery',
          profileDefaultEnabled: true,
        },
      ],
    },
    {
      stageId: 'url_collection',
      steps: [
        {
          stageId: 'url_collection',
          stepId: 'url_collection',
          engineId: 'engine.lunafox.url_collection',
          profileDefaultEnabled: true,
        },
      ],
    },
    {
      stageId: 'screenshot',
      steps: [
        {
          stageId: 'screenshot',
          stepId: 'screenshot',
          engineId: 'engine.lunafox.screenshot',
          profileDefaultEnabled: true,
        },
      ],
    },
    {
      stageId: 'directory_scan',
      steps: [
        {
          stageId: 'directory_scan',
          stepId: 'directory_scan',
          engineId: 'engine.lunafox.directory_scan',
          profileDefaultEnabled: true,
        },
      ],
    },
    {
      stageId: 'nuclei_vulnerability',
      steps: [
        {
          stageId: 'nuclei_vulnerability',
          stepId: 'nuclei_vulnerability',
          engineId: 'engine.lunafox.nuclei_vulnerability',
          profileDefaultEnabled: false,
        },
      ],
    },
]

export const mockScanWorkflows: ScanWorkflow[] = [
  {
    name: 'scanWorkflows/default',
    displayName: 'Default Scan',
    description: 'Run the default built-in scan workflow: discover subdomains, scan target-scoped hosts for open ports, probe websites, collect URLs, capture screenshots, then scan directories.',
    stages: defaultScanWorkflowStages,
    steps: defaultScanWorkflowStages.flatMap((stage) => stage.steps),
    isBuiltin: true,
    isExecutable: true,
    etag: 'mock-default-v1',
    createTime: '2026-01-01T00:00:00Z',
    updateTime: '2026-01-01T00:00:00Z',
  },
]

export function getMockScanWorkflows(): ScanWorkflow[] {
  return mockScanWorkflows.map((workflow) => ({ ...workflow }))
}

export function getMockScanWorkflowByName(name: string): ScanWorkflow | undefined {
  return mockScanWorkflows.find((workflow) => workflow.name === name)
}

/** Returns a request-ready configuration with every current Workflow Step explicit. */
export function getMockCanonicalWorkflowConfiguration() {
  const workflow = mockScanWorkflows[0]
  const profile = getMockScanWorkflowProfile(workflow.name)
  if (!profile) throw new Error(`Mock Profile is unavailable for ${workflow.name}`)
  const profileSteps = profile.configuration.steps
  if (typeof profileSteps !== "object" || profileSteps === null || Array.isArray(profileSteps)) {
    throw new Error(`Mock Profile steps are unavailable for ${workflow.name}`)
  }

  return {
    steps: Object.fromEntries(workflow.steps.map((step) => {
      const profileStep = (profileSteps as Record<string, unknown>)[step.stepId]
      if (!profileStep || typeof profileStep !== "object" || Array.isArray(profileStep)) {
        throw new Error(`Mock Profile Step is unavailable for ${step.stepId}`)
      }
      const profileStepRecord = profileStep as Record<string, unknown>
      return [step.stepId, { enabled: true, engineConfig: profileStepRecord.engineConfig }]
    })),
  }
}

export function getMockScanWorkflowProfile(scanWorkflow: string): ScanWorkflowProfile | undefined {
  const parentName = scanWorkflow.startsWith("scanWorkflows/")
    ? scanWorkflow
    : `scanWorkflows/${scanWorkflow}`
  const workflow = getMockScanWorkflowByName(parentName)
  if (!workflow) return undefined
  return {
    name: `${workflow.name}/profile`,
    scanWorkflow: workflow.name,
    configuration: {
      steps: Object.fromEntries(workflow.steps.map((step) => [
        step.stepId,
        { enabled: step.profileDefaultEnabled, engineConfig: materializeMockEngineConfig(step.engineId) },
      ])),
    },
  }
}

function materializeMockEngineConfig(engineId: string): Record<string, unknown> {
  const engine = getMockEngineCatalogDetail(engineId)
  if (!engine) return {}

  const config: Record<string, unknown> = {}
  for (const section of engine.execution.configSections) {
    // Engine manifests treat an omitted defaultEnabled declaration as false.
    const enabled = section.defaultEnabled ?? false
    const sectionConfig: Record<string, unknown> = { enabled }
    if (enabled) {
      for (const param of section.params) {
        if (param.default !== undefined) sectionConfig[param.key] = param.default
      }
    }
    config[section.id] = sectionConfig
  }
  return config
}
