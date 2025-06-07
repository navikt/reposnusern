package parser

import (
	"testing"
)

func TestParsePyProjectPEP621(t *testing.T) {
	input := []byte(`
[project]
dependencies = [
  "requests >=2.24.0",
  "numpy ==1.18.5"
]
`)

	expected := []Dependency{
		{Name: "requests", Version: ">=2.24.0", Type: "pyproject", Path: "pyproject.toml"},
		{Name: "numpy", Version: "==1.18.5", Type: "pyproject", Path: "pyproject.toml"},
	}

	deps, err := PyProjectParser{}.ParseFile("pyproject.toml", input)
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

func TestParsePyProjectPoetry(t *testing.T) {
	input := []byte(`
[tool.poetry.dependencies]
python = "^3.10"
requests = "^2.24.0"
numpy = "^1.18.5"
`)

	expected := []Dependency{
		{Name: "requests", Version: "^2.24.0", Type: "pyproject", Path: "pyproject.toml"},
		{Name: "numpy", Version: "^1.18.5", Type: "pyproject", Path: "pyproject.toml"},
	}

	deps, err := PyProjectParser{}.ParseFile("pyproject.toml", input)
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
