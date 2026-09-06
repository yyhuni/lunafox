package repository

import (
	"testing"

	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/datatypes"
)

func TestScanWorkflowModelToDomainRejectsMalformedOrInvalidTopology(t *testing.T) {
	base := model.ScanWorkflow{
		ScanWorkflowID: "wf-123e4567-e89b-12d3-a456-426614174000",
		DisplayName:    "Discovery",
		Description:    "Run discovery.",
		IsBuiltin:      false,
		Version:        1,
	}
	for name, stages := range map[string]datatypes.JSON{
		"malformed json":              datatypes.JSON(`{`),
		"empty topology":              datatypes.JSON(`[]`),
		"missing profile default":     datatypes.JSON(`[{"stageId":"one","steps":[{"stepId":"same","engineId":"engine.lunafox.one"}]}]`),
		"null profile default":        datatypes.JSON(`[{"stageId":"one","steps":[{"stepId":"same","engineId":"engine.lunafox.one","profileDefaultEnabled":null}]}]`),
		"non-boolean profile default": datatypes.JSON(`[{"stageId":"one","steps":[{"stepId":"same","engineId":"engine.lunafox.one","profileDefaultEnabled":"true"}]}]`),
		"duplicate step":              datatypes.JSON(`[{"stageId":"one","steps":[{"stepId":"same","engineId":"engine.lunafox.one","profileDefaultEnabled":true}]},{"stageId":"two","steps":[{"stepId":"same","engineId":"engine.lunafox.two","profileDefaultEnabled":true}]}]`),
	} {
		t.Run(name, func(t *testing.T) {
			record := base
			record.Stages = stages
			if _, err := scanWorkflowModelToDomain(&record); err == nil {
				t.Fatal("expected persisted topology to fail closed")
			}
		})
	}
}

func TestScanWorkflowDomainToModelPreservesOrderedStages(t *testing.T) {
	workflow := &catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: "wf-123e4567-e89b-12d3-a456-426614174000",
		DisplayName:    "Discovery",
		Description:    "Run discovery.",
		Stages: []scanworkflow.Stage{
			{StageID: "first", Steps: []scanworkflow.Step{{StepID: "first_step", EngineID: "engine.lunafox.first", ProfileDefaultEnabled: true}}},
			{StageID: "second", Steps: []scanworkflow.Step{{StepID: "second_step", EngineID: "engine.lunafox.second", ProfileDefaultEnabled: false}}},
		},
	}
	record, err := scanWorkflowDomainToModel(workflow)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := scanWorkflowModelToDomain(record)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Stages[0].StageID != "first" || decoded.Stages[1].Steps[0].StepID != "second_step" {
		t.Fatalf("workflow stage order changed: %+v", decoded.Stages)
	}
	if decoded.Stages[0].Steps[0].ProfileDefaultEnabled != true || decoded.Stages[1].Steps[0].ProfileDefaultEnabled != false {
		t.Fatalf("workflow Profile defaults changed during persistence round-trip: %+v", decoded.Stages)
	}
}
