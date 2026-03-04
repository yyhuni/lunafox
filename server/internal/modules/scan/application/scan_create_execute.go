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
		return nil, WrapSchemaInvalid("", "failed to parse configuration YAML", err)
	}
	if root == nil {
		return nil, WrapSchemaInvalid("", "configuration YAML must be an object", nil)
	}

	engineNames := append([]string(nil), input.EngineNames...)
	if err := validateEngineNamesStrict(engineNames); err != nil {
		return nil, err
	}
	if err := validateEngineIdentityConsistency(input.EngineIDs, engineNames); err != nil {
		return nil, err
	}
	if err := service.validateEngineCatalogCoverage(engineNames); err != nil {
		return nil, err
	}

	for _, engine := range engineNames {
		engine = strings.TrimSpace(engine)
		if engine == "" {
			continue
		}
		if err := engineschema.ValidateYAML(engine, []byte(configYAML)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, WrapSchemaInvalid(engine, fmt.Sprintf("engine %s does not support this configuration version", engine), err)
			}
			return nil, WrapSchemaInvalid(engine, fmt.Sprintf("engine %s configuration failed schema validation", engine), err)
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

	tasks, err := buildScanTasks(engineNames, root)
	if err != nil {
		return nil, err
	}
	if err := service.scanStore.CreateWithTasks(scan, tasks); err != nil {
		return nil, err
	}
	scan.Target = &TargetRef{ID: target.ID, Name: target.Name, Type: target.Type, CreatedAt: timeutil.ToUTC(target.CreatedAt), LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt), DeletedAt: timeutil.ToUTCPtr(target.DeletedAt)}
	return scan, nil
}
