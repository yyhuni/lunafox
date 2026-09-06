// Package executionartifact owns the closed, versioned representation
// identities used by Server-produced Engine execution inputs and resources.
package executionartifact

import (
	"fmt"
	"path"

	"github.com/yyhuni/lunafox/engine-go/protocol"
)

// Role identifies one typed execution-artifact position. It is not an
// authorization, source, path, cache, mount, sensitivity, retry, or size policy.
type Role uint8

const (
	RoleSubdomainsInput Role = iota + 1
	RoleHostPortsInput
	RoleWebsiteURLsInput
	RoleEndpointURLsInput
	RoleWordlistConfigResource
	RoleSubfinderProviderConfigPlatformResource
	RoleFingerprintLibraryFingerPrintHub
	RoleRuntimeArtifact
)

const (
	FileKindArtifactFile       = "artifactFile"
	FileKindWordlist           = "wordlist"
	FileKindToolProviderConfig = "toolProviderConfig"
	FileKindFingerprintLibrary = "fingerprintLibrary"
	FileKindRuntimeArtifact    = "nucleiTemplates"

	RoleSubdomains   = protocol.ExecutionInputRoleSubdomains
	RoleHostPorts    = protocol.ExecutionInputRoleHostPorts
	RoleWebsiteURLs  = protocol.ExecutionInputRoleWebsiteURLs
	RoleEndpointURLs = protocol.ExecutionInputRoleEndpointURLs

	ContentTypeSubdomains                       = "application/vnd.lunafox.subdomains.v1"
	ContentTypeHostPorts                        = "application/vnd.lunafox.host-ports.v1"
	ContentTypeWebsiteURLs                      = "application/vnd.lunafox.website-urls.v1"
	ContentTypeEndpointURLs                     = "application/vnd.lunafox.endpoint-urls.v1"
	ContentTypeWordlist                         = "application/vnd.lunafox.wordlist.v1"
	ContentTypeSubfinderProviderConfig          = "application/vnd.lunafox.subfinder-provider-config.v1+yaml"
	ContentTypeFingerprintLibraryFingerPrintHub = "application/vnd.lunafox.fingerprint-library.fingerprinthub.v1+json"
	// ContentTypeNucleiTemplates describes the declaration-gated directory
	// capability.  It is intentionally not a bundle/archive media type: the
	// bytes are exchanged as individual digest-addressed YAML blobs and the
	// Engine consumes the Agent-published directory.
	ContentTypeRuntimeArtifact = "application/vnd.lunafox.nucleiTemplates.v1"
)

// RecordCountPresence defines whether transfer integrity must carry a record
// count. It does not establish an aggregate record limit.
type RecordCountPresence uint8

const (
	RecordCountRequired RecordCountPresence = iota + 1
	RecordCountAbsent
)

// Descriptor is the passive representation contract for one typed role.
type Descriptor struct {
	Role                  Role
	FileKind              string
	ContentType           string
	RecordCountPresence   RecordCountPresence
	PlatformResourceID    string
	Filename              string
	Library               string
	NativeHTTPContentType string
	ContainerPath         string
	RoleID                string
	// ArtifactID is populated for runtime artifacts. It is the closed plan
	// binding key; callers cannot register or select arbitrary IDs.
	ArtifactID string
}

// canonicalInputDescriptors is the complete, closed Engine execution-input
// Registry. It intentionally contains only non-sensitive Scan products; keep
// this order stable because generated Engine handles and wire validation use it.
var canonicalInputDescriptors = [...]Descriptor{
	{
		Role:                RoleSubdomainsInput,
		RoleID:              RoleSubdomains,
		FileKind:            FileKindArtifactFile,
		ContentType:         ContentTypeSubdomains,
		RecordCountPresence: RecordCountRequired,
		ContainerPath:       protocol.SubdomainsInputPath,
	},
	{
		Role:                RoleHostPortsInput,
		RoleID:              RoleHostPorts,
		FileKind:            FileKindArtifactFile,
		ContentType:         ContentTypeHostPorts,
		RecordCountPresence: RecordCountRequired,
		ContainerPath:       protocol.HostPortsInputPath,
	},
	{
		Role:                RoleWebsiteURLsInput,
		RoleID:              RoleWebsiteURLs,
		FileKind:            FileKindArtifactFile,
		ContentType:         ContentTypeWebsiteURLs,
		RecordCountPresence: RecordCountRequired,
		ContainerPath:       protocol.WebsiteURLsInputPath,
	},
	{
		Role:                RoleEndpointURLsInput,
		RoleID:              RoleEndpointURLs,
		FileKind:            FileKindArtifactFile,
		ContentType:         ContentTypeEndpointURLs,
		RecordCountPresence: RecordCountRequired,
		ContainerPath:       protocol.EndpointURLsInputPath,
	},
}

// canonicalResourceDescriptors contains eager config/platform resources. It is
// a separate registry so resource metadata cannot accidentally become a lazy
// execution-input role or a generated Engine handle.
var canonicalResourceDescriptors = [...]Descriptor{
	{
		Role:                RoleWordlistConfigResource,
		FileKind:            FileKindWordlist,
		ContentType:         ContentTypeWordlist,
		RecordCountPresence: RecordCountRequired,
	},
	{
		Role:                  RoleSubfinderProviderConfigPlatformResource,
		FileKind:              FileKindToolProviderConfig,
		ContentType:           ContentTypeSubfinderProviderConfig,
		RecordCountPresence:   RecordCountAbsent,
		PlatformResourceID:    "subfinderProviderConfig",
		Filename:              "config.yaml",
		NativeHTTPContentType: "application/yaml",
		ContainerPath:         path.Join(protocol.PlatformResourcesDirectoryPath, "subfinderProviderConfig", "config.yaml"),
	},
	{Role: RoleFingerprintLibraryFingerPrintHub, FileKind: FileKindFingerprintLibrary, ContentType: ContentTypeFingerprintLibraryFingerPrintHub, RecordCountPresence: RecordCountRequired, PlatformResourceID: "fingerprintLibraryFingerPrintHub", Filename: "fingerprinthub_web.json", Library: "fingerprinthub", NativeHTTPContentType: "application/json", ContainerPath: path.Join(protocol.PlatformResourcesDirectoryPath, "fingerprintLibraryFingerPrintHub", "fingerprinthub_web.json")},
}

