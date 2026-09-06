package agentexecution

import (
	"strings"
	"testing"

	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	"github.com/yyhuni/lunafox/contracts/results"
	engineexecutionpb "github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestResolvedPlanHasExactlyElevenClosedTopLevelGroups(t *testing.T) {
	descriptor := (&agentexecutionv1.ResolvedEngineExecutionPlan{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, descriptor, []descriptorField{
		{name: "execution", number: 1, kind: protoreflect.StringKind},
		{name: "task", number: 2, kind: protoreflect.StringKind},
		{name: "target", number: 3, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.CanonicalTarget"},
		{name: "workflow_step", number: 4, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.WorkflowStepScope"},
		{name: "engine_release", number: 5, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.EngineRelease"},
		{name: "runtime_image", number: 6, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.RuntimeImage"},
		{name: "config", number: 7, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.FinalEngineConfig"},
		{name: "config_resource_bindings", number: 9, kind: protoreflect.MessageKind, cardinality: protoreflect.Repeated, message: "lunafox.agent.execution.v1.ConfigResourceBinding"},
		{name: "platform_resource_bindings", number: 10, kind: protoreflect.MessageKind, cardinality: protoreflect.Repeated, message: "lunafox.agent.execution.v1.PlatformResourceBinding"},
		{name: "limits", number: 11, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.ExecutionLimits"},
		{name: "runtime_artifact_bindings", number: 12, kind: protoreflect.MessageKind, cardinality: protoreflect.Repeated, message: "lunafox.agent.execution.v1.RuntimeArtifactBinding"},
	})
}

func TestResolvedPlanNestedProtocolGroupsRemainClosed(t *testing.T) {
	assertDescriptorFields(t, (&agentexecutionv1.CanonicalTarget{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "resource", number: 1, kind: protoreflect.StringKind},
		{name: "type", number: 2, kind: protoreflect.EnumKind},
		{name: "value", number: 3, kind: protoreflect.StringKind},
	})
	assertDescriptorFields(t, (&agentexecutionv1.RuntimeImage{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "refs", number: 1, kind: protoreflect.StringKind, cardinality: protoreflect.Repeated},
	})
	assertDescriptorFields(t, (&agentexecutionv1.WorkflowStepScope{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "scan", number: 1, kind: protoreflect.StringKind},
		{name: "workflow", number: 2, kind: protoreflect.StringKind},
		{name: "stage_id", number: 3, kind: protoreflect.StringKind},
		{name: "step_id", number: 4, kind: protoreflect.StringKind},
	})
	assertDescriptorFields(t, (&agentexecutionv1.EngineRelease{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "engine", number: 1, kind: protoreflect.StringKind},
		{name: "package_digest", number: 2, kind: protoreflect.StringKind},
		{name: "engine_api_major", number: 3, kind: protoreflect.Uint32Kind},
		{name: "compatibility_revision", number: 4, kind: protoreflect.StringKind},
	})
	assertDescriptorFields(t, (&agentexecutionv1.FinalEngineConfig{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "sections", number: 1, kind: protoreflect.MessageKind, cardinality: protoreflect.Repeated, message: "lunafox.agent.execution.v1.EngineConfigSection"},
	})
	assertDescriptorFields(t, (&agentexecutionv1.EngineConfigSection{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "section_id", number: 1, kind: protoreflect.StringKind},
		{name: "enabled", number: 2, kind: protoreflect.BoolKind},
		{name: "params", number: 3, kind: protoreflect.MessageKind, cardinality: protoreflect.Repeated, message: "lunafox.agent.execution.v1.EngineConfigParam"},
	})
	assertDescriptorFields(t, (&agentexecutionv1.EngineConfigParam{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "key", number: 1, kind: protoreflect.StringKind},
		{name: "value", number: 2, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.EngineConfigScalar"},
	})
	scalar := (&agentexecutionv1.EngineConfigScalar{}).ProtoReflect().Descriptor()
	assertDescriptorFields(t, scalar, []descriptorField{
		{name: "bool_value", number: 1, kind: protoreflect.BoolKind},
		{name: "integer_value", number: 2, kind: protoreflect.Sint64Kind},
		{name: "string_value", number: 3, kind: protoreflect.StringKind},
		{name: "string_array_value", number: 4, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.StringArrayValue"},
	})
	if scalar.Oneofs().Len() != 1 || scalar.Oneofs().Get(0).Name() != "value" {
		t.Fatalf("EngineConfigScalar oneofs = %d, want value", scalar.Oneofs().Len())
	}
	assertDescriptorFields(t, (&agentexecutionv1.StringArrayValue{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "values", number: 1, kind: protoreflect.StringKind, cardinality: protoreflect.Repeated},
	})
	assertDescriptorFields(t, (&agentexecutionv1.ConfigResourceBinding{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "section_id", number: 1, kind: protoreflect.StringKind},
		{name: "param_key", number: 2, kind: protoreflect.StringKind},
		{name: "content_type", number: 3, kind: protoreflect.StringKind},
		{name: "wordlist", number: 4, kind: protoreflect.MessageKind, message: "lunafox.agent.execution.v1.WordlistDescriptor"},
	})
	assertDescriptorFields(t, (&agentexecutionv1.WordlistDescriptor{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "resource", number: 1, kind: protoreflect.StringKind},
		{name: "basename", number: 2, kind: protoreflect.StringKind},
		{name: "size_bytes", number: 3, kind: protoreflect.Uint64Kind},
		{name: "sha256_digest", number: 4, kind: protoreflect.StringKind},
		{name: "line_count", number: 5, kind: protoreflect.Uint64Kind},
	})
	assertDescriptorFields(t, (&agentexecutionv1.PlatformResourceBinding{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "resource_id", number: 1, kind: protoreflect.StringKind},
		{name: "content_type", number: 2, kind: protoreflect.StringKind},
	})
	assertDescriptorFields(t, (&agentexecutionv1.ExecutionLimits{}).ProtoReflect().Descriptor(), []descriptorField{
		{name: "max_execution_duration", number: 1, kind: protoreflect.MessageKind, message: "google.protobuf.Duration"},
		{name: "progress_message_max_bytes", number: 2, kind: protoreflect.Uint32Kind},
		{name: "result_batch_max_items", number: 3, kind: protoreflect.Uint32Kind},
		{name: "result_batch_max_bytes", number: 4, kind: protoreflect.Uint32Kind},
	})
}

func TestResolvedPlanValidationRejectsNodeAndLegacyFacts(t *testing.T) {
	plan := validPlanForTest()
	plan.RuntimeImage.Refs = []string{"docker.io/example/engine:latest"}
	if err := ValidateResolvedEngineExecutionPlan(plan); err == nil {
		t.Fatal("expected tag-only runtime image to be rejected")
	}
	plan = validPlanForTest()
	plan.Target.Type = agentexecutionv1.TargetType_TARGET_TYPE_IP
	plan.Target.Value = "2001:db8::1"
	if err := ValidateResolvedEngineExecutionPlan(plan); err == nil {
		t.Fatal("expected IPv6 target to be rejected")
	}
	plan = validPlanForTest()
	plan.RuntimeImage.Refs[0] = "docker.io/yyhuni/lunafox-engine-runtime-port-scan @ sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := ValidateResolvedEngineExecutionPlan(plan); err == nil {
		t.Fatal("expected non-canonical runtime image spacing to be rejected")
	}
}

func TestResolvedPlanValidationAcceptsClosedImmutableScope(t *testing.T) {
	if err := ValidateResolvedEngineExecutionPlan(validPlanForTest()); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
}

func TestResolvedPlanValidationRequiresPositiveValidLimits(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*agentexecutionv1.ExecutionLimits)
	}{
		{name: "missing duration", mutate: func(limits *agentexecutionv1.ExecutionLimits) { limits.MaxExecutionDuration = nil }},
		{name: "zero duration", mutate: func(limits *agentexecutionv1.ExecutionLimits) { limits.MaxExecutionDuration = &durationpb.Duration{} }},
		{name: "negative duration", mutate: func(limits *agentexecutionv1.ExecutionLimits) {
			limits.MaxExecutionDuration = &durationpb.Duration{Seconds: -1}
		}},
		{name: "invalid nanos", mutate: func(limits *agentexecutionv1.ExecutionLimits) {
			limits.MaxExecutionDuration = &durationpb.Duration{Nanos: 1_000_000_000}
		}},
		{name: "protobuf range overflow", mutate: func(limits *agentexecutionv1.ExecutionLimits) {
			limits.MaxExecutionDuration = &durationpb.Duration{Seconds: 315_576_000_001}
		}},
		{name: "zero progress bytes", mutate: func(limits *agentexecutionv1.ExecutionLimits) { limits.ProgressMessageMaxBytes = 0 }},
		{name: "zero result items", mutate: func(limits *agentexecutionv1.ExecutionLimits) { limits.ResultBatchMaxItems = 0 }},
		{name: "zero result bytes", mutate: func(limits *agentexecutionv1.ExecutionLimits) { limits.ResultBatchMaxBytes = 0 }},
		{name: "result items exceed platform maximum", mutate: func(limits *agentexecutionv1.ExecutionLimits) {
			limits.ResultBatchMaxItems = uint32(results.DefaultResultBatchMaxItems + 1)
		}},
		{name: "result bytes exceed platform maximum", mutate: func(limits *agentexecutionv1.ExecutionLimits) {
			limits.ResultBatchMaxBytes = uint32(results.DefaultResultBatchMaxBytes + 1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validPlanForTest()
			test.mutate(plan.Limits)
			if err := ValidateResolvedEngineExecutionPlan(plan); err == nil {
				t.Fatal("invalid limits were accepted")
			}
		})
	}
}

func TestResolvedPlanValidationRejectsNonCanonicalScopeAndBindingRelations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*agentexecutionv1.ResolvedEngineExecutionPlan)
		want   string
	}{
		{name: "execution resource", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.Execution = " executions/1 " }, want: "canonical identity"},
		{name: "execution control byte", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.Execution = "executions/1\x00" }, want: "canonical identity"},
		{name: "task resource", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.Task = "scanTasks/1" }, want: "canonical task"},
		{name: "task scan mismatch", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.WorkflowStep.Scan = "scans/2" }, want: "same scan"},
		{name: "target resource", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.Target.Resource = " targets/1 " }, want: "target.resource"},
		{name: "domain value", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.Target.Value = "bad_label.example.com" }, want: "canonical domain"},
		{name: "workflow resource", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.WorkflowStep.Workflow = "scanWorkflows/Default"
		}, want: "scan workflow"},
		{name: "stage ID", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.WorkflowStep.StageId = "Bad Stage" }, want: "stage_id"},
		{name: "step ID", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.WorkflowStep.StepId = "bad.step" }, want: "step_id"},
		{name: "engine identity", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.EngineRelease.Engine = "engine.example.unknown-name "
		}, want: "invalid engineId"},
		{name: "config section identity", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) { plan.Config.Sections[1].SectionId = "Naabu" }, want: "config section id"},
		{name: "config param identity", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.Config.Sections[1].Params[0].Key = "thread_count"
		}, want: "config param key"},
		{name: "binding section identity", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.ConfigResourceBindings[0].SectionId = "brute-force"
		}, want: "config section id"},
		{name: "binding param identity", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.ConfigResourceBindings[0].ParamKey = "word_list"
		}, want: "config param key"},
		{name: "wordlist resource", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.ConfigResourceBindings[0].Wordlist.Resource = "not-a-wordlist-resource"
		}, want: "canonical wordlist resource"},
		{name: "wordlist resource control byte", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.ConfigResourceBindings[0].Wordlist.Resource = "wordlists/default\x00"
		}, want: "canonical identity"},
		{name: "wordlist basename control byte", mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
			plan.ConfigResourceBindings[0].Wordlist.Basename = "words\x00.txt"
		}, want: "plain filename"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validPlanForTest()
			test.mutate(plan)
			err := ValidateResolvedEngineExecutionPlan(plan)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateResolvedEngineExecutionPlan() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolvedPlanValidationRequiresConfigBindingsToReferenceEnabledSections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*agentexecutionv1.ResolvedEngineExecutionPlan)
		want   string
	}{
		{
			name: "unknown section",
			mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
				plan.ConfigResourceBindings[0].SectionId = "dns"
			},
			want: "unknown config section",
		},
		{
			name: "disabled section",
			mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
				plan.Config.Sections[0].Enabled = false
			},
			want: "disabled config section",
		},
		{
			name: "duplicate binding",
			mutate: func(plan *agentexecutionv1.ResolvedEngineExecutionPlan) {
				plan.ConfigResourceBindings = append(plan.ConfigResourceBindings, plan.ConfigResourceBindings[0])
			},
			want: "duplicate key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validPlanForTest()
			test.mutate(plan)
			err := ValidateResolvedEngineExecutionPlan(plan)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateResolvedEngineExecutionPlan() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolvedPlanValidationAllowsDisabledSectionWithoutBindingAndStillValidatesPlatformResources(t *testing.T) {
	plan := validPlanForTest()
	plan.Config.Sections[0].Enabled = false
	plan.ConfigResourceBindings = nil
	if err := ValidateResolvedEngineExecutionPlan(plan); err != nil {
		t.Fatalf("disabled config section without binding rejected: %v", err)
	}

	plan.PlatformResourceBindings[0].ContentType = "application/octet-stream"
	err := ValidateResolvedEngineExecutionPlan(plan)
	if err == nil || !strings.Contains(err.Error(), "unsupported platform resource binding") {
		t.Fatalf("ValidateResolvedEngineExecutionPlan() error = %v, want platform resource validation error", err)
	}
}

func TestResolvedPlanValidationAcceptsDynamicallyNamedEngineIdentity(t *testing.T) {
	plan := validPlanForTest()
	plan.EngineRelease.Engine = "engine.lunafox.http2_probe"
	plan.RuntimeImage.Refs = []string{
		"docker.io/yyhuni/lunafox-engine-runtime-http2-probe@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}

	if err := ValidateResolvedEngineExecutionPlan(plan); err != nil {
		t.Fatalf("ValidateResolvedEngineExecutionPlan() error = %v", err)
	}
}

func validPlanForTest() *agentexecutionv1.ResolvedEngineExecutionPlan {
	return &agentexecutionv1.ResolvedEngineExecutionPlan{
		Execution: "executions/1",
		Task:      "scans/1/tasks/1",
		Target: &agentexecutionv1.CanonicalTarget{
			Resource: "targets/1",
			Type:     agentexecutionv1.TargetType_TARGET_TYPE_DOMAIN,
			Value:    "example.com",
		},
		WorkflowStep: &agentexecutionv1.WorkflowStepScope{
			Scan: "scans/1", Workflow: "scanWorkflows/default", StageId: "ports", StepId: "port_scan",
		},
		EngineRelease: &agentexecutionv1.EngineRelease{
			Engine: "engine.lunafox.port_scan", PackageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", EngineApiMajor: 2, CompatibilityRevision: engineexecutionpb.EngineExecutionDiagnosticsCompatibilityRevision,
		},
		RuntimeImage: &agentexecutionv1.RuntimeImage{Refs: []string{
			"docker.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}},
		Config: &agentexecutionv1.FinalEngineConfig{Sections: []*agentexecutionv1.EngineConfigSection{
			{SectionId: "bruteforce", Enabled: true},
			{
				SectionId: "naabu_active", Enabled: true, Params: []*agentexecutionv1.EngineConfigParam{{
					Key: "threads", Value: &agentexecutionv1.EngineConfigScalar{Value: &agentexecutionv1.EngineConfigScalar_IntegerValue{IntegerValue: 25}},
				}},
			},
		}},
		ConfigResourceBindings: []*agentexecutionv1.ConfigResourceBinding{{
			SectionId: "bruteforce", ParamKey: "wordlist", ContentType: "application/vnd.lunafox.wordlist.v1",
			Wordlist: &agentexecutionv1.WordlistDescriptor{Resource: "wordlists/1", Basename: "words.txt", SizeBytes: 4, Sha256Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", LineCount: 1},
		}},
		PlatformResourceBindings: []*agentexecutionv1.PlatformResourceBinding{{
			ResourceId: "subfinderProviderConfig", ContentType: "application/vnd.lunafox.subfinder-provider-config.v1+yaml",
		}},
		Limits: &agentexecutionv1.ExecutionLimits{
			MaxExecutionDuration: durationpb.New(60 * 1000000000), ProgressMessageMaxBytes: 1024, ResultBatchMaxItems: 10, ResultBatchMaxBytes: 1024,
		},
	}
}
