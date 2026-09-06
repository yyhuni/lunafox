import type {
  NucleiPocDetail,
  NucleiPocFilterOptionField,
  NucleiPocFilterOptionsResponse,
  NucleiPocListItem,
  NucleiPocListQuery,
  NucleiPocListResponse,
  NucleiPocSource,
  NucleiPocSyncState,
  NucleiPocSyncTask,
  SyncNucleiPocSourceRequest,
} from "@/types/nuclei-poc.types"

const MOCK_TIME = "2026-08-18T08:00:00.000Z"
const INITIAL_SOURCE: NucleiPocSource = {
  name: "nucleiPocSources/current",
  sourceType: "git",
  repoUrl: "https://github.com/projectdiscovery/nuclei-templates.git",
  commitSha: "7b7b2e94a2f8c4d9e3a1b6f08d2c7e5a9f4b1c6d",
  syncedAt: MOCK_TIME,
  createdAt: MOCK_TIME,
  updatedAt: MOCK_TIME,
}

const initialPocs: NucleiPocDetail[] = [
  {
    name: "nucleiPocs/CVE-2024-3400",
    templateId: "CVE-2024-3400",
    displayName: "PAN-OS GlobalProtect Command Injection",
    severity: "critical",
    tags: ["cve", "rce", "paloalto", "globalprotect"],
    author: "pdteam",
    description: "Detects the command injection condition in PAN-OS GlobalProtect firewall deployments.",
    cve: ["CVE-2024-3400"],
    cwe: ["CWE-78"],
    references: ["https://nvd.nist.gov/vuln/detail/CVE-2024-3400"],
    remediation: "Upgrade PAN-OS to a vendor-fixed release.",
    relativePath: "http/cves/2024/CVE-2024-3400.yaml",
    contentSha256: "0ebec6d61b20d95d3a9508f351d0f4b0d84f13ab8d4774f5a58b25fdca580e03",
    isEnabled: true,
    createdAt: MOCK_TIME,
    updatedAt: MOCK_TIME,
    content: `id: CVE-2024-3400

info:
  name: PAN-OS GlobalProtect Command Injection
  author: pdteam
  severity: critical
  tags: cve,rce,paloalto,globalprotect
  classification:
    cve-id: CVE-2024-3400
    cwe-id: CWE-78

http:
  - method: GET
    path:
      - "{{BaseURL}}/global-protect/login.esp"
`,
  },
  {
    name: "nucleiPocs/http-missing-security-headers",
    templateId: "http-missing-security-headers",
    displayName: "Missing Security Headers",
    severity: "low",
    tags: ["misconfig", "headers", "http"],
    author: "pdteam",
    description: "Checks for baseline HTTP response security headers.",
    cve: [],
    cwe: [],
    references: ["https://owasp.org/www-project-secure-headers/"],
    remediation: "Set the baseline response security headers.",
    relativePath: "http/misconfiguration/http-missing-security-headers.yaml",
    contentSha256: "f8cc23d68ac72e8d287501af66d00703b90ca71abafc6a8bcfaec9a56f642fb0",
    isEnabled: true,
    createdAt: MOCK_TIME,
    updatedAt: MOCK_TIME,
    content: `id: http-missing-security-headers

info:
  name: Missing Security Headers
  author: pdteam
  severity: low
  tags: misconfig,headers,http

http:
  - method: GET
    path:
      - "{{BaseURL}}/"
`,
  },
  {
    name: "nucleiPocs/lunafox-gateway-actuator-exposure",
    templateId: "lunafox-gateway-actuator-exposure",
    displayName: "Gateway Actuator Exposure",
    severity: "high",
    tags: ["spring", "actuator", "exposure"],
    author: "security-team",
    description: "Detects exposed Spring Boot actuator endpoints.",
    cve: [],
    cwe: ["CWE-200"],
    references: [],
    remediation: "Restrict actuator exposure to authenticated administrative networks.",
    relativePath: "http/exposures/lunafox-gateway-actuator-exposure.yaml",
    contentSha256: "b71f1c4f9d7b2b1e8824e6803ed0f7c821a1c7b5e1cf13b5c821b6f4f0c68713",
    isEnabled: false,
    createdAt: MOCK_TIME,
    updatedAt: MOCK_TIME,
    content: `id: lunafox-gateway-actuator-exposure

info:
  name: Gateway Actuator Exposure
  author: security-team
  severity: high
  tags: spring,actuator,exposure

http:
  - method: GET
    path:
      - "{{BaseURL}}/actuator/env"
`,
  },
]

