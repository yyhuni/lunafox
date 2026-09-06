import type { EngineCatalogDetail } from "@/types/engine-catalog.types"

// Keep built-in mock strings aligned with the packaged Engine locale resources.
const engines: EngineCatalogDetail[] = [
  createSubdomainDiscoveryEngine(),
  createEngine("port_scan", {
    displayName: "Port Scan", description: "Scan Target baselines plus finalized subdomain facts for open network ports and report host-port findings.", sectionName: "Naabu Active", sectionDescription: "Actively probe target-scoped hosts for open TCP ports using naabu.", timeoutDescription: "Runtime timeout in seconds for the active naabu process.",
  }, "naabu_active"),
  createEngine("web_crawling", {
    displayName: "Web Crawling", description: "Crawl websites to discover URLs, forms, and content for downstream scanning.", sectionName: "Crawler", sectionDescription: "Configure website crawling.", timeoutDescription: "Scan timeout in seconds",
  }, "crawler"),
  createDirectoryEngine(),
  createScreenshotEngine(),
  createNucleiVulnerabilityEngine(),
  createEngine("website_discovery", {
    displayName: "Website Discovery", description: "Probe target-scoped URL lists for live websites and report website findings.", sectionName: "HTTPX", sectionDescription: "Probe Server-materialized URLs using httpx.", timeoutDescription: "Runtime timeout in seconds for the httpx process.",
  }, "httpx"),
  createEngine("url_collection", {
    displayName: "URL Collection", description: "Collect target-scoped URLs from historical sources and crawling, then report endpoint findings.", sectionName: "Waymore", sectionDescription: "Collect historical URLs for domain Targets using Waymore.", timeoutDescription: "Runtime timeout in seconds for the Waymore process.",
  }, "waymore"),
]

let operatorEngine: EngineCatalogDetail | undefined

type MockEngineInstallRequest = {
  artifactRef?: unknown
  allowReplacement?: unknown
}

type MockEngineInstallConflict = {
  engineId: string
  currentPackageDigest: string
  proposedPackageDigest: string
}

export type MockEngineInstallResult =
  | { engine: EngineCatalogDetail }
  | { error: string; conflict?: MockEngineInstallConflict }

export function getMockEngineCatalog() {
  return [...engines, ...(operatorEngine ? [operatorEngine] : [])].map(({ execution, ...engine }) => {
    return {
      ...engine,
      execution: {
        engineApiMajor: execution.engineApiMajor,
        supportedTargetTypes: execution.supportedTargetTypes,
        ...(execution.executionResources === undefined ? {} : { executionResources: execution.executionResources }),
      },
    }
  })
}

export function getMockEngineCatalogDetail(engineId: string) {
  return [...engines, ...(operatorEngine ? [operatorEngine] : [])].find((engine) => engine.engineId === engineId)
}

export function installMockEngine(request: MockEngineInstallRequest): MockEngineInstallResult {
  if (typeof request.artifactRef !== "string" || typeof request.allowReplacement !== "boolean") {
    return { error: "artifactRef and allowReplacement are required" }
  }
  const artifactRef = request.artifactRef
  const digest = getCanonicalArtifactDigest(artifactRef)
  if (!digest) return { error: "A canonical OCI digest reference is required" }

  if (!operatorEngine) {
    operatorEngine = createOperatorEngine(artifactRef, digest)
    return { engine: operatorEngine }
  }
  if (operatorEngine.artifactRef === artifactRef) return { engine: operatorEngine }
  if (!request.allowReplacement) {
    return {
      error: "Engine replacement requires confirmation",
      conflict: {
        engineId: operatorEngine.engineId,
        currentPackageDigest: operatorEngine.packageDigest,
        proposedPackageDigest: digest,
      },
    }
  }

  operatorEngine = { ...operatorEngine, artifactRef, packageDigest: digest }
  return { engine: operatorEngine }
}

export function resetMockEngineCatalog() {
  operatorEngine = undefined
}

type MockEngineLocale = {
  displayName: string
  description: string
  sectionName: string
  sectionDescription: string
  timeoutDescription: string
}

