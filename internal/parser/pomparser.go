package parser

import (
	"encoding/xml"
	"regexp"
	"strings"
)

type PomParser struct{}

func (p PomParser) CanParse(filename string) bool {
	return strings.HasSuffix(filename, ".xml")
}

func (p PomParser) ParseFile(path string, data []byte) ([]Dependency, error) {
	deps, err := ParseSinglePomFile(data)
	if err != nil {
		return nil, err
	}
	for i := range deps {
		deps[i].Path = path
	}
	return deps, nil
}

func (p PomParser) ParseRepo(files map[string][]byte) ([]Dependency, error) {
	return ParseRepoPomFiles(files)
}

// Project representerer en Maven POM-fil.
type Project struct {
	XMLName              xml.Name          `xml:"http://maven.apache.org/POM/4.0.0 project"`
	ModelVersion         string            `xml:"modelVersion"`
	GroupID              string            `xml:"groupId"`
	ArtifactID           string            `xml:"artifactId"`
	Version              string            `xml:"version"`
	Packaging            string            `xml:"packaging"`
	Name                 string            `xml:"name"`
	Modules              []string          `xml:"modules>module"`
	Properties           Properties        `xml:"properties"`
	Dependencies         []mavenDependency `xml:"dependencies>dependency"`
	DependencyManagement struct {
		Dependencies []mavenDependency `xml:"dependencies>dependency"`
	} `xml:"dependencyManagement"`
}

type Properties struct {
	Entries []KV `xml:",any"`
}

type KV struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// Dependency representerer en Maven dependency.
// Path refererer til filen dependencyen ble funnet i.
type mavenDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Type       string `xml:"type"`
}

// --- Intern versjonsoppløsing ---
func interpolate(version string, props map[string]string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(version, -1)

	for _, match := range matches {
		key := match[1]
		if _, ok := props[key]; !ok {
			return "unparsed-version"
		}
	}

	result := re.ReplaceAllStringFunc(version, func(s string) string {
		key := re.FindStringSubmatch(s)[1]
		return props[key] // kan være tom, og det er OK
	})

	return result
}

func propsToMap(p Properties) map[string]string {
	m := make(map[string]string)
	for _, kv := range p.Entries {
		m[kv.XMLName.Local] = kv.Value
	}
	return m
}

// ParseSinglePomFile parser en enkelt pom.xml-fil og returnerer løste dependencies.
// Interpolerer versjonsvariabler lokalt i filen.
func ParseSinglePomFile(data []byte) ([]Dependency, error) {
	var proj Project
	if err := xml.Unmarshal(data, &proj); err != nil {
		return nil, err
	}
	props := propsToMap(proj.Properties)
	props["project.version"] = interpolate(proj.Version, props)

	var all []mavenDependency
	all = append(all, proj.Dependencies...)
	all = append(all, proj.DependencyManagement.Dependencies...)

	var resolved []Dependency
	for _, dep := range all {
		if dep.Version == "" {
			continue
		}
		version := interpolate(dep.Version, props)
		if version == "" {
			version = "unparsed-version"
		}
		resolved = append(resolved, Dependency{
			Name:    dep.ArtifactID,
			Group:   dep.GroupID,
			Version: version,
			Type:    "maven",
			Path:    "", // ikke kjent i enkeltfil
		})
	}
	return resolved, nil
}

// ParseRepoPomFiles parser flere pom.xml-filer og løser dependencies
// på tvers av filer ved å bruke samlet props og project.version.
func ParseRepoPomFiles(files map[string][]byte) ([]Dependency, error) {
	allProjects := make(map[string]Project)
	allProps := make(map[string]string)
	projectVersions := make(map[string]string)

	// Første pass: samle alle props og project.version per fil
	for path, data := range files {
		var proj Project
		if err := xml.Unmarshal(data, &proj); err != nil {
			return nil, err
		}
		allProjects[path] = proj

		props := propsToMap(proj.Properties)
		for k, v := range props {
			allProps[k] = v
		}
		projectVersions[path] = proj.Version
	}

	// Andre pass: parse dependencies og resolve versjoner
	var allDeps []Dependency
	for path, proj := range allProjects {
		// Legg til interpolert project.version for akkurat denne filen
		localProps := make(map[string]string)
		for k, v := range allProps {
			localProps[k] = v
		}
		localProps["project.version"] = interpolate(projectVersions[path], allProps)

		all := append(proj.Dependencies, proj.DependencyManagement.Dependencies...)
		for _, dep := range all {
			if dep.Version == "" {
				continue
			}
			version := interpolate(dep.Version, localProps)
			if version == "" {
				version = "unparsed-version"
			}
			allDeps = append(allDeps, Dependency{
				Name:    dep.ArtifactID,
				Group:   dep.GroupID,
				Version: version,
				Type:    "maven",
				Path:    path,
			})
		}

	}

	return allDeps, nil
}
