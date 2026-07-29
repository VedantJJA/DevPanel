package docker

import (
	"gopkg.in/yaml.v3"
)

// parseBlueprintYAML automatically translates Render-style YAML into DevPanel Blueprint format.
func parseBlueprintYAML(data []byte, v interface{}) error {
	var generic map[string]interface{}
	if err := yaml.Unmarshal(data, &generic); err != nil {
		return yaml.Unmarshal(data, v) // Fallback to raw unmarshal if not a map
	}

	servicesRaw, ok := generic["services"]
	if !ok {
		return yaml.Unmarshal(data, v)
	}

	// Detect if 'services' is a list (Render format) instead of a map (DevPanel format)
	if svcList, isList := servicesRaw.([]interface{}); isList {
		svcMap := make(map[string]interface{})

		for _, sRaw := range svcList {
			s, ok := sRaw.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := s["name"].(string)
			if name == "" {
				name = "web"
			}

			renderType, _ := s["type"].(string) // "web", "worker", "psql"
			renderEnv, _ := s["env"].(string)   // "node", "static", "python"

			// Translate Render to DevPanel types
			devType := "web"
			if renderType == "worker" {
				devType = "worker"
			} else if renderEnv == "static" {
				devType = "static"
			}

			buildMap := make(map[string]interface{})
			if cmd, ok := s["buildCommand"].(string); ok {
				buildMap["command"] = cmd
			}
			if out, ok := s["staticPublishPath"].(string); ok {
				buildMap["output_dir"] = out
			}
			if renderEnv != "" {
				buildMap["engine"] = renderEnv
			}

			deployMap := make(map[string]interface{})
			if cmd, ok := s["startCommand"].(string); ok {
				deployMap["command"] = cmd
			}

			// Translate envVars list to env map
			if envVars, ok := s["envVars"].([]interface{}); ok {
				envMap := make(map[string]interface{})
				for _, evRaw := range envVars {
					if ev, ok := evRaw.(map[string]interface{}); ok {
						key, _ := ev["key"].(string)
						if val, ok := ev["value"].(string); ok {
							envMap[key] = val
						} else if gen, ok := ev["generateValue"].(bool); ok && gen {
							envMap[key] = "generated_devpanel_secret"
						}
					}
				}
				deployMap["env"] = envMap
			}

			sourceMap := make(map[string]interface{})
			if root, ok := s["rootDir"].(string); ok {
				sourceMap["directory"] = root
			}

			// Assemble DevPanel service config
			svcMap[name] = map[string]interface{}{
				"type":   devType,
				"build":  buildMap,
				"deploy": deployMap,
				"source": sourceMap,
			}
		}

		// Replace the list with the generated map
		generic["services"] = svcMap
	}

	// Re-marshal to bytes and unmarshal into the target struct
	translated, err := yaml.Marshal(generic)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(translated, v)
}

func yamlLibUnmarshal(data []byte, v interface{}) error {
	return parseBlueprintYAML(data, v)
}