function createEngine(localId: string, locale: MockEngineLocale, sectionId: string): EngineCatalogDetail {
  const engineId = `engine.lunafox.${localId}`
  const paramKey = "timeout"
  return {
    name: `engines/${engineId}`,
    engineId,
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "0.1.0",
    artifactRef: `docker.io/lunafox/lunafox-engine-${localId.replaceAll("_", "-")}@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`,
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: supportedTargetTypesFor(localId),
      ...(localId === "subdomain_discovery" ? { executionResources: ["subfinderProviderConfig"] } : {}),
      configSections: [{
        id: sectionId,
        defaultEnabled: true,
        params: [{ key: paramKey, type: "integer", default: 3600, minimum: 1 }],
      }],
    },
    localeResources: buildEnglishLocaleResources(() => buildLocale(locale, sectionId)),
  }
}

function createNucleiVulnerabilityEngine(): EngineCatalogDetail {
  const engineId = "engine.lunafox.nuclei_vulnerability"
  const configSections = [{
    id: "nuclei",
    defaultEnabled: true,
    params: [
      { key: "scan-targets", type: "stringArray" as const, default: ["website"], enum: ["website", "endpoint"], minItems: 1, maxItems: 2 },
      { key: "timeout", type: "integer" as const, default: 3600, minimum: 60, maximum: 604800 },
      { key: "concurrency", type: "integer" as const, default: 25, minimum: 1, maximum: 100 },
      { key: "rate-limit", type: "integer" as const, default: 150, minimum: 1, maximum: 1000 },
      { key: "request-timeout", type: "integer" as const, default: 5, minimum: 1, maximum: 120 },
      { key: "bulk-size", type: "integer" as const, default: 25, minimum: 1, maximum: 100 },
      { key: "retries", type: "integer" as const, default: 1, minimum: 0, maximum: 5 },
      { key: "severity", type: "stringArray" as const, default: ["medium", "high", "critical"], enum: ["info", "low", "medium", "high", "critical"] },
      { key: "tags", type: "stringArray" as const, default: [] as string[] },
      { key: "exclude-tags", type: "stringArray" as const, default: [] as string[] },
    ],
  }]
  return {
    name: `engines/${engineId}`,
    engineId,
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "0.1.0",
    artifactRef: "docker.io/lunafox/lunafox-engine-nuclei-vulnerability@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain", "ip", "cidr"],
      executionResources: ["nucleiTemplates"],
      configSections,
    },
    localeResources: buildEnglishLocaleResources(() => ({
      engine: {
        displayName: "Nuclei vulnerability scan",
        description: "Run the controlled Nuclei Website and Endpoint vulnerability scan.",
      },
      sections: {
        nuclei: {
          name: "Nuclei",
          description: "Run the anonymous Website and Endpoint scan with approved templates.",
          params: {
            "scan-targets": { description: "Asset sources to scan" },
            timeout: { description: "Wall-clock scan timeout in seconds" },
            concurrency: { description: "Concurrent targets" },
            "rate-limit": { description: "Global requests per second" },
            "request-timeout": { description: "Per-request timeout in seconds" },
            "bulk-size": { description: "Target bulk size" },
            retries: { description: "Retries per request" },
            severity: { description: "Allowed template severities" },
            tags: { description: "Included template tags" },
            "exclude-tags": { description: "Excluded template tags" },
          },
        },
      },
    })),
  }
}

function createSubdomainDiscoveryEngine(): EngineCatalogDetail {
  return {
    name: "engines/engine.lunafox.subdomain_discovery",
    engineId: "engine.lunafox.subdomain_discovery",
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "0.1.0",
    artifactRef: "docker.io/lunafox/lunafox-engine-subdomain-discovery@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain"],
      executionResources: ["subfinderProviderConfig"],
      configSections: [
        {
          id: "recon",
          defaultEnabled: true,
          params: [
            { key: "timeout", type: "integer", default: 3600, minimum: 1 },
            { key: "threads", type: "integer", default: 10, minimum: 1 },
          ],
        },
        {
          id: "bruteforce",
          params: [
            { key: "timeout", type: "integer", default: 3600, minimum: 1 },
            { key: "wordlist", type: "string", default: "subdomains-top1million-110000.txt", resource: { kind: "wordlist" } },
            { key: "resolvers", type: "string", default: "resolvers.txt", resource: { kind: "wordlist" } },
            { key: "threads", type: "integer", default: 100, minimum: 1 },
            { key: "rate-limit", type: "integer", default: 150, minimum: 1 },
            { key: "wildcard-filter", type: "boolean", default: false },
            { key: "wildcard-probe-count", type: "integer", default: 50, minimum: 1 },
            { key: "wildcard-batch", type: "integer", default: 1000000, minimum: 1 },
          ],
        },
        {
          id: "resolve",
          defaultEnabled: true,
          requiredEnabled: true,
          params: [
            { key: "timeout", type: "integer", default: 3600, minimum: 1 },
            { key: "threads", type: "integer", default: 100, minimum: 1 },
            { key: "rate-limit", type: "integer", default: 150, minimum: 1 },
            { key: "wildcard-filter", type: "boolean", default: false },
            { key: "resolvers", type: "string", default: "resolvers.txt", resource: { kind: "wordlist" } },
          ],
        },
      ],
    },
    localeResources: buildEnglishLocaleResources(buildSubdomainDiscoveryLocale),
  }
}

