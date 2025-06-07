package parser

import (
	"encoding/json"
	"strings"
)

type PackageJSONParser struct{}

func (p PackageJSONParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, "package.json") || strings.HasSuffix(filename, "package-lock.json")
}

func (p PackageJSONParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	return ParseSinglePJFile(path, data)
}

func (p PackageJSONParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	return ParseRepoPJFiles(files)
}

// ParseSinglePJFile parser én fil (enten package.json eller package-lock.json)
func ParseSinglePJFile(path string, data []byte) ([]Dependency, error) {
	var deps []Dependency
	var err error

	if strings.HasSuffix(path, "package.json") {
		deps, err = ParsePackageJSON(path, data)
	} else if strings.HasSuffix(path, "package-lock.json") {
		deps, err = ParsePackageLockJSON(path, data)
	} else {
		return nil, nil // ignorer andre filer
	}

	for i := range deps {
		deps[i].Path = path
		deps[i].Type = "npm"
	}
	return deps, err
}

// ParseRepoPJFiles parser alle relevante npm-filer i et repo
func ParseRepoPJFiles(files map[string][]byte) ([]Dependency, error) {
	var all []Dependency
	for path, data := range files {
		deps, err := ParseSinglePJFile(path, data)
		if err != nil {
			return nil, err
		}
		all = append(all, deps...)
	}
	return all, nil
}

// ParsePackageJSON parser dependencies og devDependencies fra en package.json
func ParsePackageJSON(path string, data []byte) ([]Dependency, error) {
	var raw struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var deps []Dependency
	for name, version := range raw.Dependencies {
		if version != "" {
			group, base := splitNpmScopedName(name)
			deps = append(deps, Dependency{
				Group:   group,
				Name:    base,
				Version: version,
				Type:    "npm",
				Path:    path,
			})
		}
	}
	for name, version := range raw.DevDependencies {
		if version != "" {
			group, base := splitNpmScopedName(name)
			deps = append(deps, Dependency{
				Group:   group,
				Name:    base,
				Version: version,
				Type:    "npm",
				Path:    path,
			})
		}
	}
	return deps, nil
}

// ParsePackageLockJSON parser resolved versjoner fra en package-lock.json
func ParsePackageLockJSON(path string, data []byte) ([]Dependency, error) {
	var raw struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var deps []Dependency
	for pkgPath, entry := range raw.Packages {
		if entry.Version == "" || !strings.HasPrefix(pkgPath, "node_modules/") {
			continue
		}
		name := strings.TrimPrefix(pkgPath, "node_modules/")
		group, base := splitNpmScopedName(name)
		deps = append(deps, Dependency{
			Group:   group,
			Name:    base,
			Version: entry.Version,
			Type:    "npm",
			Path:    path,
		})
	}
	return deps, nil
}

// splitNpmScopedName deler @scope/navn i ("@scope", "navn")
// returnerer ("", fullName) hvis ikke scoped
func splitNpmScopedName(fullName string) (string, string) {
	if strings.HasPrefix(fullName, "@") {
		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return "", fullName
}
