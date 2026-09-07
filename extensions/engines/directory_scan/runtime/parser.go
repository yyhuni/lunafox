package directoryscanruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
	"io"
	"os"
)

const maximumFFUFPhysicalRecordBytes = 4 * 1024 * 1024

type DirectoryObservation struct {
	Item                  enginecontract.Directory
	CandidateOrdinal      uint64
	PhysicalRecordOrdinal uint64
}

type FFUFParseSummary struct {
	SourceRecords    uint64
	ParsedItems      uint64
	MalformedRecords uint64
	InvalidRecords   uint64
	OversizedRecords uint64
}

func (summary FFUFParseSummary) AllRecordsRejected() bool {
	return summary.SourceRecords > 0 && summary.ParsedItems == 0
}

func (summary *FFUFParseSummary) add(other FFUFParseSummary) {
	summary.SourceRecords += other.SourceRecords
	summary.ParsedItems += other.ParsedItems
	summary.MalformedRecords += other.MalformedRecords
	summary.InvalidRecords += other.InvalidRecords
	summary.OversizedRecords += other.OversizedRecords
}

func ParseFFUFArtifact(
	ctx context.Context,
	path string,
	candidateOrdinal uint64,
	visit func(DirectoryObservation) error,
) (summary FFUFParseSummary, err error) {
	if ctx == nil {
		return summary, errors.New("FFUF parser context is required")
	}
	if path == "" {
		return summary, errors.New("FFUF raw artifact path is required")
	}
	if visit == nil {
		return summary, errors.New("Directory observation callback is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return summary, fmt.Errorf("open FFUF raw artifact: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close FFUF raw artifact: %w", closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return summary, fmt.Errorf("stat FFUF raw artifact: %w", err)
	}
	if !info.Mode().IsRegular() {
		return summary, errors.New("FFUF raw artifact must be a regular file")
	}
	return parseFFUFStream(ctx, file, candidateOrdinal, visit)
}

func parseFFUFStream(
	ctx context.Context,
	stream io.Reader,
	candidateOrdinal uint64,
	visit func(DirectoryObservation) error,
) (FFUFParseSummary, error) {
	if stream == nil {
		return FFUFParseSummary{}, errors.New("FFUF raw artifact reader is required")
	}
	reader := bufio.NewReaderSize(stream, 64*1024)
	var summary FFUFParseSummary
	for {
		record, oversized, err := readFFUFPhysicalRecord(ctx, reader)
		if errors.Is(err, io.EOF) {
			return summary, nil
		}
		if err != nil {
			return summary, fmt.Errorf("read FFUF raw artifact: %w", err)
		}
		physicalOrdinal := summary.SourceRecords
		summary.SourceRecords++
		if oversized {
			summary.OversizedRecords++
			continue
		}
		item, outcome := decodeFFUFRecord(record)
		switch outcome {
		case ffufRecordMalformed:
			summary.MalformedRecords++
			continue
		case ffufRecordInvalid:
			summary.InvalidRecords++
			continue
		case ffufRecordValid:
		default:
			return summary, errors.New("unknown FFUF record outcome")
		}
		if err := visit(DirectoryObservation{
			Item:                  item,
			CandidateOrdinal:      candidateOrdinal,
			PhysicalRecordOrdinal: physicalOrdinal,
		}); err != nil {
			return summary, err
		}
		summary.ParsedItems++
	}
}

func readFFUFPhysicalRecord(ctx context.Context, reader *bufio.Reader) ([]byte, bool, error) {
	record := make([]byte, 0, min(maximumFFUFPhysicalRecordBytes, 64*1024))
	oversized := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		fragment, isPrefix, err := reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) && oversized {
				return nil, true, nil
			}
			return nil, false, err
		}
		if !oversized {
			if len(fragment) > maximumFFUFPhysicalRecordBytes-len(record) {
				oversized = true
				record = nil
			} else {
				record = append(record, fragment...)
			}
		}
		if !isPrefix {
			if oversized {
				return nil, true, nil
			}
			return record, false, nil
		}
	}
}

type ffufRecordOutcome uint8

const (
	ffufRecordMalformed ffufRecordOutcome = iota
	ffufRecordInvalid
	ffufRecordValid
)

func decodeFFUFRecord(record []byte) (enginecontract.Directory, ffufRecordOutcome) {
	// encoding/json accepts unpaired surrogate escapes by substituting U+FFFD.
	// Validate the raw JSON text first so the URL contract can reject that
	// lossy transport value instead of turning it into a distinct literal URL.
	if err := validateJSONText(record); err != nil {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	if !json.Valid(record) {
		return enginecontract.Directory{}, ffufRecordMalformed
	}
	fields, ok := decodeFFUFObjectFields(record)
	if !ok {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	url, ok := decodeRequiredString(fields, "url")
	if !ok {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	status, ok := decodeRequiredInt64(fields, "status")
	if !ok {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	contentLength, ok := decodeRequiredInt64(fields, "length")
	if !ok {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	contentType, ok := decodeRequiredString(fields, "content-type")
	if !ok {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	duration, ok := decodeRequiredInt64(fields, "duration")
	if !ok {
		return enginecontract.Directory{}, ffufRecordInvalid
	}
	item := enginecontract.Directory{
		URL:           url,
		Status:        int(status),
		ContentLength: contentLength,
		ContentType:   contentType,
		Duration:      duration,
	}
	return item, ffufRecordValid
}

func decodeFFUFObjectFields(record []byte) (map[string]json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(record))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, false
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return nil, false
		}
		field, ok := fieldToken.(string)
		if !ok {
			return nil, false
		}
		if _, duplicate := fields[field]; duplicate {
			return nil, false
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, false
		}
		fields[field] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, false
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	return fields, true
}

func decodeRequiredString(fields map[string]json.RawMessage, field string) (string, bool) {
	raw, ok := fields[field]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func decodeRequiredInt64(fields map[string]json.RawMessage, field string) (int64, bool) {
	raw, ok := fields[field]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return 0, false
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	return value, true
}
