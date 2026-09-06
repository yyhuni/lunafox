package websitediscoveryruntime

import (
	"bufio"
	"bytes"
	"container/heap"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
)

const (
	resultDedupDefaultChunkByteBudget int64 = 8 * 1024 * 1024
	resultDedupDefaultMergeFanIn            = 8
	resultDedupRecordOverhead         int64 = 64
)

// resultDedupOptions configures workspace-local external result staging.
// Zero chunk and fan-in values select the fixed runtime defaults.
type resultDedupOptions struct {
	workspaceDir    string
	chunkByteBudget int64
	mergeFanIn      int
}

// resultDedupRecord is a raw observed website record emitted after global
// exact-URL deduplication.
type resultDedupRecord struct {
	key     string
	payload []byte
}

// resultDedupStats describes records accepted by Add and emitted by Stream.
type resultDedupStats struct {
	inputCount     int
	uniqueCount    int
	duplicateCount int
}

// resultDedupStager sorts raw observed URL records in workspace-local chunks. It keeps
// only one chunk in memory and opens no more than mergeFanIn source files per merge.
type resultDedupStager struct {
	tempDir         string
	chunkByteBudget int64
	mergeFanIn      int

	chunk       []storedResultRecord
	chunkBytes  int64
	chunkCount  int
	nextOrdinal uint64
	inputCount  int

	streamed bool
	closed   bool
	closeErr error

	// These are retained for focused tests without exposing diagnostics through
	// the Engine result-reporting surface.
	mergePasses         int
	maxOpenChunkReaders int
	maxLoadedPayloads   int
}

type storedResultRecord struct {
	key     string
	payload []byte
	ordinal uint64
}

func newResultDedupStager(options resultDedupOptions) (*resultDedupStager, error) {
	if options.workspaceDir == "" {
		return nil, errors.New("result deduplication workspace is required")
	}

	chunkByteBudget := options.chunkByteBudget
	if chunkByteBudget == 0 {
		chunkByteBudget = resultDedupDefaultChunkByteBudget
	}
	if chunkByteBudget < 0 {
		return nil, errors.New("result deduplication chunk byte budget must be > 0")
	}

	mergeFanIn := options.mergeFanIn
	if mergeFanIn == 0 {
		mergeFanIn = resultDedupDefaultMergeFanIn
	}
	if mergeFanIn < 2 {
		return nil, errors.New("result deduplication merge fan-in must be >= 2")
	}

	tempDir, err := os.MkdirTemp(options.workspaceDir, ".result-dedup-*")
	if err != nil {
		return nil, fmt.Errorf("create result deduplication temp directory: %w", err)
	}

	return &resultDedupStager{
		tempDir:         tempDir,
		chunkByteBudget: chunkByteBudget,
		mergeFanIn:      mergeFanIn,
	}, nil
}

// Add stages one already-admitted record. Source order determines the
// deterministic duplicate winner when records share the same exact URL key.
func (s *resultDedupStager) Add(ctx context.Context, exactURLKey string, payload []byte) error {
	if err := s.ensureActive(ctx); err != nil {
		return err
	}
	if exactURLKey == "" {
		return s.abort(errors.New("result deduplication exact URL key is required"))
	}
	if s.nextOrdinal == math.MaxUint64 {
		return s.abort(errors.New("result deduplication source ordinal overflow"))
	}

	storedPayload := bytes.Clone(payload)
	recordBytes := estimateResultDedupRecordBytes(exactURLKey, storedPayload)
	if len(s.chunk) > 0 && recordBytes > s.chunkByteBudget-s.chunkBytes {
		if err := s.flushChunk(ctx); err != nil {
			return s.abort(err)
		}
	}

	s.chunk = append(s.chunk, storedResultRecord{
		key:     exactURLKey,
		payload: storedPayload,
		ordinal: s.nextOrdinal,
	})
	s.chunkBytes += recordBytes
	s.nextOrdinal++
	s.inputCount++
	return nil
}