function buildSubdomainDiscoveryLocale() {
  return {
    engine: {
      displayName: "Subdomain Discovery",
      description: "Discover target-domain subdomains through reconnaissance, optional dictionary bruteforce, and DNS resolution",
    },
    sections: {
      recon: {
        name: "Reconnaissance",
        description: "Collect subdomains from multiple data sources without directly probing the target.",
        params: {
          timeout: { description: "Scan timeout in seconds." },
          threads: { description: "Number of concurrent threads." },
        },
      },
      bruteforce: {
        name: "Dictionary Bruteforce",
        description: "Bruteforce domains using dictionaries to discover unpublished subdomains.",
        params: {
          timeout: { description: "Scan timeout in seconds." },
          wordlist: { description: "Subdomain wordlist selected from the Server catalog." },
          resolvers: { description: "Resolver-list Wordlist selected for PureDNS Bruteforce." },
          threads: { description: "Number of concurrent threads." },
          "rate-limit": { description: "Rate limit per second." },
          "wildcard-filter": { description: "Enable PureDNS wildcard-response filtering for this stage." },
          "wildcard-probe-count": { description: "PureDNS wildcard tests per domain level." },
          "wildcard-batch": { description: "PureDNS wildcard-filter precache batch size." },
        },
      },
      resolve: {
        name: "Resolution",
        description: "Verify discovered subdomains are currently DNS-resolvable before ingest.",
        params: {
          timeout: { description: "Scan timeout in seconds." },
          resolvers: { description: "Resolver-list Wordlist selected for PureDNS Resolve." },
          threads: { description: "Number of concurrent threads." },
          "rate-limit": { description: "Rate limit per second." },
          "wildcard-filter": { description: "Enable PureDNS wildcard-response filtering for this stage." },
        },
      },
    },
  }
}

function createDirectoryEngine(): EngineCatalogDetail {
  const engineId = "engine.lunafox.directory_scan"
  return {
    name: `engines/${engineId}`,
    engineId,
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "0.1.0",
    artifactRef: "docker.io/lunafox/lunafox-engine-directory-scan@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain", "ip", "cidr"],
      configSections: [{
        id: "ffuf",
        defaultEnabled: true,
        params: [
          { key: "wordlist", type: "string", default: "dir_default.txt", resource: { kind: "wordlist" } },
          { key: "recursion", type: "boolean", default: false },
          { key: "recursion-depth", type: "integer", default: 1, minimum: 1, maximum: 5 },
          { key: "recursion-strategy", type: "string", default: "default", enum: ["default", "greedy"] },
          { key: "auto-calibration", type: "boolean", default: true },
          { key: "auto-calibration-mode", type: "string", default: "ac", enum: ["ac", "ach"] },
          { key: "match-codes", type: "string", default: "200-299,301,302,307,401,403,405,500" },
          { key: "concurrency", type: "integer", default: 5, minimum: 1, maximum: 20 },
          { key: "threads", type: "integer", default: 10, minimum: 1, maximum: 100 },
          { key: "rate", type: "integer", default: 0, minimum: 0, maximum: 1000 },
          { key: "delay", type: "string", default: "0.1-2.0" },
          { key: "request-timeout", type: "integer", default: 10, minimum: 1, maximum: 120 },
          { key: "timeout", type: "integer", default: 86400, minimum: 60, maximum: 604800 },
          { key: "follow-redirects", type: "boolean", default: false },
          { key: "http2", type: "boolean", default: false },
        ],
      }],
    },
    localeResources: buildEnglishLocaleResources(buildDirectoryLocale),
  }
}

