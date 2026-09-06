package urlcollectionruntime

import (
	"bufio"
	"container/heap"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const (
	urlSortChunkByteBudget = 8 * 1024 * 1024
	urlSortMergeFanIn      = 8
)

type urlSortOptions struct {
	chunkByteBudget int
	mergeFanIn      int
}

func defaultURLSortOptions() urlSortOptions {
	return urlSortOptions{chunkByteBudget: urlSortChunkByteBudget, mergeFanIn: urlSortMergeFanIn}
}

// sortAndDeduplicateURLs externally sorts exact raw URL lines in the task
// workspace. Only one bounded chunk and at most urlSortMergeFanIn readers are
// resident at a time, so a large collector result cannot turn into an
// application-wide in-memory URL set.
func sortAndDeduplicateURLs(ctx context.Context, path string) (int, int, error) {
	unique, total, err := sortURLLines(ctx, path, true)
	if err != nil {
		return 0, 0, err
	}
	return unique, total - unique, nil
}

// sortURLLines sorts bounded newline-delimited records. Its no-dedup mode is
// used by Endpoint staging where the ordinal suffix must preserve the final
// input record for each exact URL.
func sortURLLines(ctx context.Context, path string, deduplicate bool) (result int, total int, err error) {
	return sortURLLinesWithOptions(ctx, path, deduplicate, defaultURLSortOptions())
}

func sortURLLinesWithOptions(ctx context.Context, path string, deduplicate bool, options urlSortOptions) (result int, total int, err error) {
	if ctx == nil || path == "" {
		return 0, 0, errors.New("URL sort context and path are required")
	}
	if options.chunkByteBudget <= 0 || options.mergeFanIn < 2 {
		return 0, 0, errors.New("URL sort options are invalid")
	}
	temporaryDirectory, err := os.MkdirTemp(filepath.Dir(path), ".url-sort-*")
	if err != nil {
		return 0, 0, fmt.Errorf("create URL sort workspace: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(temporaryDirectory); removeErr != nil && err == nil {
			result, total = 0, 0
			err = fmt.Errorf("remove URL sort workspace: %w", removeErr)
		} else if removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove URL sort workspace: %w", removeErr))
		}
	}()

	chunks, total, err := writeURLSortChunks(ctx, path, temporaryDirectory, options.chunkByteBudget)
	if err != nil {
		return 0, 0, err
	}
	for len(chunks) > options.mergeFanIn {
		chunks, err = reduceURLSortChunks(ctx, chunks, temporaryDirectory, options.mergeFanIn)
		if err != nil {
			return 0, 0, err
		}
	}
	result, err = mergeSortedURLChunks(ctx, chunks, path, deduplicate)
	if err != nil {
		return 0, 0, err
	}
	if !deduplicate {
		result = total
	}
	return result, total, nil
}

func writeURLSortChunks(ctx context.Context, inputPath, directory string, chunkByteBudget int) (chunks []string, total int, err error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close URL sort input: %w", closeErr)
		}
	}()

	reader := bufio.NewReaderSize(input, 64*1024)
	chunk := make([]string, 0, 4096)
	chunkBytes, total, chunkNumber := 0, 0, 0
	chunks = make([]string, 0)
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		sort.Strings(chunk)
		chunkNumber++
		path := filepath.Join(directory, fmt.Sprintf("chunk-%08d.txt", chunkNumber))
		if err := writeSortedURLChunk(ctx, path, chunk); err != nil {
			return err
		}
		chunks = append(chunks, path)
		chunk = nil
		chunkBytes = 0
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		line, oversized, readErr := readBoundedLine(reader)
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, 0, readErr
		}
		if oversized {
			return nil, 0, errors.New("candidate record exceeds 4 MiB")
		}
		if line == "" {
			continue
		}
		if chunkBytes > 0 && len(line)+1 > chunkByteBudget-chunkBytes {
			if err := flush(); err != nil {
				return nil, 0, err
			}
		}
		chunk = append(chunk, line)
		chunkBytes += len(line) + 1
		total++
	}
	if err := flush(); err != nil {
		return nil, 0, err
	}
	return chunks, total, nil
}