// Stream emits one record for each exact URL key in ascending byte order. It
// removes its private temporary directory after success, cancellation, or error.
func (s *resultDedupStager) Stream(ctx context.Context, emit func(resultDedupRecord) error) (stats resultDedupStats, err error) {
	if emit == nil {
		return stats, s.abort(errors.New("result deduplication emit callback is required"))
	}
	if err := s.ensureActive(ctx); err != nil {
		return stats, err
	}
	if s.streamed {
		return stats, s.abort(errors.New("result deduplication records have already been streamed"))
	}
	s.streamed = true
	stats.inputCount = s.inputCount

	defer func() {
		if closeErr := s.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			} else {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	if err = s.flushChunk(ctx); err != nil {
		return stats, err
	}

	finalPaths, err := s.reduceChunks(ctx)
	if err != nil {
		return stats, err
	}

	var winner storedResultRecord
	hasWinner := false
	err = mergeResultDedupRecords(ctx, finalPaths, func(record storedResultRecord) error {
		if !hasWinner {
			winner = record
			hasWinner = true
			return nil
		}
		if record.key == winner.key {
			stats.duplicateCount++
			winner = record
			return nil
		}
		stats.uniqueCount++
		if err := emit(resultDedupRecord{key: winner.key, payload: winner.payload}); err != nil {
			return err
		}
		winner = record
		return nil
	}, s.observeOpenChunkReaders, s.observeLoadedPayloads)
	if err != nil {
		return stats, err
	}
	if hasWinner {
		stats.uniqueCount++
		if err := emit(resultDedupRecord{key: winner.key, payload: winner.payload}); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

// Close removes the private staging directory. A failed removal may be retried
// after the underlying filesystem condition is repaired.
func (s *resultDedupStager) Close() error {
	if s == nil {
		return nil
	}
	if s.closed && s.closeErr == nil {
		return nil
	}
	s.closed = true
	s.chunk = nil
	if err := os.RemoveAll(s.tempDir); err != nil {
		s.closeErr = fmt.Errorf("remove result deduplication temp directory: %w", err)
		return s.closeErr
	}
	s.closeErr = nil
	return nil
}

func (s *resultDedupStager) ensureActive(ctx context.Context) error {
	if s == nil {
		return errors.New("result deduplication stager is required")
	}
	if s.closed {
		return errors.New("result deduplication stager is closed")
	}
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return s.abort(err)
	}
	return nil
}

func (s *resultDedupStager) abort(err error) error {
	if err == nil {
		return nil
	}
	if closeErr := s.Close(); closeErr != nil {
		return errors.Join(err, closeErr)
	}
	return err
}

func (s *resultDedupStager) flushChunk(ctx context.Context) error {
	if len(s.chunk) == 0 {
		return nil
	}
	if err := resultDedupContextError(ctx); err != nil {
		return err
	}

	sort.Slice(s.chunk, func(i, j int) bool {
		if s.chunk[i].key == s.chunk[j].key {
			return s.chunk[i].ordinal < s.chunk[j].ordinal
		}
		return s.chunk[i].key < s.chunk[j].key
	})
	if err := resultDedupContextError(ctx); err != nil {
		return err
	}

	chunkNumber := s.chunkCount + 1
	chunkPath := s.runPath(0, chunkNumber)
	if err := writeResultDedupChunk(ctx, chunkPath, s.chunk); err != nil {
		return err
	}
	s.chunkCount = chunkNumber
	s.chunk = nil
	s.chunkBytes = 0
	return nil
}

func (s *resultDedupStager) reduceChunks(ctx context.Context) ([]string, error) {
	round := 0
	runCount := s.chunkCount
	for runCount > s.mergeFanIn {
		if err := resultDedupContextError(ctx); err != nil {
			return nil, err
		}

		nextCount := (runCount + s.mergeFanIn - 1) / s.mergeFanIn
		for outputIndex := 1; outputIndex <= nextCount; outputIndex++ {
			if err := resultDedupContextError(ctx); err != nil {
				return nil, err
			}
			start := (outputIndex-1)*s.mergeFanIn + 1
			end := min(start+s.mergeFanIn-1, runCount)
			group := s.runPaths(round, start, end)
			mergedPath := s.runPath(round+1, outputIndex)
			if len(group) == 1 {
				if err := os.Rename(group[0], mergedPath); err != nil {
					return nil, fmt.Errorf("rename result deduplication merge input: %w", err)
				}
				continue
			}

			if err := mergeResultDedupRecordsToFile(ctx, group, mergedPath, s.observeOpenChunkReaders, s.observeLoadedPayloads); err != nil {
				return nil, err
			}
			if err := removeResultDedupFiles(group); err != nil {
				return nil, err
			}
		}
		runCount = nextCount
		round++
		s.mergePasses++
	}
	return s.runPaths(round, 1, runCount), nil
}

func (s *resultDedupStager) runPaths(round, start, end int) []string {
	if end < start {
		return nil
	}
	paths := make([]string, 0, end-start+1)
	for index := start; index <= end; index++ {
		paths = append(paths, s.runPath(round, index))
	}
	return paths
}

func (s *resultDedupStager) runPath(round, index int) string {
	if round == 0 {
		return filepath.Join(s.tempDir, fmt.Sprintf("chunk-%09d.bin", index))
	}
	return filepath.Join(s.tempDir, fmt.Sprintf("merge-%03d-%09d.bin", round, index))
}

func (s *resultDedupStager) observeOpenChunkReaders(openReaders int) {
	if openReaders > s.maxOpenChunkReaders {
		s.maxOpenChunkReaders = openReaders
	}
}

func (s *resultDedupStager) observeLoadedPayloads(payloads int) {
	if payloads > s.maxLoadedPayloads {
		s.maxLoadedPayloads = payloads
	}
}

func estimateResultDedupRecordBytes(key string, payload []byte) int64 {
	return int64(len(key)) + int64(len(payload)) + resultDedupRecordOverhead
}

func writeResultDedupChunk(ctx context.Context, path string, records []storedResultRecord) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create result deduplication chunk: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close result deduplication chunk: %w", closeErr)
			if err == nil {
				err = closeErr
			} else {
				err = errors.Join(err, closeErr)
			}
		}
		if err != nil {
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove result deduplication chunk: %w", removeErr))
			}
		}
	}()

	writer := bufio.NewWriter(file)
	for _, record := range records {
		if err := resultDedupContextError(ctx); err != nil {
			return err
		}
		if err := writeStoredResultRecord(writer, record); err != nil {
			return fmt.Errorf("write result deduplication chunk: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush result deduplication chunk: %w", err)
	}
	return nil
}

