package application

import (
	"strings"

	"gopkg.in/yaml.v3"
)

func parseYAMLMapping(bytes []byte) (map[string]any, error) {
	var root map[string]any
	if err := yaml.Unmarshal(bytes, &root); err != nil {
		return nil, err
	}
	return root, nil
}

func marshalYAMLMapping(root map[string]any) (string, error) {
	payload, err := yaml.Marshal(root)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}
