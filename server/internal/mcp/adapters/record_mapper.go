package adapters

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	securitydomain "github.com/yyhuni/lunafox/server/internal/modules/security/domain"
)

func utcPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func scanRecord(scan scandomain.QueryScan) tools.ScanRecord {
	record := tools.ScanRecord{
		ID: scan.ID, TargetID: scan.TargetID, Workflow: scan.ScanWorkflowID, Status: scan.Status,
		InputSource: string(scan.InputSource), TriggerType: string(scan.TriggerType), AssignmentMode: scan.AssignmentMode,
		Progress: scan.Progress, CurrentStage: scan.CurrentStage, CreatedAt: scan.CreatedAt.UTC(), StoppedAt: utcPtr(scan.StoppedAt),
		AgentID: scan.AgentID, AgentName: scan.AgentName, AgentStatus: scan.AgentStatus, AgentDeleted: scan.AgentDeleted,
		CachedAssets: tools.AssetCount{
			Subdomains: scan.CachedSubdomainsCount, Websites: scan.CachedWebsitesCount, Endpoints: scan.CachedEndpointsCount,
			IPs: scan.CachedIPsCount, Directories: scan.CachedDirectoriesCount, Screenshots: scan.CachedScreenshotsCount,
			Vulnerabilities: scan.CachedVulnsTotal,
		},
	}
	if scan.Target != nil {
		record.TargetName = scan.Target.Name
	}
	if scan.Failure != nil {
		record.Failure = &tools.ScanFailure{Kind: scan.Failure.Kind, Message: scan.Failure.DisplayMessage}
		if record.Failure.Message == "" {
			record.Failure.Message = scan.Failure.Message
		}
	}
	if len(scan.RuntimeTasks) > 0 {
		record.RuntimeTasks = make([]tools.RuntimeTaskRecord, 0, len(scan.RuntimeTasks))
		for _, task := range scan.RuntimeTasks {
			record.RuntimeTasks = append(record.RuntimeTasks, tools.RuntimeTaskRecord{
				ID: task.ID, StepID: task.StepID, StageID: task.StageID, EngineID: task.EngineID, Status: task.Status,
				SkipReason: task.SkipReason, Order: task.Order, StartedAt: utcPtr(task.StartedAt), CompletedAt: utcPtr(task.CompletedAt),
				Duration: task.Duration, FailureKind: task.FailureKind,
			})
		}
	}
	return record
}

func websiteRecord(value assetdomain.Website) tools.WebsiteRecord {
	return tools.WebsiteRecord{ID: value.ID, TargetID: value.TargetID, URL: value.URL, Host: value.Host, Location: value.Location,
		CreatedAt: value.CreatedAt.UTC(), Title: value.Title, Webserver: value.Webserver, ContentType: value.ContentType,
		Tech: append([]string(nil), value.Tech...), StatusCode: value.StatusCode, ContentLength: value.ContentLength, Vhost: value.Vhost}
}

func subdomainRecord(value assetdomain.Subdomain) tools.SubdomainRecord {
	return tools.SubdomainRecord{ID: value.ID, TargetID: value.TargetID, DNSName: value.DNSName, CreatedAt: value.CreatedAt.UTC()}
}

func endpointRecord(value assetdomain.Endpoint) tools.EndpointRecord {
	return tools.EndpointRecord{ID: value.ID, TargetID: value.TargetID, URL: value.URL, Host: value.Host, Location: value.Location,
		CreatedAt: value.CreatedAt.UTC(), Title: value.Title, Webserver: value.Webserver, ContentType: value.ContentType,
		Tech: append([]string(nil), value.Tech...), StatusCode: value.StatusCode, ContentLength: value.ContentLength, Vhost: value.Vhost}
}

func directoryRecord(value assetdomain.Directory) tools.DirectoryRecord {
	return tools.DirectoryRecord{ID: value.ID, TargetID: value.TargetID, URL: value.URL, Status: value.Status,
		ContentLength: value.ContentLength, ContentType: value.ContentType, Duration: value.Duration, CreatedAt: value.CreatedAt.UTC()}
}

func hostPortRecord(value assetdomain.IPAggregationRow, hosts []string, ports []int) tools.HostPortRecord {
	return tools.HostPortRecord{IP: value.IP, Hosts: append([]string(nil), hosts...), Ports: append([]int(nil), ports...), CreatedAt: value.CreatedAt.UTC()}
}

func vulnerabilityRecord(value securitydomain.Vulnerability) tools.VulnerabilityRecord {
	record := tools.VulnerabilityRecord{ID: value.ID, TargetID: value.TargetID, URL: value.URL, VulnType: value.VulnType,
		Severity: value.Severity, Source: value.Source, Description: value.Description, Reviewed: value.Reviewed, CreatedAt: value.CreatedAt.UTC()}
	if value.CVSSScore != nil {
		if score, err := strconv.ParseFloat(value.CVSSScore.String(), 64); err == nil {
			record.CVSSScore = &score
		}
	}
	if len(value.RawOutput) > 0 {
		record.RawOutput = append(json.RawMessage(nil), value.RawOutput...)
	}
	return record
}