// canonicalRuntimeArtifactDescriptors contains task-start artifacts that are
// neither saved-plan resources nor lazy execution inputs. They deliberately
// remain out of ExecutionInputRegistry so a late-bound template selection
// cannot become a frozen product binding.
var canonicalRuntimeArtifactDescriptors = [...]Descriptor{
	{
		Role:                RoleRuntimeArtifact,
		FileKind:            FileKindRuntimeArtifact,
		ContentType:         ContentTypeRuntimeArtifact,
		RecordCountPresence: RecordCountAbsent,
		Filename:            "templates",
		ContainerPath:       protocol.NucleiTemplatesPath,
		ArtifactID:          "nucleiTemplates",
	},
}

func allDescriptors() []Descriptor {
	result := make([]Descriptor, 0, len(canonicalInputDescriptors)+len(canonicalResourceDescriptors)+len(canonicalRuntimeArtifactDescriptors))
	result = append(result, canonicalInputDescriptors[:]...)
	result = append(result, canonicalResourceDescriptors[:]...)
	result = append(result, canonicalRuntimeArtifactDescriptors[:]...)
	return result
}

// Descriptors returns the complete registry in canonical order. The returned
// slice is detached so callers cannot mutate the package-owned registry.
func Descriptors() []Descriptor {
	return allDescriptors()
}

// Lookup returns the exact phase-one representation contract for a typed role.
func Lookup(role Role) (Descriptor, bool) {
	for _, descriptor := range allDescriptors() {
		if descriptor.Role == role {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

// LookupRoleID resolves the closed, wire-visible role identity. Role IDs are
// intentionally strings at protocol boundaries while the registry keeps an
// internal typed key for owner-local dispatch.
func LookupRoleID(roleID string) (Descriptor, bool) {
	for _, descriptor := range canonicalInputDescriptors {
		if descriptor.RoleID == roleID {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

// InputDescriptors returns only the four scan-fact products in canonical
// role order. Resource descriptors remain available through Descriptors.
func InputDescriptors() []Descriptor {
	return append([]Descriptor(nil), canonicalInputDescriptors[:]...)
}

// ExecutionInputRegistry is the canonical typed Registry consumed by code
// generation and runtime authorization. It never returns eager resources.
func ExecutionInputRegistry() []Descriptor { return InputDescriptors() }

func InputRoleIDs() []string {
	descriptors := InputDescriptors()
	result := make([]string, len(descriptors))
	for index, descriptor := range descriptors {
		result[index] = descriptor.RoleID
	}
	return result
}

// LookupPlatformResource resolves only platform-resource roles. The closed
// mapping prevents caller-provided IDs from selecting arbitrary files.
func LookupPlatformResource(resourceID string) (Descriptor, bool) {
	for _, descriptor := range canonicalResourceDescriptors {
		if descriptor.PlatformResourceID == resourceID {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

// LookupRuntimeArtifact resolves the closed set of late-bound task-start
// artifacts. Runtime artifacts cannot be selected through any generic role or
// resource-id request.
func LookupRuntimeArtifact(role Role) (Descriptor, bool) {
	for _, descriptor := range canonicalRuntimeArtifactDescriptors {
		if descriptor.Role == role {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

func LookupRuntimeArtifactID(artifactID string) (Descriptor, bool) {
	for _, descriptor := range canonicalRuntimeArtifactDescriptors {
		if descriptor.ArtifactID != "" && descriptor.ArtifactID == artifactID {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

func RuntimeArtifactID(role Role) (string, bool) {
	if _, ok := LookupRuntimeArtifact(role); !ok {
		return "", false
	}
	for _, descriptor := range canonicalRuntimeArtifactDescriptors {
		if descriptor.Role == role && descriptor.ArtifactID != "" {
			return descriptor.ArtifactID, true
		}
	}
	return "", false
}

// LookupRuntimeArtifactByContainerPath resolves a runtime mount target from
// the closed descriptor registry. Agent mount validation uses this inverse
// lookup so container paths never become an Engine-specific switch.
func LookupRuntimeArtifactByContainerPath(containerPath string) (Descriptor, bool) {
	for _, descriptor := range canonicalRuntimeArtifactDescriptors {
		if descriptor.ContainerPath == containerPath {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

func IsFingerprintLibraryRole(role Role) bool {
	return role == RoleFingerprintLibraryFingerPrintHub
}

func IsRuntimeArtifactRole(role Role) bool {
	_, ok := LookupRuntimeArtifact(role)
	return ok
}

// ValidateBinding rejects an unknown role, legacy/unknown content identity, or
// a file-kind/content-type value copied from another role.
func ValidateBinding(role Role, fileKind, contentType string) error {
	descriptor, ok := Lookup(role)
	if !ok {
		return fmt.Errorf("execution artifact role is unsupported")
	}
	if fileKind != descriptor.FileKind {
		return fmt.Errorf("execution artifact file kind does not match typed role")
	}
	if contentType != descriptor.ContentType {
		return fmt.Errorf("execution artifact content type does not match typed role")
	}
	return nil
}