type MockTaskRecord = {
  task: NucleiPocSyncTask
  polls: number
  mode: "success" | "failure" | "expired"
  request: SyncNucleiPocSourceRequest
}

let currentSource: NucleiPocSource | null = { ...INITIAL_SOURCE }
let pocs = initialPocs.map(cloneDetail)
let activeTaskName: string | null = null
let taskCounter = 1
const tasks = new Map<string, MockTaskRecord>()

type MockPageToken = {
  v: 1
  s: number
  f: string
  r: string
  c: string
  i: string
}

function requestError(status: number, code: string, message: string) {
  const error = new TypeError(message) as TypeError & { status?: number; code?: string }
  error.status = status
  error.code = code
  return error
}

function normalizeSyncRequest(request: SyncNucleiPocSourceRequest): SyncNucleiPocSourceRequest {
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(request.requestId) || request.requestId === "00000000-0000-0000-0000-000000000000") {
    throw requestError(400, "INVALID_ARGUMENT", "requestId must be a canonical UUID")
  }
  if (request.sourceType !== "git" && request.sourceType !== "gitee" && request.sourceType !== "custom") {
    throw requestError(400, "INVALID_ARGUMENT", "sourceType must be git, gitee, or custom")
  }
  let parsed: URL
  try {
    parsed = new URL(request.repoUrl.trim())
  } catch {
    throw requestError(400, "INVALID_ARGUMENT", "repoUrl must be a public HTTPS repository URL")
  }
  const host = parsed.hostname.toLowerCase().replace(/\.$/, "")
  const ipLiteral = /^\d{1,3}(?:\.\d{1,3}){3}$/.test(host) || host.includes(":")
  let decodedPath = parsed.pathname
  try {
    decodedPath = decodeURIComponent(decodedPath)
  } catch {
    throw requestError(400, "INVALID_ARGUMENT", "repoUrl must be a public HTTPS repository URL")
  }
  if (parsed.protocol !== "https:" || parsed.username || parsed.password || parsed.search || parsed.hash || (parsed.port && parsed.port !== "443") || !parsed.pathname || parsed.pathname === "/" || ipLiteral || host.includes("_") || host === "localhost" || host.endsWith(".localhost") || host.endsWith(".local") || !/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/.test(host) || /[\u0000-\u001f\u007f\\]/.test(decodedPath) || decodedPath.split("/").some((segment) => segment === "." || segment === "..")) {
    throw requestError(400, "INVALID_ARGUMENT", "repoUrl must be a public HTTPS repository URL")
  }
  if (request.sourceType === "gitee" && host !== "gitee.com") {
    throw requestError(400, "INVALID_ARGUMENT", "Gitee source must use gitee.com")
  }
  parsed.hostname = host
  parsed.port = ""
  return { ...request, repoUrl: parsed.toString() }
}

export function getMockNucleiPocSource(empty = false): NucleiPocSource | null {
  return empty ? null : currentSource ? { ...currentSource } : null
}

export function createMockNucleiPocSync(
  request: SyncNucleiPocSourceRequest,
  forceFailure = false,
): { task?: NucleiPocSyncTask; conflictTaskName?: string } {
	request = normalizeSyncRequest(request)
  const replay = Array.from(tasks.values()).find((record) => record.task.requestId === request.requestId)
  if (replay) {
	    if (replay.mode === "expired") {
	      throw requestError(410, "SYNC_REQUEST_EXPIRED", "The sync request record has expired; start a new synchronization.")
	    }
    if (replay.request.sourceType === request.sourceType && replay.request.repoUrl === request.repoUrl) {
      return { task: cloneTask(replay.task) }
    }
    throw requestError(409, "SYNC_REQUEST_CONFLICT", "The requestId was already used for a different source.")
  }
  if (activeTaskName) {
    const activeRecord = tasks.get(activeTaskName)
    if (activeRecord?.mode === "expired") activeTaskName = null
    const active = activeTaskName ? tasks.get(activeTaskName)?.task : undefined
    if (active && !isTerminal(active.state)) return { conflictTaskName: active.name }
  }

  const id = `00000000-0000-4000-8000-${String(taskCounter++).padStart(12, "0")}`
  const name = `nucleiPocSyncTasks/${id}` as const
  const mode = request.repoUrl.includes("expired")
    ? "expired"
    : forceFailure || request.repoUrl.includes("fail")
      ? "failure"
      : "success"
  const task: NucleiPocSyncTask = {
    name,
    requestId: request.requestId,
    sourceType: request.sourceType,
    state: "VALIDATING_SOURCE",
    phase: "VALIDATING_SOURCE",
    counters: {
      filesSeen: null,
      yamlFilesSeen: null,
      templatesValidated: null,
      bytesRead: null,
    },
    diagnostics: { samples: [], total: 0, truncated: false },
    cleanupStatus: "pending",
    createdAt: MOCK_TIME,
    startedAt: MOCK_TIME,
    completedAt: null,
    updatedAt: MOCK_TIME,
  }
  tasks.set(name, { task, polls: 0, mode, request: { ...request } })
  activeTaskName = name
  return { task: cloneTask(task) }
}

