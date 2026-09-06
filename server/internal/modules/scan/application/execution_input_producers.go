package application

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
)

// ExecutionTarget remains the canonical planning projection. Target baseline
// expansion is intentionally owned by Engines and is never used by these
// finalized Scan-fact producers.
type ExecutionTarget struct {
	Resource string
	Type     string
	Value    string
}

const (
	ExecutionTargetTypeDomain = "domain"
	ExecutionTargetTypeIP     = "ip"
	ExecutionTargetTypeCIDR   = "cidr"
)

// DNSNameCursor is the finalized Scan snapshot projection for subdomains.
type DNSNameCursor interface {
	ForEachDNSNameByScanID(ctx context.Context, scanID int, visit func(string) error) error
}

// TargetDNSNameCursor streams the current Target inventory for subdomains.
type TargetDNSNameCursor interface {
	ForEachDNSNameByTargetID(ctx context.Context, targetID int, visit func(string) error) error
}

// HostPortEvidence is the complete persisted HostPort fact. It deliberately
// does not expose an effective-host or URL-derived projection.
type HostPortEvidence struct {
	Host string
	IP   string
	Port int
}

// HostPortCursor is the finalized Scan snapshot projection for hostPorts.
type HostPortCursor interface {
	ForEachHostPortByScanID(ctx context.Context, scanID int, visit func(HostPortEvidence) error) error
}

// TargetHostPortCursor streams the current Target inventory for hostPorts.
type TargetHostPortCursor interface {
	ForEachHostPortByTargetID(ctx context.Context, targetID int, visit func(HostPortEvidence) error) error
}

// WebsiteURLCursor is the finalized Scan snapshot projection for websiteURLs.
type WebsiteURLCursor interface {
	ForEachWebsiteURLByScanID(ctx context.Context, scanID int, visit func(string) error) error
}

// TargetWebsiteURLCursor streams the current Target inventory for websiteURLs.
type TargetWebsiteURLCursor interface {
	ForEachWebsiteURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error
}

// EndpointURLCursor is the finalized Scan snapshot projection for endpointURLs.
type EndpointURLCursor interface {
	ForEachEndpointURLByScanID(ctx context.Context, scanID int, visit func(string) error) error
}

// TargetEndpointURLCursor streams the current Target Endpoint URL inventory.
type TargetEndpointURLCursor interface {
	ForEachEndpointURLByTargetID(ctx context.Context, targetID int, visit func(string) error) error
}

type SubdomainsProducer func(emit func(string) error) error
type HostPortsProducer func(emit func(HostPortEvidence) error) error
type WebsiteURLsProducer func(emit func(string) error) error
type EndpointURLsProducer func(emit func(string) error) error

func NewSubdomainsProducer(ctx context.Context, scanID int, cursor DNSNameCursor, blacklist *ExecutionInputBlacklistFilter) (SubdomainsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("subdomains Scan snapshot cursor is required")
	}
	return newSubdomainsProducer(ctx, scanID, "Scan snapshot", cursor.ForEachDNSNameByScanID, blacklist)
}

// NewTargetInventorySubdomainsProducer uses the current Target asset cursor
// for one authorized input request. It deliberately does not retain a prior
// collection, so a retry observes the Target state at retry time.
func NewTargetInventorySubdomainsProducer(ctx context.Context, targetID int, cursor TargetDNSNameCursor, blacklist *ExecutionInputBlacklistFilter) (SubdomainsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("subdomains Target inventory cursor is required")
	}
	return newSubdomainsProducer(ctx, targetID, "Target inventory", cursor.ForEachDNSNameByTargetID, blacklist)
}

func newSubdomainsProducer(ctx context.Context, sourceID int, sourceName string, forEach func(context.Context, int, func(string) error) error, blacklist *ExecutionInputBlacklistFilter) (SubdomainsProducer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("subdomains producer context is required")
	}
	if sourceID <= 0 || forEach == nil {
		return nil, fmt.Errorf("subdomains %s cursor is required", sourceName)
	}
	if blacklist == nil {
		return nil, fmt.Errorf("subdomains blacklist filter is required")
	}
	return func(emit func(string) error) error {
		if emit == nil {
			return fmt.Errorf("subdomains emit callback is required")
		}
		return forEach(ctx, sourceID, func(value string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			excluded, err := blacklist.ShouldExcludeHostname(value)
			if err != nil {
				return err
			}
			if excluded {
				return nil
			}
			// Finalization owns canonicalization and ordering. The producer emits
			// the stored projection verbatim and does not repair historical rows.
			if err := emit(value); err != nil {
				return err
			}
			return blacklist.RecordEmitted()
		})
	}, nil
}

