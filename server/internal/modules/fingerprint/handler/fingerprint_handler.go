// Package handler adapts the fingerprint AIP HTTP boundary to its application
// facade. It owns transport limits and typed import diagnostics, not rules.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	fingerprintapp "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	fingerprintdto "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

const (
	MaxFingerprintFileSizeBytes    int64 = 30 * 1024 * 1024
	maxFingerprintRequestSizeBytes int64 = 50 * 1024 * 1024
	fingerprintImportDiagnosticURL       = "type.googleapis.com/lunafox.v1.FingerprintImportDiagnostic"
)

var (
	fingerprintHTTPArtifactMeter     = otel.Meter("lunafox.fingerprint.artifacts")
	fingerprintHTTPArtifactTransfers = mustHTTPArtifactCounter("fingerprint_artifact_http_transfers_total")
	fingerprintHTTPArtifactBytes     = mustHTTPArtifactCounter("fingerprint_artifact_http_bytes_total")
	fingerprintHTTPArtifactDuration  = mustHTTPArtifactHistogram("fingerprint_artifact_http_transfer_duration_seconds")
)

func mustHTTPArtifactCounter(name string) metric.Int64Counter {
	counter, err := fingerprintHTTPArtifactMeter.Int64Counter(name)
	if err != nil {
		return metricnoop.Int64Counter{}
	}
	return counter
}

func mustHTTPArtifactHistogram(name string) metric.Float64Histogram {
	histogram, err := fingerprintHTTPArtifactMeter.Float64Histogram(name)
	if err != nil {
		return metricnoop.Float64Histogram{}
	}
	return histogram
}

type FingerprintHandler struct {
	facade *fingerprintapp.Facade
}

func NewFingerprintHandler(facade *fingerprintapp.Facade) *FingerprintHandler {
	return &FingerprintHandler{facade: facade}
}

// GET /v1/fingerprintLibraries/{library}/fingerprints
func (handler *FingerprintHandler) List(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	var query fingerprintdto.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid fingerprint list query.")
		return
	}
	result, err := handler.facade.List(c.Request.Context(), library, fingerprintapp.ListInput{
		PageSize:  query.PageSize,
		PageToken: query.PageToken,
		Filter:    query.Filter,
		OrderBy:   query.OrderBy,
	})
	if err != nil {
		handler.writeApplicationError(c, library, err)
		return
	}
	results := make([]fingerprintdto.Fingerprint, 0, len(result.Records))
	for _, record := range result.Records {
		response, err := fingerprintResponse(record, false)
		if err != nil {
			handler.writePresentationError(c, record, err)
			return
		}
		results = append(results, response)
	}
	c.JSON(http.StatusOK, fingerprintdto.ListResponse{Results: results, NextPageToken: result.NextPageToken, TotalSize: result.TotalSize})
}

// FilterOptions returns a complete-library aggregation for one approved
// facet. GET /v1/fingerprintLibraries/{library}/fingerprints/filterOptions?field={field}
func (handler *FingerprintHandler) FilterOptions(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	var query fingerprintdto.FilterOptionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid fingerprint filter-options query.")
		return
	}
	options, err := handler.facade.ListFilterOptions(c.Request.Context(), library, query.Field)
	if err != nil {
		handler.writeApplicationError(c, library, err)
		return
	}
	results := make([]httpdto.FilterOption, 0, len(options))
	for _, option := range options {
		results = append(results, httpdto.FilterOption{Value: option.Value, Label: option.Label, Count: option.Count})
	}
	httpdto.Success(c, httpdto.FilterOptionsResponse{Results: results})
}

