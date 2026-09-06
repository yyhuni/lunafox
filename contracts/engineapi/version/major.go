// Package version owns the one-to-one mapping between an Engine API major and
// the versioned Engine Execution Context wire message selected by that major.
// It deliberately contains no runtime, Agent, or package-loading behavior.
package version

import "fmt"

const (
	// EngineAPIMajor is the first active Engine Container API major.
	EngineAPIMajor uint32 = 2

	// EngineExecutionContextProtoPackage is the canonical protobuf package
	// selected by EngineAPIMajor.
	EngineExecutionContextProtoPackage = "lunafox.engine.execution.v2"
	// EngineExecutionContextProtoMessage is the canonical fully-qualified
	// Context message selected by EngineAPIMajor.
	EngineExecutionContextProtoMessage = "lunafox.engine.execution.v2.EngineExecutionContext"
	// EngineExecutionContextMappingLabel is the stable logical mapping label
	// used in diagnostics and compatibility documentation.
	EngineExecutionContextMappingLabel = "engine.execution-context.v2"
)

// ContextBinding describes the protocol identity selected by one Engine API
// major. Values are immutable strings; callers receive a value copy.
type ContextBinding struct {
	EngineAPIMajor uint32
	ProtoPackage   string
	ProtoMessage   string
	MappingLabel   string
}

var contextBindings = [...]ContextBinding{{
	EngineAPIMajor: EngineAPIMajor,
	ProtoPackage:   EngineExecutionContextProtoPackage,
	ProtoMessage:   EngineExecutionContextProtoMessage,
	MappingLabel:   EngineExecutionContextMappingLabel,
}}

// ContextForEngineAPIMajor returns the sole Context binding for a supported
// major. Unknown majors return false and must be rejected by callers.
func ContextForEngineAPIMajor(major uint32) (ContextBinding, bool) {
	for _, binding := range contextBindings {
		if binding.EngineAPIMajor == major {
			return binding, true
		}
	}
	return ContextBinding{}, false
}

// RequireContextBinding returns a typed error for an unsupported major. It is
// intended for planner/generator boundaries that must fail closed.
func RequireContextBinding(major uint32) (ContextBinding, error) {
	binding, ok := ContextForEngineAPIMajor(major)
	if !ok {
		return ContextBinding{}, fmt.Errorf("unsupported Engine API major %d", major)
	}
	return binding, nil
}

// SupportedEngineAPIMajors returns a detached, deterministic list. A fresh
// slice prevents callers from mutating the code-owned capability set.
func SupportedEngineAPIMajors() []uint32 {
	majors := make([]uint32, len(contextBindings))
	for index, binding := range contextBindings {
		majors[index] = binding.EngineAPIMajor
	}
	return majors
}
