package parser

import (
	"testing"
)

func TestParseGemfile(t *testing.T) {
	input := []byte(`
source 'https://rubygems.org'

gem 'rails', '6.1.4'
gem 'pg'
gem 'puma', '~> 5.0'
`)

	expected := []Dependency{
		{Name: "rails", Version: "6.1.4", Type: "gem", Path: "Gemfile"},
		{Name: "pg", Version: "", Type: "gem", Path: "Gemfile"},
		{Name: "puma", Version: "~> 5.0", Type: "gem", Path: "Gemfile"},
	}

	deps, err := GemfileParser{}.ParseFile("Gemfile", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != len(expected) {
		t.Fatalf("expected %d deps, got %d", len(expected), len(deps))
	}

	for i := range deps {
		if deps[i] != expected[i] {
			t.Errorf("mismatch at %d: got %+v, want %+v", i, deps[i], expected[i])
		}
	}
}
