package directoryscanruntime

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"net/netip"
	"os"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

// WebsiteCandidate is one stable pass-two item. Website remains the exact
// baseline or finalized fact string and is never parsed or rebuilt.
type WebsiteCandidate struct {
	Ordinal uint64
	Website string
}

// CandidatePlan is created only after the complete first pass succeeds. It
// retains no candidate collection; pass two reopens the immutable facts file.
type CandidatePlan struct {
	enumerator      candidateEnumerator
	candidateCount  uint64
	candidateDigest [sha256.Size]byte
}

// CandidateReplay owns the concurrency-sized pass-two queue and its producer.
type CandidateReplay struct {
	Candidates <-chan WebsiteCandidate
	done       chan struct{}
	cancel     context.CancelFunc
	err        error
}

type candidateEnumerator struct {
	targetType      enginecontract.TargetType
	targetValue     string
	websiteURLsPath string
	cidrStart       uint32
	cidrEnd         uint32
}

// PreflightCandidates validates and counts every raw candidate before a replay
// queue or scanner process can exist. FFUF's appended FUZZ token is tool
// syntax; literal candidate bytes are never rewritten or rejected for it.
func PreflightCandidates(ctx context.Context, target enginecontract.Target, websiteURLsPath string) (*CandidatePlan, error) {
	if ctx == nil {
		return nil, errors.New("candidate context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	enumerator, err := newCandidateEnumerator(target, websiteURLsPath)
	if err != nil {
		return nil, err
	}
	var count uint64
	digest := sha256.New()
	if err := enumerator.forEach(ctx, func(candidate string) error {
		if count == math.MaxUint64 {
			return errors.New("candidate count overflow")
		}
		writeCandidateDigest(digest, candidate)
		count++
		return nil
	}); err != nil {
		return nil, err
	}
	plan := &CandidatePlan{enumerator: enumerator, candidateCount: count}
	copy(plan.candidateDigest[:], digest.Sum(nil))
	return plan, nil
}

func (plan *CandidatePlan) CandidateCount() uint64 {
	if plan == nil {
		return 0
	}
	return plan.candidateCount
}

// StartReplay allocates exactly one bounded queue and reproduces pass one in a
// cancellation-aware producer. Content or count drift fails the replay rather
// than accepting a sequence different from the completed preflight.
func (plan *CandidatePlan) StartReplay(ctx context.Context, queueCapacity int) (*CandidateReplay, error) {
	if plan == nil {
		return nil, errors.New("candidate plan is required")
	}
	if ctx == nil {
		return nil, errors.New("candidate replay context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if queueCapacity < 1 {
		return nil, errors.New("candidate queue capacity must be positive")
	}

	replayCtx, cancel := context.WithCancel(ctx)
	queue := make(chan WebsiteCandidate, queueCapacity)
	replay := &CandidateReplay{
		Candidates: queue,
		done:       make(chan struct{}),
		cancel:     cancel,
	}
	go func() {
		defer close(replay.done)
		defer close(queue)
		replay.err = plan.replay(replayCtx, queue)
	}()
	return replay, nil
}

func (plan *CandidatePlan) replay(ctx context.Context, queue chan<- WebsiteCandidate) error {
	var ordinal uint64
	digest := sha256.New()
	err := plan.enumerator.forEach(ctx, func(candidate string) error {
		if ordinal >= plan.candidateCount {
			return errors.New("candidate input changed between passes")
		}
		writeCandidateDigest(digest, candidate)
		item := WebsiteCandidate{Ordinal: ordinal, Website: candidate}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case queue <- item:
			ordinal++
			return nil
		}
	})
	if err != nil {
		return err
	}
	if ordinal != plan.candidateCount {
		return errors.New("candidate input changed between passes")
	}
	if !bytes.Equal(digest.Sum(nil), plan.candidateDigest[:]) {
		return errors.New("candidate input changed between passes")
	}
	return nil
}

func (replay *CandidateReplay) Stop() {
	if replay != nil && replay.cancel != nil {
		replay.cancel()
	}
}

func (replay *CandidateReplay) Wait() error {
	if replay == nil || replay.done == nil {
		return errors.New("candidate replay is required")
	}
	<-replay.done
	return replay.err
}

func newCandidateEnumerator(target enginecontract.Target, websiteURLsPath string) (candidateEnumerator, error) {
	if websiteURLsPath == "" {
		return candidateEnumerator{}, errors.New("websiteURLs facts path is required")
	}
	if target.Value == "" {
		return candidateEnumerator{}, errors.New("Target value is required")
	}
	enumerator := candidateEnumerator{
		targetType:      target.Type,
		targetValue:     target.Value,
		websiteURLsPath: websiteURLsPath,
	}
	switch target.Type {
	case enginecontract.TargetTypeDomain:
	case enginecontract.TargetTypeIP:
		address, err := netip.ParseAddr(target.Value)
		if err != nil || !address.Is4() || address.String() != target.Value {
			return candidateEnumerator{}, errors.New("Target IP must be a canonical IPv4 address")
		}
	case enginecontract.TargetTypeCIDR:
		prefix, err := netip.ParsePrefix(target.Value)
		if err != nil || !prefix.Addr().Is4() || prefix != prefix.Masked() || prefix.String() != target.Value {
			return candidateEnumerator{}, errors.New("Target CIDR must be a canonical IPv4 prefix")
		}
		startBytes := prefix.Addr().As4()
		start := binary.BigEndian.Uint32(startBytes[:])
		span := uint64(1) << uint(32-prefix.Bits())
		end := uint64(start) + span - 1
		if end > math.MaxUint32 {
			return candidateEnumerator{}, errors.New("Target IPv4 CIDR upper bound overflow")
		}
		enumerator.cidrStart = start
		enumerator.cidrEnd = uint32(end)
	default:
		return candidateEnumerator{}, fmt.Errorf("unsupported Target type %q", target.Type)
	}
	return enumerator, nil
}

func (enumerator candidateEnumerator) forEach(ctx context.Context, emit func(string) error) error {
	if emit == nil {
		return errors.New("candidate emit callback is required")
	}
	if err := enumerator.forEachBaseline(ctx, emit); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return enumerator.forEachWebsiteURL(ctx, func(candidate string) error {
		if enumerator.isEmittedBaseline(candidate) {
			return nil
		}
		return emit(candidate)
	})
}

func (enumerator candidateEnumerator) forEachBaseline(ctx context.Context, emit func(string) error) error {
	switch enumerator.targetType {
	case enginecontract.TargetTypeDomain, enginecontract.TargetTypeIP:
		if err := emitCandidate(ctx, "http://"+enumerator.targetValue, emit); err != nil {
			return err
		}
		return emitCandidate(ctx, "https://"+enumerator.targetValue, emit)
	case enginecontract.TargetTypeCIDR:
		for current := uint64(enumerator.cidrStart); ; current++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			address := netip.AddrFrom4([4]byte{
				byte(current >> 24), byte(current >> 16), byte(current >> 8), byte(current),
			}).String()
			if err := emitCandidate(ctx, "http://"+address, emit); err != nil {
				return err
			}
			if err := emitCandidate(ctx, "https://"+address, emit); err != nil {
				return err
			}
			if current == uint64(enumerator.cidrEnd) {
				return nil
			}
			if current == math.MaxUint32 {
				return errors.New("Target IPv4 CIDR iteration overflow")
			}
		}
	default:
		return fmt.Errorf("unsupported Target type %q", enumerator.targetType)
	}
}

func (enumerator candidateEnumerator) forEachWebsiteURL(ctx context.Context, emit func(string) error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(enumerator.websiteURLsPath)
	if err != nil {
		return fmt.Errorf("open websiteURLs facts: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close websiteURLs facts: %w", closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat websiteURLs facts: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("websiteURLs facts must be a regular file")
	}

	readErr := readWebsiteURLFacts(file, func(candidate string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := emit(candidate); err != nil {
			return err
		}
		return nil
	})
	if readErr != nil {
		return fmt.Errorf("read websiteURLs facts: %w", readErr)
	}
	return ctx.Err()
}

// readWebsiteURLFacts preserves confirmed candidate bytes while enforcing only
// the LF framing and bounded line size needed by the FFUF input file.
func readWebsiteURLFacts(reader io.Reader, visit func(string) error) error {
	if reader == nil || visit == nil {
		return errors.New("websiteURLs fact reader and visitor are required")
	}
	buffered := bufio.NewReaderSize(reader, maximumFFUFPhysicalRecordBytes+1)
	for {
		line, err := buffered.ReadSlice('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maximumFFUFPhysicalRecordBytes)
		}
		if err != nil {
			return fmt.Errorf("read websiteURLs facts: %w", err)
		}
		if len(line) == 0 || line[len(line)-1] != '\n' {
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		line = line[:len(line)-1]
		if len(line) > maximumFFUFPhysicalRecordBytes {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maximumFFUFPhysicalRecordBytes)
		}
		if strings.ContainsAny(string(line), "\x00\r") {
			return errors.New("websiteURLs fact line contains unsafe control characters")
		}
		if err := visit(string(line)); err != nil {
			return err
		}
	}
}

func (enumerator candidateEnumerator) isEmittedBaseline(candidate string) bool {
	switch enumerator.targetType {
	case enginecontract.TargetTypeDomain, enginecontract.TargetTypeIP:
		return candidate == "http://"+enumerator.targetValue || candidate == "https://"+enumerator.targetValue
	case enginecontract.TargetTypeCIDR:
		host := ""
		if value, ok := strings.CutPrefix(candidate, "http://"); ok {
			host = value
		} else if value, ok := strings.CutPrefix(candidate, "https://"); ok {
			host = value
		}
		address, err := netip.ParseAddr(host)
		if err != nil || !address.Is4() || address.String() != host {
			return false
		}
		bytes := address.As4()
		value := binary.BigEndian.Uint32(bytes[:])
		return value >= enumerator.cidrStart && value <= enumerator.cidrEnd
	default:
		return false
	}
}

func emitCandidate(ctx context.Context, candidate string, emit func(string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return emit(candidate)
}

func writeCandidateDigest(digest hash.Hash, candidate string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(candidate)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(candidate))
}
