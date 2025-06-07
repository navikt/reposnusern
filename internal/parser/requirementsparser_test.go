package parser

import "testing"

func TestParseRequirements(t *testing.T) {
	input := []byte(`
# Kommentar
requests==2.31.0
flask
    `)

	deps, err := parseRequirements("reqs.txt", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 deps, got %d", len(deps))
	}

	if deps[0].Name != "requests" || deps[0].Version != "2.31.0" {
		t.Errorf("unexpected first dep: %+v", deps[0])
	}

	if deps[1].Name != "flask" || deps[1].Version != "unparsed-version" {
		t.Errorf("unexpected second dep: %+v", deps[1])
	}
}
