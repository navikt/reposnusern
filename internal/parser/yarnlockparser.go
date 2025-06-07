package parser

import (
	"strings"
)

type YarnLockParser struct{}

func (p YarnLockParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "yarn.lock")
}

func (p YarnLockParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	return parseYarnLock(path, string(data)), nil
}

func (p YarnLockParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if p.CanParse(path) {
			all = append(all, parseYarnLock(path, string(data))...)
		}
	}
	return all, nil
}

func parseYarnLock(path, content string) []Dependency {
	var deps []Dependency
	lines := strings.Split(content, "\n")
	var currentName string
	var currentVersion string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Ny blokk starter
		if strings.HasSuffix(line, ":") {
			keyLine := strings.TrimSuffix(line, ":")
			entries := strings.Split(keyLine, ",")
			for _, entry := range entries {
				entry = strings.Trim(entry, ` "`)
				atIndex := strings.LastIndex(entry, "@")
				if atIndex > 0 {
					currentName = entry[:atIndex]
					break
				}
			}
		} else if strings.HasPrefix(line, "version ") && currentName != "" {
			currentVersion = strings.Trim(strings.TrimPrefix(line, "version "), `"`)
			deps = append(deps, Dependency{
				Name:    currentName,
				Version: currentVersion,
				Type:    "yarn",
				Path:    path,
			})
			// reset currentName for neste blokk
			currentName = ""
		}
	}
	return deps
}
