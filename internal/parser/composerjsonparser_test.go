package parser

import (
	"testing"
)

func TestParseComposerFile(t *testing.T) {
	input := []byte(`{
		"require": {
			"php": "^7.4 || ^8.0"
		},
		"require-dev": {
			"friendsofphp/php-cs-fixer": "^3.0"
		}
	}`)

	expected := []Dependency{
		{Name: "php", Group: "", Version: "^7.4 || ^8.0", Type: "composer", Path: "composer.json"},
		{Name: "php-cs-fixer", Group: "friendsofphp", Version: "^3.0", Type: "composer", Path: "composer.json"},
	}

	deps, err := ParseSingleComposerFile("composer.json", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != len(expected) {
		t.Fatalf("expected %d dependencies, got %d", len(expected), len(deps))
	}

	for i := range deps {
		if deps[i] != expected[i] {
			t.Errorf("mismatch at index %d:\ngot:      %+v\nexpected: %+v", i, deps[i], expected[i])
		}
	}
}
