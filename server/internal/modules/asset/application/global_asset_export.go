package application

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

const (
	GlobalAssetExportMaxRows  int64 = 100000
	GlobalAssetExportMaxBytes int64 = 256 << 20
	GlobalAssetExportGrantTTL       = 2 * time.Minute
)

var (
	ErrInvalidGlobalAssetExport = errors.New("invalid global asset export request")
	ErrGlobalAssetExportGrant   = errors.New("invalid or expired global asset export grant")
	ErrGlobalAssetExportRows    = errors.New("global asset export row limit exceeded")
	ErrGlobalAssetExportBytes   = errors.New("global asset export byte limit exceeded")
)

// GlobalAssetExportGrant is the Redis value bound to one submitted search.
type GlobalAssetExportGrant struct {
	UserID       int                        `json:"userId"`
	TokenVersion int                        `json:"tokenVersion"`
	AssetType    GlobalAssetSearchAssetType `json:"assetType"`
	Query        string                     `json:"query"`
	AST          GlobalAssetSearchAST       `json:"ast"`
	CreatedAt    time.Time                  `json:"createdAt"`
	ExpiresAt    time.Time                  `json:"expiresAt"`
}

// GlobalAssetExportGrantStore owns opaque, one-time download grants.
type GlobalAssetExportGrantStore interface {
	Issue(context.Context, GlobalAssetExportGrant, time.Duration) (string, error)
	Consume(context.Context, string) (*GlobalAssetExportGrant, error)
}

// GlobalAssetExportService validates submitted searches and issues grants.
type GlobalAssetExportService struct {
	grants GlobalAssetExportGrantStore
	ttl    time.Duration
	now    func() time.Time
}

func NewGlobalAssetExportService(grants GlobalAssetExportGrantStore) *GlobalAssetExportService {
	return &GlobalAssetExportService{grants: grants, ttl: GlobalAssetExportGrantTTL, now: time.Now}
}

func (service *GlobalAssetExportService) Issue(ctx context.Context, userID, tokenVersion int, query string, assetType GlobalAssetSearchAssetType) (string, time.Time, error) {
	if service == nil || service.grants == nil || userID <= 0 || tokenVersion < 0 || !isGlobalAssetSearchAssetType(assetType) {
		return "", time.Time{}, ErrInvalidGlobalAssetExport
	}
	ast, err := ParseGlobalAssetSearchQuery(query)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %v", ErrInvalidGlobalAssetExport, err)
	}
	if ctx == nil {
		return "", time.Time{}, context.Canceled
	}
	now := service.now().UTC()
	expiresAt := now.Add(service.ttl)
	normalizedQuery := strings.TrimSpace(query)
	token, err := service.grants.Issue(ctx, GlobalAssetExportGrant{
		UserID: userID, TokenVersion: tokenVersion, AssetType: assetType,
		Query: normalizedQuery, AST: ast, CreatedAt: now, ExpiresAt: expiresAt,
	}, service.ttl)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (service *GlobalAssetExportService) Consume(ctx context.Context, token string) (*GlobalAssetExportGrant, error) {
	if service == nil || service.grants == nil || token == "" {
		return nil, ErrGlobalAssetExportGrant
	}
	grant, err := service.grants.Consume(ctx, token)
	if err != nil || grant == nil || grant.UserID <= 0 || !isGlobalAssetSearchAssetType(grant.AssetType) || !grant.ExpiresAt.After(service.now().UTC()) {
		return nil, ErrGlobalAssetExportGrant
	}
	return grant, nil
}

// NewOpaqueExportToken returns an unguessable URL token.
func NewOpaqueExportToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// MarshalExportGrant is kept in application so storage implementations cannot
// silently choose a second grant schema.
func MarshalExportGrant(grant GlobalAssetExportGrant) ([]byte, error) { return json.Marshal(grant) }

