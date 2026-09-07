import type { Tool, GetToolsResponse, CreateToolRequest, UpdateToolRequest } from '@/types/tool.types'

export const mockTools: Tool[] = [
  {
    id: 1,
    name: 'subfinder',
    type: 'opensource',
    repoUrl: 'https://github.com/projectdiscovery/subfinder',
    version: 'v2.6.3',
    description: 'Fast passive subdomain enumeration tool.',
    categoryNames: ['subdomain', 'recon'],
    directory: '/opt/tools/subfinder',
    installCommand: 'go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@v2.12.0',
    updateCommand: 'go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@v2.12.0',
    versionCommand: 'subfinder -version',
    createdAt: '2024-12-20T10:00:00Z',
    updatedAt: '2024-12-28T10:00:00Z',
  },
  {
    id: 2,
    name: 'httpx',
    type: 'opensource',
    repoUrl: 'https://github.com/projectdiscovery/httpx',
    version: 'v1.6.0',
    description: 'Fast and multi-purpose HTTP toolkit.',
    categoryNames: ['http', 'recon'],
    directory: '/opt/tools/httpx',
    installCommand: 'go install -v github.com/projectdiscovery/httpx/cmd/httpx@v1.8.1',
    updateCommand: 'go install -v github.com/projectdiscovery/httpx/cmd/httpx@v1.8.1',
    versionCommand: 'httpx -version',
    createdAt: '2024-12-20T10:01:00Z',
    updatedAt: '2024-12-28T10:01:00Z',
  },
  {
    id: 3,
    name: 'nuclei',
    type: 'opensource',
    repoUrl: 'https://github.com/projectdiscovery/nuclei',
    version: 'v3.1.0',
    description: 'Fast and customizable vulnerability scanner.',
    categoryNames: ['vulnerability'],
    directory: '/opt/tools/nuclei',
    installCommand: 'go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@v3.7.0',
    updateCommand: 'go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@v3.7.0',
    versionCommand: 'nuclei -version',
    createdAt: '2024-12-20T10:02:00Z',
    updatedAt: '2024-12-28T10:02:00Z',
  },
  {
    id: 4,
    name: 'naabu',
    type: 'opensource',
    repoUrl: 'https://github.com/projectdiscovery/naabu',
    version: 'v2.2.1',
    description: 'Fast port scanner written in go.',
    categoryNames: ['port', 'network'],
    directory: '/opt/tools/naabu',
    installCommand: 'go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@v2.4.0',
    updateCommand: 'go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@v2.4.0',
    versionCommand: 'naabu -version',
    createdAt: '2024-12-20T10:03:00Z',
    updatedAt: '2024-12-28T10:03:00Z',
  },
  {
    id: 5,
    name: 'katana',
    type: 'opensource',
    repoUrl: 'https://github.com/projectdiscovery/katana',
    version: 'v1.0.4',
    description: 'Next-generation crawling and spidering framework.',
    categoryNames: ['crawler', 'recon'],
    directory: '/opt/tools/katana',
    installCommand: 'go install github.com/projectdiscovery/katana/cmd/katana@v1.4.0',
    updateCommand: 'go install github.com/projectdiscovery/katana/cmd/katana@v1.4.0',
    versionCommand: 'katana -version',
    createdAt: '2024-12-20T10:04:00Z',
    updatedAt: '2024-12-28T10:04:00Z',
  },
  {
    id: 6,
    name: 'ffuf',
    type: 'opensource',
    repoUrl: 'https://github.com/ffuf/ffuf',
    version: 'v2.1.0',
    description: 'Fast web fuzzer written in Go.',
    categoryNames: ['directory', 'fuzzer'],
    directory: '/opt/tools/ffuf',
    installCommand: 'go install github.com/ffuf/ffuf/v2@v2.1.0',
    updateCommand: 'go install github.com/ffuf/ffuf/v2@v2.1.0',
    versionCommand: 'ffuf -V',
    createdAt: '2024-12-20T10:05:00Z',
    updatedAt: '2024-12-28T10:05:00Z',
  },
  {
    id: 7,
    name: 'amass',
    type: 'opensource',
    repoUrl: 'https://github.com/owasp-amass/amass',
    version: 'v4.2.0',
    description: 'In-depth attack surface mapping and asset discovery.',
    categoryNames: ['subdomain', 'recon'],
    directory: '/opt/tools/amass',
    installCommand: 'go install -v github.com/owasp-amass/amass/v4/...@v4.2.0',
    updateCommand: 'go install -v github.com/owasp-amass/amass/v4/...@v4.2.0',
    versionCommand: 'amass -version',
    createdAt: '2024-12-20T10:06:00Z',
    updatedAt: '2024-12-28T10:06:00Z',
  },
  {
    id: 8,
    name: 'xingfinger',
    type: 'custom',
    repoUrl: '',
    version: '1.0.0',
    description: 'Custom fingerprint detection tool',
    categoryNames: ['recon'],
    directory: '/opt/tools/xingfinger',
    installCommand: '',
    updateCommand: '',
    versionCommand: '',
    createdAt: '2024-12-20T10:07:00Z',
    updatedAt: '2024-12-28T10:07:00Z',
  },
]

let nextMockToolId = Math.max(...mockTools.map((tool) => tool.id)) + 1

function cloneTool(tool: Tool): Tool {
  return {
    ...tool,
    categoryNames: [...tool.categoryNames],
  }
}

export function getMockTools(params?: {
  page?: number
  pageSize?: number
}): GetToolsResponse {
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10

  const total = mockTools.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const tools = mockTools.slice(start, start + pageSize).map(cloneTool)

  return {
    tools,
    total,
    page,
    pageSize,
    totalPages,
  }
}

export function getMockToolById(id: number): Tool | undefined {
  const tool = mockTools.find(t => t.id === id)
  return tool ? cloneTool(tool) : undefined
}

export function createMockTool(data: CreateToolRequest): Tool {
  const timestamp = new Date().toISOString()
  const tool: Tool = {
    id: nextMockToolId++,
    name: data.name,
    type: data.type,
    repoUrl: data.repoUrl ?? '',
    version: data.version ?? '',
    description: data.description ?? '',
    categoryNames: data.categoryNames ?? [],
    directory: data.directory ?? '',
    installCommand: data.installCommand ?? '',
    updateCommand: data.updateCommand ?? '',
    versionCommand: data.versionCommand ?? '',
    createdAt: timestamp,
    updatedAt: timestamp,
  }

  mockTools.unshift(tool)
  return cloneTool(tool)
}

export function updateMockTool(id: number, data: UpdateToolRequest): Tool {
  const index = mockTools.findIndex((tool) => tool.id === id)
  if (index === -1) {
    throw new Error(`Mock tool not found: ${id}`)
  }

  mockTools[index] = {
    ...mockTools[index],
    ...data,
    categoryNames: data.categoryNames ?? mockTools[index].categoryNames,
    updatedAt: new Date().toISOString(),
  }

  return cloneTool(mockTools[index])
}

export function deleteMockTool(id: number): void {
  const index = mockTools.findIndex((tool) => tool.id === id)
  if (index !== -1) {
    mockTools.splice(index, 1)
  }
}
