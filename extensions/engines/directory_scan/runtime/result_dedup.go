package directoryscanruntime

import (
	"bufio"
	"container/heap"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

const (
	directoryResultChunkByteBudget   int64 = 8 * 1024 * 1024
	directoryResultMergeFanIn              = 8
	directoryStoredRecordHeaderBytes       = 4 * 8
)

type directoryDedupWriteCloser interface {
	io.Writer
	io.Closer
}

type directoryDedupReadCloser interface {
	io.Reader
	io.Closer
}

type directoryDedupFileOps struct {
	create    func(string) (directoryDedupWriteCloser, error)
	open      func(string) (directoryDedupReadCloser, error)
	remove    func(string) error
	removeAll func(string) error
}

type directoryDedupOptions struct {
	Workspace       string
	ChunkByteBudget int64
	MergeFanIn      int
	FileOps         directoryDedupFileOps
}

type directoryDedupStats struct {
	InputItems       uint64
	WinnerItems      uint64
	SkippedDuplicate uint64
}

type directoryStoredRecord struct {
	key                   string
	payload               []byte
	candidateOrdinal      uint64
	physicalRecordOrdinal uint64
}

type directoryStoredRecordHeader struct {
	key                   string
	payloadLength         int
	candidateOrdinal      uint64
	physicalRecordOrdinal uint64
}

// directoryDedupStager is deliberately Engine-local. It keeps only one
// byte-bounded chunk in memory and names runs by ordinal instead of retaining
// a result-sized path collection.
type directoryDedupStager struct {
	tempDir         string
	chunkByteBudget int64
	mergeFanIn      int
	files           directoryDedupFileOps

	chunk      []directoryStoredRecord
	chunkBytes int64
	runCount   uint64
	inputItems uint64
	finalized  bool
	closed     bool
	closeErr   error

	mergePasses        int
	maxOpenReaders     int
	maxLoadedRecords   int
	maxBufferedRecords int
}

type directoryWinnerSet struct {
	path          string
	expectedItems uint64
	open          func(string) (directoryDedupReadCloser, error)
	streamed      bool
}

func newDirectoryDedupStager(options directoryDedupOptions) (*directoryDedupStager, error) {
	workspace, err := requireDirectoryWorkspace(options.Workspace)
	if err != nil {
		return nil, err
	}
	chunkBudget := options.ChunkByteBudget
	if chunkBudget == 0 {
		chunkBudget = directoryResultChunkByteBudget
	}
	if chunkBudget < 1 {
		return nil, errors.New("Directory result deduplication chunk byte budget must be positive")
	}
	mergeFanIn := options.MergeFanIn
	if mergeFanIn == 0 {
		mergeFanIn = directoryResultMergeFanIn
	}
	if mergeFanIn < 2 {
		return nil, errors.New("Directory result deduplication merge fan-in must be at least 2")
	}
	files := completeDirectoryDedupFileOps(options.FileOps)
	tempDir, err := os.MkdirTemp(workspace, ".directory-result-dedup-*")
	if err != nil {
		return nil, fmt.Errorf("create Directory result deduplication staging: %w", err)
	}
	return &directoryDedupStager{
		tempDir:         tempDir,
		chunkByteBudget: chunkBudget,
		mergeFanIn:      mergeFanIn,
		files:           files,
	}, nil
}

func completeDirectoryDedupFileOps(ops directoryDedupFileOps) directoryDedupFileOps {
	if ops.create == nil {
		ops.create = func(path string) (directoryDedupWriteCloser, error) {
			return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		}
	}
	if ops.open == nil {
		ops.open = func(path string) (directoryDedupReadCloser, error) { return os.Open(path) }
	}
	if ops.remove == nil {
		ops.remove = os.Remove
	}
	if ops.removeAll == nil {
		ops.removeAll = os.RemoveAll
	}
	return ops
}

func (stager *directoryDedupStager) Add(ctx context.Context, observation DirectoryObservation) error {
	if err := stager.ensureMutable(ctx); err != nil {
		return err
	}
	payload, err := json.Marshal(observation.Item)
	if err != nil {
		return fmt.Errorf("encode Directory result for deduplication: %w", err)
	}
	if stager.inputItems == math.MaxUint64 {
		return errors.New("Directory result count overflow")
	}
	record := directoryStoredRecord{
		key:                   strings.Clone(observation.Item.URL),
		payload:               payload,
		candidateOrdinal:      observation.CandidateOrdinal,
		physicalRecordOrdinal: observation.PhysicalRecordOrdinal,
	}
	recordBytes := directoryStoredRecordBytes(record)
	if len(stager.chunk) > 0 && recordBytes > stager.chunkByteBudget-stager.chunkBytes {
		if err := stager.flushChunk(ctx); err != nil {
			return err
		}
	}
	stager.chunk = append(stager.chunk, record)
	stager.chunkBytes += recordBytes
	stager.inputItems++
	if len(stager.chunk) > stager.maxBufferedRecords {
		stager.maxBufferedRecords = len(stager.chunk)
	}
	return nil
}

// Finalize creates and closes one immutable winner file before returning it.
// Submission must not begin until this method succeeds.
func (stager *directoryDedupStager) Finalize(ctx context.Context) (*directoryWinnerSet, directoryDedupStats, error) {
	var stats directoryDedupStats
	if err := stager.ensureMutable(ctx); err != nil {
		return nil, stats, err
	}
	stager.finalized = true
	stats.InputItems = stager.inputItems
	if err := stager.flushChunk(ctx); err != nil {
		return nil, stats, err
	}
	paths, err := stager.reduceRuns(ctx)
	if err != nil {
		return nil, stats, err
	}
	winnerPath := filepath.Join(stager.tempDir, "winners.bin")
	stats, err = stager.writeWinnerFile(ctx, paths, winnerPath)
	if err != nil {
		return nil, stats, err
	}
	if err := stager.removeFiles(paths); err != nil {
		return nil, stats, err
	}
	if stats.InputItems != stats.WinnerItems+stats.SkippedDuplicate {
		return nil, stats, errors.New("Directory result deduplication count invariant failed")
	}
	return &directoryWinnerSet{
		path:          winnerPath,
		expectedItems: stats.WinnerItems,
		open:          stager.files.open,
	}, stats, nil
}

// Close removes only the private Directory deduplication directory. A failed
// removal remains retryable, but no staging operation may resume afterward.
func (stager *directoryDedupStager) Close() error {
	if stager == nil {
		return nil
	}
	if stager.closed && stager.closeErr == nil {
		return nil
	}
	stager.closed = true
	stager.chunk = nil
	stager.chunkBytes = 0
	if err := stager.files.removeAll(stager.tempDir); err != nil {
		stager.closeErr = fmt.Errorf("remove Directory result deduplication staging: %w", err)
		return stager.closeErr
	}
	stager.closeErr = nil
	return nil
}

func (stager *directoryDedupStager) ensureMutable(ctx context.Context) error {
	if stager == nil {
		return errors.New("Directory result deduplication stager is required")
	}
	if ctx == nil {
		return errors.New("Directory result deduplication context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if stager.closed {
		return errors.New("Directory result deduplication stager is closed")
	}
	if stager.finalized {
		return errors.New("Directory result deduplication is already finalized")
	}
	return nil
}

func (stager *directoryDedupStager) flushChunk(ctx context.Context) error {
	if len(stager.chunk) == 0 {
		return nil
	}
	if err := directoryDedupContextError(ctx); err != nil {
		return err
	}
	sort.Slice(stager.chunk, func(left, right int) bool {
		return compareDirectoryStoredRecords(stager.chunk[left], stager.chunk[right]) < 0
	})
	if stager.runCount == math.MaxUint64 {
		return errors.New("Directory result deduplication run count overflow")
	}
	path := stager.runPath(0, stager.runCount+1)
	if err := stager.writeRecordFile(ctx, path, func(writer *bufio.Writer) error {
		for _, record := range stager.chunk {
			if err := directoryDedupContextError(ctx); err != nil {
				return err
			}
			if err := writeDirectoryStoredRecord(writer, record); err != nil {
				return fmt.Errorf("write Directory result deduplication chunk: %w", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	stager.runCount++
	stager.chunk = nil
	stager.chunkBytes = 0
	return nil
}

func (stager *directoryDedupStager) reduceRuns(ctx context.Context) ([]string, error) {
	round := uint64(0)
	runCount := stager.runCount
	for runCount > uint64(stager.mergeFanIn) {
		if err := directoryDedupContextError(ctx); err != nil {
			return nil, err
		}
		fanIn := uint64(stager.mergeFanIn)
		nextCount := runCount / fanIn
		if runCount%fanIn != 0 {
			nextCount++
		}
		for output := uint64(1); output <= nextCount; output++ {
			start := (output-1)*fanIn + 1
			groupSize := min(fanIn, runCount-start+1)
			paths := stager.runPaths(round, start, start+groupSize-1)
			outputPath := stager.runPath(round+1, output)
			if err := stager.mergeRecordsToFile(ctx, paths, outputPath); err != nil {
				return nil, err
			}
			if err := stager.removeFiles(paths); err != nil {
				return nil, err
			}
		}
		round++
		runCount = nextCount
		stager.mergePasses++
	}
	return stager.runPaths(round, 1, runCount), nil
}

func (stager *directoryDedupStager) mergeRecordsToFile(ctx context.Context, paths []string, outputPath string) error {
	return stager.writeRecordFile(ctx, outputPath, func(writer *bufio.Writer) error {
		return stager.mergeRecords(ctx, paths, func(record directoryStoredRecord) error {
			if err := writeDirectoryStoredRecord(writer, record); err != nil {
				return fmt.Errorf("write Directory result deduplication merge: %w", err)
			}
			return nil
		})
	})
}

func (stager *directoryDedupStager) writeWinnerFile(ctx context.Context, paths []string, outputPath string) (stats directoryDedupStats, err error) {
	stats.InputItems = stager.inputItems
	err = stager.writeRecordFile(ctx, outputPath, func(writer *bufio.Writer) error {
		var winner directoryStoredRecord
		hasWinner := false
		writeWinner := func() error {
			if !hasWinner {
				return nil
			}
			if stats.WinnerItems == math.MaxUint64 {
				return errors.New("Directory winner count overflow")
			}
			if err := writeDirectoryStoredRecord(writer, winner); err != nil {
				return fmt.Errorf("write Directory result winner: %w", err)
			}
			stats.WinnerItems++
			return nil
		}
		if err := stager.mergeRecords(ctx, paths, func(record directoryStoredRecord) error {
			if !hasWinner {
				winner = record
				hasWinner = true
				stager.observeLoadedRecords(1)
				return nil
			}
			stager.observeLoadedRecords(2)
			if record.key == winner.key {
				if compareDirectorySourceOrdinal(record, winner) == 0 {
					return fmt.Errorf("Directory result %q has duplicate source ordinal", record.key)
				}
				if stats.SkippedDuplicate == math.MaxUint64 {
					return errors.New("Directory duplicate count overflow")
				}
				stats.SkippedDuplicate++
				winner = record
				stager.observeLoadedRecords(1)
				return nil
			}
			if err := writeWinner(); err != nil {
				return err
			}
			winner = record
			stager.observeLoadedRecords(1)
			return nil
		}); err != nil {
			return err
		}
		return writeWinner()
	})
	return stats, err
}

func (stager *directoryDedupStager) writeRecordFile(ctx context.Context, path string, write func(*bufio.Writer) error) (err error) {
	file, err := stager.files.create(path)
	if err != nil {
		return fmt.Errorf("create Directory result deduplication file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close Directory result deduplication file: %w", closeErr)
			err = errors.Join(err, closeErr)
		}
		if err != nil {
			if removeErr := stager.files.remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove incomplete Directory result deduplication file: %w", removeErr))
			}
		}
	}()
	writer := bufio.NewWriter(file)
	if err := write(writer); err != nil {
		return err
	}
	if err := directoryDedupContextError(ctx); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush Directory result deduplication file: %w", err)
	}
	return nil
}

func (stager *directoryDedupStager) mergeRecords(ctx context.Context, paths []string, emit func(directoryStoredRecord) error) (err error) {
	if len(paths) == 0 {
		return nil
	}
	readers, err := stager.openReaders(paths)
	if err != nil {
		return err
	}
	defer func() {
		for index := range readers {
			if closeErr := readers[index].file.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close Directory result merge input: %w", closeErr))
			}
		}
	}()
	if len(readers) > stager.maxOpenReaders {
		stager.maxOpenReaders = len(readers)
	}

	records := &directoryRecordHeap{}
	heap.Init(records)
	for index := range readers {
		header, ok, err := readers[index].nextHeader()
		if err != nil {
			return err
		}
		if ok {
			heap.Push(records, directoryHeapRecord{header: header, readerIndex: index})
		}
	}
	for records.Len() > 0 {
		if err := directoryDedupContextError(ctx); err != nil {
			return err
		}
		next := heap.Pop(records).(directoryHeapRecord)
		record, err := readers[next.readerIndex].readRecord(next.header)
		if err != nil {
			return err
		}
		stager.observeLoadedRecords(1)
		if err := emit(record); err != nil {
			return err
		}
		header, ok, err := readers[next.readerIndex].nextHeader()
		if err != nil {
			return err
		}
		if ok {
			heap.Push(records, directoryHeapRecord{header: header, readerIndex: next.readerIndex})
		}
	}
	return nil
}

type directoryChunkReader struct {
	file   directoryDedupReadCloser
	reader *bufio.Reader
}

func (stager *directoryDedupStager) openReaders(paths []string) ([]directoryChunkReader, error) {
	readers := make([]directoryChunkReader, 0, len(paths))
	for _, path := range paths {
		file, err := stager.files.open(path)
		if err != nil {
			for index := range readers {
				_ = readers[index].file.Close()
			}
			return nil, fmt.Errorf("open Directory result merge input: %w", err)
		}
		readers = append(readers, directoryChunkReader{file: file, reader: bufio.NewReader(file)})
	}
	return readers, nil
}

func (reader *directoryChunkReader) nextHeader() (directoryStoredRecordHeader, bool, error) {
	header, err := readDirectoryStoredRecordHeader(reader.reader)
	if errors.Is(err, io.EOF) {
		return directoryStoredRecordHeader{}, false, nil
	}
	if err != nil {
		return directoryStoredRecordHeader{}, false, fmt.Errorf("read Directory result merge input: %w", err)
	}
	return header, true, nil
}

func (reader *directoryChunkReader) readRecord(header directoryStoredRecordHeader) (directoryStoredRecord, error) {
	payload := make([]byte, header.payloadLength)
	if _, err := io.ReadFull(reader.reader, payload); err != nil {
		return directoryStoredRecord{}, fmt.Errorf("read Directory result merge payload: %w", err)
	}
	return directoryStoredRecord{
		key:                   header.key,
		payload:               payload,
		candidateOrdinal:      header.candidateOrdinal,
		physicalRecordOrdinal: header.physicalRecordOrdinal,
	}, nil
}

type directoryHeapRecord struct {
	header      directoryStoredRecordHeader
	readerIndex int
}

type directoryRecordHeap []directoryHeapRecord

func (records directoryRecordHeap) Len() int { return len(records) }

func (records directoryRecordHeap) Less(left, right int) bool {
	comparison := compareDirectoryStoredHeaders(records[left].header, records[right].header)
	if comparison != 0 {
		return comparison < 0
	}
	return records[left].readerIndex < records[right].readerIndex
}

func (records directoryRecordHeap) Swap(left, right int) {
	records[left], records[right] = records[right], records[left]
}

func (records *directoryRecordHeap) Push(value any) {
	*records = append(*records, value.(directoryHeapRecord))
}

func (records *directoryRecordHeap) Pop() any {
	old := *records
	last := len(old) - 1
	value := old[last]
	*records = old[:last]
	return value
}

func (stager *directoryDedupStager) runPaths(round, start, end uint64) []string {
	if end < start {
		return nil
	}
	paths := make([]string, 0, int(end-start+1))
	for index := start; index <= end; index++ {
		paths = append(paths, stager.runPath(round, index))
	}
	return paths
}

func (stager *directoryDedupStager) runPath(round, index uint64) string {
	if round == 0 {
		return filepath.Join(stager.tempDir, fmt.Sprintf("chunk-%020d.bin", index))
	}
	return filepath.Join(stager.tempDir, fmt.Sprintf("merge-%03d-%020d.bin", round, index))
}

func (stager *directoryDedupStager) removeFiles(paths []string) error {
	for _, path := range paths {
		if err := stager.files.remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove Directory result merge input: %w", err)
		}
	}
	return nil
}

func (stager *directoryDedupStager) observeLoadedRecords(count int) {
	if count > stager.maxLoadedRecords {
		stager.maxLoadedRecords = count
	}
}

func directoryStoredRecordBytes(record directoryStoredRecord) int64 {
	return directoryStoredRecordHeaderBytes + int64(len(record.key)) + int64(len(record.payload))
}

func compareDirectoryStoredRecords(left, right directoryStoredRecord) int {
	if left.key < right.key {
		return -1
	}
	if left.key > right.key {
		return 1
	}
	return compareDirectorySourceOrdinal(left, right)
}

func compareDirectoryStoredHeaders(left, right directoryStoredRecordHeader) int {
	if left.key < right.key {
		return -1
	}
	if left.key > right.key {
		return 1
	}
	if left.candidateOrdinal < right.candidateOrdinal {
		return -1
	}
	if left.candidateOrdinal > right.candidateOrdinal {
		return 1
	}
	if left.physicalRecordOrdinal < right.physicalRecordOrdinal {
		return -1
	}
	if left.physicalRecordOrdinal > right.physicalRecordOrdinal {
		return 1
	}
	return 0
}

func compareDirectorySourceOrdinal(left, right directoryStoredRecord) int {
	if left.candidateOrdinal < right.candidateOrdinal {
		return -1
	}
	if left.candidateOrdinal > right.candidateOrdinal {
		return 1
	}
	if left.physicalRecordOrdinal < right.physicalRecordOrdinal {
		return -1
	}
	if left.physicalRecordOrdinal > right.physicalRecordOrdinal {
		return 1
	}
	return 0
}

func writeDirectoryStoredRecord(writer io.Writer, record directoryStoredRecord) error {
	for _, value := range []uint64{
		uint64(len(record.key)), record.candidateOrdinal, record.physicalRecordOrdinal, uint64(len(record.payload)),
	} {
		if err := writeDirectoryUint64(writer, value); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(writer, record.key); err != nil {
		return err
	}
	_, err := writer.Write(record.payload)
	return err
}

func readDirectoryStoredRecordHeader(reader io.Reader) (directoryStoredRecordHeader, error) {
	keyLength, err := readDirectoryUint64(reader)
	if err != nil {
		return directoryStoredRecordHeader{}, err
	}
	candidateOrdinal, err := readDirectoryUint64(reader)
	if err != nil {
		return directoryStoredRecordHeader{}, err
	}
	physicalRecordOrdinal, err := readDirectoryUint64(reader)
	if err != nil {
		return directoryStoredRecordHeader{}, err
	}
	payloadLength, err := readDirectoryUint64(reader)
	if err != nil {
		return directoryStoredRecordHeader{}, err
	}
	if keyLength > maximumFFUFPhysicalRecordBytes || payloadLength > maximumFFUFPhysicalRecordBytes || payloadLength > uint64(maximumDirectoryInt()) {
		return directoryStoredRecordHeader{}, errors.New("Directory result deduplication record length is invalid")
	}
	key := make([]byte, int(keyLength))
	if _, err := io.ReadFull(reader, key); err != nil {
		return directoryStoredRecordHeader{}, err
	}
	return directoryStoredRecordHeader{
		key:                   string(key),
		payloadLength:         int(payloadLength),
		candidateOrdinal:      candidateOrdinal,
		physicalRecordOrdinal: physicalRecordOrdinal,
	}, nil
}

func writeDirectoryUint64(writer io.Writer, value uint64) error {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, err := writer.Write(encoded[:])
	return err
}

func readDirectoryUint64(reader io.Reader) (uint64, error) {
	var encoded [8]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(encoded[:]), nil
}

func (winners *directoryWinnerSet) Stream(ctx context.Context, emit func(enginecontract.Directory) error) (err error) {
	if winners == nil || winners.open == nil || winners.path == "" {
		return errors.New("Directory winner set is required")
	}
	if ctx == nil {
		return errors.New("Directory winner stream context is required")
	}
	if emit == nil {
		return errors.New("Directory winner callback is required")
	}
	if winners.streamed {
		return errors.New("Directory winner set has already been streamed")
	}
	winners.streamed = true
	file, err := winners.open(winners.path)
	if err != nil {
		return fmt.Errorf("open Directory winner set: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close Directory winner set: %w", closeErr))
		}
	}()
	reader := &directoryChunkReader{file: file, reader: bufio.NewReader(file)}
	var count uint64
	previousURL := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, ok, err := reader.nextHeader()
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		record, err := reader.readRecord(header)
		if err != nil {
			return err
		}
		var item enginecontract.Directory
		if err := json.Unmarshal(record.payload, &item); err != nil {
			return fmt.Errorf("decode staged Directory winner: %w", err)
		}
		if item.URL != record.key || (count > 0 && item.URL <= previousURL) {
			return errors.New("Directory winner set identity or ordering is invalid")
		}
		if count == math.MaxUint64 {
			return errors.New("Directory winner stream count overflow")
		}
		if err := emit(item); err != nil {
			return err
		}
		previousURL = item.URL
		count++
	}
	if count != winners.expectedItems {
		return errors.New("Directory winner stream count does not match finalized count")
	}
	return nil
}

func directoryDedupContextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("Directory result deduplication context is required")
	}
	return ctx.Err()
}

func maximumDirectoryInt() int {
	return int(^uint(0) >> 1)
}
