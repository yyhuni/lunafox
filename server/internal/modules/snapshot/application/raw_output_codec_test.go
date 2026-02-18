package application

import (
	"encoding/json"

	"gorm.io/datatypes"
)

type vulnerabilityRawOutputCodecStub struct{}

func newVulnerabilityRawOutputCodecStub() *vulnerabilityRawOutputCodecStub {
	return &vulnerabilityRawOutputCodecStub{}
}

func (stub *vulnerabilityRawOutputCodecStub) Encode(rawOutput map[string]any) (datatypes.JSON, error) {
	if rawOutput == nil {
		return datatypes.JSON([]byte("{}")), nil
	}
	jsonBytes, err := json.Marshal(rawOutput)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(jsonBytes), nil
}