func removeResultDedupFiles(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove result deduplication merge input: %w", err)
		}
	}
	return nil
}

func mergeResultDedupRecordsToFile(ctx context.Context, paths []string, outputPath string, observeOpenReaders func(int), observeLoadedPayloads func(int)) (err error) {
	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create result deduplication merge file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close result deduplication merge file: %w", closeErr)
			if err == nil {
				err = closeErr
			} else {
				err = errors.Join(err, closeErr)
			}
		}
		if err != nil {
			if removeErr := os.Remove(outputPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove result deduplication merge file: %w", removeErr))
			}
		}
	}()

	writer := bufio.NewWriter(file)
	err = mergeResultDedupRecords(ctx, paths, func(record storedResultRecord) error {
		if err := writeStoredResultRecord(writer, record); err != nil {
			return fmt.Errorf("write result deduplication merge file: %w", err)
		}
		return nil
	}, observeOpenReaders, observeLoadedPayloads)
	if err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush result deduplication merge file: %w", err)
	}
	return nil
}

func mergeResultDedupRecords(ctx context.Context, paths []string, emit func(storedResultRecord) error, observeOpenReaders func(int), observeLoadedPayloads func(int)) (err error) {
	if len(paths) == 0 {
		return nil
	}
	readers, err := openResultDedupChunkReaders(paths)
	if err != nil {
		return err
	}
	defer func() {
		for _, reader := range readers {
			if closeErr := reader.file.Close(); closeErr != nil {
				closeErr = fmt.Errorf("close result deduplication merge input: %w", closeErr)
				if err == nil {
					err = closeErr
				} else {
					err = errors.Join(err, closeErr)
				}
			}
		}
	}()
	if observeOpenReaders != nil {
		observeOpenReaders(len(readers))
	}

	records := &resultDedupRecordHeap{}
	heap.Init(records)
	for index := range readers {
		if err := resultDedupContextError(ctx); err != nil {
			return err
		}
		header, ok, err := readers[index].nextHeader()
		if err != nil {
			return err
		}
		if ok {
			heap.Push(records, resultDedupHeapRecord{header: header, readerIndex: index})
		}
	}

	for records.Len() > 0 {
		if err := resultDedupContextError(ctx); err != nil {
			return err
		}
		item := heap.Pop(records).(resultDedupHeapRecord)
		if err := emitStoredResultRecord(&readers[item.readerIndex], item.header, emit, observeLoadedPayloads); err != nil {
			return err
		}
		if err := resultDedupContextError(ctx); err != nil {
			return err
		}
		header, ok, err := readers[item.readerIndex].nextHeader()
		if err != nil {
			return err
		}
		if ok {
			heap.Push(records, resultDedupHeapRecord{header: header, readerIndex: item.readerIndex})
		}
	}
	return nil
}

func emitStoredResultRecord(reader *resultDedupChunkReader, header storedResultRecordHeader, emit func(storedResultRecord) error, observeLoadedPayloads func(int)) error {
	if observeLoadedPayloads != nil {
		observeLoadedPayloads(1)
	}
	record, err := reader.readRecord(header)
	if err == nil {
		err = emit(record)
	}
	if observeLoadedPayloads != nil {
		observeLoadedPayloads(0)
	}
	return err
}

