package results

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"unicode/utf8"
)

const (
	ResultKindAssetSubdomain         = "asset.subdomain.v1"
	ResultKindAssetHostPort          = "asset.host_port.v1"
	ResultKindAssetWebsite           = "asset.website.v1"
	ResultKindAssetWebsiteTechnology = "asset.website_technology.v1"
	ResultKindAssetEndpoint          = "asset.endpoint.v1"
	ResultKindAssetScreenshot        = "asset.screenshot.v1"
	ResultKindAssetDirectory         = "asset.directory.v1"
	ResultKindAssetVulnerability     = "asset.vulnerability.v1"

	ResultEncoderAssetSubdomain         = "asset.subdomain.v1.canonical-json"
	ResultEncoderAssetHostPort          = "asset.host_port.v1.canonical-json"
	ResultEncoderAssetWebsite           = "asset.website.v1.canonical-json"
	ResultEncoderAssetWebsiteTechnology = "asset.website_technology.v1.canonical-json"
	ResultEncoderAssetEndpoint          = "asset.endpoint.v1.canonical-json"
	ResultEncoderAssetScreenshot        = "asset.screenshot.v1.canonical-json"
	ResultEncoderAssetDirectory         = "asset.directory.v1.canonical-json"
	ResultEncoderAssetVulnerability     = "asset.vulnerability.v1.canonical-json"
)

type Descriptor struct {
	// ResultType is the explicit wire result_type value. It is never inferred
	// from an Engine, package, Go item type, or JSON shape.
	ResultType string
	// SchemaRef identifies the Server-owned item contract used for validation.
	SchemaRef string
	// AuthorResultsFieldName, ItemTypeName, and GoEncoderName are passive Go
	// generation inputs for the shared typed Results surface. The generated
	// submitter type is derived from ItemTypeName as <ItemTypeName>Submitter,
	// keeping the collection field separate from its submission capability.
	// EncoderID is the explicit language-neutral canonical encoder contract; no
	// consumer may infer either encoder identity from the Go item type or JSON
	// shape. These fields do not grant submission authority.
	AuthorResultsFieldName string
	ItemTypeName           string
	EncoderID              string
	GoEncoderName          string
}

// FieldPresence describes whether a JSON member has to be present on the
// wire. Optional fields are omitted when their Go zero value is encoded.
// Required fields are checked by the generated Schema-basics encoder before
// the item reaches the ResultSink.
type FieldPresence string

const (
	FieldPresenceRequired FieldPresence = "required"
	FieldPresenceOptional FieldPresence = "optional"
)

// FieldSchemaBasics is the deliberately small set of immutable checks that an
// Engine-local generated encoder may perform. Cross-field, target-scoped and
// persistence rules remain owned by the Server result validator.
type FieldSchemaBasics struct {
	ValidUTF8      bool
	NonEmpty       bool
	ExplicitArray  bool
	ExplicitObject bool
	HTTPURL        bool
	NoURLUserInfo  bool
	NonNegative    bool
	Trimmed        bool
	MinInt         *int64
	MaxInt         *int64
	MinNumber      *float64
	MaxNumber      *float64
	MaxBytes       int
	MaxRunes       int
	EnumValues     []string
}

// FieldDescriptor is the source metadata for one generated local item field.
// GoType is intentionally explicit: the generator must not infer a contract
// from Go AST layout or from the Server's validator implementation.
type FieldDescriptor struct {
	GoName       string
	JSONName     string
	GoType       string
	Presence     FieldPresence
	SchemaBasics FieldSchemaBasics
}

// CodegenDescriptor combines the passive registry identity with the complete
// field-level metadata consumed by engine-manifest-artifact-gen.
type CodegenDescriptor struct {
	Descriptor
	ClosedObject bool
	Fields       []FieldDescriptor
}

type Subdomain struct {
	DNSName string `json:"dnsName"`
}

