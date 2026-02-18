package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/engineschema"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func (service *ScanCreateService) CreateNormal(input *CreateNormalInput) (*CreateScan, error) {
	if input == nil {
		return nil, ErrCreateInvalidConfig
	}
	if input.TargetID == 0 {
		return nil, ErrCreateTargetNotFound
	}

	configYAML := strings.TrimSpace(input.Configuration)
	if configYAML == "" {
		return nil, ErrCreateInvalidConfig
	}

	root, err := parseYAMLMapping([]byte(configYAML))
	if err != nil {
		return nil, fmt.Errorf("%w: parse yaml: %v", ErrCreateInvalidConfig, err)
	}
	if root == nil {
		return nil, fmt.Errorf("%w: yaml must be a mapping", ErrCreateInvalidConfig)
	}

	if len(input.EngineNames) != 1 {
		return nil, ErrCreateInvalidEngineNames
	}

	engineNames, err := normalizeEngineNames(input.EngineNames)
	if err != nil {
		return nil, err
	}
	if len(engineNames) != 1 {
		return nil, ErrCreateInvalidEngineNames
	}

	for _, engine := range engineNames {
		engine = strings.TrimSpace(engine)
		if engine == "" {
			continue
		}
		if err := engineschema.ValidateYAML(engine, []byte(configYAML)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("%w: %s: %v", ErrCreateInvalidConfig, engine, err)
		}
	}

	if service.targetLookup == nil {
		return nil, ErrCreateTargetLookupNotReady
	}
	target, err := service.targetLookup(input.TargetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrCreateTargetNotFound
	}

	engineNamesJSON, err := json.Marshal(engineNames)
	if err != nil {
		return nil, err
	}

	engineIDs := make([]int64, 0, len(input.EngineIDs))
	for _, id := range input.EngineIDs {
		engineIDs = append(engineIDs, int64(id))
	}

	scan := &CreateScan{
		TargetID:          input.TargetID,
		EngineIDs:         engineIDs,
		EngineNames:       engineNamesJSON,
		YamlConfiguration: configYAML,
		ScanMode:          CreateScanModeFull,
		Status:            CreateScanStatusPending,
	}
	inputs := []CreateScanInputTarget{{Value: target.Name, InputType: target.Type}}

	tasks, err := buildScanTasks(engineNames, root)
	if err != nil {
		return nil, err
	}
	if err := service.scanStore.CreateWithInputTargetsAndTasks(scan, inputs, tasks); err != nil {
		return nil, err
	}
	scan.Target = &TargetRef{ID: target.ID, Name: target.Name, Type: target.Type, CreatedAt: timeutil.ToUTC(target.CreatedAt), LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt), DeletedAt: timeutil.ToUTCPtr(target.DeletedAt)}
	return scan, nil
}
