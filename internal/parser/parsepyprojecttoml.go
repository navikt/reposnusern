package parser

import (
	"strings"

	"github.com/BurntSushi/toml"
)

type PyProjectParser struct{}

func (p PyProjectParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "pyproject.toml")
}

func (p PyProjectParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency

	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return deps, err
	}

	// PEP 621
	if project, ok := raw["project"].(map[string]interface{}); ok {
		if dependencies, ok := project["dependencies"].([]interface{}); ok {
			for _, dep := range dependencies {
				if depStr, ok := dep.(string); ok {
					name, version := splitPythonDep(depStr)
					deps = append(deps, Dependency{
						Name:    name,
						Version: version,
						Type:    "pyproject",
						Path:    path,
					})
				}
			}
		}
	}

	// Poetry
	if tool, ok := raw["tool"].(map[string]interface{}); ok {
		if poetry, ok := tool["poetry"].(map[string]interface{}); ok {
			if dependencies, ok := poetry["dependencies"].(map[string]interface{}); ok {
				for name, versionRaw := range dependencies {
					if name == "python" {
						continue // hopp over Python-versjon
					}
					version := ""
					switch v := versionRaw.(type) {
					case string:
						version = v
					case map[string]interface{}:
						// Håndter spesialtilfeller senere
						version = "complex"
					}
					deps = append(deps, Dependency{
						Name:    name,
						Version: version,
						Type:    "pyproject",
						Path:    path,
					})
				}
			}
		}
	}

	return deps, nil
}

func (p PyProjectParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if p.CanParse(path) {
			deps, err := p.ParseFile(path, data)
			if err != nil {
				continue
			}
			all = append(all, deps...)
		}
	}
	return all, nil
}

func splitPythonDep(dep string) (string, string) {
	for _, sep := range []string{">=", "==", "<=", "~=", "!=", "<", ">", "="} {
		if parts := strings.SplitN(dep, sep, 2); len(parts) == 2 {
			return strings.TrimSpace(parts[0]), sep + strings.TrimSpace(parts[1])
		}
	}
	return dep, ""
}