type HostPort struct {
	Host string `json:"host"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type Website struct {
	URL             string   `json:"url"`
	Host            string   `json:"host,omitempty"`
	Title           string   `json:"title,omitempty"`
	StatusCode      *int     `json:"statusCode,omitempty"`
	ContentLength   *int     `json:"contentLength,omitempty"`
	Location        string   `json:"location,omitempty"`
	Webserver       string   `json:"webserver,omitempty"`
	ContentType     string   `json:"contentType,omitempty"`
	Tech            []string `json:"tech,omitempty"`
	ResponseBody    string   `json:"responseBody,omitempty"`
	Vhost           *bool    `json:"vhost,omitempty"`
	ResponseHeaders string   `json:"responseHeaders,omitempty"`
}

// WebsiteTechnology is a complete, current-only technology observation. It
// intentionally has no Host or mutable Website fields: the Server owns the
// Target-scoped upsert and replaces only Website.tech.
type WebsiteTechnology struct {
	URL  string   `json:"url"`
	Tech []string `json:"tech"`
}

// Endpoint is URL Collection evidence. URL stays exactly as observed; Host is
// a separate required assertion validated against the raw URL authority.
type Endpoint struct {
	URL                      string   `json:"url"`
	Host                     string   `json:"host"`
	Title                    string   `json:"title,omitempty"`
	StatusCode               *int     `json:"statusCode,omitempty"`
	ContentLength            *int     `json:"contentLength,omitempty"`
	Location                 string   `json:"location,omitempty"`
	Webserver                string   `json:"webserver,omitempty"`
	ContentType              string   `json:"contentType,omitempty"`
	Tech                     []string `json:"tech,omitempty"`
	ResponseBody             string   `json:"responseBody,omitempty"`
	ResponseBodyTruncated    bool     `json:"responseBodyTruncated,omitempty"`
	Vhost                    *bool    `json:"vhost,omitempty"`
	ResponseHeaders          string   `json:"responseHeaders,omitempty"`
	ResponseHeadersTruncated bool     `json:"responseHeadersTruncated,omitempty"`
}

// Screenshot is a viewport screenshot observation. URL stays exactly as
// observed. Image contains encoded WebP bytes and is standard base64 in JSON.
type Screenshot struct {
	URL        string `json:"url"`
	StatusCode *int   `json:"statusCode,omitempty"`
	Image      []byte `json:"image"`
}

// canonicalDescriptors is the single passive descriptor source shared by
// Server validation and the typed Results generator. Keep its order stable so
// generated output does not depend on map iteration.
var canonicalDescriptors = [...]Descriptor{
	{
		ResultType:             ResultKindAssetSubdomain,
		SchemaRef:              "schema.result.asset.subdomain.v1",
		AuthorResultsFieldName: "Subdomains",
		ItemTypeName:           "Subdomain",
		EncoderID:              ResultEncoderAssetSubdomain,
		GoEncoderName:          "EncodeSubdomain",
	},
	{
		ResultType:             ResultKindAssetEndpoint,
		SchemaRef:              "schema.result.asset.endpoint.v1",
		AuthorResultsFieldName: "Endpoints",
		ItemTypeName:           "Endpoint",
		EncoderID:              ResultEncoderAssetEndpoint,
		GoEncoderName:          "EncodeEndpoint",
	},
	{
		ResultType:             ResultKindAssetHostPort,
		SchemaRef:              "schema.result.asset.host_port.v1",
		AuthorResultsFieldName: "HostPorts",
		ItemTypeName:           "HostPort",
		EncoderID:              ResultEncoderAssetHostPort,
		GoEncoderName:          "EncodeHostPort",
	},
	{
		ResultType:             ResultKindAssetWebsite,
		SchemaRef:              "schema.result.asset.website.v1",
		AuthorResultsFieldName: "Websites",
		ItemTypeName:           "Website",
		EncoderID:              ResultEncoderAssetWebsite,
		GoEncoderName:          "EncodeWebsite",
	},
	{
		ResultType:             ResultKindAssetWebsiteTechnology,
		SchemaRef:              "schema.result.asset.website_technology.v1",
		AuthorResultsFieldName: "WebsiteTechnologies",
		ItemTypeName:           "WebsiteTechnology",
		EncoderID:              ResultEncoderAssetWebsiteTechnology,
		GoEncoderName:          "EncodeWebsiteTechnology",
	},
	{
		ResultType:             ResultKindAssetScreenshot,
		SchemaRef:              "schema.result.asset.screenshot.v1",
		AuthorResultsFieldName: "Screenshots",
		ItemTypeName:           "Screenshot",
		EncoderID:              ResultEncoderAssetScreenshot,
		GoEncoderName:          "EncodeScreenshot",
	},
	{
		ResultType:             ResultKindAssetDirectory,
		SchemaRef:              "schema.result.asset.directory.v1",
		AuthorResultsFieldName: "Directories",
		ItemTypeName:           "Directory",
		EncoderID:              ResultEncoderAssetDirectory,
		GoEncoderName:          "EncodeDirectory",
	},
	{
		ResultType:             ResultKindAssetVulnerability,
		SchemaRef:              "schema.result.asset.vulnerability.v1",
		AuthorResultsFieldName: "Vulnerabilities",
		ItemTypeName:           "Vulnerability",
		EncoderID:              ResultEncoderAssetVulnerability,
		GoEncoderName:          "EncodeVulnerability",
	},
}

func int64Pointer(value int64) *int64      { return &value }
func numberPointer(value float64) *float64 { return &value }

// canonicalCodegenDescriptors is kept beside canonicalDescriptors so a
// result identity cannot accidentally acquire a different generated field
// surface. Its order is the build output order for every Engine.
var canonicalCodegenDescriptors = [...]CodegenDescriptor{
	{
		Descriptor:   canonicalDescriptors[0],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "DNSName", JSONName: "dnsName", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
		},
	},
	{
		Descriptor:   canonicalDescriptors[1],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "URL", JSONName: "url", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Host", JSONName: "host", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Title", JSONName: "title", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "StatusCode", JSONName: "statusCode", GoType: "*int", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{MinInt: int64Pointer(100), MaxInt: int64Pointer(599)}},
			{GoName: "ContentLength", JSONName: "contentLength", GoType: "*int", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{NonNegative: true, MaxInt: int64Pointer(2147483647)}},
			{GoName: "Location", JSONName: "location", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "Webserver", JSONName: "webserver", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "ContentType", JSONName: "contentType", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "Tech", JSONName: "tech", GoType: "[]string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "ResponseBody", JSONName: "responseBody", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "ResponseBodyTruncated", JSONName: "responseBodyTruncated", GoType: "bool", Presence: FieldPresenceOptional},
			{GoName: "Vhost", JSONName: "vhost", GoType: "*bool", Presence: FieldPresenceOptional},
			{GoName: "ResponseHeaders", JSONName: "responseHeaders", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "ResponseHeadersTruncated", JSONName: "responseHeadersTruncated", GoType: "bool", Presence: FieldPresenceOptional},
		},
	},
	{
		Descriptor:   canonicalDescriptors[2],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "Host", JSONName: "host", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "IP", JSONName: "ip", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Port", JSONName: "port", GoType: "int", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{MinInt: int64Pointer(1), MaxInt: int64Pointer(65535)}},
		},
	},
	{
		Descriptor:   canonicalDescriptors[3],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "URL", JSONName: "url", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Host", JSONName: "host", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Title", JSONName: "title", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "StatusCode", JSONName: "statusCode", GoType: "*int", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{MinInt: int64Pointer(100), MaxInt: int64Pointer(599)}},
			{GoName: "ContentLength", JSONName: "contentLength", GoType: "*int", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{NonNegative: true}},
			{GoName: "Location", JSONName: "location", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "Webserver", JSONName: "webserver", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "ContentType", JSONName: "contentType", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "Tech", JSONName: "tech", GoType: "[]string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "ResponseBody", JSONName: "responseBody", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "Vhost", JSONName: "vhost", GoType: "*bool", Presence: FieldPresenceOptional},
			{GoName: "ResponseHeaders", JSONName: "responseHeaders", GoType: "string", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
		},
	},
	{
		Descriptor:   canonicalDescriptors[4],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "URL", JSONName: "url", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Tech", JSONName: "tech", GoType: "[]string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, ExplicitArray: true}},
		},
	},
	{
		Descriptor:   canonicalDescriptors[5],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "URL", JSONName: "url", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "StatusCode", JSONName: "statusCode", GoType: "*int", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{MinInt: int64Pointer(100), MaxInt: int64Pointer(599)}},
			{GoName: "Image", JSONName: "image", GoType: "[]byte", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{NonEmpty: true}},
		},
	},
	{
		Descriptor:   canonicalDescriptors[6],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "URL", JSONName: "url", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Status", JSONName: "status", GoType: "int", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{MinInt: int64Pointer(0), MaxInt: int64Pointer(999)}},
			{GoName: "ContentLength", JSONName: "contentLength", GoType: "int64", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{NonNegative: true}},
			{GoName: "ContentType", JSONName: "contentType", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true}},
			{GoName: "Duration", JSONName: "duration", GoType: "int64", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{NonNegative: true}},
		},
	},
	{
		Descriptor:   canonicalDescriptors[7],
		ClosedObject: true,
		Fields: []FieldDescriptor{
			{GoName: "URL", JSONName: "url", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true, HTTPURL: true, NoURLUserInfo: true}},
			{GoName: "VulnType", JSONName: "vulnType", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true, Trimmed: true}},
			{GoName: "Severity", JSONName: "severity", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true, EnumValues: []string{"unknown", "info", "low", "medium", "high", "critical"}}},
			{GoName: "Source", JSONName: "source", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true, NonEmpty: true, EnumValues: []string{"nuclei"}}},
			{GoName: "CVSSScore", JSONName: "cvssScore", GoType: "*float64", Presence: FieldPresenceOptional, SchemaBasics: FieldSchemaBasics{MinNumber: numberPointer(0), MaxNumber: numberPointer(10)}},
			{GoName: "Description", JSONName: "description", GoType: "string", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ValidUTF8: true}},
			{GoName: "RawOutput", JSONName: "rawOutput", GoType: "map[string]any", Presence: FieldPresenceRequired, SchemaBasics: FieldSchemaBasics{ExplicitObject: true}},
		},
	},
}

// CodegenDescriptors returns a detached copy of all canonical result
// descriptors. Consumers receive no pointers into the registry and therefore
// cannot change Server validation or another generator invocation.
func CodegenDescriptors() ([]CodegenDescriptor, error) {
	if err := validateCodegenDescriptors(canonicalCodegenDescriptors[:]); err != nil {
		return nil, err
	}
	cloned := make([]CodegenDescriptor, 0, len(canonicalCodegenDescriptors))
	for _, descriptor := range canonicalCodegenDescriptors {
		copyDescriptor := descriptor
		copyDescriptor.Fields = make([]FieldDescriptor, len(descriptor.Fields))
		for index, field := range descriptor.Fields {
			copyDescriptor.Fields[index] = field
			copyDescriptor.Fields[index].SchemaBasics.EnumValues = append([]string(nil), field.SchemaBasics.EnumValues...)
			if field.SchemaBasics.MinInt != nil {
				value := *field.SchemaBasics.MinInt
				copyDescriptor.Fields[index].SchemaBasics.MinInt = &value
			}
			if field.SchemaBasics.MaxInt != nil {
				value := *field.SchemaBasics.MaxInt
				copyDescriptor.Fields[index].SchemaBasics.MaxInt = &value
			}
			if field.SchemaBasics.MinNumber != nil {
				value := *field.SchemaBasics.MinNumber
				copyDescriptor.Fields[index].SchemaBasics.MinNumber = &value
			}
			if field.SchemaBasics.MaxNumber != nil {
				value := *field.SchemaBasics.MaxNumber
				copyDescriptor.Fields[index].SchemaBasics.MaxNumber = &value
			}
		}
		cloned = append(cloned, copyDescriptor)
	}
	return cloned, nil
}

// ValidateCodegenDescriptors applies the same fast-fail checks used by the
// generator-facing registry API. It is exported for generator tests and
// verification tools that intentionally exercise malformed metadata.
func ValidateCodegenDescriptors(descriptors []CodegenDescriptor) error {
	return validateCodegenDescriptors(descriptors)
}

func validateCodegenDescriptors(descriptors []CodegenDescriptor) error {
	if len(descriptors) == 0 {
		return fmt.Errorf("result codegen descriptor registry must not be empty")
	}
	if len(descriptors) != len(canonicalDescriptors) {
		return fmt.Errorf("result codegen descriptor registry is incomplete: got %d descriptors, want %d", len(descriptors), len(canonicalDescriptors))
	}
	seen := map[string]map[string]int{
		"result type": {}, "schema ref": {}, "Results field": {}, "item type": {}, "encoder id": {}, "Go encoder": {},
	}
	for index, descriptor := range descriptors {
		if descriptor.ResultType == "" || descriptor.SchemaRef == "" || descriptor.AuthorResultsFieldName == "" || descriptor.ItemTypeName == "" || descriptor.EncoderID == "" || descriptor.GoEncoderName == "" {
			return fmt.Errorf("result codegen descriptor %d has missing identity metadata", index)
		}
		for label, value := range map[string]string{
			"result type": descriptor.ResultType, "schema ref": descriptor.SchemaRef,
			"Results field": descriptor.AuthorResultsFieldName, "item type": descriptor.ItemTypeName,
			"encoder id": descriptor.EncoderID, "Go encoder": descriptor.GoEncoderName,
		} {
			if previous, exists := seen[label][value]; exists {
				return fmt.Errorf("duplicate %s %q at descriptors %d and %d", label, value, index, previous)
			}
			seen[label][value] = index
		}
		if descriptor.ResultType != canonicalDescriptors[index].ResultType || descriptor.SchemaRef != canonicalDescriptors[index].SchemaRef || descriptor.AuthorResultsFieldName != canonicalDescriptors[index].AuthorResultsFieldName || descriptor.ItemTypeName != canonicalDescriptors[index].ItemTypeName || descriptor.EncoderID != canonicalDescriptors[index].EncoderID || descriptor.GoEncoderName != canonicalDescriptors[index].GoEncoderName {
			return fmt.Errorf("descriptor %d identity is inconsistent with canonical registry", index)
		}
		if !descriptor.ClosedObject {
			return fmt.Errorf("descriptor %q must declare a closed object", descriptor.ResultType)
		}
		if len(descriptor.Fields) == 0 {
			return fmt.Errorf("descriptor %q must declare fields", descriptor.ResultType)
		}
		fieldNames := make(map[string]struct{}, len(descriptor.Fields))
		jsonNames := make(map[string]struct{}, len(descriptor.Fields))
		for fieldIndex, field := range descriptor.Fields {
			if field.GoName == "" || field.JSONName == "" || field.GoType == "" {
				return fmt.Errorf("descriptor %q field %d has missing metadata", descriptor.ResultType, fieldIndex)
			}
			if !isExportedIdentifier(field.GoName) {
				return fmt.Errorf("descriptor %q field %q is not an exported Go identifier", descriptor.ResultType, field.GoName)
			}
			if _, exists := fieldNames[field.GoName]; exists {
				return fmt.Errorf("descriptor %q has duplicate Go field %q", descriptor.ResultType, field.GoName)
			}
			if _, exists := jsonNames[field.JSONName]; exists {
				return fmt.Errorf("descriptor %q has duplicate JSON field %q", descriptor.ResultType, field.JSONName)
			}
			fieldNames[field.GoName] = struct{}{}
			jsonNames[field.JSONName] = struct{}{}
			if field.Presence != FieldPresenceRequired && field.Presence != FieldPresenceOptional {
				return fmt.Errorf("descriptor %q field %q has unsupported presence %q", descriptor.ResultType, field.GoName, field.Presence)
			}
			if !supportedCodegenGoType(field.GoType) {
				return fmt.Errorf("descriptor %q field %q has unsupported Go type %q", descriptor.ResultType, field.GoName, field.GoType)
			}
			if field.SchemaBasics.NonEmpty && field.Presence != FieldPresenceRequired {
				return fmt.Errorf("descriptor %q field %q marks optional field as non-empty", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.MinInt != nil && field.SchemaBasics.MaxInt != nil && *field.SchemaBasics.MinInt > *field.SchemaBasics.MaxInt {
				return fmt.Errorf("descriptor %q field %q has invalid integer bounds", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.MinNumber != nil && field.SchemaBasics.MaxNumber != nil && *field.SchemaBasics.MinNumber > *field.SchemaBasics.MaxNumber {
				return fmt.Errorf("descriptor %q field %q has invalid numeric bounds", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.ExplicitArray && !strings.HasPrefix(field.GoType, "[]") {
				return fmt.Errorf("descriptor %q field %q requires an array Go type", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.ExplicitObject && !strings.HasPrefix(field.GoType, "map[") {
				return fmt.Errorf("descriptor %q field %q requires an object Go type", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.HTTPURL && field.GoType != "string" {
				return fmt.Errorf("descriptor %q field %q requires a string URL type", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.NoURLUserInfo && !field.SchemaBasics.HTTPURL {
				return fmt.Errorf("descriptor %q field %q forbids URL user info without HTTP URL validation", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.Trimmed && field.GoType != "string" {
				return fmt.Errorf("descriptor %q field %q requires a string type for canonical whitespace validation", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.NonNegative && !isNumericCodegenGoType(field.GoType) {
				return fmt.Errorf("descriptor %q field %q requires a numeric Go type for non-negative validation", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.MinInt != nil || field.SchemaBasics.MaxInt != nil {
				if !isIntegerCodegenGoType(field.GoType) {
					return fmt.Errorf("descriptor %q field %q requires an integer Go type for integer bounds", descriptor.ResultType, field.GoName)
				}
			}
			if field.SchemaBasics.MinNumber != nil || field.SchemaBasics.MaxNumber != nil {
				if field.GoType != "*float64" && field.GoType != "float64" {
					return fmt.Errorf("descriptor %q field %q requires a number Go type for numeric bounds", descriptor.ResultType, field.GoName)
				}
			}
			if field.SchemaBasics.MaxBytes < 0 || field.SchemaBasics.MaxRunes < 0 {
				return fmt.Errorf("descriptor %q field %q has negative size metadata", descriptor.ResultType, field.GoName)
			}
			if field.SchemaBasics.MaxBytes > 0 || field.SchemaBasics.MaxRunes > 0 {
				if field.GoType != "string" && field.GoType != "[]string" && field.GoType != "[]byte" {
					return fmt.Errorf("descriptor %q field %q has unsupported size metadata for Go type %q", descriptor.ResultType, field.GoName, field.GoType)
				}
			}
			seenEnums := make(map[string]struct{}, len(field.SchemaBasics.EnumValues))
			for _, enumValue := range field.SchemaBasics.EnumValues {
				if field.GoType != "string" || enumValue == "" || !utf8.ValidString(enumValue) {
					return fmt.Errorf("descriptor %q field %q has unsupported enum metadata", descriptor.ResultType, field.GoName)
				}
				if _, exists := seenEnums[enumValue]; exists {
					return fmt.Errorf("descriptor %q field %q has duplicate enum value %q", descriptor.ResultType, field.GoName, enumValue)
				}
				seenEnums[enumValue] = struct{}{}
			}
		}
	}
	for index, descriptor := range descriptors {
		if descriptor.Descriptor != canonicalDescriptors[index] {
			return fmt.Errorf("descriptor %d identity is inconsistent with canonical registry", index)
		}
		if !reflect.DeepEqual(descriptor.Fields, canonicalCodegenDescriptors[index].Fields) || descriptor.ClosedObject != canonicalCodegenDescriptors[index].ClosedObject {
			return fmt.Errorf("descriptor %d field metadata is inconsistent with canonical registry", index)
		}
	}
	return nil
}

func supportedCodegenGoType(value string) bool {
	switch value {
	case "string", "int", "int64", "bool", "[]string", "[]byte", "*int", "*bool", "*float64", "map[string]any":
		return true
	default:
		return false
	}
}

func isIntegerCodegenGoType(value string) bool {
	return value == "int" || value == "int64" || value == "*int" || value == "*int64"
}

func isNumericCodegenGoType(value string) bool {
	return isIntegerCodegenGoType(value) || value == "float64" || value == "*float64"
}

func isExportedIdentifier(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	for index, r := range value {
		if index == 0 {
			if r < 'A' || r > 'Z' {
				return false
			}
			continue
		}
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

// Descriptors returns the complete platform result registry in canonical
// generation order. The returned slice is detached from the registry.
func Descriptors() []Descriptor {
	return append([]Descriptor(nil), canonicalDescriptors[:]...)
}

// Lookup returns a descriptor only for an exact canonical result_type value.
// Whitespace trimming here would create an implicit wire alias and make the
// Server and generated typed ports disagree about the submitted identity.
func Lookup(kind string) (Descriptor, bool) {
	if kind == "" || kind != strings.TrimSpace(kind) {
		return Descriptor{}, false
	}
	for _, descriptor := range canonicalDescriptors {
		if descriptor.ResultType == kind {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

func Validate(kind string, item any) error {
	if kind == "" {
		return fmt.Errorf("result kind is required")
	}
	if kind != strings.TrimSpace(kind) {
		return fmt.Errorf("result kind %q must be canonical", kind)
	}
	descriptor, ok := Lookup(kind)
	if !ok {
		return fmt.Errorf("unknown result kind %q", kind)
	}
	switch descriptor.ResultType {
	case ResultKindAssetSubdomain:
		subdomain, ok := item.(Subdomain)
		if !ok {
			return fmt.Errorf("result kind %s requires results.Subdomain item", kind)
		}
		canonical, valid := NormalizeSubdomainDNSName(subdomain.DNSName)
		if !valid || canonical != subdomain.DNSName {
			return fmt.Errorf("subdomain dnsName must be a canonical DNS name")
		}
		return nil
	case ResultKindAssetHostPort:
		hostPort, ok := item.(HostPort)
		if !ok {
			return fmt.Errorf("result kind %s requires results.HostPort item", kind)
		}
		canonicalHost, validHost := normalizeHostPortHost(hostPort.Host)
		if !validHost || canonicalHost != hostPort.Host {
			return fmt.Errorf("host-port host must be canonical")
		}
		ip := net.ParseIP(hostPort.IP)
		if ip == nil {
			return fmt.Errorf("host-port ip is invalid")
		}
		if strings.Contains(hostPort.IP, ":") || ip.To4() == nil {
			return fmt.Errorf("host-port IPv6 is not supported")
		}
		if ip.To4().String() != hostPort.IP {
			return fmt.Errorf("host-port ip must be canonical")
		}
		if hostPort.Port < 1 || hostPort.Port > 65535 {
			return fmt.Errorf("host-port port must be between 1 and 65535")
		}
		return nil
	case ResultKindAssetWebsite:
		website, ok := item.(Website)
		if !ok {
			return fmt.Errorf("result kind %s requires results.Website item", kind)
		}
		_, err := ValidateObservedAssetURL(website.URL)
		if err != nil {
			return err
		}
		urlHost, err := DeriveObservedAssetURLHost(website.URL)
		if err != nil {
			return err
		}
		if website.Host != urlHost {
			return fmt.Errorf("website host conflicts with url authority")
		}
		if err := validateWebsiteOptionalFields(website); err != nil {
			return err
		}
		return nil
	case ResultKindAssetWebsiteTechnology:
		technology, ok := item.(WebsiteTechnology)
		if !ok {
			return fmt.Errorf("result kind %s requires results.WebsiteTechnology item", kind)
		}
		return validateWebsiteTechnology(technology)
	case ResultKindAssetEndpoint:
		endpoint, ok := item.(Endpoint)
		if !ok {
			return fmt.Errorf("result kind %s requires results.Endpoint item", kind)
		}
		return validateEndpoint(endpoint)
	case ResultKindAssetScreenshot:
		screenshot, ok := item.(Screenshot)
		if !ok {
			return fmt.Errorf("result kind %s requires results.Screenshot item", kind)
		}
		return ValidateScreenshot(screenshot)
	case ResultKindAssetDirectory:
		directory, ok := item.(Directory)
		if !ok {
			return fmt.Errorf("result kind %s requires results.Directory item", kind)
		}
		return validateDirectory(directory)
	case ResultKindAssetVulnerability:
		vulnerability, ok := item.(Vulnerability)
		if !ok {
			return fmt.Errorf("result kind %s requires results.Vulnerability item", kind)
		}
		return validateVulnerability(vulnerability)
	default:
		return fmt.Errorf("unknown result kind %q", kind)
	}
}

func validateEndpoint(endpoint Endpoint) error {
	if _, err := ValidateObservedAssetURL(endpoint.URL); err != nil {
		return fmt.Errorf("endpoint url is invalid")
	}
	host, err := DeriveObservedAssetURLHost(endpoint.URL)
	if err != nil {
		return fmt.Errorf("endpoint url authority is invalid")
	}
	if endpoint.Host != host {
		return fmt.Errorf("endpoint host conflicts with url authority")
	}
	if endpoint.StatusCode != nil && (*endpoint.StatusCode < 100 || *endpoint.StatusCode > 599) {
		return fmt.Errorf("endpoint statusCode must be between 100 and 599")
	}
	if endpoint.ContentLength != nil && (*endpoint.ContentLength < 0 || *endpoint.ContentLength > 2147483647) {
		return fmt.Errorf("endpoint contentLength is out of range")
	}
	for _, field := range []struct {
		name, value string
		limit       int
	}{
		{"title", endpoint.Title, 2000}, {"location", endpoint.Location, 2000},
		{"webserver", endpoint.Webserver, 1024}, {"contentType", endpoint.ContentType, 1024},
	} {
		if strings.Contains(field.value, "\x00") || len([]byte(field.value)) > field.limit {
			return fmt.Errorf("endpoint %s is invalid", field.name)
		}
	}
	for _, field := range []struct {
		name, value string
		limit       int
	}{
		{"responseBody", endpoint.ResponseBody, 2000}, {"responseHeaders", endpoint.ResponseHeaders, 64 * 1024},
	} {
		if strings.Contains(field.value, "\x00") || len([]byte(field.value)) > field.limit {
			return fmt.Errorf("endpoint %s is invalid", field.name)
		}
	}
	for _, tech := range endpoint.Tech {
		if strings.TrimSpace(tech) == "" || strings.Contains(tech, "\x00") || utf8.RuneCountInString(tech) > 100 {
			return fmt.Errorf("endpoint tech is invalid")
		}
	}
	return nil
}

func validateWebsiteOptionalFields(website Website) error {
	if website.StatusCode != nil && (*website.StatusCode < 100 || *website.StatusCode > 599) {
		return fmt.Errorf("website statusCode must be between 100 and 599")
	}
	if website.ContentLength != nil && *website.ContentLength < 0 {
		return fmt.Errorf("website contentLength must be non-negative")
	}
	for _, value := range []struct {
		name  string
		value string
	}{
		{name: "title", value: website.Title},
		{name: "location", value: website.Location},
		{name: "webserver", value: website.Webserver},
		{name: "contentType", value: website.ContentType},
	} {
		if strings.Contains(value.value, "\x00") {
			return fmt.Errorf("website %s must be NUL-safe", value.name)
		}
	}
	// HTTP response evidence is intentionally multiline but cannot contain NUL.
	for _, value := range []struct {
		name  string
		value string
	}{
		{name: "responseBody", value: website.ResponseBody},
		{name: "responseHeaders", value: website.ResponseHeaders},
	} {
		if strings.Contains(value.value, "\x00") {
			return fmt.Errorf("website %s must be NUL-safe", value.name)
		}
	}
	for _, tech := range website.Tech {
		if strings.TrimSpace(tech) == "" || strings.Contains(tech, "\x00") {
			return fmt.Errorf("website tech values must be non-empty and NUL-safe")
		}
	}
	return nil
}

func validateWebsiteTechnology(item WebsiteTechnology) error {
	if _, err := ValidateObservedAssetURL(item.URL); err != nil {
		return fmt.Errorf("website technology url is invalid")
	}
	if item.Tech == nil {
		return fmt.Errorf("website technology tech must be an explicit string array")
	}
	for _, tech := range item.Tech {
		if !utf8.ValidString(tech) || strings.TrimSpace(tech) == "" || strings.Contains(tech, "\x00") {
			return fmt.Errorf("website technology values must be nonblank, valid UTF-8, and NUL-free")
		}
		if utf8.RuneCountInString(tech) > 100 {
			return fmt.Errorf("website technology value exceeds 100 Unicode code points")
		}
	}
	return nil
}