func NewHostPortsProducer(ctx context.Context, scanID int, cursor HostPortCursor, blacklist *ExecutionInputBlacklistFilter) (HostPortsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("hostPorts Scan snapshot cursor is required")
	}
	return newHostPortsProducer(ctx, scanID, "Scan snapshot", cursor.ForEachHostPortByScanID, blacklist)
}

// NewTargetInventoryHostPortsProducer uses the current Target asset cursor
// for one authorized input request.
func NewTargetInventoryHostPortsProducer(ctx context.Context, targetID int, cursor TargetHostPortCursor, blacklist *ExecutionInputBlacklistFilter) (HostPortsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("hostPorts Target inventory cursor is required")
	}
	return newHostPortsProducer(ctx, targetID, "Target inventory", cursor.ForEachHostPortByTargetID, blacklist)
}

func newHostPortsProducer(ctx context.Context, sourceID int, sourceName string, forEach func(context.Context, int, func(HostPortEvidence) error) error, blacklist *ExecutionInputBlacklistFilter) (HostPortsProducer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("hostPorts producer context is required")
	}
	if sourceID <= 0 || forEach == nil {
		return nil, fmt.Errorf("hostPorts %s cursor is required", sourceName)
	}
	if blacklist == nil {
		return nil, fmt.Errorf("hostPorts blacklist filter is required")
	}
	return func(emit func(HostPortEvidence) error) error {
		if emit == nil {
			return fmt.Errorf("hostPorts emit callback is required")
		}
		return forEach(ctx, sourceID, func(value HostPortEvidence) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			excluded, err := blacklist.ShouldExcludeHostPort(value.Host, value.IP)
			if err != nil {
				return err
			}
			if excluded {
				return nil
			}
			if err := emit(value); err != nil {
				return err
			}
			return blacklist.RecordEmitted()
		})
	}, nil
}

func NewWebsiteURLsProducer(ctx context.Context, scanID int, cursor WebsiteURLCursor, blacklist *ExecutionInputBlacklistFilter) (WebsiteURLsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("websiteURLs Scan snapshot cursor is required")
	}
	return newWebsiteURLsProducer(ctx, scanID, "Scan snapshot", cursor.ForEachWebsiteURLByScanID, blacklist)
}

// NewTargetInventoryWebsiteURLsProducer uses the current Target asset cursor
// for one authorized input request.
func NewTargetInventoryWebsiteURLsProducer(ctx context.Context, targetID int, cursor TargetWebsiteURLCursor, blacklist *ExecutionInputBlacklistFilter) (WebsiteURLsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("websiteURLs Target inventory cursor is required")
	}
	return newWebsiteURLsProducer(ctx, targetID, "Target inventory", cursor.ForEachWebsiteURLByTargetID, blacklist)
}

func newWebsiteURLsProducer(ctx context.Context, sourceID int, sourceName string, forEach func(context.Context, int, func(string) error) error, blacklist *ExecutionInputBlacklistFilter) (WebsiteURLsProducer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("websiteURLs producer context is required")
	}
	if sourceID <= 0 || forEach == nil {
		return nil, fmt.Errorf("websiteURLs %s cursor is required", sourceName)
	}
	if blacklist == nil {
		return nil, fmt.Errorf("websiteURLs blacklist filter is required")
	}
	return func(emit func(string) error) error {
		if emit == nil {
			return fmt.Errorf("websiteURLs emit callback is required")
		}
		return forEach(ctx, sourceID, func(value string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			// A selected record must already satisfy the line-product contract.
			// Do not repair it: a failed stream prevents Agent publication and
			// therefore cannot turn damaged stored evidence into a new input.
			if _, err := contractresults.ValidateObservedAssetURL(value); err != nil {
				return fmt.Errorf("websiteURLs %s record is invalid: %w", sourceName, err)
			}
			excluded, err := blacklist.ShouldExcludeURL(value)
			if err != nil {
				return err
			}
			if excluded {
				return nil
			}
			if err := emit(value); err != nil {
				return err
			}
			return blacklist.RecordEmitted()
		})
	}, nil
}

func NewEndpointURLsProducer(ctx context.Context, scanID int, cursor EndpointURLCursor, blacklist *ExecutionInputBlacklistFilter) (EndpointURLsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("endpointURLs Scan snapshot cursor is required")
	}
	return newEndpointURLsProducer(ctx, scanID, "Scan snapshot", cursor.ForEachEndpointURLByScanID, blacklist)
}