function buildDirectoryLocale() {
  return {
    engine: {
      displayName: "Directory Scan",
      description: "Discover target-scoped directory paths with FFUF and a selected wordlist.",
    },
    sections: {
      ffuf: {
        name: "FFUF",
        description: "Configure FFUF directory discovery.",
        params: {
          wordlist: { description: "Server-managed wordlist selected for directory discovery." },
          recursion: { description: "Enable recursive directory discovery." },
          "recursion-depth": { description: "Maximum recursive discovery depth." },
          "recursion-strategy": { description: "FFUF recursion strategy." },
          "auto-calibration": { description: "Enable FFUF automatic calibration." },
          "auto-calibration-mode": { description: "Select FFUF ac or per-host-and-parent-path ach automatic calibration." },
          "match-codes": { description: "HTTP status matcher expression passed unchanged to FFUF." },
          concurrency: { description: "Maximum number of Website scans running at once." },
          threads: { description: "FFUF worker threads for each Website process." },
          rate: { description: "Per-process request rate limit; zero keeps FFUF unlimited." },
          delay: { description: "Fixed or ranged request delay in seconds passed unchanged to FFUF." },
          "request-timeout": { description: "Per-request timeout in seconds." },
          timeout: { description: "Deadline in seconds for each Website FFUF process." },
          "follow-redirects": { description: "Follow HTTP redirects during scanning." },
          http2: { description: "Allow HTTP/2 for HTTPS requests with HTTP/1 fallback." },
        },
      },
    },
  }
}

function createScreenshotEngine(): EngineCatalogDetail {
  const engineId = "engine.lunafox.screenshot"
  return {
    name: `engines/${engineId}`,
    engineId,
    manifestVersion: "engine.v5",
    publisher: "lunafox",
    packageVersion: "0.1.0",
    artifactRef: "docker.io/lunafox/lunafox-engine-screenshot@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    packageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: supportedTargetTypesFor("screenshot"),
      configSections: [{
        id: "capture",
        defaultEnabled: true,
        params: [
          { key: "page-timeout", type: "integer", default: 15, minimum: 1, maximum: 120 },
          { key: "concurrency", type: "integer", default: 5, minimum: 1, maximum: 20 },
          { key: "retries", type: "integer", default: 1, minimum: 0, maximum: 3 },
        ],
      }],
    },
    localeResources: buildEnglishLocaleResources(() => ({
      engine: { displayName: "Screenshot", description: "Capture viewport screenshots for finalized website URLs." },
      sections: {
        capture: {
          name: "Screenshot",
          description: "Capture bounded viewport screenshots.",
          params: {
            "page-timeout": { description: "Page wait timeout in seconds" },
            concurrency: { description: "Concurrent browser pages" },
            retries: { description: "Additional screenshot attempts per URL" },
          },
        },
      },
    })),
  }
}

function createOperatorEngine(artifactRef: string, packageDigest: string): EngineCatalogDetail {
  return {
    name: "engines/engine.example.operator_scanner",
    engineId: "engine.example.operator_scanner",
    manifestVersion: "engine.v5",
    publisher: "example",
    packageVersion: "1.0.0",
    artifactRef,
    packageDigest,
    execution: {
      engineApiMajor: 2,
      supportedTargetTypes: ["domain"],
      configSections: [{
        id: "scan",
        defaultEnabled: true,
        params: [{ key: "timeout", type: "integer", default: 3600, minimum: 1 }],
      }],
    },
    localeResources: buildEnglishLocaleResources(() => buildLocale({
      displayName: "Operator Example Scanner",
      description: "A mock Engine for exercising the public OCI installation flow.",
      sectionName: "Scan",
      sectionDescription: "Configure the example scan.",
      timeoutDescription: "Scan timeout in seconds",
    }, "scan")),
  }
}

function getCanonicalArtifactDigest(artifactRef: string) {
  const match = artifactRef.match(/^[^/@\s]+(?:\/[^/@\s]+)+@(sha256:[a-f0-9]{64})$/)
  return match?.[1]
}

function supportedTargetTypesFor(localId: string): string[] {
  return localId === "subdomain_discovery" ? ["domain"] : ["domain", "ip", "cidr"]
}

function buildLocale(locale: MockEngineLocale, sectionId: string) {
  return {
    engine: { displayName: locale.displayName, description: locale.description },
    sections: {
      [sectionId]: {
        name: locale.sectionName,
        description: locale.sectionDescription,
        params: { timeout: { description: locale.timeoutDescription } },
      },
    },
  }
}

// Runtime mock payloads stay English in every locale; UI translations belong in message catalogs.
function buildEnglishLocaleResources<T>(buildLocale: () => T): { zh: T; en: T } {
  return {
    zh: buildLocale(),
    en: buildLocale(),
  }
}
