package parser

import (
	"testing"
)

func TestParseGoMod(t *testing.T) {
	data := `
module example.com/my/module

go 1.20

require (
	github.com/sirupsen/logrus v1.9.0
	golang.org/x/net v0.17.0
	github.com/stretchr/testify v1.8.4
)
`
	deps, err := parseGoMod("go.mod", []byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []Dependency{
		{Group: "github.com/sirupsen", Name: "logrus", Version: "v1.9.0", Type: "go", Path: "go.mod"},
		{Group: "golang.org/x", Name: "net", Version: "v0.17.0", Type: "go", Path: "go.mod"},
		{Group: "github.com/stretchr", Name: "testify", Version: "v1.8.4", Type: "go", Path: "go.mod"},
	}

	if len(deps) != len(expected) {
		t.Fatalf("expected %d dependencies, got %d", len(expected), len(deps))
	}

	for i, dep := range deps {
		if dep != expected[i] {
			t.Errorf("dependency %d: got %+v, expected %+v", i, dep, expected[i])
		}
	}
}
