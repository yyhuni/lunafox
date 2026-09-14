package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

const (
	hostJournalSchema      = 1
	hostJournalCurrentFile = "current.json"
	hostJournalHistoryDir  = "operations"
	hostJournalReceiptDir  = "receipts"
	hostJournalDirectory   = ".lunafox/upgrade"
)

// JournalEventReader is the server-side recovery boundary for the independent
// host upgrader. It returns only validated, non-secret observations.
type JournalEventReader interface {
	ReadCurrent(context.Context) (HostUpgradeEvent, error)
}

// FileJournalEventReader reads the deployment-scoped host journal. The reader
// never follows symlinks and requires the same private modes enforced by the
// host process, so a writable deployment path cannot silently redirect state.
type FileJournalEventReader struct {
	root string
}

func NewFileJournalEventReader(deploymentRoot string) (*FileJournalEventReader, error) {
	if strings.TrimSpace(deploymentRoot) == "" {
		return nil, fmt.Errorf("upgrade deployment root is required")
	}
	root, err := filepath.Abs(filepath.Clean(deploymentRoot))
	if err != nil {
		return nil, fmt.Errorf("resolve upgrade deployment root: %w", err)
	}
	return &FileJournalEventReader{root: root}, nil
}

func (reader *FileJournalEventReader) ReadCurrent(ctx context.Context) (HostUpgradeEvent, error) {
	if reader == nil || reader.root == "" {
		return HostUpgradeEvent{}, fmt.Errorf("upgrade journal reader is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return HostUpgradeEvent{}, err
	}
	privateRoot := filepath.Join(reader.root, hostJournalDirectory)
	if err := validateJournalDirectory(privateRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HostUpgradeEvent{}, os.ErrNotExist
		}
		return HostUpgradeEvent{}, fmt.Errorf("validate upgrade journal directory: %w", err)
	}
	current, err := readHostJournal(filepath.Join(privateRoot, hostJournalCurrentFile))
	if err != nil {
		return HostUpgradeEvent{}, err
	}
	// The per-operation copy is an additional integrity fence. current.json is
	// written first by the host, so an equal or older history checkpoint is
	// expected; a newer history checkpoint indicates tampering or a partial
	// replacement and must not be replayed as truth.
	history, historyErr := readHostJournal(filepath.Join(privateRoot, hostJournalHistoryDir, current.OperationID+".json"))
	if historyErr == nil {
		if history.ManifestDigest != current.ManifestDigest || history.OperationID != current.OperationID {
			return HostUpgradeEvent{}, fmt.Errorf("journal history identity mismatch")
		}
		if history.UpdatedAt.After(current.UpdatedAt) {
			return HostUpgradeEvent{}, fmt.Errorf("journal history is newer than current checkpoint")
		}
	} else if !errors.Is(historyErr, os.ErrNotExist) {
		return HostUpgradeEvent{}, historyErr
	}
	event := HostUpgradeEvent{
		OperationID:     current.OperationID,
		ManifestDigest:  current.ManifestDigest,
		Stage:           string(current.Stage),
		Diagnostic:      current.Diagnostic,
		Migration:       current.MigrationStatus,
		UpdatedAt:       current.UpdatedAt,
		ObservedDigests: map[string]string{},
		FromJournal:     true,
	}
	// Receipt is deliberately only supplemental evidence. It can provide the
	// Compose observed digests, but never changes the host stage by itself.
	receiptPath := filepath.Join(privateRoot, hostJournalReceiptDir, current.OperationID+".json")
	if receipt, receiptErr := readHostReceipt(receiptPath); receiptErr == nil {
		if receipt.OperationID != current.OperationID || receipt.ManifestDigest != current.ManifestDigest {
			return HostUpgradeEvent{}, fmt.Errorf("journal receipt identity mismatch")
		}
		for service, digest := range receipt.ObservedImages {
			event.ObservedDigests[service] = digest
		}
	} else if !errors.Is(receiptErr, os.ErrNotExist) {
		return HostUpgradeEvent{}, receiptErr
	}
	return event, nil
}

