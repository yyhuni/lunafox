import type { GetToolsResponse, Tool } from "@/types/tool.types"
import { mockTools } from "../data/tools"
import { getMockScenario } from "../scenarios"

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function paginate(items: Tool[], page = 1, pageSize = 10): GetToolsResponse {
  const total = items.length
  const totalPages = total === 0 ? 0 : Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize

  return {
    tools: items.slice(start, start + pageSize),
    total,
    page,
    pageSize,
    totalPages,
  }
}

function buildStressTools(): Tool[] {
  return [
    {
      id: 9901,
      name: "enterprise-shared-security-platform-orchestrator",
      type: "opensource",
      repoUrl: "https://github.com/example/enterprise-shared-security-platform-orchestrator",
      version: "v2026.05.27-enterprise",
      description: "Long tool name and description used to validate card truncation, wrapping, and stable button layout in dense content.",
      categoryNames: ["subdomain", "recon", "http", "crawler", "vulnerability"],
      directory: "/opt/tools/enterprise-shared-security-platform-orchestrator/bin/current",
      installCommand: "curl -fsSL https://example.com/install-enterprise-shared-security-platform-orchestrator.sh | bash",
      updateCommand: "enterprise-shared-security-platform-orchestrator self-update --channel stable",
      versionCommand: "enterprise-shared-security-platform-orchestrator version --long",
      createdAt: "2025-01-05T09:00:00Z",
      updatedAt: "2026-05-27T08:00:00Z",
    },
    {
      id: 9902,
      name: "customer-facing-shared-gateway-postscan-hooks",
      type: "custom",
      repoUrl: "",
      version: "2026.05-hotfix.4",
      description: "Used to validate layout stability for custom tool cards with long paths, long descriptions, and multiple tags.",
      categoryNames: ["recon", "http", "other", "network"],
      directory: "/srv/lunafox/custom-tools/customer-facing-shared-gateway-postscan-hooks/releases/current",
      installCommand: "",
      updateCommand: "",
      versionCommand: "",
      createdAt: "2025-02-10T11:30:00Z",
      updatedAt: "2026-05-27T09:15:00Z",
    },
  ]
}

function buildEdgeTools(): Tool[] {
  return [
    {
      id: 9903,
      name: "Edge Tool",
      type: "custom",
      repoUrl: "",
      version: "",
      description: "",
      categoryNames: [],
      directory: "/srv/tools/edge-tool",
      installCommand: "",
      updateCommand: "",
      versionCommand: "",
      createdAt: "2025-03-03T03:03:03Z",
      updatedAt: "2026-05-27T10:00:00Z",
    },
  ]
}

export function buildTools(params?: {
  page?: number
  pageSize?: number
}): GetToolsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const scenario = getMockScenario()

  if (scenario === "empty") {
    return paginate([], page, pageSize)
  }

  const base = clone(mockTools)

  if (scenario === "stress") {
    return paginate([...buildStressTools(), ...base], page, pageSize)
  }

  if (scenario === "edge") {
    return paginate([...buildEdgeTools(), ...base], page, pageSize)
  }

  return paginate(base, page, pageSize)
}
