package parser

type MultiParser struct {
	all []Parser
}

func NewMultiParser() *MultiParser {
	return &MultiParser{
		all: []Parser{
			&PackageJSONParser{},
			&GoModParser{},
			&PomParser{},
			&GradleGroovyParser{},
			&BuildGradleKtsParser{},
			&GemfileParser{},
			&CargoTomlParser{},
			&RequirementsParser{},
			&PyProjectParser{},
			&PnpmLockParser{},
			&ComposerJSONParser{},
			&YarnLockParser{},
		},
	}
}

func (m *MultiParser) ParseFiles(files map[string][]byte) ([]Dependency, error) {
	var result []Dependency
	var usedPaths = make(map[string]bool)

	for _, parser := range m.all {
		supportedPaths := make(map[string][]byte)

		// Filtrer filer som parseren sier den kan parse
		for path, content := range files {
			if parser.CanParse(path) {
				supportedPaths[path] = content
			}
		}

		// Hopp over hvis ingen filer matcher denne parseren
		if len(supportedPaths) == 0 {
			continue
		}

		// Bruk ParseRepo hvis implementert
		if deps, err := parser.ParseRepo(supportedPaths); err == nil && deps != nil {
			// slog.Info("✅ ParseRepo OK", "parser", fmt.Sprintf("%T", parser), "filer", len(supportedPaths), "funnet", len(deps))
			result = append(result, deps...)
			// Merk alle paths som brukt
			for path := range supportedPaths {
				usedPaths[path] = true
			}
			continue
		}

		// Fallback: parse filer én og én med ParseFile
		for path, content := range supportedPaths {
			deps, err := parser.ParseFile(path, content)
			if err != nil {
				// slog.Warn("❗️ParseFile feilet", "parser", fmt.Sprintf("%T", parser), "fil", path, "error", err)
				continue
			}
			// slog.Info("✅ ParseFile OK", "parser", fmt.Sprintf("%T", parser), "fil", path, "funnet", len(deps))
			result = append(result, deps...)
			usedPaths[path] = true
		}
	}

	// Logg filer som ikke ble håndtert av noen parser
	for path := range files {
		if !usedPaths[path] {
			// slog.Debug("📂 Umatchet fil", "path", path)
		}
	}

	return result, nil
}
