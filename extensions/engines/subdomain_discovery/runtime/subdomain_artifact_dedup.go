package subdomaindiscoveryruntime

import (
	"bufio"
	"container/heap"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yyhuni/lunafox/engines/subdomain_discovery/internal/validator"
)

const subdomainDedupChunkSize = 100000

func deduplicateSubdomainArtifacts(ctx context.Context, workspaceDir string, inputArtifactPaths []string) (string, error) {
	return deduplicateSubdomainArtifactsWithChunkSize(ctx, workspaceDir, inputArtifactPaths, subdomainDedupChunkSize)
}

func deduplicateSubdomainArtifactsWithChunkSize(ctx context.Context, workspaceDir string, inputArtifactPaths []string, chunkSize int) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	workspaceAbs, err := requireWorkspaceAbs(workspaceDir, "subdomain artifact deduplication workspace")
	if err != nil {
		return "", err
	}
	if chunkSize <= 0 {
		return "", fmt.Errorf("subdomain deduplication chunk size must be > 0")
	}
	inputs, err := resolveInputArtifactPaths(workspaceAbs, inputArtifactPaths)
	if err != nil {
		return "", err
	}
	if len(inputs) == 0 {
		return "", fmt.Errorf("subdomain artifact paths are required")
	}

	tempDir, err := os.MkdirTemp(workspaceAbs, ".subdomain-dedup-*")
	if err != nil {
		return "", fmt.Errorf("create subdomain dedup temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	chunkPaths, err := writeSubdomainDedupChunks(ctx, inputs, tempDir, chunkSize)
	if err != nil {
		return "", err
	}
	outputPath, err := allocateSubdomainDedupOutputFile(workspaceAbs)
	if err != nil {
		return "", err
	}
	if _, err := mergeSubdomainDedupChunks(ctx, chunkPaths, outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
}

func resolveInputArtifactPaths(workspaceAbs string, paths []string) ([]string, error) {
	var resolved []string
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		candidate := path
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(workspaceAbs, candidate)
		}
		clean, err := filepath.Abs(candidate)
		if err != nil {
			return nil, fmt.Errorf("resolve subdomain artifact path %q: %w", raw, err)
		}
		resolved = append(resolved, clean)
	}
	return resolved, nil
}

func writeSubdomainDedupChunks(ctx context.Context, inputPaths []string, tempDir string, chunkSize int) ([]string, error) {
	chunk := make([]string, 0, chunkSize)
	chunkSeen := make(map[string]struct{}, chunkSize)
	var chunkPaths []string
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		sort.Strings(chunk)
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%06d.txt", len(chunkPaths)+1))
		if err := writeSortedSubdomainChunk(chunkPath, chunk); err != nil {
			return err
		}
		chunkPaths = append(chunkPaths, chunkPath)
		chunk = chunk[:0]
		clear(chunkSeen)
		return nil
	}

	for _, path := range inputPaths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open subdomain artifact %s: %w", path, err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if err := ctx.Err(); err != nil {
				_ = file.Close()
				return nil, err
			}
			subdomain, ok := normalizeSubdomainArtifactLine(scanner.Text())
			if !ok {
				continue
			}
			if _, exists := chunkSeen[subdomain]; exists {
				continue
			}
			chunkSeen[subdomain] = struct{}{}
			chunk = append(chunk, subdomain)
			if len(chunk) >= chunkSize {
				if err := flush(); err != nil {
					_ = file.Close()
					return nil, err
				}
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			_ = file.Close()
			return nil, fmt.Errorf("scan subdomain artifact %s: %w", path, scanErr)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close subdomain artifact %s: %w", path, err)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return chunkPaths, nil
}

func normalizeSubdomainArtifactLine(s string) (string, bool) {
	return validator.NormalizeSubdomainLine(s)
}

func writeSortedSubdomainChunk(path string, lines []string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create subdomain dedup chunk: %w", err)
	}
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			_ = file.Close()
			return fmt.Errorf("write subdomain dedup chunk: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		return fmt.Errorf("flush subdomain dedup chunk: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close subdomain dedup chunk: %w", err)
	}
	return nil
}

func allocateSubdomainDedupOutputFile(workspaceAbs string) (string, error) {
	for i := 1; i <= 999999; i++ {
		path := filepath.Join(workspaceAbs, fmt.Sprintf("deduplicated_subdomains_%03d.txt", i))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect subdomain dedup output candidate: %w", err)
		}
	}
	return "", fmt.Errorf("allocate subdomain dedup output file: exhausted generated names")
}

func mergeSubdomainDedupChunks(ctx context.Context, chunkPaths []string, outputPath string) (int, error) {
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return 0, fmt.Errorf("create deduplicated subdomain artifact: %w", err)
	}
	defer func() { _ = out.Close() }()
	writer := bufio.NewWriter(out)
	defer func() { _ = writer.Flush() }()

	readers, err := openSubdomainChunkReaders(chunkPaths)
	if err != nil {
		return 0, err
	}
	defer func() {
		for _, reader := range readers {
			_ = reader.file.Close()
		}
	}()

	h := &subdomainChunkHeap{}
	heap.Init(h)
	for i := range readers {
		if readers[i].scanner.Scan() {
			heap.Push(h, subdomainChunkItem{value: readers[i].scanner.Text(), readerIndex: i})
		} else if err := readers[i].scanner.Err(); err != nil {
			return 0, fmt.Errorf("scan subdomain dedup chunk: %w", err)
		}
	}

	last := ""
	uniqueCount := 0
	for h.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return uniqueCount, err
		}
		item := heap.Pop(h).(subdomainChunkItem)
		if item.value != last {
			if _, err := fmt.Fprintln(writer, item.value); err != nil {
				return uniqueCount, fmt.Errorf("write deduplicated subdomain artifact: %w", err)
			}
			last = item.value
			uniqueCount++
		}
		reader := &readers[item.readerIndex]
		if reader.scanner.Scan() {
			heap.Push(h, subdomainChunkItem{value: reader.scanner.Text(), readerIndex: item.readerIndex})
		} else if err := reader.scanner.Err(); err != nil {
			return uniqueCount, fmt.Errorf("scan subdomain dedup chunk: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return uniqueCount, fmt.Errorf("flush deduplicated subdomain artifact: %w", err)
	}
	return uniqueCount, nil
}

type subdomainChunkReader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func openSubdomainChunkReaders(paths []string) ([]subdomainChunkReader, error) {
	readers := make([]subdomainChunkReader, 0, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			for _, reader := range readers {
				_ = reader.file.Close()
			}
			return nil, fmt.Errorf("open subdomain dedup chunk: %w", err)
		}
		readers = append(readers, subdomainChunkReader{file: file, scanner: bufio.NewScanner(file)})
	}
	return readers, nil
}

type subdomainChunkItem struct {
	value       string
	readerIndex int
}

type subdomainChunkHeap []subdomainChunkItem

func (h subdomainChunkHeap) Len() int { return len(h) }

func (h subdomainChunkHeap) Less(i, j int) bool {
	if h[i].value == h[j].value {
		return h[i].readerIndex < h[j].readerIndex
	}
	return h[i].value < h[j].value
}

func (h subdomainChunkHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *subdomainChunkHeap) Push(x any) {
	*h = append(*h, x.(subdomainChunkItem))
}

func (h *subdomainChunkHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
