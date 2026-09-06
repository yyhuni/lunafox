package fingerprintdetectionruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
)

type Runtime struct {
	executor observerWardExecutor
}

func New() *Runtime {
	return &Runtime{executor: containerObserverWardExecutor{}}
}

func NewWithExecutor(executor observerWardExecutor) *Runtime {
	return &Runtime{executor: executor}
}

func (runtime *Runtime) Execute(ctx context.Context, execution *enginecontract.Execution) (err error) {
	if ctx == nil {
		return errors.New("execution context is required")
	}
	if execution == nil {
		return errors.New("execution is required")
	}
	if execution.Progress == nil || execution.Results.WebsiteTechnologies == nil {
		return errors.New("Fingerprint Detection progress and result ports are required")
	}
	config := execution.Config.ObserverWard
	if err := validateObserverWardConfig(config); err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	if err := preflightFingerprintHub(execution.PlatformResources.FingerprintLibraryFingerPrintHub); err != nil {
		return err
	}
	if execution.Input.WebsiteURLs == nil {
		return errors.New("typed WebsiteURLs input handle is required")
	}
	websiteURLsPath, err := execution.Input.WebsiteURLs.Path(ctx)
	if err != nil {
		return fmt.Errorf("materialize WebsiteURLs input: %w", err)
	}
	candidatePath, candidateCount, err := MaterializeCandidates(ctx, execution.Target, websiteURLsPath, execution.Workspace)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := cleanupObserverCandidateArtifacts(execution.Workspace); cleanupErr != nil {
			if err == nil {
				err = cleanupErr
			} else {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	command, err := buildObserverWardCommand(candidatePath, execution.PlatformResources.FingerprintLibraryFingerPrintHub, config)
	if err != nil {
		return err
	}
	if runtime == nil || runtime.executor == nil {
		return errors.New("Observer Ward executor is not configured")
	}
	outputRecords := uint64(0)
	validRecords := uint64(0)
	malformedRecords := uint64(0)
	trustedResponses := uint64(0)
	matchedCandidates := uint64(0)
	zeroMatchCandidates := uint64(0)
	if err := runtime.executor.Run(ctx, command, func(line []byte) error {
		if outputRecords == ^uint64(0) {
			return errors.New("Observer Ward output record count overflow")
		}
		outputRecords++
		result, err := parseObserverWardResult(line)
		if err != nil {
			if malformedRecords == ^uint64(0) {
				return errors.New("Observer Ward malformed record count overflow")
			}
			malformedRecords++
			return nil
		}
		if validRecords == ^uint64(0) {
			return errors.New("Observer Ward valid record count overflow")
		}
		validRecords++
		trusted, tech := trustedObserverWardTechnology(result)
		if !trusted {
			return nil
		}
		if trustedResponses == ^uint64(0) {
			return errors.New("Observer Ward trusted response count overflow")
		}
		trustedResponses++
		if len(tech) == 0 {
			zeroMatchCandidates++
		} else {
			matchedCandidates++
		}
		item := enginecontract.WebsiteTechnology{URL: result.Target, Tech: tech}
		items := make(chan enginecontract.WebsiteTechnology, 1)
		items <- item
		close(items)
		if err := execution.Results.WebsiteTechnologies.Submit(ctx, items); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if outputRecords > 0 && validRecords == 0 {
		return errors.New("Observer Ward output contains no valid record")
	}
	reportedCandidates := candidateCount
	if trustedResponses > reportedCandidates {
		// Duplicate tool observations are submitted independently. Keep the
		// aggregate partition meaningful even when output rows outnumber input
		// candidate lines.
		reportedCandidates = trustedResponses
	}
	failedRequests := reportedCandidates - trustedResponses
	if err := execution.Progress.Report(ctx, fmt.Sprintf("complete result reporting candidates=%d outputRecords=%d malformedRecords=%d trustedResponses=%d failedRequests=%d matchedCandidates=%d zeroMatchCandidates=%d submittedItems=%d", reportedCandidates, outputRecords, malformedRecords, trustedResponses, failedRequests, matchedCandidates, zeroMatchCandidates, trustedResponses)); err != nil {
		return err
	}
	return nil
}

func preflightFingerprintHub(path string) error {
	if path == "" {
		return errors.New("FingerprintHub platform resource is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect FingerprintHub platform resource: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("FingerprintHub platform resource must be a regular file")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read FingerprintHub platform resource: %w", err)
	}
	if len(payload) == 0 {
		return errors.New("FingerprintHub corpus is empty")
	}
	var aggregate []json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&aggregate); err != nil {
		return fmt.Errorf("parse FingerprintHub corpus: %w", err)
	}
	if len(aggregate) == 0 {
		return errors.New("FingerprintHub corpus contains zero templates")
	}
	for index, template := range aggregate {
		if len(bytes.TrimSpace(template)) == 0 || bytes.Equal(bytes.TrimSpace(template), []byte("null")) {
			return fmt.Errorf("FingerprintHub corpus template %d is empty", index)
		}
		if err := validateFingerprintHubTemplate(template); err != nil {
			return fmt.Errorf("FingerprintHub corpus template %d is invalid: %w", index, err)
		}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("FingerprintHub corpus has trailing JSON content")
		}
		return fmt.Errorf("FingerprintHub corpus has trailing JSON content: %w", err)
	}
	return nil
}

// validateFingerprintHubTemplate mirrors the platform's minimum native import
// shape. The Engine does not reimplement Observer Ward parsing; it only keeps
// a syntactically non-empty but unusable aggregate from reaching Observer Ward,
// where zero templates can trigger its implicit updater behavior.
func validateFingerprintHubTemplate(payload json.RawMessage) error {
	fields, err := decodeFingerprintHubObject(payload)
	if err != nil {
		return err
	}
	if _, err := requiredFingerprintHubString(fields, "id"); err != nil {
		return err
	}
	infoPayload, ok := fields["info"]
	if !ok {
		return errors.New("info is required")
	}
	info, err := decodeFingerprintHubObject(infoPayload)
	if err != nil {
		return fmt.Errorf("info: %w", err)
	}
	if _, err := requiredFingerprintHubString(info, "name"); err != nil {
		return fmt.Errorf("info.%w", err)
	}
	httpPayload, ok := fields["http"]
	if !ok {
		return errors.New("http is required")
	}
	var requests []json.RawMessage
	if err := json.Unmarshal(httpPayload, &requests); err != nil || len(requests) == 0 {
		return errors.New("http must be a non-empty array")
	}
	for index, requestPayload := range requests {
		request, err := decodeFingerprintHubObject(requestPayload)
		if err != nil {
			return fmt.Errorf("http[%d]: %w", index, err)
		}
		matcherPayload, ok := request["matchers"]
		if !ok {
			return fmt.Errorf("http[%d].matchers is required", index)
		}
		var matchers []json.RawMessage
		if err := json.Unmarshal(matcherPayload, &matchers); err != nil || len(matchers) == 0 {
			return fmt.Errorf("http[%d].matchers must be a non-empty array", index)
		}
		for matcherIndex, matcherPayload := range matchers {
			matcher, err := decodeFingerprintHubObject(matcherPayload)
			if err != nil {
				return fmt.Errorf("http[%d].matchers[%d]: %w", index, matcherIndex, err)
			}
			if _, err := requiredFingerprintHubString(matcher, "type"); err != nil {
				return fmt.Errorf("http[%d].matchers[%d].%w", index, matcherIndex, err)
			}
		}
	}
	return nil
}

func decodeFingerprintHubObject(payload []byte) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil || len(fields) == 0 {
		return nil, errors.New("must be a non-empty object")
	}
	return fields, nil
}

func requiredFingerprintHubString(fields map[string]json.RawMessage, name string) (string, error) {
	payload, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return "", fmt.Errorf("%s is required", name)
	}
	var value string
	if err := json.Unmarshal(payload, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty string", name)
	}
	return value, nil
}
