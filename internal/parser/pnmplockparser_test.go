package parser

import (
	"testing"
)

func TestParsePnpmLock(t *testing.T) {
	input := []byte(`
lockfileVersion: 5.4

importers:
  .:
    dependencies:
      '@org/round-icons':
        specifier: ^7.22.0
        version: 7.22.0
      react:
        specifier: ^19.1.0
        version: 19.1.0
`)

	expected := []Dependency{
		{Group: "@org", Name: "round-icons", Version: "7.22.0", Type: "pnpm", Path: "pnpm-lock.yaml"},
		{Group: "", Name: "react", Version: "19.1.0", Type: "pnpm", Path: "pnpm-lock.yaml"},
	}

	deps, err := PnpmLockParser{}.ParseFile("pnpm-lock.yaml", input)
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