// GET /v1/fingerprintLibraries/{library}/fingerprints/{fingerprint}
func (handler *FingerprintHandler) Get(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	resourceID := strings.TrimSpace(c.Param("fingerprint"))
	if _, _, err := domain.ParseCanonicalName(library.CanonicalName(resourceID)); err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid fingerprint resource name.")
		return
	}
	record, err := handler.facade.Get(c.Request.Context(), library, resourceID)
	if err != nil {
		handler.writeApplicationError(c, library, err)
		return
	}
	response, err := fingerprintResponse(*record, true)
	if err != nil {
		handler.writePresentationError(c, *record, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

// POST /v1/fingerprintLibraries/{library}:import
func (handler *FingerprintHandler) Import(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	contents, err := readFingerprintUpload(c, library)
	if err != nil {
		handler.writeImportError(c, library, err)
		return
	}
	result, err := handler.facade.Import(c.Request.Context(), library, contents)
	if err != nil {
		var diagnostic *domain.ImportDiagnosticError
		if errors.As(err, &diagnostic) {
			handler.writeImportDiagnostic(c, library, http.StatusBadRequest, "FINGERPRINT_IMPORT_INVALID", diagnostic)
			return
		}
		// Storage failures never expose SQL or internal identity details.
		httpdto.InternalError(c, "Fingerprint import failed.")
		return
	}
	c.JSON(http.StatusOK, fingerprintdto.ImportResponse{
		CreatedCount:   result.CreatedCount,
		UpdatedCount:   result.UpdatedCount,
		UnchangedCount: result.UnchangedCount,
	})
}

// POST /v1/fingerprintLibraries/{library}/fingerprints:batchDelete
func (handler *FingerprintHandler) BatchDelete(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	var request fingerprintdto.BatchDeleteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid fingerprint delete request.")
		return
	}
	deletedCount, err := handler.facade.Delete(c.Request.Context(), library, request.Names)
	if err != nil {
		handler.writeApplicationError(c, library, err)
		return
	}
	c.JSON(http.StatusOK, fingerprintdto.DeleteResponse{DeletedCount: deletedCount})
}

// POST /v1/fingerprintLibraries/{library}:clear
func (handler *FingerprintHandler) Clear(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	if !isEmptyClearRequestBody(c.Request.Body) {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Fingerprint clear request must not contain fields.")
		return
	}
	deletedCount, err := handler.facade.Clear(c.Request.Context(), library)
	if err != nil {
		handler.writeApplicationError(c, library, err)
		return
	}
	c.JSON(http.StatusOK, fingerprintdto.DeleteResponse{DeletedCount: deletedCount})
}

// isEmptyClearRequestBody rejects accidental selected-delete payloads. A clear
// action is destructive for the entire library, so it may only carry no body
// or an empty JSON object; ignored fields would otherwise widen a deletion.
func isEmptyClearRequestBody(body io.Reader) bool {
	if body == nil {
		return true
	}
	decoder := json.NewDecoder(body)
	firstToken, err := decoder.Token()
	if errors.Is(err, io.EOF) {
		return true
	}
	openingDelimiter, ok := firstToken.(json.Delim)
	if err != nil || !ok || openingDelimiter != '{' || decoder.More() {
		return false
	}
	closingDelimiter, err := decoder.Token()
	if err != nil || closingDelimiter != json.Delim('}') {
		return false
	}
	var trailing any
	return errors.Is(decoder.Decode(&trailing), io.EOF)
}

// GET /v1/fingerprintLibraries/{library}/exportFiles/current
func (handler *FingerprintHandler) Export(c *gin.Context) {
	library, ok := handler.libraryFromContext(c)
	if !ok {
		return
	}
	if len(c.Request.URL.Query()) != 0 {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Fingerprint export does not support selection parameters.")
		return
	}
	artifact, err := handler.facade.OpenCurrentArtifact(c.Request.Context(), library)
	if err != nil {
		handler.writeApplicationError(c, library, err)
		return
	}
	defer artifact.Reader.Close()
	startedAt := time.Now()
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": artifact.Descriptor.Filename}))
	c.Header("ETag", fmt.Sprintf("\"%s\"", artifact.Descriptor.SHA256Digest))
	c.Header("Content-Length", fmt.Sprintf("%d", artifact.Descriptor.SizeBytes))
	c.DataFromReader(http.StatusOK, artifact.Descriptor.SizeBytes, artifact.Descriptor.ContentType, artifact.Reader, nil)
	attributes := metric.WithAttributes(attribute.String("fingerprint.library", string(library)), attribute.String("fingerprint.artifact.transport", "http"), attribute.String("fingerprint.artifact.outcome", "completed"))
	fingerprintHTTPArtifactTransfers.Add(c.Request.Context(), 1, attributes)
	fingerprintHTTPArtifactBytes.Add(c.Request.Context(), artifact.Descriptor.SizeBytes, attributes)
	fingerprintHTTPArtifactDuration.Record(c.Request.Context(), time.Since(startedAt).Seconds(), attributes)
	pkg.Info("fingerprint artifact HTTP transfer completed", zap.String("fingerprint.library", string(library)), zap.String("fingerprint.artifact.transport", "http"), zap.String("fingerprint.artifact.outcome", "completed"), zap.Int64("fingerprint.artifact.bytes", artifact.Descriptor.SizeBytes))
}

// GET /v1/fingerprintLibraryStatistics
func (handler *FingerprintHandler) Statistics(c *gin.Context) {
	statistics, err := handler.facade.Statistics(c.Request.Context())
	if err != nil {
		httpdto.InternalError(c, "Fingerprint statistics failed.")
		return
	}
	c.JSON(http.StatusOK, fingerprintdto.LibraryStatistics{
		FingerPrintHub: statistics.FingerPrintHub,
	})
}

func (handler *FingerprintHandler) libraryFromContext(c *gin.Context) (domain.Library, bool) {
	library, err := domain.ParseLibrary(c.Param("library"))
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Unsupported fingerprint library.")
		return "", false
	}
	return library, true
}