type hostJournal struct {
	SchemaVersion     int        `json:"schemaVersion"`
	OperationID       string     `json:"operationId"`
	ManifestDigest    string     `json:"manifestDigest"`
	Stage             string     `json:"stage"`
	RepairStage       string     `json:"repairStage,omitempty"`
	MigrationID       string     `json:"migrationId,omitempty"`
	MigrationChecksum string     `json:"migrationChecksum,omitempty"`
	MigrationStatus   string     `json:"migrationStatus,omitempty"`
	StartedAt         time.Time  `json:"startedAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	ExitCode          *int       `json:"exitCode,omitempty"`
	Diagnostic        string     `json:"diagnostic,omitempty"`
}

type hostReceipt struct {
	SchemaVersion  int               `json:"schemaVersion"`
	OperationID    string            `json:"operationId"`
	ManifestDigest string            `json:"manifestDigest"`
	CompletedAt    time.Time         `json:"completedAt"`
	Services       []string          `json:"services"`
	ObservedImages map[string]string `json:"observedImages"`
}

func readHostJournal(path string) (hostJournal, error) {
	data, err := readPrivateJSON(path)
	if err != nil {
		return hostJournal{}, err
	}
	var journal hostJournal
	if err := decodeStrictJSON(data, &journal); err != nil {
		return hostJournal{}, fmt.Errorf("invalid upgrade journal: %w", err)
	}
	if journal.SchemaVersion != hostJournalSchema {
		return hostJournal{}, fmt.Errorf("unsupported upgrade journal schema version %d", journal.SchemaVersion)
	}
	if _, err := uuid.Parse(journal.OperationID); err != nil || uuid.MustParse(journal.OperationID).String() != journal.OperationID {
		return hostJournal{}, fmt.Errorf("upgrade journal operationId is invalid")
	}
	if !isSHA256Digest(journal.ManifestDigest) {
		return hostJournal{}, fmt.Errorf("upgrade journal manifestDigest is invalid")
	}
	if !validHostStage(journal.Stage) {
		return hostJournal{}, fmt.Errorf("upgrade journal stage is invalid")
	}
	if journal.StartedAt.IsZero() || journal.UpdatedAt.IsZero() || journal.UpdatedAt.Before(journal.StartedAt) {
		return hostJournal{}, fmt.Errorf("upgrade journal timestamps are invalid")
	}
	if journal.CompletedAt != nil && journal.CompletedAt.Before(journal.StartedAt) {
		return hostJournal{}, fmt.Errorf("upgrade journal completedAt is invalid")
	}
	if journal.MigrationStatus != "" && !validMigrationStatus(journal.MigrationStatus) {
		return hostJournal{}, fmt.Errorf("upgrade journal migrationStatus is invalid")
	}
	if journal.MigrationChecksum != "" && !isSHA256Digest(journal.MigrationChecksum) {
		return hostJournal{}, fmt.Errorf("upgrade journal migrationChecksum is invalid")
	}
	if journal.Diagnostic != "" && (strings.ContainsAny(journal.Diagnostic, "\r\n") || len([]rune(journal.Diagnostic)) > 512) {
		return hostJournal{}, fmt.Errorf("upgrade journal diagnostic is invalid")
	}
	return journal, nil
}

func readHostReceipt(path string) (hostReceipt, error) {
	data, err := readPrivateJSON(path)
	if err != nil {
		return hostReceipt{}, err
	}
	var receipt hostReceipt
	if err := decodeStrictJSON(data, &receipt); err != nil {
		return hostReceipt{}, fmt.Errorf("invalid upgrade receipt: %w", err)
	}
	if receipt.SchemaVersion != hostJournalSchema {
		return hostReceipt{}, fmt.Errorf("unsupported upgrade receipt schema version %d", receipt.SchemaVersion)
	}
	if _, err := uuid.Parse(receipt.OperationID); err != nil || uuid.MustParse(receipt.OperationID).String() != receipt.OperationID {
		return hostReceipt{}, fmt.Errorf("upgrade receipt operationId is invalid")
	}
	if !isSHA256Digest(receipt.ManifestDigest) || receipt.CompletedAt.IsZero() {
		return hostReceipt{}, fmt.Errorf("upgrade receipt identity or timestamp is invalid")
	}
	allowed := map[string]bool{"server": true, "frontend": true, "nginx": true}
	if len(receipt.Services) == 0 {
		return hostReceipt{}, fmt.Errorf("upgrade receipt services are required")
	}
	seen := map[string]bool{}
	for _, service := range receipt.Services {
		if !allowed[service] || seen[service] {
			return hostReceipt{}, fmt.Errorf("upgrade receipt service is invalid")
		}
		seen[service] = true
	}
	for service, digest := range receipt.ObservedImages {
		if !seen[service] || !isSHA256Digest(digest) {
			return hostReceipt{}, fmt.Errorf("upgrade receipt observed image is invalid")
		}
	}
	for service := range seen {
		if _, ok := receipt.ObservedImages[service]; !ok {
			return hostReceipt{}, fmt.Errorf("upgrade receipt is missing observed image")
		}
	}
	return receipt, nil
}

func readPrivateJSON(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("private upgrade file has invalid mode or type")
	}
	return os.ReadFile(path)
}

func validateJournalDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return fmt.Errorf("private upgrade directory must be a 0700 directory")
	}
	for _, child := range []string{filepath.Join(path, hostJournalHistoryDir), filepath.Join(path, hostJournalReceiptDir)} {
		childInfo, childErr := os.Lstat(child)
		if childErr != nil {
			return childErr
		}
		if childInfo.Mode()&os.ModeSymlink != 0 || !childInfo.IsDir() || childInfo.Mode().Perm() != 0o700 {
			return fmt.Errorf("private upgrade child directory must be a 0700 directory")
		}
	}
	return nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func isSHA256Digest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, char := range value[len("sha256:"):] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func validMigrationStatus(value string) bool {
	switch value {
	case string(domain.MigrationStatusNotStarted), string(domain.MigrationStatusRunning), string(domain.MigrationStatusSucceeded), string(domain.MigrationStatusFailed), string(domain.MigrationStatusUnknown):
		return true
	default:
		return false
	}
}

func validHostStage(value string) bool {
	switch value {
	case string(domain.StatusQueued), string(domain.StatusStopping), string(domain.StatusPreflight), string(domain.StatusUpdating), string(domain.StatusMigrating), string(domain.StatusRestarting), string(domain.StatusAgentVerifying), string(domain.StatusVerifying), string(domain.StatusSucceeded), string(domain.StatusFailed), string(domain.StatusNeedsRecovery), string(domain.StatusNeedsAttention):
		return true
	default:
		return false
	}
}
