package parser

import (
	"testing"
)

func TestParseCargoToml(t *testing.T) {
	input := []byte(`
[dependencies]
serde = "1.0"
tokio = { version = "1.5", features = ["full"] }

[dev-dependencies]
rand = "0.8"

[build-dependencies]
cc = { version = "1.0" }
`)

	expected := []Dependency{
		{Name: "serde", Version: "1.0", Type: "cargo", Path: "Cargo.toml"},
		{Name: "tokio", Version: "1.5", Type: "cargo", Path: "Cargo.toml"},
		{Name: "rand", Version: "0.8", Type: "cargo", Path: "Cargo.toml"},
		{Name: "cc", Version: "1.0", Type: "cargo", Path: "Cargo.toml"},
	}

	deps, err := CargoTomlParser{}.ParseFile("Cargo.toml", input)
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
