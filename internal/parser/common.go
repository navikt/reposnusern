package parser

// Dependency er en generisk representasjon av et avhengighetsforhold i et prosjekt.
// Kan komme fra Maven, npm, pip, osv.
type Dependency struct {
	Name    string
	Group   string
	Version string
	Type    string
	Path    string
}

// Parser støtter begge metoder: enkeltfiler og repo-nivå.
// ParseRepoFiles er valgfri – returner nil hvis ikke støttet.
type Parser interface {
	CanParse(filename string) bool
	ParseFile(path string, content []byte) ([]Dependency, error)
	ParseRepo(files map[string][]byte) ([]Dependency, error)
}
