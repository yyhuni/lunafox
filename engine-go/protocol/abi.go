package protocol

import "fmt"

const (
	// EngineAPIMajor is the sole active Engine Container protocol major.
	EngineAPIMajor uint32 = 2

	RuntimeRootPath = "/run/lunafox"

	ContextDirectoryPath    = RuntimeRootPath + "/context"
	ContextFilePath         = ContextDirectoryPath + "/execution.pb"
	CredentialDirectoryPath = RuntimeRootPath + "/credential"
	CredentialFilePath      = CredentialDirectoryPath + "/token"
	SocketDirectoryPath     = RuntimeRootPath + "/socket"
	ExecutionEndpointPath   = SocketDirectoryPath + "/engine.sock"
	WorkspacePath           = "/workspace"

	InputsDirectoryPath            = RuntimeRootPath + "/inputs"
	SubdomainsInputPath            = InputsDirectoryPath + "/subdomains.txt"
	HostPortsInputPath             = InputsDirectoryPath + "/host-ports.jsonl"
	WebsiteURLsInputPath           = InputsDirectoryPath + "/website-urls.txt"
	EndpointURLsInputPath          = InputsDirectoryPath + "/endpoint-urls.txt"
	ResourcesDirectoryPath         = RuntimeRootPath + "/resources"
	ConfigResourcesDirectoryPath   = ResourcesDirectoryPath + "/config"
	PlatformResourcesDirectoryPath = ResourcesDirectoryPath + "/platform"
	RuntimeResourcesDirectoryPath  = ResourcesDirectoryPath + "/runtime"
	// NucleiTemplatesPath is the fixed directory consumed by the Nuclei
	// runtime.  The directory is published atomically by the Agent after a
	// complete manifest/blob exchange; no bundle file is part of the ABI.
	NucleiTemplatesPath = RuntimeResourcesDirectoryPath + "/nuclei/templates"
)

const (
	// ExecutionInputRoleSubdomains is the stable wire identity of the
	// non-sensitive subdomains product. Registry ownership of membership,
	// content, and paths remains in contracts/executionartifact.
	ExecutionInputRoleSubdomains = "subdomains"
	// ExecutionInputRoleHostPorts is the stable wire identity of the
	// non-sensitive hostPorts product.
	ExecutionInputRoleHostPorts = "hostPorts"
	// ExecutionInputRoleWebsiteURLs is the stable wire identity of the
	// non-sensitive websiteURLs product.
	ExecutionInputRoleWebsiteURLs = "websiteURLs"
	// ExecutionInputRoleEndpointURLs is the stable wire identity of the
	// non-sensitive Endpoint URL product.
	ExecutionInputRoleEndpointURLs = "endpointURLs"
)

const (
	// Protocol hard ceilings prevent a malformed or mixed-revision Context from
	// expanding the transport surface beyond what every implementation accepts.
	ProgressMessageMaxBytesCeiling uint32 = 64 * 1024
	ResultBatchMaxItemsCeiling     uint32 = 4096
	ResultBatchMaxBytesCeiling     uint32 = 4 * 1024 * 1024
)

// ContextBinding identifies the protocol message selected by a major.
type ContextBinding struct {
	EngineAPIMajor uint32
	ProtoPackage   string
	ProtoMessage   string
}

var contextBinding = ContextBinding{
	EngineAPIMajor: EngineAPIMajor,
	ProtoPackage:   "lunafox.engine.execution.v2",
	ProtoMessage:   "lunafox.engine.execution.v2.EngineExecutionContext",
}

// RequireContextBinding rejects unsupported protocol majors.
func RequireContextBinding(major uint32) (ContextBinding, error) {
	if major != EngineAPIMajor {
		return ContextBinding{}, fmt.Errorf("unsupported Engine API major %d", major)
	}
	return contextBinding, nil
}

// SupportedEngineAPIMajors returns the immutable supported-major set.
func SupportedEngineAPIMajors() []uint32 { return []uint32{EngineAPIMajor} }
