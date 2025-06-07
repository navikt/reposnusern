package parser

import (
	"regexp"
	"strings"
)

type BuildGradleKtsParser struct{}

func (p BuildGradleKtsParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "build.gradle.kts")
}

func (p BuildGradleKtsParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	return ParseSingleBGKFile(path, data)
}

func (p BuildGradleKtsParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	return ParseRepoBGKFiles(files)
}

func ParseSingleBGKFile(path string, data []byte) ([]Dependency, error) {
	deps, err := ParseGradleKTS(data)
	if err != nil {
		return nil, err
	}
	for i := range deps {
		deps[i].Path = path
		deps[i].Type = "gradle"
	}
	return deps, nil
}

func ParseRepoBGKFiles(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if strings.HasSuffix(path, "build.gradle.kts") {
			deps, err := ParseSingleBGKFile(path, data)
			if err != nil {
				return nil, err
			}
			all = append(all, deps...)
		}
	}
	return all, nil
}

// ParseGradleKTS parser en .kts-fil og returnerer dependencies med interpolerte versjoner
func ParseGradleKTS(data []byte) ([]Dependency, error) {
	content := string(data)
	props := make(map[string]string)

	valDefRegex := regexp.MustCompile(`(?m)val\s+(\w+)\s*=\s*"([^"]*)"`)
	valByRegex := regexp.MustCompile(`(?m)val\s+(\w+)\s+by\s+project`)
	depRegex := regexp.MustCompile(`(?m)^\s*(implementation|api|testImplementation)\(\s*["']([^:"']+):([^:"']+):([^"']+)["']\s*\)`)
	varRefRegex := regexp.MustCompile(`\$(\w+)`)

	// Samle val x = "1.2.3"
	for _, match := range valDefRegex.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 {
			props[match[1]] = match[2]
		}
	}

	// Sett tom string for val x by project
	for _, match := range valByRegex.FindAllStringSubmatch(content, -1) {
		if len(match) >= 2 {
			props[match[1]] = ""
		}
	}

	// Finn dependencies
	matches := depRegex.FindAllStringSubmatch(content, -1)
	var deps []Dependency

	for _, m := range matches {
		if len(m) < 5 {
			continue // Beskyttelse mot kort match
		}
		group := m[2]
		name := m[3]
		versionRaw := m[4]
		version := varRefRegex.ReplaceAllStringFunc(versionRaw, func(s string) string {
			key := s[1:] // fjern $
			if val, ok := props[key]; ok {
				return val
			}
			return "unparsed-version"
		})
		if version == "" {
			version = "unparsed-version"
		}
		deps = append(deps, Dependency{
			Group:   group,
			Name:    name,
			Version: version,
		})
	}

	return deps, nil
}
