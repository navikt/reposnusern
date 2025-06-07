package parser

import (
	"strings"

	"github.com/pelletier/go-toml"
)

type CargoTomlParser struct{}

func (p CargoTomlParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "Cargo.toml")
}

func (p CargoTomlParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency

	tree, err := toml.LoadBytes(data)
	if err != nil {
		return deps, err
	}

	for _, section := range []string{"dependencies", "dev-dependencies", "build-dependencies"} {
		if subTree := tree.Get(section); subTree != nil {
			if m, ok := subTree.(*toml.Tree); ok {
				for name, value := range m.ToMap() {
					version := ""
					switch v := value.(type) {
					case string:
						version = v
					case map[string]interface{}:
						if ver, ok := v["version"].(string); ok {
							version = ver
						}
					}
					deps = append(deps, Dependency{
						Name:    name,
						Version: version,
						Group:   "", // Ikke brukt i Cargo
						Type:    "cargo",
						Path:    path,
					})
				}
			}
		}
	}

	return deps, nil
}

func (p CargoTomlParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
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
