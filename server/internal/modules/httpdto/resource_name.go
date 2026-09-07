package httpdto

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
)

func AgentName(id int) string {
	return fmt.Sprintf("agents/%d", id)
}

func AgentRegistrationTokenName(id int) string {
	return fmt.Sprintf("agentRegistrationTokens/%d", id)
}

func AgentClusterSummaryName() string {
	return "agentClusterSummaries/current"
}

func AgentLocationMapName() string {
	return "agentLocationMaps/current"
}

func ParseResourceIDSegment(value string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid resource id segment")
	}
	return id, nil
}

func ParseResourceNameID(value string, collection string) (int, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || parts[0] != collection {
		return 0, fmt.Errorf("invalid resource name")
	}
	return ParseResourceIDSegment(parts[1])
}

func ParseResourceNameIDs(values []string, collection string) ([]int, error) {
	ids := make([]int, 0, len(values))
	for _, value := range values {
		id, err := ParseResourceNameID(value, collection)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func ParseNestedResourceNameID(value string, parentCollection string, childCollection string) (int, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 4 || parts[0] != parentCollection || parts[2] != childCollection {
		return 0, fmt.Errorf("invalid nested resource name")
	}
	if _, err := ParseResourceIDSegment(parts[1]); err != nil {
		return 0, err
	}
	return ParseResourceIDSegment(parts[3])
}

func ParseNestedResourceNameIDs(values []string, parentCollection string, childCollection string) ([]int, error) {
	ids := make([]int, 0, len(values))
	for _, value := range values {
		id, err := ParseNestedResourceNameID(value, parentCollection, childCollection)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func OrganizationName(id int) string {
	return fmt.Sprintf("organizations/%d", id)
}

func TargetName(id int) string {
	return resourcenames.Target(id)
}

func UserName(id int) string {
	return fmt.Sprintf("users/%d", id)
}

func ScanName(id int) string {
	return fmt.Sprintf("scans/%d", id)
}

func ScheduledScanName(id int) string {
	return fmt.Sprintf("scheduledScans/%d", id)
}

func ScanTaskName(scanID, taskID int) string {
	return resourcenames.Task(scanID, taskID)
}

func VulnerabilityName(id int) string {
	return fmt.Sprintf("vulnerabilities/%d", id)
}

func WebsiteName(targetID, id int) string {
	return fmt.Sprintf("targets/%d/websites/%d", targetID, id)
}

func SubdomainName(targetID, id int) string {
	return fmt.Sprintf("targets/%d/subdomains/%d", targetID, id)
}

func EndpointName(targetID, id int) string {
	return fmt.Sprintf("targets/%d/endpoints/%d", targetID, id)
}

func DirectoryName(targetID, id int) string {
	return fmt.Sprintf("targets/%d/directories/%d", targetID, id)
}

func HostPortName(targetID, id int) string {
	return fmt.Sprintf("targets/%d/hostPorts/%d", targetID, id)
}

func HostPortIPName(targetID int, ip string) string {
	return fmt.Sprintf("targets/%d/hostPorts/%s", targetID, ip)
}

func ScreenshotName(targetID, id int) string {
	return fmt.Sprintf("targets/%d/screenshots/%d", targetID, id)
}

func WebsiteSnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/websiteSnapshots/%d", scanID, id)
}

func SubdomainSnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/subdomainSnapshots/%d", scanID, id)
}

func EndpointSnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/endpointSnapshots/%d", scanID, id)
}

func DirectorySnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/directorySnapshots/%d", scanID, id)
}

func HostPortSnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/hostPortSnapshots/%d", scanID, id)
}

func HostPortSnapshotIPName(scanID int, ip string) string {
	return fmt.Sprintf("scans/%d/hostPorts/%s", scanID, ip)
}

func ScreenshotSnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/screenshotSnapshots/%d", scanID, id)
}

func VulnerabilitySnapshotName(scanID, id int) string {
	return fmt.Sprintf("scans/%d/vulnerabilitySnapshots/%d", scanID, id)
}

func WordlistName(id int) string {
	return resourcenames.Wordlist(id)
}

func WordlistTextName(id int) string {
	return fmt.Sprintf("%s/text", WordlistName(id))
}

func WordlistTagName(displayName string) string {
	return fmt.Sprintf("wordlistTags/%s", url.PathEscape(displayName))
}

func ScanWorkflowName(workflowID string) string {
	return resourcenames.ScanWorkflow(workflowID)
}

func ParseScanWorkflowName(value string) (string, error) {
	return resourcenames.ParseScanWorkflow(value)
}
