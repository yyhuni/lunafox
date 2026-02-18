import type { ScanEngine, PresetEngine } from '@/types/engine.types'

// Preset engines (system-defined, read-only)
export const mockPresetEngines: PresetEngine[] = [
  {
    id: 'full_scan',
    name: '完整扫描',
    description: '执行所有扫描阶段，包括子域名发现、端口扫描、站点扫描、指纹识别、目录扫描、截图、URL抓取和漏洞扫描',
    configuration: `# 完整扫描配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 10
      assetfinder:
        enabled: true
        timeout-runtime: 3600
  bruteforce:
    enabled: true
    tools:
      subdomain-bruteforce:
        enabled: true
        timeout-runtime: 7200
        threads-cli: 50
        rate-limit-cli: 1000
        wildcard-tests-cli: 5
        wildcard-batch-cli: 10000
        subdomain-wordlist-name-runtime: "default"
  permutation:
    enabled: true
    tools:
      subdomain-permutation-resolve:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 50
        rate-limit-cli: 1000
        wildcard-tests-cli: 5
        wildcard-batch-cli: 10000
        wildcard-sample-timeout-runtime: 300
        wildcard-sample-multiplier-runtime: 2
        wildcard-expansion-threshold-runtime: 50
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 50
        rate-limit-cli: 1000

port_scan:
  enabled: true
  ports: "1-65535"
  rate: 1000

site_scan:
  enabled: true

fingerprint_detect:
  enabled: true

directory_scan:
  enabled: true

screenshot:
  enabled: true

url_fetch:
  enabled: true

vuln_scan:
  enabled: true
  severity: "critical,high,medium"
`,
  },
  {
    id: 'quick_scan',
    name: '快速扫描',
    description: '仅执行子域名发现和端口扫描，适合快速侦查',
    configuration: `# 快速扫描配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 1800
        threads-cli: 20
      assetfinder:
        enabled: true
        timeout-runtime: 1800
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 1800
        threads-cli: 50
        rate-limit-cli: 2000

port_scan:
  enabled: true
  ports: "80,443,8080,8443,22,21,3306,5432,6379,27017"
  rate: 2000
`,
  },
  {
    id: 'subdomain_only',
    name: '子域名发现',
    description: '专注于子域名枚举，包括被动收集、字典爆破和排列组合',
    configuration: `# 子域名发现配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 10
      assetfinder:
        enabled: true
        timeout-runtime: 3600
  bruteforce:
    enabled: true
    tools:
      subdomain-bruteforce:
        enabled: true
        timeout-runtime: 7200
        threads-cli: 100
        rate-limit-cli: 2000
        wildcard-tests-cli: 5
        wildcard-batch-cli: 10000
        subdomain-wordlist-name-runtime: "large"
  permutation:
    enabled: true
    tools:
      subdomain-permutation-resolve:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 100
        rate-limit-cli: 2000
        wildcard-tests-cli: 5
        wildcard-batch-cli: 10000
        wildcard-sample-timeout-runtime: 300
        wildcard-sample-multiplier-runtime: 2
        wildcard-expansion-threshold-runtime: 50
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 100
        rate-limit-cli: 2000
`,
  },
  {
    id: 'vuln_only',
    name: '漏洞扫描',
    description: '仅执行漏洞扫描，适合对已知资产进行安全检测',
    configuration: `# 漏洞扫描配置
vuln_scan:
  enabled: true
  severity: "critical,high,medium,low"
  rate-limit: 100
  bulk-size: 25
  timeout: 10
`,
  },
  {
    id: 'web_scan',
    name: 'Web应用扫描',
    description: '针对Web应用的安全扫描，包括站点发现、指纹识别、目录扫描和漏洞检测',
    configuration: `# Web应用扫描配置
site_scan:
  enabled: true

fingerprint_detect:
  enabled: true

directory_scan:
  enabled: true
  threads: 50
  timeout: 10

vuln_scan:
  enabled: true
  severity: "critical,high,medium"
  tags: "cve,oast,xss,sqli,lfi,rce"
`,
  },
  {
    id: 'api_scan',
    name: 'API安全扫描',
    description: '针对API端点的安全检测，包括端口扫描、URL抓取和漏洞扫描',
    configuration: `# API安全扫描配置
port_scan:
  enabled: true
  ports: "80,443,8080,8443,3000,5000,8000"
  rate: 1000

url_fetch:
  enabled: true
  depth: 3
  timeout: 10

vuln_scan:
  enabled: true
  severity: "critical,high"
  tags: "api,jwt,auth,injection"
`,
  },
  {
    id: 'recon_passive',
    name: '被动侦查',
    description: '仅使用被动收集技术进行侦查，不主动发送请求到目标',
    configuration: `# 被动侦查配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 10
      assetfinder:
        enabled: true
        timeout-runtime: 3600
  bruteforce:
    enabled: false
  permutation:
    enabled: false
  resolve:
    enabled: false
`,
  },
  {
    id: 'port_full',
    name: '全端口扫描',
    description: '扫描所有65535个端口，适合深度端口发现',
    configuration: `# 全端口扫描配置
port_scan:
  enabled: true
  ports: "1-65535"
  rate: 5000
  timeout: 3
`,
  },
  {
    id: 'screenshot_only',
    name: '截图采集',
    description: '仅对目标进行截图采集，用于资产可视化',
    configuration: `# 截图采集配置
screenshot:
  enabled: true
  timeout: 30
  width: 1920
  height: 1080
  full-page: false
`,
  },
  {
    id: 'compliance_scan',
    name: '合规检测',
    description: '针对合规要求的安全检测，包括SSL/TLS配置、安全头部等',
    configuration: `# 合规检测配置
site_scan:
  enabled: true

fingerprint_detect:
  enabled: true

vuln_scan:
  enabled: true
  severity: "critical,high,medium,low,info"
  tags: "ssl,tls,headers,misconfig,exposure"
`,
  },
  {
    id: 'ci_scan',
    name: 'CI/CD集成扫描',
    description: '适用于CI/CD流水线的快速安全扫描，超时时间短',
    configuration: `# CI/CD集成扫描配置
port_scan:
  enabled: true
  ports: "80,443,8080,8443"
  rate: 3000
  timeout: 2

vuln_scan:
  enabled: true
  severity: "critical,high"
  timeout: 5
  rate-limit: 200
`,
  },
]

