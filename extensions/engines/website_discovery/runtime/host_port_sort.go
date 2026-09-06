package websitediscoveryruntime

import (
	"bufio"
	"container/heap"
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
	"golang.org/x/net/idna"
)

const (
	hostPortSortChunkByteBudget = 8 * 1024 * 1024
	hostPortSortMergeFanIn      = 8
	maxHostPortFactLineBytes    = 4 * 1024 * 1024
)

func writeHostPortSortChunks(ctx context.Context, factsPath, directory string, target websitediscoverycontract.Target) (chunkPaths []string, total uint64, err error) {
	if ctx == nil || factsPath == "" || directory == "" {
		return nil, 0, errors.New("HostPort sort inputs are required")
	}
	facts, err := os.Open(factsPath)
	if err != nil {
		return nil, 0, fmt.Errorf("open hostPorts facts: %w", err)
	}
	defer func() {
		if closeErr := facts.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close hostPorts facts: %w", closeErr)
		}
	}()
	reader := bufio.NewReaderSize(facts, 64*1024)
	chunk := make([]string, 0, 4096)
	chunkBytes := 0
	chunkNumber := 0
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		sort.Strings(chunk)
		chunkNumber++
		path := filepath.Join(directory, fmt.Sprintf("hostports-chunk-%08d.txt", chunkNumber))
		if err := writeHostPortChunk(ctx, path, chunk); err != nil {
			return err
		}
		chunkPaths = append(chunkPaths, path)
		chunk = make([]string, 0, 4096)
		chunkBytes = 0
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		line, err := readHostPortFactLine(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		fact, err := parseHostPortFact([]byte(line))
		if err != nil {
			return nil, 0, fmt.Errorf("parse hostPorts fact: %w", err)
		}
		if targetBaselineOverlapsHostPort(target, fact.Host, fact.Port) {
			continue
		}
		key := hostPortSortKey(fact.Host, fact.Port)
		if chunkBytes > 0 && chunkBytes+len(key)+1 > hostPortSortChunkByteBudget {
			if err := flush(); err != nil {
				return nil, 0, err
			}
		}
		chunk = append(chunk, key)
		chunkBytes += len(key) + 1
		if total == ^uint64(0) {
			return nil, 0, errors.New("HostPort record count overflow")
		}
		total++
	}
	if err := flush(); err != nil {
		return nil, 0, err
	}
	return chunkPaths, total, nil
}

func readHostPortFactLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if errors.Is(err, io.EOF) {
		if line == "" {
			return "", io.EOF
		}
		return "", errors.New("hostPorts fact line must be LF-terminated")
	}
	if err != nil {
		return "", fmt.Errorf("read hostPorts facts: %w", err)
	}
	line = strings.TrimSuffix(line, "\n")
	if len(line) == 0 || len(line) > maxHostPortFactLineBytes || strings.ContainsAny(line, "\r\n") {
		return "", errors.New("hostPorts fact line is invalid")
	}
	return line, nil
}

func writeHostPortChunk(ctx context.Context, path string, values []string) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
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

func mergeHostPortChunks(ctx context.Context, chunks []string, directory string) (string, error) {
	if len(chunks) == 0 {
		path := filepath.Join(directory, "hostports-merged-empty.txt")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return "", err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return "", err
		}
		return path, nil
	}
	for len(chunks) > hostPortSortMergeFanIn {
		next := make([]string, 0, (len(chunks)+hostPortSortMergeFanIn-1)/hostPortSortMergeFanIn)
		for start := 0; start < len(chunks); start += hostPortSortMergeFanIn {
			end := start + hostPortSortMergeFanIn
			if end > len(chunks) {
				end = len(chunks)
			}
			output := filepath.Join(directory, fmt.Sprintf("hostports-merge-%08d-%08d.txt", len(chunks), len(next)+1))
			if err := mergeHostPortChunkGroup(ctx, chunks[start:end], output); err != nil {
				return "", err
			}
			for _, path := range chunks[start:end] {
				if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					return "", err
				}
			}
			next = append(next, output)
		}
		chunks = next
	}
	output := filepath.Join(directory, "hostports-merged.txt")
	if err := mergeHostPortChunkGroup(ctx, chunks, output); err != nil {
		return "", err
	}
	return output, nil
}

