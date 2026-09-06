package application

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strings"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
)

type dynamicConfiguration struct {
	Steps map[string]dynamicStepConfiguration
}

type dynamicStepConfiguration struct {
	Enabled      bool
	EngineConfig map[string]any
}

func (service *ScanCreateService) validateRequestedScanWorkflow(ctx context.Context, workflow string) (ScanCreateWorkflowManifest, error) {
	scanWorkflowID, err := resourcenames.ParseScanWorkflow(workflow)
	if err != nil {
		return ScanCreateWorkflowManifest{}, invalidScanWorkflowf("scanWorkflow must use scanWorkflows/{workflow}")
	}
	if service == nil || service.workflowReader == nil {
		return ScanCreateWorkflowManifest{}, ErrCreateInvalidScanWorkflow
	}
	manifest, err := service.workflowReader.GetScanWorkflowManifest(ctx, scanWorkflowID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ScanCreateWorkflowManifest{}, WrapSchemaInvalid(scanWorkflowID, "scan workflow "+scanWorkflowID+" not found in available scan workflow definitions", fs.ErrNotExist)
		}
		return ScanCreateWorkflowManifest{}, err
	}
	return manifest, nil
}

// normalizePlanTaskConfiguration requires one submitted config per workflow
// step. Engine defaults are materialized by Profile, never during Scan create.
func normalizePlanTaskConfiguration(configuration map[string]any, manifest ScanCreateWorkflowManifest) (dynamicConfiguration, error) {
	stepConfigs, err := workflowconfig.Decode(configuration, manifestStepSet(manifest))
	if err != nil {
		schemaErr := WrapSchemaInvalid(manifest.ScanWorkflowID, err.Error(), err)
		if violation, ok := workflowconfig.FirstViolation(err); ok && violation.Reason == workflowconfig.ReasonFieldValueInvalid && violation.Path == "configuration.steps" {
			return dynamicConfiguration{}, errors.Join(ErrCreateNoScanWorkflows, schemaErr)
		}
		return dynamicConfiguration{}, schemaErr
	}

	normalized := dynamicConfiguration{Steps: make(map[string]dynamicStepConfiguration, len(stepConfigs))}
	for stepID, stepConfig := range stepConfigs {
		normalized.Steps[stepID] = dynamicStepConfiguration{
			Enabled:      stepConfig.Enabled,
			EngineConfig: cloneMap(stepConfig.EngineConfig),
		}
	}
	return normalized, nil
}

func manifestStepSet(manifest ScanCreateWorkflowManifest) map[string]struct{} {
	out := map[string]struct{}{}
	for _, stage := range manifest.Stages {
		for _, step := range stage.Steps {
			out[step.StepID] = struct{}{}
		}
	}
	return out
}

func invalidScanWorkflowf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrCreateInvalidScanWorkflow, fmt.Sprintf(format, args...))
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		cloned := cloneConfigurationValue(reflect.ValueOf(value))
		if !cloned.IsValid() {
			out[key] = nil
			continue
		}
		out[key] = cloned.Interface()
	}
	return out
}

// cloneConfigurationValue copies the JSON-shaped configuration graph rather
// than only its outer map. Scan plans are frozen after validation; retaining a
// caller-owned nested map/slice would let a later mutation rewrite the saved
// configuration or its execution fingerprint.
func cloneConfigurationValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := cloneConfigurationValue(value.Elem())
		out := reflect.New(value.Type()).Elem()
		if !cloned.IsValid() {
			return out
		}
		out.Set(cloned)
		return out
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			key := cloneConfigurationValue(iter.Key())
			item := cloneConfigurationValue(iter.Value())
			if !key.IsValid() || !item.IsValid() {
				continue
			}
			out.SetMapIndex(key, item)
		}
		return out
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			out.Index(index).Set(cloneConfigurationValue(value.Index(index)))
		}
		return out
	case reflect.Array:
		out := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			out.Index(index).Set(cloneConfigurationValue(value.Index(index)))
		}
		return out
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.New(value.Type().Elem())
		out.Elem().Set(cloneConfigurationValue(value.Elem()))
		return out
	default:
		// Scalars and immutable value structs (for example time.Duration) do
		// not contain a mutable JSON child graph.
		return value
	}
}
