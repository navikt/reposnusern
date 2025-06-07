package parser

import (
	"bufio"
	"bytes"
	"strings"
)

type RequirementsParser struct{}

func (p RequirementsParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "requirements.txt")
}

func (p RequirementsParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	return parseRequirements(path, data)
}

func (p RequirementsParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if p.CanParse(path) {
			deps, err := parseRequirements(path, data)
			if err != nil {
				return nil, err
			}
			all = append(all, deps...)
		}
	}
	return all, nil
}

func parseRequirements(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "==") {
			parts := strings.SplitN(line, "==", 2)
			deps = append(deps, Dependency{
				Name:    strings.TrimSpace(parts[0]),
				Version: strings.TrimSpace(parts[1]),
				Type:    "pip",
				Path:    path,
			})
		} else {
			// Ingen versjon spesifisert
			deps = append(deps, Dependency{
				Name:    line,
				Version: "unparsed-version",
				Type:    "pip",
				Path:    path,
			})
		}
	}

	return deps, scanner.Err()
}
