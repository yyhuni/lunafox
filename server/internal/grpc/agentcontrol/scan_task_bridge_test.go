package agentcontrol

import (
	"context"
	"testing"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanTaskBridgeStub struct{}

func (scanTaskBridgeStub) ReportTerminalTaskResult(context.Context, int, string, int64, int, string, *scanapp.FailureDetail) error {
	return nil
}

func TestScanTaskBridgeNamingContract(t *testing.T) {
	var _ ScanTaskBridge = scanTaskBridgeStub{}
}
