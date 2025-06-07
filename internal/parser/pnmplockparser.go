package parser

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type PnpmLockParser struct{}

func (p PnpmLockParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "pnpm-lock.yaml")
}

func (p PnpmLockParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency
	var root map[string]interface{}

	if err := yaml.Unmarshal(data, &root); err != nil {
		return deps, err
	}

	importers, ok := root["importers"].(map[string]interface{})
	if !ok {
		return deps, nil
	}

	for _, importer := range importers {
		importerMap, ok := importer.(map[string]interface{})
		if !ok {
			continue
		}

		for _, section := range []string{"dependencies", "devDependencies"} {
			if entries, ok := importerMap[section].(map[string]interface{}); ok {
				for rawName, entry := range entries {
					entryMap, ok := entry.(map[string]interface{})
					if !ok {
						continue
					}
					version, _ := entryMap["version"].(string)
					version = cleanPnpmVersion(version)

					group, name := splitScoped(rawName)

					deps = append(deps, Dependency{
						Group:   group,
						Name:    name,
						Version: version,
						Type:    "pnpm",
						Path:    path,
					})
				}
			}
		}
	}

	return deps, nil
}

func (p PnpmLockParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
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

// --- Hjelpefunksjoner ---

func cleanPnpmVersion(v string) string {
	if i := strings.Index(v, "("); i != -1 {
		return strings.TrimSpace(v[:i])
	}
	return v
}

func splitScoped(name string) (group, actual string) {
	if strings.HasPrefix(name, "@") {
		if strings.Contains(name, "/") {
			parts := strings.SplitN(name, "/", 2)
			return parts[0], parts[1]
		}
		if strings.Contains(name, ":") {
			parts := strings.SplitN(name, ":", 2)
			return parts[0], parts[1]
		}
	}
	return "", name
}
