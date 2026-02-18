package domain

import "strings"

type WorkflowName string

const (
	WorkflowSubdomainDiscovery WorkflowName = "subdomain_discovery"
	WorkflowPortScan           WorkflowName = "port_scan"
	WorkflowSiteScan           WorkflowName = "site_scan"
	WorkflowFingerprintDetect  WorkflowName = "fingerprint_detect"
	WorkflowURLFetch           WorkflowName = "url_fetch"
	WorkflowDirectoryScan      WorkflowName = "directory_scan"
	WorkflowScreenshot         WorkflowName = "screenshot"
	WorkflowVulnScan           WorkflowName = "vuln_scan"
)

func ParseWorkflowName(value string) (WorkflowName, bool) {
	normalized := WorkflowName(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case WorkflowSubdomainDiscovery,
		WorkflowPortScan,
		WorkflowSiteScan,
		WorkflowFingerprintDetect,
		WorkflowURLFetch,
		WorkflowDirectoryScan,
		WorkflowScreenshot,
		WorkflowVulnScan:
		return normalized, true
	default:
		return "", false
	}
}
