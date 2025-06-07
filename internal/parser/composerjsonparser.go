package parser

import (
	"encoding/json"
	"strings"
)

type ComposerJSONParser struct{}

func (p ComposerJSONParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "composer.json")
}

func (p ComposerJSONParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	return ParseSingleComposerFile(path, data)
}

func (p ComposerJSONParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if p.CanParse(path) {
			deps, err := ParseSingleComposerFile(path, data)
			if err != nil {
				return nil, err
			}
			all = append(all, deps...)
		}
	}
	return all, nil
}

func ParseSingleComposerFile(path string, data []byte) ([]Dependency, error) {
	var raw struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var deps []Dependency
	for name, version := range raw.Require {
		group, pkg := splitComposerName(name)
		deps = append(deps, Dependency{
			Name:    pkg,
			Group:   group,
			Version: version,
			Type:    "composer",
			Path:    path,
		})
	}
	for name, version := range raw.RequireDev {
		group, pkg := splitComposerName(name)
		deps = append(deps, Dependency{
			Name:    pkg,
			Group:   group,
			Version: version,
			Type:    "composer",
			Path:    path,
		})
	}
	return deps, nil
}

func splitComposerName(name string) (group string, pkg string) {
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		return parts[0], parts[1]
	}
	return "", name
}
