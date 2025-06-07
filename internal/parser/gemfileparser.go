package parser

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

type GemfileParser struct{}

var gemRegex = regexp.MustCompile(`gem ['"]([^'"]+)['"](?:,\s*['"]([^'"]+)['"])?`)

func (p GemfileParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "Gemfile")
}

func (p GemfileParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := gemRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			version := ""
			if len(matches) >= 3 {
				version = matches[2]
			}
			deps = append(deps, Dependency{
				Name:    name,
				Version: version,
				Group:   "", // Ikke brukt i Ruby
				Type:    "gem",
				Path:    path,
			})
		}
	}

	return deps, nil
}

func (p GemfileParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
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