// GlobalAssetExportSnapshot is a single repeatable-read transaction.
type GlobalAssetExportSnapshot interface {
	ForEachWebsite(context.Context, func(assetdomain.Website) error) error
	ForEachEndpoint(context.Context, func(assetdomain.Endpoint) error) error
}

// GlobalAssetExportSnapshotStore opens one snapshot and commits it only after
// both encoding passes have completed.
type GlobalAssetExportSnapshotStore interface {
	WithSnapshot(context.Context, GlobalAssetSearchAST, GlobalAssetSearchAssetType, func(GlobalAssetExportSnapshot) error) error
}

// GlobalAssetExportServiceRunner performs the deterministic two-pass encoding.
type GlobalAssetExportServiceRunner struct {
	store GlobalAssetExportSnapshotStore
}

func NewGlobalAssetExportServiceRunner(store GlobalAssetExportSnapshotStore) *GlobalAssetExportServiceRunner {
	return &GlobalAssetExportServiceRunner{store: store}
}

// Export preflights without writing, then invokes begin exactly once and streams
// the same snapshot through a bounded tail writer.
func (service *GlobalAssetExportServiceRunner) Export(ctx context.Context, ast GlobalAssetSearchAST, assetType GlobalAssetSearchAssetType, begin func(int64) (io.Writer, error)) error {
	if service == nil || service.store == nil || begin == nil {
		return ErrInvalidGlobalAssetExport
	}
	var pending *exportTailWriter
	err := service.store.WithSnapshot(ctx, ast, assetType, func(snapshot GlobalAssetExportSnapshot) error {
		count, size, err := countExportRows(snapshot, ctx, assetType)
		if err != nil {
			return err
		}
		if count > GlobalAssetExportMaxRows {
			return ErrGlobalAssetExportRows
		}
		if size > GlobalAssetExportMaxBytes {
			return ErrGlobalAssetExportBytes
		}
		output, err := begin(size)
		if err != nil {
			return err
		}
		pending = newExportTailWriter(output, 64<<10)
		rows, bytes, err := streamExportRows(snapshot, ctx, assetType, pending, size)
		if err != nil {
			return err
		}
		if rows != count || bytes != size {
			return fmt.Errorf("global asset export snapshot changed during encoding")
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if pending == nil {
		return errors.New("global asset export did not create output")
	}
	if err := pending.Commit(); err != nil {
		return err
	}
	return nil
}

func countExportRows(snapshot GlobalAssetExportSnapshot, ctx context.Context, assetType GlobalAssetSearchAssetType) (int64, int64, error) {
	var rows int64
	counter := &exportCountingWriter{limit: GlobalAssetExportMaxBytes}
	if err := writeCSVExport(snapshot, ctx, assetType, counter, &rows); err != nil {
		return 0, 0, err
	}
	return rows, counter.n, nil
}

func streamExportRows(snapshot GlobalAssetExportSnapshot, ctx context.Context, assetType GlobalAssetSearchAssetType, output io.Writer, expected int64) (int64, int64, error) {
	var rows int64
	counter := &exportCountingWriter{writer: output, limit: expected}
	if err := writeCSVExport(snapshot, ctx, assetType, counter, &rows); err != nil {
		return 0, 0, err
	}
	return rows, counter.n, nil
}

type exportCountingWriter struct {
	writer io.Writer
	limit  int64
	n      int64
}

func (writer *exportCountingWriter) Write(data []byte) (int, error) {
	if int64(len(data))+writer.n > writer.limit {
		return 0, ErrGlobalAssetExportBytes
	}
	writer.n += int64(len(data))
	if writer.writer == nil {
		return len(data), nil
	}
	written, err := writer.writer.Write(data)
	if err == nil && written != len(data) {
		return written, io.ErrShortWrite
	}
	return written, err
}

type exportTailWriter struct {
	writer io.Writer
	buffer []byte
	limit  int
}

func newExportTailWriter(writer io.Writer, limit int) *exportTailWriter {
	return &exportTailWriter{writer: writer, limit: limit, buffer: make([]byte, 0, limit)}
}

func (writer *exportTailWriter) Write(data []byte) (int, error) {
	if len(data) >= writer.limit {
		if _, err := writer.flush(writer.buffer); err != nil {
			return 0, err
		}
		prefix := data[:len(data)-writer.limit]
		if _, err := writer.flush(prefix); err != nil {
			return 0, err
		}
		writer.buffer = append(writer.buffer[:0], data[len(data)-writer.limit:]...)
		return len(data), nil
	}
	writer.buffer = append(writer.buffer, data...)
	if len(writer.buffer) <= writer.limit {
		return len(data), nil
	}
	cut := len(writer.buffer) - writer.limit
	if _, err := writer.flush(writer.buffer[:cut]); err != nil {
		return 0, err
	}
	copy(writer.buffer, writer.buffer[cut:])
	writer.buffer = writer.buffer[:writer.limit]
	return len(data), nil
}

func (writer *exportTailWriter) flush(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	written, err := writer.writer.Write(data)
	if err == nil && written != len(data) {
		return written, io.ErrShortWrite
	}
	return written, err
}

func (writer *exportTailWriter) Commit() error {
	if len(writer.buffer) == 0 {
		return nil
	}
	_, err := writer.flush(writer.buffer)
	return err
}

var globalAssetExportHeaders = []string{
	"id", "target_id", "url", "host", "location", "title", "status_code",
	"content_length", "content_type", "webserver", "tech", "response_body",
	"response_headers", "vhost", "created_at",
}

func writeCSVExport(snapshot GlobalAssetExportSnapshot, ctx context.Context, assetType GlobalAssetSearchAssetType, output io.Writer, rows *int64) error {
	if _, err := output.Write(csvUTF8BOM); err != nil {
		return err
	}
	writer := csv.NewWriter(output)
	if err := writer.Write(globalAssetExportHeaders); err != nil {
		return err
	}
	write := func(fields []string) error {
		if err := writer.Write(fields); err != nil {
			return err
		}
		(*rows)++
		if *rows > GlobalAssetExportMaxRows {
			return ErrGlobalAssetExportRows
		}
		return nil
	}
	var err error
	switch assetType {
	case GlobalAssetSearchAssetTypeWebsite:
		err = snapshot.ForEachWebsite(ctx, func(item assetdomain.Website) error { return write(globalWebsiteExportRow(item)) })
	case GlobalAssetSearchAssetTypeEndpoint:
		err = snapshot.ForEachEndpoint(ctx, func(item assetdomain.Endpoint) error { return write(globalEndpointExportRow(item)) })
	default:
		return ErrInvalidGlobalAssetExport
	}
	if err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

var csvUTF8BOM = []byte{0xEF, 0xBB, 0xBF}

func globalWebsiteExportRow(item assetdomain.Website) []string {
	return []string{
		strconv.Itoa(item.ID), strconv.Itoa(item.TargetID), item.URL, item.Host, item.Location,
		item.Title, optionalInt64(item.StatusCode), optionalInt64(item.ContentLength), item.ContentType,
		item.Webserver, strings.Join(item.Tech, "|"), item.ResponseBody, item.ResponseHeaders,
		optionalBool(item.Vhost), timeutil.FormatRFC3339NanoUTC(item.CreatedAt),
	}
}

func globalEndpointExportRow(item assetdomain.Endpoint) []string {
	return []string{
		strconv.Itoa(item.ID), strconv.Itoa(item.TargetID), item.URL, item.Host, item.Location,
		item.Title, optionalInt64(item.StatusCode), optionalInt64(item.ContentLength), item.ContentType,
		item.Webserver, strings.Join(item.Tech, "|"), item.ResponseBody, item.ResponseHeaders,
		optionalBool(item.Vhost), timeutil.FormatRFC3339NanoUTC(item.CreatedAt),
	}
}

func optionalInt64(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func optionalBool(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}
