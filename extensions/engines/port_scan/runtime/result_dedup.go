package portscanruntime

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

// resultDedupOptions configures workspace-local result staging.
type resultDedupOptions struct {
	WorkspaceDir    string
	ChunkByteBudget int64
	MergeFanIn      int
}

type resultDedupRecord struct {
	Key     string
	Payload []byte
}

type resultDedupStats struct {
	InputCount     int
	UniqueCount    int
	DuplicateCount int
}

// resultDedupStager keeps port-scan result records in byte-bounded chunks
// before they are globally deduplicated for typed submission.
type resultDedupStager struct {
	tempDir         string
	chunkByteBudget int64
	mergeFanIn      int

	chunk       []resultDedupStoredRecord
	chunkBytes  int64
	chunkCount  int
	nextOrdinal uint64
	inputCount  int

	streamed bool
	closed   bool
	closeErr error

	// Kept on the instance so focused tests can verify bounded multi-pass
	// reduction without exposing diagnostics to Engine callers.
	mergePasses         int
	maxOpenChunkReaders int
	maxLoadedPayloads   int
}

type resultDedupStoredRecord struct {
	key     string
	payload []byte
	ordinal uint64
}

func newResultDedupStager(options resultDedupOptions) (*resultDedupStager, error) {
	if options.WorkspaceDir == "" {
		return nil, errors.New("result deduplication workspace is required")
	}

	chunkByteBudget := options.ChunkByteBudget
	if chunkByteBudget == 0 {
		chunkByteBudget = resultDedupDefaultChunkByteBudget
	}
	if chunkByteBudget < 0 {
		return nil, errors.New("result deduplication chunk byte budget must be > 0")
	}

	mergeFanIn := options.MergeFanIn
	if mergeFanIn == 0 {
		mergeFanIn = resultDedupDefaultMergeFanIn
	}
	if mergeFanIn < 2 {
		return nil, errors.New("result deduplication merge fan-in must be >= 2")
	}

	tempDir, err := os.MkdirTemp(options.WorkspaceDir, ".result-dedup-*")
	if err != nil {
		return nil, fmt.Errorf("create result deduplication temp directory: %w", err)
	}

	return &resultDedupStager{
		tempDir:         tempDir,
		chunkByteBudget: chunkByteBudget,
		mergeFanIn:      mergeFanIn,
	}, nil
}

// Add stages one canonical record. The source ordinal is assigned in Add call
// order so Stream can deterministically retain the latest valid record for a key.
func (s *resultDedupStager) Add(ctx context.Context, canonicalKey string, payload []byte) error {
	if err := s.ensureActive(ctx); err != nil {
		return err
	}
	if canonicalKey == "" {
		return s.abort(errors.New("result deduplication canonical key is required"))
	}
	if s.nextOrdinal == math.MaxUint64 {
		return s.abort(errors.New("result deduplication source ordinal overflow"))
	}

	storedPayload := bytes.Clone(payload)
	recordBytes := resultDedupEstimateRecordBytes(canonicalKey, storedPayload)
	if len(s.chunk) > 0 && recordBytes > s.chunkByteBudget-s.chunkBytes {
		if err := s.flushChunk(ctx); err != nil {
			return s.abort(err)
		}
	}

	s.chunk = append(s.chunk, resultDedupStoredRecord{
		key:     canonicalKey,
		payload: storedPayload,
		ordinal: s.nextOrdinal,
	})
	s.chunkBytes += recordBytes
	s.nextOrdinal++
	s.inputCount++
	return nil
}

// Stream globally deduplicates all staged records and calls emit once per key
// in ascending canonical-key order. It automatically removes all temporary
// files before returning, including after cancellation or an emit failure.
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
	stats.InputCount = s.inputCount

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

	var finalPaths []string
	finalPaths, err = s.reduceChunks(ctx)
	if err != nil {
		return stats, err
	}

	var winner resultDedupStoredRecord
	hasWinner := false
	err = resultDedupMergeRecords(ctx, finalPaths, func(record resultDedupStoredRecord) error {
		if !hasWinner {
			winner = record
			hasWinner = true
			return nil
		}
		if record.key == winner.key {
			stats.DuplicateCount++
			winner = record
			return nil
		}
		stats.UniqueCount++
		if err := emit(resultDedupRecord{Key: winner.key, Payload: winner.payload}); err != nil {
			return err
		}
		winner = record
		return nil
	}, s.observeOpenChunkReaders, s.observeLoadedPayloads)
	if err != nil {
		return stats, err
	}
	if hasWinner {
		stats.UniqueCount++
		if err := emit(resultDedupRecord{Key: winner.key, Payload: winner.payload}); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

// Close removes the private temporary directory. It is safe to call more than
// once, including after Stream has already performed automatic cleanup; a
// transient removal failure may be retried after the underlying condition heals.
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
	if err := resultDedupWriteChunk(ctx, chunkPath, s.chunk); err != nil {
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

			if err := resultDedupMergeRecordsToFile(ctx, group, mergedPath, s.observeOpenChunkReaders, s.observeLoadedPayloads); err != nil {
				return nil, err
			}
			if err := resultDedupRemoveFiles(group); err != nil {
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

func resultDedupEstimateRecordBytes(key string, payload []byte) int64 {
	return int64(len(key)) + int64(len(payload)) + resultDedupRecordOverhead
}

func resultDedupWriteChunk(ctx context.Context, path string, records []resultDedupStoredRecord) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
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
		if err := resultDedupWriteStoredRecord(writer, record); err != nil {
			return fmt.Errorf("write result deduplication chunk: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush result deduplication chunk: %w", err)
	}
	return nil
}

func resultDedupRemoveFiles(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove result deduplication merge input: %w", err)
		}
	}
	return nil
}

