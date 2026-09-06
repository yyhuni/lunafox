package application

import (
	"encoding/json"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

const templateVersionOne = 1

// Templates owns the first-phase server-side zh/en render snapshots.
type Templates struct{}

func NewTemplates() *Templates { return &Templates{} }

func (templates *Templates) Render(event domain.ValidatedEvent, locale domain.Locale) (domain.RenderSnapshot, error) {
	if err := domain.ValidateLocale(locale); err != nil {
		return domain.RenderSnapshot{}, err
	}
	title, message, err := renderTemplateV1(event, locale)
	if err != nil {
		return domain.RenderSnapshot{}, err
	}
	payload, err := json.Marshal(map[string]any{
		"title": title, "message": message, "locale": locale, "templateVersion": templateVersionOne,
	})
	if err != nil {
		return domain.RenderSnapshot{}, err
	}
	return domain.RenderSnapshot{Locale: locale, TemplateVersion: templateVersionOne, Title: title, Message: message, ProviderPayload: payload}, nil
}

func renderTemplateV1(event domain.ValidatedEvent, locale domain.Locale) (string, string, error) {
	switch payload := event.Payload.(type) {
	case domain.ScanSucceededPayload:
		if locale == domain.LocaleChinese {
			return "扫描完成", fmt.Sprintf("目标 %s 的扫描已成功完成。", payload.TargetName), nil
		}
		return "Scan completed", fmt.Sprintf("Scan for %s completed successfully.", payload.TargetName), nil
	case domain.ScanFailedPayload:
		if locale == domain.LocaleChinese {
			return "扫描失败", fmt.Sprintf("目标 %s 的扫描失败：%s。", payload.TargetName, payload.FailureMessage), nil
		}
		return "Scan failed", fmt.Sprintf("Scan for %s failed: %s.", payload.TargetName, payload.FailureMessage), nil
	case domain.VulnerabilityObservedPayload:
		if locale == domain.LocaleChinese {
			return "发现漏洞", fmt.Sprintf("目标 %s 发现 %s 级别的 %s。", payload.TargetName, payload.Severity, payload.VulnType), nil
		}
		return "Vulnerability observed", fmt.Sprintf("%s %s observed for %s.", payload.Severity, payload.VulnType, payload.TargetName), nil
	case domain.AgentOfflinePayload:
		if locale == domain.LocaleChinese {
			return "Agent 离线", fmt.Sprintf("Agent %s 已离线。", payload.DisplayName), nil
		}
		return "Agent offline", fmt.Sprintf("Agent %s is offline.", payload.DisplayName), nil
	case domain.NucleiPOCSyncSucceededPayload:
		commitPrefix := payload.CommitSHA[:12]
		if locale == domain.LocaleChinese {
			return "Nuclei POC 同步完成", fmt.Sprintf("已导入 %d 个 Nuclei POC，提交 %s。", payload.CommittedPOCCount, commitPrefix), nil
		}
		return "Nuclei POC sync completed", fmt.Sprintf("Imported %d Nuclei POCs from commit %s.", payload.CommittedPOCCount, commitPrefix), nil
	case domain.NucleiPOCSyncFailedPayload:
		if locale == domain.LocaleChinese {
			return "Nuclei POC 同步失败", fmt.Sprintf("同步失败（%s）：%s", payload.FailureCode, payload.FailureSummary), nil
		}
		return "Nuclei POC sync failed", fmt.Sprintf("Sync failed (%s): %s", payload.FailureCode, payload.FailureSummary), nil
	default:
		return "", "", fmt.Errorf("no template for notification kind %q", event.Kind)
	}
}

var _ TemplateRenderer = (*Templates)(nil)