export function getMockNucleiPocSyncTask(name: string): NucleiPocSyncTask | null {
  const record = tasks.get(name)
  if (!record) return null
  if (record.mode === "expired") {
    // An expired retained record no longer occupies the singleton slot; a
    // replay is still rejected by the request tombstone branch above.
    if (activeTaskName === name) activeTaskName = null
    return null
  }
  if (!isTerminal(record.task.state)) {
    record.polls += 1
    advanceTask(record)
  }
  return cloneTask(record.task)
}

export function getMockNucleiPocs(query: NucleiPocListQuery): NucleiPocListResponse {
	const pageSize = query.pageSize ?? 50
	if (!Number.isInteger(pageSize) || pageSize < 1 || pageSize > 1000) throw requestError(400, "INVALID_ARGUMENT", "Invalid pageSize")
	const filter = normalizeMockFilter(query.filter)
	const orderBy = normalizeMockOrderBy(query.orderBy)
	const cursor = decodePageToken(query.pageToken, { pageSize, filter, orderBy })
	const filtered = applyFilter(pocs, filter)
	const ordered = orderPocs(filtered, orderBy)
	const start = cursor ? ordered.findIndex((item) => isAfterMockCursor(item, orderBy, cursor)) : 0
	const from = cursor && start < 0 ? ordered.length : Math.max(start, 0)
	const results = ordered.slice(from, from + pageSize).map(toListItem)
	const hasMore = from + results.length < ordered.length
	return {
		results,
		totalSize: ordered.length,
		...(hasMore && results.length > 0 ? { nextPageToken: encodePageToken({ v: 1, s: pageSize, f: filter, r: orderBy, c: mockSortValue(results[results.length - 1]!, orderBy), i: results[results.length - 1]!.templateId }) } : {}),
	}
}

export function getMockNucleiPocFilterOptions(
  field: NucleiPocFilterOptionField,
): NucleiPocFilterOptionsResponse {
  if (field !== "tags") {
    throw requestError(400, "INVALID_ARGUMENT", "Nuclei POC filter options support only the tags field")
  }
  const counts = new Map<string, number>()
  for (const poc of pocs) {
    const tags = new Set(
      poc.tags
        .map((tag) => tag.trim().toLowerCase())
        .filter(Boolean),
    )
    for (const tag of tags) counts.set(tag, (counts.get(tag) ?? 0) + 1)
  }
  return {
    results: [...counts.entries()]
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([value, count]) => ({ value, label: value, count })),
  }
}

export function getMockNucleiPoc(name: string, detailError = false): NucleiPocDetail | null {
  if (detailError) return null
  const poc = pocs.find((item) => item.name === name)
  return poc ? cloneDetail(poc) : null
}

export function updateMockNucleiPocEnabled(name: string, isEnabled: boolean): NucleiPocDetail | null {
  const index = pocs.findIndex((item) => item.name === name)
  if (index < 0) return null
  pocs[index] = { ...pocs[index], isEnabled, updatedAt: new Date().toISOString() }
  return cloneDetail(pocs[index])
}

export function setMockNucleiPocActivation(
  enabled: boolean,
  forceFailure = false,
  emptyCatalog = false,
  names?: unknown,
): { enabled: boolean; affectedCount: number } {
  const active = activeTaskName ? tasks.get(activeTaskName) : undefined
  if (active && !isTerminal(active.task.state)) {
    const error = requestError(409, "SYNC_ALREADY_RUNNING", "Another Nuclei POC sync is already running.") as TypeError & { taskName?: string }
    error.taskName = active.task.name
    throw error
  }
  if (forceFailure) throw requestError(500, "INTERNAL", "Nuclei POC activation failed")
  const selectedNames = names === undefined ? undefined : normalizeMockActivationNames(names)
  if (selectedNames) {
    const knownNames = new Set(pocs.map((poc) => poc.name))
    if (selectedNames.some((name) => !knownNames.has(name))) {
      throw requestError(404, "NOT_FOUND", "Nuclei POC not found")
    }
  } else if (emptyCatalog) {
    return { enabled, affectedCount: 0 }
  }
  const selectedNameSet = selectedNames ? new Set(selectedNames) : null
  let affectedCount = 0
  const updatedAt = new Date().toISOString()
  pocs = pocs.map((poc) => {
    if (selectedNameSet && !selectedNameSet.has(poc.name)) return poc
    if (poc.isEnabled === enabled) return poc
    affectedCount += 1
    return { ...poc, isEnabled: enabled, updatedAt }
  })
  return { enabled, affectedCount }
}

