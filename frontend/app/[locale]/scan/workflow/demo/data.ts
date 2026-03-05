
export const FEATURE_LIST = [
  { key: "subdomain_discovery", label: "Subdomain Discovery", icon: "🌐" },
  { key: "port_scan", label: "Port Scan", icon: "🔌" },
  { key: "site_scan", label: "Site Scan", icon: "📄" },
  { key: "fingerprint_detect", label: "Fingerprint", icon: "🔍" },
  { key: "directory_scan", label: "Directory Scan", icon: "📂" },
  { key: "screenshot", label: "Screenshot", icon: "📸" },
  { key: "url_fetch", label: "URL Fetch", icon: "🔗" },
  { key: "vuln_scan", label: "Vuln Scan", icon: "🛡️" },
] as const

export const MOCK_WORKFLOWS = [
  {
    id: 1,
    name: "Full Scan Profile",
    description: "Comprehensive scanning with all features enabled",
    type: "preset",
    updatedAt: "2023-10-01T12:00:00Z",
    isValid: true,
    features: ["subdomain_discovery", "port_scan", "site_scan", "fingerprint_detect", "vuln_scan"],
    stats: { runs: 120, avgTime: "45m" }
  },
  {
    id: 2,
    name: "Quick Discovery",
    description: "Fast reconnaissance for subdomains and ports",
    type: "preset",
    updatedAt: "2023-10-05T09:30:00Z",
    isValid: true,
    features: ["subdomain_discovery", "port_scan"],
    stats: { runs: 850, avgTime: "5m" }
  },
  {
    id: 3,
    name: "Web Vulnerability Check",
    description: "Focus on web application vulnerabilities",
    type: "user",
    updatedAt: "2024-01-15T14:20:00Z",
    isValid: true,
    features: ["site_scan", "url_fetch", "vuln_scan", "screenshot"],
    stats: { runs: 45, avgTime: "25m" }
  },
  {
    id: 4,
    name: "Custom Asset Audit",
    description: "Legacy configuration for quarterly audits",
    type: "user",
    updatedAt: "2023-11-20T10:15:00Z",
    isValid: false, // Needs update
    features: ["subdomain_discovery", "fingerprint_detect"],
    stats: { runs: 12, avgTime: "15m" }
  },
  {
    id: 5,
    name: "Nightly Monitor",
    description: "Automated low-impact monitoring scan",
    type: "user",
    updatedAt: "2024-02-01T02:00:00Z",
    isValid: true,
    features: ["port_scan", "screenshot"],
    stats: { runs: 300, avgTime: "8m" }
  }
]