// User engines (stored in database, editable)
export const mockEngines: ScanEngine[] = [
  {
    id: 1,
    name: '我的完整扫描',
    isValid: true,
    configuration: `# 我的完整扫描配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 10
      assetfinder:
        enabled: true
        timeout-runtime: 3600
  bruteforce:
    enabled: true
    tools:
      subdomain-bruteforce:
        enabled: true
        timeout-runtime: 7200
        threads-cli: 50
        rate-limit-cli: 1000
        wildcard-tests-cli: 5
        wildcard-batch-cli: 10000
        subdomain-wordlist-name-runtime: "default"
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 3600
        threads-cli: 50
        rate-limit-cli: 1000

port_scan:
  enabled: true
  ports: "1-10000"
  rate: 500

site_scan:
  enabled: true

fingerprint_detect:
  enabled: true

vuln_scan:
  enabled: true
  severity: "critical,high"
`,
    createdAt: '2024-01-15T08:00:00Z',
    updatedAt: '2024-12-20T10:30:00Z',
  },
  {
    id: 2,
    name: '快速侦查',
    isValid: true,
    configuration: `# 快速侦查配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 1800
        threads-cli: 20
  bruteforce:
    enabled: false
    tools:
      subdomain-bruteforce:
        enabled: false
  permutation:
    enabled: false
    tools:
      subdomain-permutation-resolve:
        enabled: false
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 1800
        threads-cli: 50
        rate-limit-cli: 2000

port_scan:
  enabled: true
  ports: "80,443,8080,8443"
  rate: 2000
`,
    createdAt: '2024-02-10T09:00:00Z',
    updatedAt: '2024-12-18T14:00:00Z',
  },
  {
    id: 3,
    name: '旧版配置',
    isValid: false,  // Simulate configuration incompatibility
    configuration: `# 旧版配置 - 格式已过期
stages:
  - name: vulnerability_scan
    tools:
      - nuclei
    options:
      severity: critical,high,medium
`,
    createdAt: '2024-03-05T11:00:00Z',
    updatedAt: '2024-12-15T16:20:00Z',
  },
  {
    id: 4,
    name: '内网渗透',
    isValid: true,
    configuration: `# 内网渗透配置
port_scan:
  enabled: true
  ports: "21,22,23,25,53,80,110,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,6379,8080,27017"
  rate: 500

fingerprint_detect:
  enabled: true

vuln_scan:
  enabled: true
  severity: "critical,high"
  tags: "default-login,cve,exposure"
`,
    createdAt: '2024-04-10T10:00:00Z',
    updatedAt: '2024-12-19T09:00:00Z',
  },
  {
    id: 5,
    name: '子域名深度枚举',
    isValid: true,
    configuration: `# 子域名深度枚举配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 7200
        threads-cli: 20
      assetfinder:
        enabled: true
        timeout-runtime: 7200
  bruteforce:
    enabled: true
    tools:
      subdomain-bruteforce:
        enabled: true
        timeout-runtime: 14400
        threads-cli: 200
        rate-limit-cli: 5000
        subdomain-wordlist-name-runtime: "huge"
  permutation:
    enabled: true
    tools:
      subdomain-permutation-resolve:
        enabled: true
        timeout-runtime: 7200
        threads-cli: 200
        rate-limit-cli: 5000
  resolve:
    enabled: true
    tools:
      subdomain-resolve:
        enabled: true
        timeout-runtime: 7200
        threads-cli: 200
        rate-limit-cli: 5000
`,
    createdAt: '2024-05-20T14:00:00Z',
    updatedAt: '2024-12-17T11:30:00Z',
  },
  {
    id: 6,
    name: 'Web漏洞检测',
    isValid: true,
    configuration: `# Web漏洞检测配置
site_scan:
  enabled: true

directory_scan:
  enabled: true
  threads: 30

url_fetch:
  enabled: true
  depth: 5

vuln_scan:
  enabled: true
  severity: "critical,high,medium"
  tags: "xss,sqli,lfi,rfi,rce,ssrf,ssti"
`,
    createdAt: '2024-06-15T16:00:00Z',
    updatedAt: '2024-12-16T08:45:00Z',
  },
  {
    id: 7,
    name: '旧版Web扫描',
    isValid: false,
    configuration: `# 旧版Web扫描 - 格式已过期
web_scan:
  enabled: true
  spider: true
  audit: true
`,
    createdAt: '2024-01-20T09:00:00Z',
    updatedAt: '2024-10-05T14:00:00Z',
  },
  {
    id: 8,
    name: '资产发现',
    isValid: true,
    configuration: `# 资产发现配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 3600
  resolve:
    enabled: true

port_scan:
  enabled: true
  ports: "80,443,8080,8443"
  rate: 1000

site_scan:
  enabled: true

screenshot:
  enabled: true
`,
    createdAt: '2024-07-01T10:00:00Z',
    updatedAt: '2024-12-20T15:00:00Z',
  },
  {
    id: 9,
    name: '高危漏洞扫描',
    isValid: true,
    configuration: `# 高危漏洞扫描配置
vuln_scan:
  enabled: true
  severity: "critical"
  tags: "cve,rce,auth-bypass"
  rate-limit: 50
  timeout: 15
`,
    createdAt: '2024-08-10T11:00:00Z',
    updatedAt: '2024-12-18T10:00:00Z',
  },
  {
    id: 10,
    name: '定时监控',
    isValid: true,
    configuration: `# 定时监控配置
subdomain_discovery:
  recon:
    enabled: true
    tools:
      subfinder:
        enabled: true
        timeout-runtime: 1800
  resolve:
    enabled: true

port_scan:
  enabled: true
  ports: "80,443"
  rate: 2000

vuln_scan:
  enabled: true
  severity: "critical,high"
`,
    createdAt: '2024-09-05T08:00:00Z',
    updatedAt: '2024-12-21T06:00:00Z',
  },
]

export function getMockEngines(): ScanEngine[] {
  return mockEngines
}

export function getMockEngineById(id: number): ScanEngine | undefined {
  return mockEngines.find(e => e.id === id)
}

export function getMockPresetEngines(): PresetEngine[] {
  return mockPresetEngines
}

export function getMockPresetEngineById(id: string): PresetEngine | undefined {
  return mockPresetEngines.find(e => e.id === id)
}