// NewTargetInventoryEndpointURLsProducer uses current Target endpoint rows for each request.
func NewTargetInventoryEndpointURLsProducer(ctx context.Context, targetID int, cursor TargetEndpointURLCursor, blacklist *ExecutionInputBlacklistFilter) (EndpointURLsProducer, error) {
	if cursor == nil {
		return nil, fmt.Errorf("endpointURLs Target inventory cursor is required")
	}
	return newEndpointURLsProducer(ctx, targetID, "Target inventory", cursor.ForEachEndpointURLByTargetID, blacklist)
}

func newEndpointURLsProducer(ctx context.Context, sourceID int, sourceName string, forEach func(context.Context, int, func(string) error) error, blacklist *ExecutionInputBlacklistFilter) (EndpointURLsProducer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("endpointURLs producer context is required")
	}
	if sourceID <= 0 || forEach == nil {
		return nil, fmt.Errorf("endpointURLs %s cursor is required", sourceName)
	}
	if blacklist == nil {
		return nil, fmt.Errorf("endpointURLs blacklist filter is required")
	}
	return func(emit func(string) error) error {
		if emit == nil {
			return fmt.Errorf("endpointURLs emit callback is required")
		}
		return forEach(ctx, sourceID, func(value string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if _, err := contractresults.ValidateObservedAssetURL(value); err != nil {
				return fmt.Errorf("endpointURLs %s record is invalid: %w", sourceName, err)
			}
			excluded, err := blacklist.ShouldExcludeURL(value)
			if err != nil {
				return err
			}
			if excluded {
				return nil
			}
			if err := emit(value); err != nil {
				return err
			}
			return blacklist.RecordEmitted()
		})
	}, nil
}

// WriteObservedURLLineStream serializes accepted observed URL values as the
// websiteURLs line product. It validates each value without rewriting it, so
// malformed historical rows fail production instead of being silently repaired.
func WriteObservedURLLineStream(ctx context.Context, writer io.Writer, produce func(func(string) error) error) (uint64, error) {
	return writeLFLineStream(ctx, writer, func(emit func(string) error) error {
		return produce(func(value string) error {
			if _, err := contractresults.ValidateObservedAssetURL(value); err != nil {
				return fmt.Errorf("websiteURLs line record is invalid: %w", err)
			}
			return emit(value)
		})
	})
}

// WriteCanonicalLineStream encodes a finalized text-line product with exact
// LF termination. It permits a legitimate zero-record product.
func WriteCanonicalLineStream(ctx context.Context, writer io.Writer, produce func(func(string) error) error) (recordCount uint64, err error) {
	return writeLFLineStream(ctx, writer, func(emit func(string) error) error {
		return produce(func(value string) error {
			if value == "" || strings.ContainsAny(value, "\r\n") {
				return fmt.Errorf("canonical line record must be non-empty and newline-safe")
			}
			return emit(value)
		})
	})
}

func writeLFLineStream(ctx context.Context, writer io.Writer, produce func(func(string) error) error) (recordCount uint64, err error) {
	if ctx == nil || writer == nil || produce == nil {
		return 0, fmt.Errorf("line stream inputs are required")
	}
	buffered := bufio.NewWriterSize(writer, 64*1024)
	err = produce(func(value string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := buffered.WriteString(value); err != nil {
			return err
		}
		if err := buffered.WriteByte('\n'); err != nil {
			return err
		}
		if recordCount == ^uint64(0) {
			return fmt.Errorf("canonical line record counter overflow")
		}
		recordCount++
		return nil
	})
	if err != nil {
		return recordCount, err
	}
	if err := buffered.Flush(); err != nil {
		return recordCount, err
	}
	return recordCount, nil
}

// WriteHostPortJSONL encodes complete HostPort facts with stable field order.
func WriteHostPortJSONL(ctx context.Context, writer io.Writer, produce HostPortsProducer) (recordCount uint64, err error) {
	if ctx == nil || writer == nil || produce == nil {
		return 0, fmt.Errorf("hostPorts JSONL inputs are required")
	}
	buffered := bufio.NewWriterSize(writer, 64*1024)
	err = produce(func(value HostPortEvidence) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Struct field declaration order is the wire order required by the
		// representation contract: host, ip, port.
		payload, err := json.Marshal(struct {
			Host string `json:"host"`
			IP   string `json:"ip"`
			Port int    `json:"port"`
		}{Host: value.Host, IP: value.IP, Port: value.Port})
		if err != nil {
			return err
		}
		if _, err := buffered.Write(payload); err != nil {
			return err
		}
		if err := buffered.WriteByte('\n'); err != nil {
			return err
		}
		if recordCount == ^uint64(0) {
			return fmt.Errorf("hostPorts record counter overflow")
		}
		recordCount++
		return nil
	})
	if err != nil {
		return recordCount, err
	}
	if err := buffered.Flush(); err != nil {
		return recordCount, err
	}
	return recordCount, nil
}