type resultDedupChunkReader struct {
	file   *os.File
	reader *bufio.Reader
}

func openResultDedupChunkReaders(paths []string) ([]resultDedupChunkReader, error) {
	readers := make([]resultDedupChunkReader, 0, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			for _, reader := range readers {
				_ = reader.file.Close()
			}
			return nil, fmt.Errorf("open result deduplication merge input: %w", err)
		}
		readers = append(readers, resultDedupChunkReader{file: file, reader: bufio.NewReader(file)})
	}
	return readers, nil
}

func (r *resultDedupChunkReader) nextHeader() (storedResultRecordHeader, bool, error) {
	header, err := readStoredResultRecordHeader(r.reader)
	if errors.Is(err, io.EOF) {
		return storedResultRecordHeader{}, false, nil
	}
	if err != nil {
		return storedResultRecordHeader{}, false, fmt.Errorf("read result deduplication merge input: %w", err)
	}
	return header, true, nil
}

func (r *resultDedupChunkReader) readRecord(header storedResultRecordHeader) (storedResultRecord, error) {
	payload := make([]byte, header.payloadLength)
	if _, err := io.ReadFull(r.reader, payload); err != nil {
		return storedResultRecord{}, fmt.Errorf("read result deduplication merge input: %w", err)
	}
	return storedResultRecord{key: header.key, payload: payload, ordinal: header.ordinal}, nil
}

type resultDedupHeapRecord struct {
	header      storedResultRecordHeader
	readerIndex int
}

type storedResultRecordHeader struct {
	key           string
	ordinal       uint64
	payloadLength int
}

type resultDedupRecordHeap []resultDedupHeapRecord

func (h resultDedupRecordHeap) Len() int { return len(h) }

func (h resultDedupRecordHeap) Less(i, j int) bool {
	left, right := h[i], h[j]
	if left.header.key != right.header.key {
		return left.header.key < right.header.key
	}
	if left.header.ordinal != right.header.ordinal {
		return left.header.ordinal < right.header.ordinal
	}
	return left.readerIndex < right.readerIndex
}

func (h resultDedupRecordHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *resultDedupRecordHeap) Push(value any) {
	*h = append(*h, value.(resultDedupHeapRecord))
}

func (h *resultDedupRecordHeap) Pop() any {
	old := *h
	last := len(old) - 1
	item := old[last]
	*h = old[:last]
	return item
}

func writeStoredResultRecord(writer io.Writer, record storedResultRecord) error {
	if err := writeResultDedupUint64(writer, uint64(len(record.key))); err != nil {
		return err
	}
	if err := writeResultDedupUint64(writer, record.ordinal); err != nil {
		return err
	}
	if err := writeResultDedupUint64(writer, uint64(len(record.payload))); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, record.key); err != nil {
		return err
	}
	_, err := writer.Write(record.payload)
	return err
}

func readStoredResultRecordHeader(reader io.Reader) (storedResultRecordHeader, error) {
	keyLength, err := readResultDedupUint64(reader)
	if err != nil {
		return storedResultRecordHeader{}, err
	}
	ordinal, err := readResultDedupUint64(reader)
	if err != nil {
		return storedResultRecordHeader{}, err
	}
	payloadLength, err := readResultDedupUint64(reader)
	if err != nil {
		return storedResultRecordHeader{}, err
	}
	if keyLength > uint64(resultDedupMaxInt()) || payloadLength > uint64(resultDedupMaxInt()) {
		return storedResultRecordHeader{}, errors.New("result deduplication record exceeds platform allocation limit")
	}

	key := make([]byte, int(keyLength))
	if _, err := io.ReadFull(reader, key); err != nil {
		return storedResultRecordHeader{}, err
	}
	return storedResultRecordHeader{key: string(key), ordinal: ordinal, payloadLength: int(payloadLength)}, nil
}

func writeResultDedupUint64(writer io.Writer, value uint64) error {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, err := writer.Write(encoded[:])
	return err
}

func readResultDedupUint64(reader io.Reader) (uint64, error) {
	var encoded [8]byte
	_, err := io.ReadFull(reader, encoded[:])
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(encoded[:]), nil
}

func resultDedupContextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func isContextTermination(err error) bool {
	if err == nil {
		return false
	}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		causes := multi.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !isContextTermination(cause) {
				return false
			}
		}
		return true
	}
	if single, ok := err.(interface{ Unwrap() error }); ok {
		if cause := single.Unwrap(); cause != nil {
			return isContextTermination(cause)
		}
	}
	return err == context.Canceled || err == context.DeadlineExceeded
}

func resultDedupMaxInt() int {
	return int(^uint(0) >> 1)
}