function normalizeMockActivationNames(names: unknown): `nucleiPocs/${string}`[] {
  if (!Array.isArray(names) || names.length < 1 || names.length > 1000) {
    throw requestError(400, "INVALID_ARGUMENT", "Nuclei POC activation names must contain between 1 and 1000 entries")
  }
  const seen = new Set<string>()
  return names.map((name, index) => {
    if (typeof name !== "string" || name !== name.trim()) {
      throw requestError(400, "INVALID_ARGUMENT", `Invalid Nuclei POC activation name at names[${index}]`)
    }
    const match = /^nucleiPocs\/([^/]+)$/.exec(name)
    const templateId = match?.[1]
    if (!templateId || /[\\\u0000-\u001f\u007f]/.test(templateId)) {
      throw requestError(400, "INVALID_ARGUMENT", `Invalid Nuclei POC activation name at names[${index}]`)
    }
    if (seen.has(name)) {
      throw requestError(400, "INVALID_ARGUMENT", `Duplicate Nuclei POC activation name at names[${index}]`)
    }
    seen.add(name)
    return name as `nucleiPocs/${string}`
  })
}

export function resetMockNucleiPocs() {
  currentSource = { ...INITIAL_SOURCE }
  pocs = initialPocs.map(cloneDetail)
  activeTaskName = null
  taskCounter = 1
  tasks.clear()
}

function advanceTask(record: MockTaskRecord) {
  const nextState = record.mode === "failure"
    ? failureState(record.polls)
    : successState(record.polls)
  const terminal = isTerminal(nextState)
  record.task = {
    ...record.task,
    state: nextState,
    phase: nextState,
    counters: countersForState(nextState),
    ...(terminal ? {
      completedAt: MOCK_TIME,
      cleanupStatus: "clean" as const,
      ...(nextState === "SUCCEEDED"
        ? { commitSha: "c6e72d94f1ab8e0d7c2a9f3b5e1d6c8a4b7f0e2d", committedPocCount: pocs.length }
        : {
        failureCode: "TEMPLATE_INVALID",
            failureSummary: "One or more templates could not be validated.",
            diagnostics: {
              samples: [{ category: "yaml", relativePath: "http/example.yaml", reasonCode: "TEMPLATE_INVALID" }],
              total: 1,
              truncated: false,
            },
          }),
    } : {}),
  }
  if (nextState === "SUCCEEDED") {
    currentSource = {
      name: "nucleiPocSources/current",
      sourceType: record.request.sourceType,
      repoUrl: record.request.repoUrl,
      commitSha: "c6e72d94f1ab8e0d7c2a9f3b5e1d6c8a4b7f0e2d",
      syncedAt: MOCK_TIME,
      createdAt: currentSource?.createdAt ?? MOCK_TIME,
      updatedAt: MOCK_TIME,
    }
  }
  if (terminal && activeTaskName === record.task.name) activeTaskName = null
}

function successState(polls: number): NucleiPocSyncState {
  if (polls === 1) return "CLONING"
  if (polls === 2) return "SCANNING_FILES"
  if (polls === 3) return "VALIDATING_TEMPLATES"
  if (polls === 4) return "COMMITTING"
  if (polls === 5) return "CLEANING"
  return "SUCCEEDED"
}

function failureState(polls: number): NucleiPocSyncState {
  if (polls === 1) return "CLONING"
  if (polls === 2) return "VALIDATING_TEMPLATES"
  return "FAILED"
}

function countersForState(state: NucleiPocSyncState) {
  const filesSeen = state === "VALIDATING_SOURCE" || state === "CLONING" ? null : 37
  const yamlFilesSeen = state === "VALIDATING_SOURCE" || state === "CLONING" ? null : 12
  const templatesValidated = ["VALIDATING_SOURCE", "CLONING", "SCANNING_FILES"].includes(state) ? null : 12
  const bytesRead = state === "VALIDATING_SOURCE" || state === "CLONING" ? null : 42_880
  return { filesSeen, yamlFilesSeen, templatesValidated, bytesRead }
}

