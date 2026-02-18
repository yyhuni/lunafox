package infrastructure

import (
	"encoding/json"

	"gorm.io/datatypes"
)

type VulnerabilityJSONRawOutputCodec struct{}

func NewVulnerabilityJSONRawOutputCodec() *VulnerabilityJSONRawOutputCodec {
	return &VulnerabilityJSONRawOutputCodec{}
}

func (codec *VulnerabilityJSONRawOutputCodec) Encode(rawOutput map[string]any) (datatypes.JSON, error) {
	if rawOutput == nil {
		return datatypes.JSON([]byte("{}")), nil
	}
	jsonBytes, err := json.Marshal(rawOutput)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(jsonBytes), nil
}