func resultDedupMergeRecordsToFile(ctx context.Context, paths []string, outputPath string, observeOpenReaders func(int), observeLoadedPayloads func(int)) (err error) {
	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
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
	err = resultDedupMergeRecords(ctx, paths, func(record resultDedupStoredRecord) error {
		if err := resultDedupWriteStoredRecord(writer, record); err != nil {
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

func resultDedupMergeRecords(ctx context.Context, paths []string, emit func(resultDedupStoredRecord) error, observeOpenReaders func(int), observeLoadedPayloads func(int)) (err error) {
	if len(paths) == 0 {
		return nil
	}
	readers, err := resultDedupOpenChunkReaders(paths)
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
		if err := resultDedupEmitStoredRecord(&readers[item.readerIndex], item.header, emit, observeLoadedPayloads); err != nil {
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

func resultDedupEmitStoredRecord(reader *resultDedupChunkReader, header resultDedupStoredRecordHeader, emit func(resultDedupStoredRecord) error, observeLoadedPayloads func(int)) error {
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

func resultDedupOpenChunkReaders(paths []string) ([]resultDedupChunkReader, error) {
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

func (r *resultDedupChunkReader) nextHeader() (resultDedupStoredRecordHeader, bool, error) {
	header, err := resultDedupReadStoredRecordHeader(r.reader)
	if errors.Is(err, io.EOF) {
		return resultDedupStoredRecordHeader{}, false, nil
	}
	if err != nil {
		return resultDedupStoredRecordHeader{}, false, fmt.Errorf("read result deduplication merge input: %w", err)
	}
	return header, true, nil
}

func (r *resultDedupChunkReader) readRecord(header resultDedupStoredRecordHeader) (resultDedupStoredRecord, error) {
	payload := make([]byte, header.payloadLength)
	if _, err := io.ReadFull(r.reader, payload); err != nil {
		return resultDedupStoredRecord{}, fmt.Errorf("read result deduplication merge input: %w", err)
	}
	return resultDedupStoredRecord{key: header.key, payload: payload, ordinal: header.ordinal}, nil
}

type resultDedupHeapRecord struct {
	header      resultDedupStoredRecordHeader
	readerIndex int
}

type resultDedupStoredRecordHeader struct {
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

func resultDedupWriteStoredRecord(writer io.Writer, record resultDedupStoredRecord) error {
	if err := resultDedupWriteUint64(writer, uint64(len(record.key))); err != nil {
		return err
	}
	if err := resultDedupWriteUint64(writer, record.ordinal); err != nil {
		return err
	}
	if err := resultDedupWriteUint64(writer, uint64(len(record.payload))); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, record.key); err != nil {
		return err
	}
	_, err := writer.Write(record.payload)
	return err
}

func resultDedupReadStoredRecordHeader(reader io.Reader) (resultDedupStoredRecordHeader, error) {
	keyLength, err := resultDedupReadUint64(reader)
	if err != nil {
		return resultDedupStoredRecordHeader{}, err
	}
	ordinal, err := resultDedupReadUint64(reader)
	if err != nil {
		return resultDedupStoredRecordHeader{}, err
	}
	payloadLength, err := resultDedupReadUint64(reader)
	if err != nil {
		return resultDedupStoredRecordHeader{}, err
	}
	if keyLength > uint64(resultDedupMaxInt()) || payloadLength > uint64(resultDedupMaxInt()) {
		return resultDedupStoredRecordHeader{}, errors.New("result deduplication record exceeds platform allocation limit")
	}

	key := make([]byte, int(keyLength))
	if _, err := io.ReadFull(reader, key); err != nil {
		return resultDedupStoredRecordHeader{}, err
	}
	return resultDedupStoredRecordHeader{key: string(key), ordinal: ordinal, payloadLength: int(payloadLength)}, nil
}

func resultDedupWriteUint64(writer io.Writer, value uint64) error {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, err := writer.Write(encoded[:])
	return err
}

func resultDedupReadUint64(reader io.Reader) (uint64, error) {
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

// isResultDedupContextTermination reports whether err contains only expected
// context cancellation causes. Joined errors containing another failure return false.
func isResultDedupContextTermination(err error) bool {
	if err == nil {
		return false
	}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		causes := multi.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !isResultDedupContextTermination(cause) {
				return false
			}
		}
		return true
	}
	if single, ok := err.(interface{ Unwrap() error }); ok {
		if cause := single.Unwrap(); cause != nil {
			return isResultDedupContextTermination(cause)
		}
	}
	return err == context.Canceled || err == context.DeadlineExceeded
}

// resultDedupSubmissionError retains operational streaming and cleanup failures
// alongside the typed submission error. Context termination is expected after
// cancellation and must not obscure the failed submission.
func resultDedupSubmissionError(submissionErr, streamErr, closeErr error) error {
	errorsToReturn := []error{submissionErr}
	if closeErr != nil {
		errorsToReturn = append(errorsToReturn, fmt.Errorf("clean up host-port result deduplication staging: %w", closeErr))
	}
	if streamErr != nil && !isResultDedupContextTermination(streamErr) {
		errorsToReturn = append(errorsToReturn, fmt.Errorf("deduplicate host-port results after typed submission failure: %w", streamErr))
	}
	if len(errorsToReturn) == 1 {
		return submissionErr
	}
	return errors.Join(errorsToReturn...)
}

func resultDedupMaxInt() int {
	return int(^uint(0) >> 1)
}
