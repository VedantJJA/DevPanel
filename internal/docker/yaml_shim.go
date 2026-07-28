package docker

import "gopkg.in/yaml.v3"

func yamlLibUnmarshal(data []byte, v interface{}) error { return yaml.Unmarshal(data, v) }