func mergeHostPortChunkGroup(ctx context.Context, chunks []string, output string) (err error) {
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
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
	sources := make([]*hostPortSortSource, 0, len(chunks))
	defer func() {
		for _, source := range sources {
			if closeErr := source.file.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("close HostPort sort source: %w", closeErr)
			}
		}
	}()
	queue := make(hostPortSortHeap, 0, len(chunks))
	for index, path := range chunks {
		source, err := openHostPortSortSource(path)
		if err != nil {
			return err
		}
		sources = append(sources, source)
		if source.next() {
			heap.Push(&queue, hostPortSortHeapItem{value: source.value, index: index})
		} else if source.err != nil {
			return source.err
		}
	}
	previous := ""
	hasPrevious := false
	for queue.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		item := heap.Pop(&queue).(hostPortSortHeapItem)
		if !hasPrevious || item.value != previous {
			if _, err := writer.WriteString(item.value + "\n"); err != nil {
				return err
			}
			previous, hasPrevious = item.value, true
		}
		source := sources[item.index]
		if source.next() {
			heap.Push(&queue, hostPortSortHeapItem{value: source.value, index: item.index})
		} else if source.err != nil {
			return source.err
		}
	}
	return writer.Flush()
}

type hostPortSortSource struct {
	file   *os.File
	reader *bufio.Reader
	value  string
	err    error
}

func openHostPortSortSource(path string) (*hostPortSortSource, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &hostPortSortSource{file: file, reader: bufio.NewReaderSize(file, 64*1024)}, nil
}

func (source *hostPortSortSource) next() bool {
	line, err := source.reader.ReadString('\n')
	if errors.Is(err, io.EOF) {
		if line != "" {
			source.err = errors.New("sorted HostPort chunk record must be LF-terminated")
		}
		return false
	}
	if err != nil {
		source.err = err
		return false
	}
	source.value = strings.TrimSuffix(line, "\n")
	return true
}

type hostPortSortHeapItem struct {
	value string
	index int
}
type hostPortSortHeap []hostPortSortHeapItem

func (items hostPortSortHeap) Len() int { return len(items) }
func (items hostPortSortHeap) Less(left, right int) bool {
	if items[left].value == items[right].value {
		return items[left].index < items[right].index
	}
	return items[left].value < items[right].value
}
func (items hostPortSortHeap) Swap(left, right int) {
	items[left], items[right] = items[right], items[left]
}
func (items *hostPortSortHeap) Push(value any) { *items = append(*items, value.(hostPortSortHeapItem)) }
func (items *hostPortSortHeap) Pop() any {
	old := *items
	value := old[len(old)-1]
	*items = old[:len(old)-1]
	return value
}

func copyHostPortCandidates(ctx context.Context, input io.Reader, emit func(string) error) error {
	reader := bufio.NewReaderSize(input, 64*1024)
	for {
		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			if line == "" {
				return nil
			}
			return errors.New("merged HostPort line is unframed")
		}
		if err != nil {
			return err
		}
		line = strings.TrimSuffix(line, "\n")
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return errors.New("merged HostPort key is invalid")
		}
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return errors.New("merged HostPort port is invalid")
		}
		for _, candidate := range hostPortURLs(parts[0], port) {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := emit(candidate); err != nil {
				return fmt.Errorf("write hostPorts candidate: %w", err)
			}
		}
	}
}

func hostPortSortKey(host string, port int) string {
	return host + "\t" + fmt.Sprintf("%05d", port)
}

func canonicalHost(host string) bool {
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Is4() && ip.String() == host
	}
	if strings.ToLower(host) != host || strings.HasSuffix(host, ".") || len(host) > 253 || strings.ContainsAny(host, " \t\r\n") {
		return false
	}
	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil || ascii != host {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}
