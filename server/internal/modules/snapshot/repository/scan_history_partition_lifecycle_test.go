package repository

import (
	"testing"
	"time"
)

func TestScanHistoryRangeForIDUsesFixedTenThousandIDIntervals(t *testing.T) {
	tests := []struct {
		scanID int
		start  int
		end    int
	}{
		{scanID: 0, start: 0, end: 10_000},
		{scanID: 9_999, start: 0, end: 10_000},
		{scanID: 10_000, start: 10_000, end: 20_000},
	}
	for _, tt := range tests {
		t.Run(scanHistoryPartitionName("range", tt.start), func(t *testing.T) {
			partitionRange, err := scanHistoryRangeForID(tt.scanID)
			if err != nil {
				t.Fatalf("scanHistoryRangeForID(%d): %v", tt.scanID, err)
			}
			if partitionRange.Start != tt.start || partitionRange.End != tt.end {
				t.Fatalf("scan id %d range=%+v, want [%d,%d)", tt.scanID, partitionRange, tt.start, tt.end)
			}
		})
	}
	if _, err := scanHistoryRangeForID(-1); err == nil {
		t.Fatal("negative scan id must be rejected")
	}
}

func TestScanHistoryRetentionModeValidation(t *testing.T) {
	for _, mode := range []ScanHistoryRetentionMode{
		ScanHistoryRetentionDisabled,
		ScanHistoryRetentionReport,
		ScanHistoryRetentionEnforce,
	} {
		if err := ValidateScanHistoryRetentionMode(mode); err != nil {
			t.Fatalf("mode %q should be valid: %v", mode, err)
		}
	}
	if err := ValidateScanHistoryRetentionMode("delete-now"); err == nil {
		t.Fatal("unknown retention mode must be rejected")
	}
}

func TestScanHistoryRetentionRunOptionsValidation(t *testing.T) {
	valid := ScanHistoryRetentionRunOptions{
		TaskDeleteBatchSize:          1,
		MaxTaskDeleteBatchesPerRange: 1,
		MaxRangesPerRun:              1,
		MaxRunDuration:               time.Minute,
	}
	if err := ValidateScanHistoryRetentionRunOptions(valid); err != nil {
		t.Fatalf("valid retention run options: %v", err)
	}
	for _, test := range []struct {
		name    string
		options ScanHistoryRetentionRunOptions
	}{
		{name: "task batch size", options: ScanHistoryRetentionRunOptions{MaxTaskDeleteBatchesPerRange: 1, MaxRangesPerRun: 1, MaxRunDuration: time.Minute}},
		{name: "task batch budget", options: ScanHistoryRetentionRunOptions{TaskDeleteBatchSize: 1, MaxRangesPerRun: 1, MaxRunDuration: time.Minute}},
		{name: "range budget", options: ScanHistoryRetentionRunOptions{TaskDeleteBatchSize: 1, MaxTaskDeleteBatchesPerRange: 1, MaxRunDuration: time.Minute}},
		{name: "run duration", options: ScanHistoryRetentionRunOptions{TaskDeleteBatchSize: 1, MaxTaskDeleteBatchesPerRange: 1, MaxRangesPerRun: 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateScanHistoryRetentionRunOptions(test.options); err == nil {
				t.Fatalf("invalid retention run options must fail: %+v", test.options)
			}
		})
	}
}

func TestScanHistoryParentsIncludesAllPartitionedHistoryTables(t *testing.T) {
	parents := ScanHistoryParents()
	if len(parents) != 8 {
		t.Fatalf("scan history parent count=%d, want 8", len(parents))
	}
	if !IsScanHistoryParent("task_progress_log") || IsScanHistoryParent("scan_task") || IsScanHistoryParent("asset") {
		t.Fatal("scan history parent membership must match the partitioning scope")
	}
}
