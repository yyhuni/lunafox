package application

import "gopkg.in/yaml.v3"

func parseYAMLMapping(bytes []byte) (map[string]any, error) {
	var root map[string]any
	if err := yaml.Unmarshal(bytes, &root); err != nil {
		return nil, err
	}
	return root, nil
}