function isTerminal(state: NucleiPocSyncState) {
  return state === "SUCCEEDED" || state === "FAILED"
}

function cloneDetail(item: NucleiPocDetail): NucleiPocDetail {
  return {
    ...item,
    tags: [...item.tags],
    cve: [...item.cve],
    cwe: [...item.cwe],
    references: [...item.references],
  }
}

function cloneTask(task: NucleiPocSyncTask): NucleiPocSyncTask {
  return {
    ...task,
    counters: { ...task.counters },
    diagnostics: {
      ...task.diagnostics,
      samples: task.diagnostics.samples.map((sample) => ({ ...sample })),
    },
  }
}

function toListItem(item: NucleiPocDetail): NucleiPocListItem {
  return {
    name: item.name,
    templateId: item.templateId,
    displayName: item.displayName,
    severity: item.severity,
    tags: [...item.tags],
    author: item.author,
    description: item.description,
    cve: [...item.cve],
    cwe: [...item.cwe],
    references: [...item.references],
    remediation: item.remediation,
    relativePath: item.relativePath,
    contentSha256: item.contentSha256,
    isEnabled: item.isEnabled,
    createdAt: item.createdAt,
    updatedAt: item.updatedAt,
  }
}

function encodePageToken(token: MockPageToken) {
  return `mock-nuclei-poc-${encodeURIComponent(JSON.stringify(token))}`
}

function decodePageToken(token: string | undefined, query: { pageSize: number; filter: string; orderBy: string }): { value: string; id: string } | null {
  if (!token) return null
  const match = /^mock-nuclei-poc-(.+)$/.exec(token)
  if (!match) throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC page token")
  let payload: Partial<MockPageToken>
  try {
    payload = JSON.parse(decodeURIComponent(match[1]!)) as Partial<MockPageToken>
  } catch {
    throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC page token")
  }
  if (payload.v !== 1 || payload.s !== query.pageSize || payload.f !== query.filter || payload.r !== query.orderBy || typeof payload.c !== "string" || typeof payload.i !== "string") {
    throw requestError(400, "INVALID_ARGUMENT", "Page token query shape mismatch")
  }
  return { value: payload.c, id: payload.i }
}

function normalizeMockOrderBy(value: string | undefined) {
  const parts = (value?.trim() || "templateId asc").split(/\s+/)
  if (parts.length > 2 || !["templateId", "name", "severity", "updatedAt"].includes(parts[0]!)) throw requestError(400, "INVALID_ARGUMENT", "Invalid orderBy")
  const direction = (parts[1] || "asc").toLowerCase()
  if (direction !== "asc" && direction !== "desc") throw requestError(400, "INVALID_ARGUMENT", "Invalid orderBy")
  return `${parts[0]} ${direction}`
}

type MockFilterTerm = { field: string; operator: "=" | "=="; value: string }

function normalizeMockFilter(filter: string | undefined) {
  const groups = parseMockFilter(filter)
  return groups.map((group) => group.map((term) => `${term.field}${term.operator}${JSON.stringify(term.value.toLowerCase())}`).join(" OR ")).join(" AND ")
}

function parseMockFilter(filter: string | undefined): MockFilterTerm[][] {
  const value = filter?.trim() ?? ""
  if (!value) return []
  const andGroups = splitMockFilterOperators(value, "AND")
  const groups = andGroups.map((andGroup) => splitMockFilterOperators(andGroup, "OR").map(parseMockFilterTerm))
  for (const group of groups) {
    if (group.length === 0) throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter")
    for (const term of group) {
      if (!["templateId", "name", "author", "description", "tags", "cve", "cwe", "severity"].includes(term.field)) {
        throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter field")
      }
      if (term.operator === "==" && !["severity", "tags"].includes(term.field)) {
        throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter operator")
      }
      if (term.field === "severity" && (term.operator !== "==" || !["info", "low", "medium", "high", "critical"].includes(term.value.toLowerCase()))) {
        throw requestError(400, "INVALID_ARGUMENT", "Invalid severity")
      }
      if (!term.value.trim() || term.value.includes("*") || term.value.includes("%")) {
        throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter value")
      }
    }
  }
  return groups
}

