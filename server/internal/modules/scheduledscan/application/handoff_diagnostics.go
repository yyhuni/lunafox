package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	"go.uber.org/zap"
)

const (
	maxDiagnosticErrorNodes = 8
	maxDiagnosticTextBytes  = 1024
	maxDiagnosticSamples    = 5
)

var sensitiveDiagnosticText = regexp.MustCompile(`(?i)password|passwd|pwd|token|secret|credential|authorization|bearer|cookie|api[ _-]?key|private[ _-]?key|://`)

type diagnosticErrorNode struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	SQLState string `json:"sqlState,omitempty"`
}

type diagnosticBatchItem struct {
	Index    int    `json:"index"`
	TargetID int    `json:"targetId,omitempty"`
	Reason   string `json:"reason"`
	Message  string `json:"message,omitempty"`
}

// Diagnostic projections never become HandoffOutcome.Message: Server logs
// retain debugging context while persistence and APIs keep the safe summary.
func handoffDiagnosticFields(input FrozenDispatchInput, result *scanapp.BatchScanResult, err error) []zap.Field {
	fields := []zap.Field{
		zap.String("scan.workflow.id", diagnosticText(input.ScanWorkflowID)),
		zap.String("scan.input_source", string(input.InputSource)),
		zap.Bool("scheduled_scan.target_scoped", input.TargetScoped),
		zap.Int("scheduled_scan.target_count", len(input.TargetIDs)),
		zap.Ints("scheduled_scan.target_sample", input.TargetIDs[:min(len(input.TargetIDs), maxDiagnosticSamples)]),
		zap.Bool("scheduled_scan.target_sample_truncated", len(input.TargetIDs) > maxDiagnosticSamples),
		zap.Bool("scan.batch.result_present", result != nil),
	}
	if input.AgentID != nil {
		fields = append(fields, zap.Int("agent.id", *input.AgentID))
	}
	if result != nil {
		fields = append(fields,
			zap.Int("scan.batch.created_count", result.CreatedCount),
			zap.Int("scan.batch.scan_count", len(result.Scans)),
			zap.Int("scan.batch.failed_count", len(result.Failed)),
			zap.Int("scan.batch.skipped_count", len(result.Skipped)),
			zap.Any("scan.batch.failed_sample", diagnosticBatchSamples(result.Failed)),
			zap.Any("scan.batch.skipped_sample", diagnosticBatchSamples(result.Skipped)),
			zap.Bool("scan.batch.failed_sample_truncated", len(result.Failed) > maxDiagnosticSamples),
			zap.Bool("scan.batch.skipped_sample_truncated", len(result.Skipped) > maxDiagnosticSamples),
		)
	}
	return append(fields, diagnosticErrorFields("error", err)...)
}

func diagnosticBatchSamples(items []scanapp.CreateBatchItemOutcome) []diagnosticBatchItem {
	samples := make([]diagnosticBatchItem, 0, min(len(items), maxDiagnosticSamples))
	for _, item := range items[:min(len(items), maxDiagnosticSamples)] {
		samples = append(samples, diagnosticBatchItem{
			Index: item.Index, TargetID: item.TargetID,
			Reason: diagnosticText(item.Reason), Message: diagnosticText(item.Message),
		})
	}
	return samples
}

func diagnosticErrorFields(prefix string, err error) []zap.Field {
	if err == nil {
		return nil
	}
	nodes := make([]diagnosticErrorNode, 0, maxDiagnosticErrorNodes)
	causes := make([]error, 0, maxDiagnosticErrorNodes)
	var fields []zap.Field
	kind := "operation_failed"
	omitText := false
	truncated := false
	var walk func(error)
	walk = func(current error) {
		if current == nil {
			return
		}
		if len(nodes) == maxDiagnosticErrorNodes {
			truncated = true
			return
		}
		switch typed := current.(type) {
		case *scanapp.PlanTaskError:
			fields = append(fields, zap.String(prefix+".plan_task.kind", string(typed.Kind)))
			omitText = omitText || typed.Kind == scanapp.PlanTaskConfigError
		case *scanapp.ConfigResourceInputError:
			omitText = true
		case *workflowconfig.ValidationError:
			omitText = true
		case *scanapp.ConfigResourceValidationError:
			fields = append(fields,
				zap.String(prefix+".resource.cause", typed.ConfigResourceValidationCause()),
				zap.String(prefix+".resource.field", diagnosticText(typed.ConfigResourceValidationField())),
				zap.String(prefix+".resource.kind", diagnosticText(typed.ConfigResourceValidationKind())),
			)
		}
		switch current {
		case context.Canceled:
			kind = "canceled"
		case context.DeadlineExceeded:
			kind = "deadline_exceeded"
		case scanapp.ErrCreateInvalidConfig:
			omitText = true
		}
		node := diagnosticErrorNode{Type: fmt.Sprintf("%T", current)}
		if sqlErr, ok := current.(interface{ SQLState() string }); ok {
			node.SQLState = diagnosticText(sqlErr.SQLState())
		}
		nodes = append(nodes, node)
		causes = append(causes, current)
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range wrapped.Unwrap() {
				walk(child)
				if truncated {
					break
				}
			}
		case interface{ Unwrap() error }:
			walk(wrapped.Unwrap())
		}
	}
	walk(err)
	// Wrappers can echo a child's config values. Inspect the bounded chain
	// before formatting any node; incomplete chains must also fail closed.
	for index, cause := range causes {
		switch {
		case omitText:
			nodes[index].Message = "[REDACTED configuration input]"
		case truncated:
			nodes[index].Message = "[REDACTED truncated error chain]"
		default:
			nodes[index].Message = diagnosticText(cause.Error())
		}
	}
	return append(fields,
		zap.String(prefix+".kind", kind),
		zap.Any(prefix+".chain", nodes),
		zap.Bool(prefix+".chain_truncated", truncated),
	)
}

func diagnosticText(value string) string {
	// Fail closed for known credential representations rather than attempting
	// to guess where a secret value ends in an arbitrary wrapped error.
	if sensitiveDiagnosticText.MatchString(value) {
		return "[REDACTED sensitive diagnostic]"
	}
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(value, ""))
	if len(value) <= maxDiagnosticTextBytes {
		return value
	}
	const suffix = " [truncated]"
	value = value[:maxDiagnosticTextBytes-len(suffix)]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + suffix
}
