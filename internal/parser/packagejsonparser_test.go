package parser

import "testing"

func TestParsePackageJSON(t *testing.T) {
	jsonData := []byte(`{
		"dependencies": {
			"express": "^4.18.2",
			"@org/foobar": "^1.2.3"
		},
		"devDependencies": {
			"jest": "^29.5.0"
		}
	}`)

	deps, err := ParsePackageJSON("package.json", jsonData)
	if err != nil {
		t.Fatalf("ParsePackageJSON failed: %v", err)
	}

	expected := map[string]struct {
		group   string
		name    string
		version string
	}{
		"express":     {"", "express", "^4.18.2"},
		"jest":        {"", "jest", "^29.5.0"},
		"@org/foobar": {"@org", "foobar", "^1.2.3"},
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	for _, dep := range deps {
		full := dep.Name
		if dep.Group != "" {
			full = dep.Group + "/" + dep.Name
		}
		want, ok := expected[full]
		if !ok {
			t.Errorf("unexpected dependency: %s", full)
			continue
		}
		if dep.Version != want.version {
			t.Errorf("dependency %s: expected version %s, got %s", full, want.version, dep.Version)
		}
		if dep.Group != want.group {
			t.Errorf("dependency %s: expected group %s, got %s", full, want.group, dep.Group)
		}
	}
}

func TestParsePackageLockJSON(t *testing.T) {
	jsonData := []byte(`{
		"packages": {
			"": {},
			"node_modules/express": {
				"version": "4.18.2"
			},
			"node_modules/@org/foobar": {
				"version": "1.2.3"
			}
		}
	}`)

	deps, err := ParsePackageLockJSON("package-lock.json", jsonData)
	if err != nil {
		t.Fatalf("ParsePackageLockJSON failed: %v", err)
	}

	expected := map[string]struct {
		group   string
		name    string
		version string
	}{
		"express":     {"", "express", "4.18.2"},
		"@org/foobar": {"@org", "foobar", "1.2.3"},
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	for _, dep := range deps {
		full := dep.Name
		if dep.Group != "" {
			full = dep.Group + "/" + dep.Name
		}
		want, ok := expected[full]
		if !ok {
			t.Errorf("unexpected dependency: %s", full)
			continue
		}
		if dep.Version != want.version {
			t.Errorf("dependency %s: expected version %s, got %s", full, want.version, dep.Version)
		}
		if dep.Group != want.group {
			t.Errorf("dependency %s: expected group %s, got %s", full, want.group, dep.Group)
		}
	}
}

func TestParseRepoPJFiles(t *testing.T) {
	files := map[string][]byte{
		"package.json": []byte(`{
			"dependencies": {
				"express": "^4.18.2"
			},
			"devDependencies": {
				"jest": "^29.5.0"
			}
		}`),
		"package-lock.json": []byte(`{
			"packages": {
				"node_modules/express": {
					"version": "4.18.2"
				},
				"node_modules/jest": {
					"version": "29.5.0"
				}
			}
		}`),
		"README.md": []byte(`# should be ignored`),
	}

	deps, err := ParseRepoPJFiles(files)
	if err != nil {
		t.Fatalf("ParseRepoPJFiles failed: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	for _, dep := range deps {
		if dep.Path == "" {
			t.Errorf("dependency %s missing Path", dep.Name)
		}
		if dep.Type != "npm" {
			t.Errorf("dependency %s missing Type=npm", dep.Name)
		}
	}
}
