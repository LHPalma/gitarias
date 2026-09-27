package config

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

type fileConfig struct {
	Lang     string `yaml:"lang"`
	Branches struct {
		Protected []string `yaml:"protected"`
		Base      string   `yaml:"base"`
	} `yaml:"branches"`
}

func parseFile(path string, content []byte) (fileConfig, error) {
	var parsed fileConfig
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return fileConfig{}, fmt.Errorf("arquivo de configuração %s: %w", path, err)
	}

	if parsed.Lang != "" && parsed.Lang != "en" {
		return fileConfig{}, fmt.Errorf("arquivo de configuração %s: lang %q não suportada nesta versão; só en está implementado", path, parsed.Lang)
	}

	return parsed, nil
}
