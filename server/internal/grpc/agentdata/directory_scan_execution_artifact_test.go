package agentdata

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/contracts/executionartifact"
	agentdatav1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/data/v1"
)

func TestDirectoryWebsiteURLsArtifactAllowsFinalizedZeroRecordProduct(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.WorkflowStep.StageId = "directory_scan"
	plan.WorkflowStep.StepId = "directory_scan"
	plan.EngineRelease.Engine = "engine.lunafox.directory_scan"
	setPlanInputBindingForTest(plan, executionartifact.RoleWebsiteURLsInput, executionartifact.ContentTypeWebsiteURLs)
	resolver := newExecutionArtifactResolverForTest(
		t,
		plan,
		&executionArtifactTaskReaderStub{},
		executionArtifactDNSCursorStub{},
		executionArtifactHostPortCursorStub{},
	)

	artifact, err := resolver.AuthorizeExecutionInput(
		context.Background(),
		AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch},
		&agentdatav1.StreamExecutionInputRequest{
			Task: plan.GetTask(), Execution: plan.GetExecution(), Role: executionartifact.RoleWebsiteURLs,
		},
	)
	if err != nil {
		t.Fatalf("AuthorizeExecutionInput() error = %v", err)
	}
	if artifact.Preflight != nil || artifact.ContentType != executionartifact.ContentTypeWebsiteURLs {
		t.Fatalf("Directory WebsiteURLs artifact = %#v", artifact)
	}
	var output bytes.Buffer
	recordCount, err := artifact.Produce(context.Background(), &output)
	if err != nil || recordCount != 0 || output.Len() != 0 {
		t.Fatalf("zero-record Directory WebsiteURLs = count %d bytes %q error %v", recordCount, output.String(), err)
	}
	if plan.GetTarget().GetResource() != "targets/7" || plan.GetTarget().GetValue() != "example.com" || plan.GetWorkflowStep().GetStageId() != "directory_scan" {
		t.Fatalf("artifact resolution changed saved Directory scope: target=%#v workflow=%#v", plan.GetTarget(), plan.GetWorkflowStep())
	}
}

func TestDirectoryWebsiteURLsArtifactPreservesRawStoredOrderAndIdentity(t *testing.T) {
	plan := validExecutionArtifactPlan()
	plan.WorkflowStep.StageId = "directory_scan"
	plan.WorkflowStep.StepId = "directory_scan"
	plan.EngineRelease.Engine = "engine.lunafox.directory_scan"
	setPlanInputBindingForTest(plan, executionartifact.RoleWebsiteURLsInput, executionartifact.ContentTypeWebsiteURLs)

	const maximumPrefix = "https://outside.example/"
	maximum := maximumPrefix + strings.Repeat("x", 2000-len(maximumPrefix))
	values := []string{
		"HTTPS://Example.COM:443/path?x=%00#fragment",
		"https://example.com/?x=%0d%0aInjected",
		"https://example.com/%zz",
		"https://example.com/%zz",
		maximum,
	}
	blacklist := &executionInputBlacklistSnapshotSourceStub{}
	resolver := newExecutionArtifactResolverForTestWithExecutionInputSources(
		t,
		plan,
		&executionArtifactTaskReaderStub{},
		executionArtifactDNSCursorStub{},
		executionArtifactHostPortCursorStub{},
		executionArtifactWebsiteURLCursorStub{records: values},
		executionArtifactEndpointURLCursorStub{},
		blacklist,
		&executionWordlistSourceStub{},
		executionProviderConfigSourceStub{content: "alienvault: []\n"},
	)

	artifact, err := resolver.AuthorizeExecutionInput(
		context.Background(),
		AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch},
		&agentdatav1.StreamExecutionInputRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), Role: executionartifact.RoleWebsiteURLs},
	)
	if err != nil {
		t.Fatalf("AuthorizeExecutionInput() error = %v", err)
	}
	var output bytes.Buffer
	records, err := artifact.Produce(context.Background(), &output)
	if err != nil {
		t.Fatalf("produce raw WebsiteURLs: %v", err)
	}
	if records != uint64(len(values)) || output.String() != strings.Join(values, "\n")+"\n" {
		t.Fatalf("raw WebsiteURLs product = records %d bytes %q", records, output.String())
	}
	if blacklist.calls != 1 || blacklist.scanID != 23 {
		t.Fatalf("blacklist lookup = calls %d scanID %d, want one lookup for Scan 23", blacklist.calls, blacklist.scanID)
	}
}

func TestDirectoryWebsiteURLsArtifactRejectsUnsafeOrOversizedStoredLine(t *testing.T) {
	for name, value := range map[string]string{
		"actual control":  "https://example.com/\runsafe",
		"over 2000 bytes": "https://example.com/" + strings.Repeat("x", 2000),
	} {
		t.Run(name, func(t *testing.T) {
			plan := validExecutionArtifactPlan()
			plan.WorkflowStep.StageId = "directory_scan"
			plan.WorkflowStep.StepId = "directory_scan"
			plan.EngineRelease.Engine = "engine.lunafox.directory_scan"
			setPlanInputBindingForTest(plan, executionartifact.RoleWebsiteURLsInput, executionartifact.ContentTypeWebsiteURLs)
			resolver := newExecutionArtifactResolverForTestWithExecutionInputSources(
				t,
				plan,
				&executionArtifactTaskReaderStub{},
				executionArtifactDNSCursorStub{},
				executionArtifactHostPortCursorStub{},
				executionArtifactWebsiteURLCursorStub{records: []string{value}},
				executionArtifactEndpointURLCursorStub{},
				&executionInputBlacklistSnapshotSourceStub{},
				&executionWordlistSourceStub{},
				executionProviderConfigSourceStub{content: "alienvault: []\n"},
			)

			artifact, err := resolver.AuthorizeExecutionInput(
				context.Background(),
				AgentExecutionLease{AgentID: 42, SessionID: testAgentSessionID, SessionEpoch: testAgentSessionEpoch},
				&agentdatav1.StreamExecutionInputRequest{Task: plan.GetTask(), Execution: plan.GetExecution(), Role: executionartifact.RoleWebsiteURLs},
			)
			if err != nil {
				t.Fatalf("AuthorizeExecutionInput() error = %v", err)
			}
			var output bytes.Buffer
			if _, err := artifact.Produce(context.Background(), &output); err == nil {
				t.Fatal("unsafe WebsiteURLs record was accepted")
			}
			if output.Len() != 0 {
				t.Fatalf("unsafe WebsiteURLs record wrote %q", output.String())
			}
		})
	}
}
