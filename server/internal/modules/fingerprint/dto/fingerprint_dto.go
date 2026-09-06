// Package dto owns the versioned HTTP representation of fingerprint resources.
package dto

type ListQuery struct {
	PageSize  int    `form:"pageSize"`
	PageToken string `form:"pageToken"`
	Filter    string `form:"filter"`
	OrderBy   string `form:"orderBy"`
}

type FilterOptionsQuery struct {
	Field string `form:"field" binding:"required"`
}

// Fingerprint is the library-specific HTTP presentation matrix. The handler
// creates only the fields declared for the resource's library, so unrelated
// formats never leak null-shaped columns into another library's API response.
type Fingerprint map[string]any

type ListResponse struct {
	Results       []Fingerprint `json:"results"`
	NextPageToken string        `json:"nextPageToken"`
	TotalSize     int64         `json:"totalSize"`
}

type AdditionalField struct {
	Path  string `json:"path"`
	Value any    `json:"value"`
}

type ImportResponse struct {
	CreatedCount   int `json:"createdCount"`
	UpdatedCount   int `json:"updatedCount"`
	UnchangedCount int `json:"unchangedCount"`
}

type BatchDeleteRequest struct {
	Names []string `json:"names"`
}

type DeleteResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}

type LibraryStatistics struct {
	FingerPrintHub int64 `json:"fingerprinthub"`
}

// FingerprintImportDiagnostic is an AIP-193 typed Any payload. Position fields
// use one-based source coordinates and are omitted when a parser cannot know
// them; zero is never serialized as a fabricated source position.
type FingerprintImportDiagnostic struct {
	Type        string `json:"@type"`
	Kind        string `json:"kind"`
	Library     string `json:"library"`
	RecordIndex int    `json:"recordIndex,omitempty"`
	FieldPath   string `json:"fieldPath,omitempty"`
	Reason      string `json:"reason"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
}