func (handler *FingerprintHandler) writeApplicationError(c *gin.Context, library domain.Library, err error) {
	switch {
	case errors.Is(err, fingerprintapp.ErrFingerprintNotFound):
		httpdto.NotFound(c, "Fingerprint not found.")
	case errors.Is(err, fingerprintapp.ErrInvalidFingerprintPageToken),
		errors.Is(err, fingerprintapp.ErrUnsupportedFingerprintFilter),
		errors.Is(err, fingerprintapp.ErrUnsupportedFingerprintFacet),
		errors.Is(err, fingerprintapp.ErrUnsupportedFingerprintOrder),
		errors.Is(err, fingerprintapp.ErrInvalidBatchDelete):
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid fingerprint request.")
	default:
		httpdto.InternalError(c, "Fingerprint request failed.")
	}
}

type fingerprintUploadError struct {
	status     int
	reason     string
	diagnostic *domain.ImportDiagnosticError
}

func (err *fingerprintUploadError) Error() string {
	return err.reason
}

func readFingerprintUpload(c *gin.Context, library domain.Library) ([]byte, error) {
	// A declared oversized request must fail before multipart parsing can return
	// a smaller file-level error or invoke the synchronization facade.
	if c.Request.ContentLength > maxFingerprintRequestSizeBytes {
		return nil, uploadError(http.StatusRequestEntityTooLarge, "FINGERPRINT_IMPORT_REQUEST_TOO_LARGE", domain.DiagnosticTransport, "REQUEST_TOO_LARGE")
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return nil, uploadError(http.StatusUnsupportedMediaType, "FINGERPRINT_IMPORT_UNSUPPORTED_MEDIA_TYPE", domain.DiagnosticTransport, "UNSUPPORTED_MEDIA_TYPE")
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFingerprintRequestSizeBytes)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, uploadError(http.StatusBadRequest, "FINGERPRINT_IMPORT_INVALID", domain.DiagnosticTransport, "MULTIPART_INVALID")
	}

	var contents []byte
	seenFile := false
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			if errors.Is(nextErr, http.ErrBodyReadAfterClose) || strings.Contains(nextErr.Error(), "request body too large") {
				return nil, uploadError(http.StatusRequestEntityTooLarge, "FINGERPRINT_IMPORT_REQUEST_TOO_LARGE", domain.DiagnosticTransport, "REQUEST_TOO_LARGE")
			}
			return nil, uploadError(http.StatusBadRequest, "FINGERPRINT_IMPORT_INVALID", domain.DiagnosticTransport, "MULTIPART_INVALID")
		}
		if part.FormName() != "file" || part.FileName() == "" || seenFile {
			_ = part.Close()
			return nil, uploadError(http.StatusBadRequest, "FINGERPRINT_IMPORT_INVALID", domain.DiagnosticTransport, "FILE_PART_INVALID")
		}
		seenFile = true
		if !isAllowedUploadMIME(library, part.Header.Get("Content-Type")) {
			_ = part.Close()
			return nil, uploadError(http.StatusUnsupportedMediaType, "FINGERPRINT_IMPORT_UNSUPPORTED_MEDIA_TYPE", domain.DiagnosticTransport, "UNSUPPORTED_MEDIA_TYPE")
		}
		limited := io.LimitReader(part, MaxFingerprintFileSizeBytes+1)
		contents, err = io.ReadAll(limited)
		_ = part.Close()
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				return nil, uploadError(http.StatusRequestEntityTooLarge, "FINGERPRINT_IMPORT_REQUEST_TOO_LARGE", domain.DiagnosticTransport, "REQUEST_TOO_LARGE")
			}
			return nil, uploadError(http.StatusBadRequest, "FINGERPRINT_IMPORT_INVALID", domain.DiagnosticTransport, "FILE_READ_FAILED")
		}
		if int64(len(contents)) > MaxFingerprintFileSizeBytes {
			return nil, uploadError(http.StatusRequestEntityTooLarge, "FINGERPRINT_IMPORT_FILE_TOO_LARGE", domain.DiagnosticTransport, "FILE_TOO_LARGE")
		}
	}
	if !seenFile || len(contents) == 0 {
		return nil, uploadError(http.StatusBadRequest, "FINGERPRINT_IMPORT_INVALID", domain.DiagnosticTransport, "FILE_REQUIRED")
	}
	return contents, nil
}

