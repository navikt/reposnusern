package parser

import (
	"bufio"
	"bytes"
	"strings"
)

type GoModParser struct{}

func (p GoModParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "go.mod")
}

func (p GoModParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	return parseGoMod(path, data)
}

func (p GoModParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if p.CanParse(path) {
			deps, err := parseGoMod(path, data)
			if err != nil {
				return nil, err
			}
			all = append(all, deps...)
		}
	}
	return all, nil
}

func parseGoMod(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency
	scanner := bufio.NewScanner(bytes.NewReader(data))

	inRequireBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimPrefix(line, "require ")
		} else if !inRequireBlock {
			continue
		}

		line = strings.TrimSpace(strings.Split(line, "//")[0]) // Fjern kommentarer
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		full := parts[0]
		version := parts[1]

		group, name := splitGoImportPath(full)

		deps = append(deps, Dependency{
			Group:   group,
			Name:    name,
			Version: version,
			Type:    "go",
			Path:    path,
		})
	}
	return deps, scanner.Err()
}

func splitGoImportPath(path string) (group, name string) {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", path
	}
	group = strings.Join(parts[:len(parts)-1], "/")
	name = parts[len(parts)-1]
	return
}
