package parser

import (
	"regexp"
	"strings"
)

type GradleGroovyParser struct {
	KnownVars map[string]string // valgfri: kan settes eksternt
}

func (p GradleGroovyParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "build.gradle")
}

func (p GradleGroovyParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	content := string(data)

	// Trekk ut variabler hvis ikke allerede satt
	vars := p.KnownVars
	if vars == nil {
		vars = extractGradleVars(content)
	}

	return parseGradleGroovy(path, content, vars), nil
}

func (p GradleGroovyParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		if p.CanParse(path) {
			content := string(data)

			vars := p.KnownVars
			if vars == nil {
				vars = extractGradleVars(content)
			}

			all = append(all, parseGradleGroovy(path, content, vars)...)
		}
	}
	return all, nil
}

// Fanger alle dependency-strenger uavhengig av formatet rundt
var gradleDepLineRegex = regexp.MustCompile(`["']([\w\.-]+):([\w\.-]+):([^"']+)["']`)

func parseGradleGroovy(path, content string, vars map[string]string) []Dependency {
	var deps []Dependency

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//") || !strings.Contains(line, ":") {
			continue
		}

		// Fanger én eller flere dependencies per linje
		matches := gradleDepLineRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			group := m[1]
			name := m[2]
			version := interpolateGradle(m[3], vars)

			// Hopp over SCM/git-linjer og publisering
			if strings.HasPrefix(group, "scm") || strings.Contains(version, "scm:") {
				continue
			}

			deps = append(deps, Dependency{
				Group:   group,
				Name:    name,
				Version: version,
				Type:    "gradle",
				Path:    path,
			})
		}
	}

	return deps
}

// --- Versjonsoppløsing ---
func interpolateGradle(version string, props map[string]string) string {
	if props == nil {
		return version
	}
	re := regexp.MustCompile(`\$\{?([\w\.-]+)\}?`)
	return re.ReplaceAllStringFunc(version, func(s string) string {
		match := re.FindStringSubmatch(s)
		if len(match) > 1 {
			if val, ok := props[match[1]]; ok {
				return val
			}
		}
		return "unparsed-version"
	})
}

// --- Utpakking av variabler fra build.gradle ---
func extractGradleVars(content string) map[string]string {
	vars := make(map[string]string)

	// Fanger f.eks.:
	//   val jacksonVersion = "2.19.0"
	//   ext.foo = '1.0.0'
	//   project.version = "1.2.3"
	re := regexp.MustCompile(`(?m)^\s*(?:val|ext\.|project\.)\s*([\w\.-]+)\s*=\s*['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		vars[m[1]] = m[2]
	}
	return vars
}