function parseMockFilterTerm(value: string): MockFilterTerm {
  const match = /^([a-zA-Z]+)(==|=)\"((?:\\\\.|[^\"\\\\])*)\"$/.exec(value.trim())
  if (!match) throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter")
  let decoded: string
  try {
    decoded = JSON.parse(`\"${match[3]}\"`) as string
  } catch {
    throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter")
  }
  return { field: match[1]!, operator: match[2] as "=" | "==", value: decoded }
}

function splitMockFilterOperators(value: string, operator: "AND" | "OR") {
  const parts: string[] = []
  let start = 0
  let inQuote = false
  let escaped = false
  for (let index = 0; index < value.length; index += 1) {
    const character = value[index]
    if (inQuote) {
      if (escaped) escaped = false
      else if (character === "\\") escaped = true
      else if (character === "\"") inQuote = false
      continue
    }
    if (character === "\"") {
      inQuote = true
      continue
    }
    const symbol = operator === "AND" ? "&&" : "||"
    if (value.slice(index, index + 2) === symbol) {
      parts.push(value.slice(start, index))
      start = index + 2
      index += 1
      continue
    }
    if (value.slice(index, index + operator.length).toUpperCase() === operator) {
      const before = index === 0 || /\s/.test(value[index - 1]!)
      const afterIndex = index + operator.length
      const after = afterIndex === value.length || /\s/.test(value[afterIndex]!)
      if (before && after) {
        parts.push(value.slice(start, index))
        start = afterIndex
        index = afterIndex - 1
      }
    }
  }
  if (inQuote || escaped) throw requestError(400, "INVALID_ARGUMENT", "Invalid Nuclei POC filter")
  parts.push(value.slice(start))
  return parts
}

function applyFilter(items: NucleiPocDetail[], filter: string | undefined) {
	const groups = parseMockFilter(filter)
	if (groups.length === 0) return [...items]
	return items.filter((item) => groups.every((group) => group.some((term) => matchesFilterTerm(item, term))))
}

function matchesFilterTerm(item: NucleiPocDetail, term: MockFilterTerm) {
	const { field, operator } = term
	const value = term.value.toLocaleLowerCase()
	if (field === "severity") return operator === "==" && item.severity === value
	if (field === "tags" && operator === "==") return item.tags.some((tag) => tag.toLocaleLowerCase() === value)
  const candidates: Record<string, string[]> = {
    templateId: [item.templateId],
    name: [item.displayName],
    author: [item.author],
    description: [item.description],
    tags: item.tags,
    cve: item.cve,
    cwe: item.cwe,
  }
	return (candidates[field] ?? []).some((candidate) => candidate.toLocaleLowerCase().includes(value))
}

function orderPocs(items: NucleiPocDetail[], orderBy: string | undefined) {
  const [field = "templateId", direction = "asc"] = (orderBy || "templateId asc").split(/\s+/, 2)
  const factor = direction.toLocaleLowerCase() === "desc" ? -1 : 1
  return [...items].sort((left, right) => {
    const values: Record<string, [string, string]> = {
      templateId: [left.templateId, right.templateId],
      name: [left.displayName, right.displayName],
      severity: [left.severity, right.severity],
      updatedAt: [left.updatedAt, right.updatedAt],
    }
    const [leftValue, rightValue] = values[field] ?? values.templateId
    const difference = leftValue.localeCompare(rightValue)
    return difference === 0 ? left.templateId.localeCompare(right.templateId) : difference * factor
  })
}

function mockSortValue(item: Pick<NucleiPocListItem, "templateId" | "displayName" | "severity" | "updatedAt">, orderBy: string) {
  const [field = "templateId"] = orderBy.split(/\s+/, 1)
  switch (field) {
    case "name":
      return item.displayName.toLocaleLowerCase()
    case "severity":
      return item.severity.toLocaleLowerCase()
    case "updatedAt":
      return item.updatedAt
    default:
      return item.templateId.toLocaleLowerCase()
  }
}

function isAfterMockCursor(item: NucleiPocDetail, orderBy: string, cursor: { value: string; id: string }) {
	const [, direction = "asc"] = orderBy.split(/\s+/, 2)
	const valueComparison = mockSortValue(item, orderBy).localeCompare(cursor.value)
	const comparison = valueComparison === 0
	    ? item.templateId.localeCompare(cursor.id)
	    : valueComparison
	  // The secondary templateId ordering is always ascending, matching the
	  // server's stable tie-breaker even when the primary field is descending.
	  if (valueComparison === 0) return comparison > 0
	  return direction === "desc" ? comparison < 0 : comparison > 0
}