func uploadError(status int, reason, kind, diagnosticReason string) *fingerprintUploadError {
	return &fingerprintUploadError{
		status: status,
		reason: reason,
		diagnostic: &domain.ImportDiagnosticError{
			Kind:   kind,
			Reason: diagnosticReason,
		},
	}
}

func isAllowedUploadMIME(library domain.Library, raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(mediaType)
	if mediaType == "application/octet-stream" {
		return true
	}
	return mediaType == "application/json"
}

func (handler *FingerprintHandler) writeImportError(c *gin.Context, library domain.Library, err error) {
	var upload *fingerprintUploadError
	if errors.As(err, &upload) {
		handler.writeImportDiagnostic(c, library, upload.status, upload.reason, upload.diagnostic)
		return
	}
	httpdto.InternalError(c, "Fingerprint import failed.")
}

func (handler *FingerprintHandler) writeImportDiagnostic(c *gin.Context, library domain.Library, status int, errorReason string, diagnostic *domain.ImportDiagnosticError) {
	metadata := map[string]string{"library": string(library)}
	if status == http.StatusRequestEntityTooLarge && errorReason == "FINGERPRINT_IMPORT_FILE_TOO_LARGE" {
		metadata["maxFileSizeBytes"] = fmt.Sprintf("%d", MaxFingerprintFileSizeBytes)
	}
	httpdto.ErrorWithTypedDetails(c, status, errorReason, "Fingerprint import validation failed.", metadata, fingerprintdto.FingerprintImportDiagnostic{
		Type:        fingerprintImportDiagnosticURL,
		Kind:        diagnostic.Kind,
		Library:     string(library),
		RecordIndex: diagnostic.RecordIndex,
		FieldPath:   diagnostic.FieldPath,
		Reason:      diagnostic.Reason,
		Line:        diagnostic.Line,
		Column:      diagnostic.Column,
	})
}

func fingerprintResponse(record domain.PersistedRecord, includeDetails bool) (fingerprintdto.Fingerprint, error) {
	presentation, err := domain.PresentFingerprint(record)
	if err != nil {
		return nil, err
	}
	response := fingerprintdto.Fingerprint{
		"name":      record.CanonicalName(),
		"createdAt": record.CreatedAt.Format(time.RFC3339Nano),
	}
	for field, value := range presentation.Fields {
		response[field] = value
	}
	if includeDetails {
		additionalFields := make([]fingerprintdto.AdditionalField, 0, len(presentation.AdditionalFields))
		for _, field := range presentation.AdditionalFields {
			additionalFields = append(additionalFields, fingerprintdto.AdditionalField{
				Path:  field.Path,
				Value: field.Value,
			})
		}
		response["updatedAt"] = record.UpdatedAt.Format(time.RFC3339Nano)
		response["additionalFields"] = additionalFields
	}
	return response, nil
}

func (handler *FingerprintHandler) writePresentationError(c *gin.Context, record domain.PersistedRecord, err error) {
	pkg.Error("Fingerprint presentation failed",
		zap.String("fingerprint.library", string(record.Library)),
		zap.String("fingerprint.name", record.CanonicalName()),
		zap.Error(err),
	)
	httpdto.InternalError(c, "Fingerprint request failed.")
}
