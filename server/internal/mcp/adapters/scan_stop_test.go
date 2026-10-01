package adapters

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

func TestMapMCPScanStopErrorKeepsExistingStableVocabulary(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{name: "missing scan maps to not found", err: scanapp.ErrScanNotFound, want: mcpErrors.ErrNotFound},
		{name: "wrapped missing scan maps to not found", err: fmt.Errorf("stop failed: %w", scanapp.ErrScanNotFound), want: mcpErrors.ErrNotFound},
		{name: "terminal scan maps to business command failure", err: scanapp.ErrScanCannotStop, want: mcpErrors.ErrCommandFailed},
		{name: "wrapped terminal scan maps to business command failure", err: fmt.Errorf("stop failed: %w", scanapp.ErrScanCannotStop), want: mcpErrors.ErrCommandFailed},
		{name: "unknown failure stays a business command failure", err: errors.New("boom"), want: mcpErrors.ErrCommandFailed},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := mapMCPScanStopError(test.err); !errors.Is(got, test.want) {
				t.Fatalf("mapMCPScanStopError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
	if got := mapMCPScanStopError(nil); got != nil {
		t.Fatalf("nil error mapped to %v", got)
	}
	if got := mapMCPScanStopError(context.Canceled); !errors.Is(got, context.Canceled) {
		t.Fatalf("cancellation must pass through, got %v", got)
	}
	if got := mapMCPScanStopError(context.DeadlineExceeded); !errors.Is(got, context.DeadlineExceeded) {
		t.Fatalf("deadline must pass through, got %v", got)
	}
}

func TestParseCanonicalScanNameRejectsEveryNonCanonicalShape(t *testing.T) {
	for _, name := range []string{"12", "scans/0", "scans/012", "scans/-3", "scans/abc", "scans/1/2", "targets/1", "operations/11111111-1111-1111-1111-111111111111", ""} {
		if id, err := parseCanonicalScanName(name); err == nil || id != 0 {
			t.Fatalf("parseCanonicalScanName(%q) = (%d, %v), want rejection", name, id, err)
		}
	}
	// Trim tolerance follows the shared investigation-tool idiom for padded
	// canonical names, matching isCanonicalScanWorkflow and parseCanonicalTarget.
	if id, err := parseCanonicalScanName("scans/12"); err != nil || id != 12 {
		t.Fatalf("parseCanonicalScanName(scans/12) = (%d, %v)", id, err)
	}
	if id, err := parseCanonicalScanName(" scans/12 "); err != nil || id != 12 {
		t.Fatalf("parseCanonicalScanName(padded scans/12) = (%d, %v), want trim tolerance", id, err)
	}
}

func TestNewScanStopperGuardsMissingFacade(t *testing.T) {
	stopper := NewScanStopper(nil)
	if _, err := stopper.Stop(context.Background(), tools.ScanStopInput{Scan: "scans/1"}); !errors.Is(err, mcpErrors.ErrInternal) {
		t.Fatalf("stop without facade err = %v, want internal", err)
	}
	if _, err := stopper.BatchStop(context.Background(), tools.BatchScanStopInput{Scans: []string{"scans/1"}}); !errors.Is(err, mcpErrors.ErrInternal) {
		t.Fatalf("batch stop without facade err = %v, want internal", err)
	}
}