func writeSortedURLChunk(ctx context.Context, path string, values []string) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	writer := bufio.NewWriterSize(file, 64*1024)
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := writer.WriteString(value + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func reduceURLSortChunks(ctx context.Context, chunks []string, directory string, mergeFanIn int) ([]string, error) {
	next := make([]string, 0, (len(chunks)+mergeFanIn-1)/mergeFanIn)
	for start := 0; start < len(chunks); start += mergeFanIn {
		end := start + mergeFanIn
		if end > len(chunks) {
			end = len(chunks)
		}
		// The current round's input count is part of the private filename so
		// later reduction rounds cannot collide with their own source runs.
		output := filepath.Join(directory, fmt.Sprintf("merge-%08d-%08d.txt", len(chunks), len(next)+1))
		if _, err := mergeURLChunkGroup(ctx, chunks[start:end], output, false); err != nil {
			return nil, err
		}
		for _, path := range chunks[start:end] {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		next = append(next, output)
	}
	return next, nil
}

func mergeSortedURLChunks(ctx context.Context, chunks []string, output string, deduplicate bool) (int, error) {
	temporaryOutput := output + ".tmp"
	unique, err := mergeURLChunkGroup(ctx, chunks, temporaryOutput, deduplicate)
	if err != nil {
		_ = os.Remove(temporaryOutput)
		return 0, err
	}
	if err := os.Rename(temporaryOutput, output); err != nil {
		_ = os.Remove(temporaryOutput)
		return 0, err
	}
	return unique, nil
}

func mergeURLChunkGroup(ctx context.Context, chunks []string, output string, deduplicate bool) (result int, err error) {
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(output)
		}
	}()
	writer := bufio.NewWriterSize(file, 64*1024)
	sources := make([]*urlSortSource, 0, len(chunks))
	defer func() {
		for _, source := range sources {
			_ = source.file.Close()
		}
	}()
	queue := make(urlSortHeap, 0, len(chunks))
	for index, path := range chunks {
		source, err := openURLSortSource(path)
		if err != nil {
			return 0, err
		}
		sources = append(sources, source)
		if source.next() {
			heap.Push(&queue, urlSortHeapItem{value: source.value, index: index})
		} else if source.err != nil {
			return 0, source.err
		}
	}
	var previous string
	hasPrevious := false
	for queue.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		item := heap.Pop(&queue).(urlSortHeapItem)
		if !deduplicate || !hasPrevious || item.value != previous {
			if _, err := writer.WriteString(item.value + "\n"); err != nil {
				return 0, err
			}
			if deduplicate {
				result++
				previous = item.value
				hasPrevious = true
			}
		}
		source := sources[item.index]
		if source.next() {
			heap.Push(&queue, urlSortHeapItem{value: source.value, index: item.index})
		} else if source.err != nil {
			return 0, source.err
		}
	}
	if err := writer.Flush(); err != nil {
		return 0, err
	}
	return result, nil
}

type urlSortSource struct {
	file   *os.File
	reader *bufio.Reader
	value  string
	err    error
}

func openURLSortSource(path string) (*urlSortSource, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &urlSortSource{file: file, reader: bufio.NewReaderSize(file, 64*1024)}, nil
}

func (source *urlSortSource) next() bool {
	line, oversized, err := readBoundedLine(source.reader)
	if errors.Is(err, io.EOF) {
		if line != "" {
			source.err = errors.New("sorted URL chunk record must be LF-terminated")
		}
		return false
	}
	if err != nil {
		source.err = err
		return false
	}
	if oversized {
		source.err = errors.New("sorted URL chunk record exceeds 4 MiB")
		return false
	}
	source.value = line
	return true
}

type urlSortHeapItem struct {
	value string
	index int
}

type urlSortHeap []urlSortHeapItem

func (items urlSortHeap) Len() int { return len(items) }
func (items urlSortHeap) Less(left, right int) bool {
	if items[left].value == items[right].value {
		return items[left].index < items[right].index
	}
	return items[left].value < items[right].value
}
func (items urlSortHeap) Swap(left, right int) { items[left], items[right] = items[right], items[left] }
func (items *urlSortHeap) Push(value any)      { *items = append(*items, value.(urlSortHeapItem)) }
func (items *urlSortHeap) Pop() any {
	old := *items
	value := old[len(old)-1]
	*items = old[:len(old)-1]
	return value
}
